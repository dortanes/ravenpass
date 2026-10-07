import assert from "node:assert/strict";
import test from "node:test";
import type { MenuContent } from "../messages.ts";
import { MenuSessions, type PageFrame, type Sender } from "./sessions.ts";
import { Clock, MemoryArea } from "./test-doubles.test-support.ts";

const extensionId = "cceiadaelnccfbakmhcleifjfilkakag";
const tabId = 7;
const origin = "https://github.com";
const content: MenuContent = {
  state: "list",
  purpose: "sign-in",
  credentials: [
    {
      id: "a1",
      label: "GitHub",
      account: "alex",
      site: "github.com",
      exact: true,
      tags: [],
      strength: "strong",
    },
  ],
  passkeys: [],
};

function sessions() {
  const area = new MemoryArea();
  const clock = new Clock();
  let issued = 0;
  const menus = new MenuSessions({
    area,
    extensionId,
    now: clock.now,
    newToken: () => `token-${++issued}`,
  });
  return { area, clock, menus };
}

function pageSender(overrides: Partial<Sender> = {}): Sender {
  return {
    id: extensionId,
    url: `${origin}/login`,
    origin,
    tab: { id: tabId },
    frameId: 0,
    documentId: "page-document",
    ...overrides,
  };
}

function menuSender(overrides: Partial<Sender> = {}): Sender {
  return {
    id: extensionId,
    url: "chrome-extension://4f0e7c1a-dynamic/pages/menu.html#token-1",
    origin: "chrome-extension://4f0e7c1a-dynamic",
    tab: { id: tabId },
    frameId: 3,
    documentId: "menu-document",
    ...overrides,
  };
}

function page(): PageFrame {
  return { tabId, documentId: "page-document", origin };
}

test("a content script in a web page's frame may open a menu", () => {
  const { menus } = sessions();
  assert.deepEqual(menus.pageOf(pageSender()), page());
  assert.deepEqual(
    menus.pageOf(
      pageSender({ url: "about:blank", origin: "http://intranet.example" }),
    ),
    { tabId, documentId: "page-document", origin: "http://intranet.example" },
  );
});

test("anything but a web page's frame opens nothing", () => {
  const { menus } = sessions();
  for (const sender of [
    pageSender({ id: "another-extension" }),
    pageSender({ tab: undefined }),
    pageSender({ tab: {} }),
    pageSender({ frameId: undefined }),
    pageSender({ documentId: undefined }),
    pageSender({ origin: undefined }),
    pageSender({ origin: "null" }),
    pageSender({ origin: "file://" }),
    pageSender({ origin: "chrome-extension://4f0e7c1a-dynamic" }),
    menuSender(),
  ]) {
    assert.equal(menus.pageOf(sender), null);
  }
});

test("a menu page binds its token and keeps it", async () => {
  const { menus } = sessions();
  const token = await menus.open(page(), content);

  const bound = await menus.bind(token, menuSender());

  assert.equal(bound?.token, token);
  assert.equal(bound?.menu, "menu-document");
  assert.deepEqual(bound?.content, content);
  assert.deepEqual(bound?.page, page());
  assert.equal((await menus.bind(token, menuSender()))?.menu, "menu-document");
  assert.equal(
    await menus.bind(token, menuSender({ documentId: "other-menu" })),
    null,
  );
});

test("only this extension's menu page in the token's tab binds it", async () => {
  const { menus } = sessions();
  const token = await menus.open(page(), content);

  for (const sender of [
    pageSender(),
    menuSender({ id: "another-extension" }),
    menuSender({ tab: { id: tabId + 1 } }),
    menuSender({ tab: undefined }),
    menuSender({ documentId: undefined }),
    menuSender({ url: "chrome-extension://4f0e7c1a-dynamic/popup/index.html" }),
    menuSender({ url: "https://github.com/pages/menu.html" }),
    menuSender({ url: undefined }),
  ]) {
    assert.equal(await menus.bind(token, sender), null);
  }
  assert.equal(await menus.bind("token-unknown", menuSender()), null);
  assert.equal((await menus.bind(token, menuSender()))?.menu, "menu-document");
});

test("menu requests are served only for the bound menu page", async () => {
  const { menus } = sessions();
  const token = await menus.open(page(), content);
  assert.equal(await menus.forMenu(token, menuSender()), null);

  await menus.bind(token, menuSender());

  assert.equal((await menus.forMenu(token, menuSender()))?.token, token);
  assert.equal(
    await menus.forMenu(token, menuSender({ documentId: "other-menu" })),
    null,
  );
  assert.equal(await menus.forMenu(token, pageSender()), null);
});

test("frame requests are served only for the frame the menu opened for", async () => {
  const { menus } = sessions();
  const token = await menus.open(page(), content);

  assert.equal((await menus.forPage(token, pageSender()))?.token, token);
  for (const sender of [
    pageSender({ documentId: "document-after-navigation" }),
    pageSender({ tab: { id: tabId + 1 } }),
    pageSender({ origin: "https://example.com" }),
    menuSender(),
  ]) {
    assert.equal(await menus.forPage(token, sender), null);
  }
});

test("a token expires ten minutes after it was issued", async () => {
  const { area, clock, menus } = sessions();
  const token = await menus.open(page(), content);
  await menus.bind(token, menuSender());

  clock.time += 10 * 60_000 - 1;
  assert.equal((await menus.forMenu(token, menuSender()))?.token, token);
  clock.time += 1;
  assert.equal(await menus.forMenu(token, menuSender()), null);
  assert.equal(area.items.size, 0);
});

test("opening a menu sweeps the sessions that expired unasked", async () => {
  const { area, clock, menus } = sessions();
  await menus.open(page(), content);
  clock.time += 10 * 60_000;

  const token = await menus.open(page(), {
    state: "locked",
    purpose: "sign-in",
  });

  assert.deepEqual([...area.items.keys()], [`menu:${token}`]);
});

test("a closed menu's token ends", async () => {
  const { menus } = sessions();
  const token = await menus.open(page(), content);
  await menus.bind(token, menuSender());

  await menus.end(token);

  assert.equal(await menus.forMenu(token, menuSender()), null);
  assert.equal(await menus.forPage(token, pageSender()), null);
});

test("closing a tab ends its menus and only its menus", async () => {
  const { menus } = sessions();
  const closing = await menus.open(page(), content);
  const other = await menus.open(
    { tabId: tabId + 1, documentId: "other-page", origin },
    content,
  );

  await menus.endTab(tabId);

  assert.equal(await menus.forPage(closing, pageSender()), null);
  assert.equal(
    (
      await menus.forPage(
        other,
        pageSender({ tab: { id: tabId + 1 }, documentId: "other-page" }),
      )
    )?.token,
    other,
  );
});
