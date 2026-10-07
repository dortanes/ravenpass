import assert from "node:assert/strict";
import test from "node:test";
import type { CredentialMenuContent } from "../messages.ts";
import type { TabMessenger } from "./menus.ts";
import { MenuSessions, type PageFrame } from "./sessions.ts";
import { SignInCards } from "./sign-in-cards.ts";
import { Clock, extensionId, MemoryArea } from "./test-doubles.test-support.ts";

const tabId = 7;

const accounts: CredentialMenuContent = {
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

function frame(documentId: string): PageFrame {
  return { tabId, documentId, origin: "https://github.com" };
}

type Relayed = [Parameters<TabMessenger>[0], Parameters<TabMessenger>[1]];

/** The top frame shows every card unless `refuses`; `present` answers form presence by document. */
function signInCards({
  refuses = false,
  present = new Map<string, boolean>(),
} = {}) {
  const area = new MemoryArea();
  let issued = 0;
  const sessions = new MenuSessions({
    area,
    extensionId,
    now: new Clock().now,
    newToken: () => `token-${++issued}`,
  });
  const relayed: Relayed[] = [];
  const cards = new SignInCards({
    sessions,
    relay: async (target, message) => {
      relayed.push([target, message]);
      if (message.kind === "card-show") return { shown: !refuses };
      if (message.kind === "form-present" && "documentId" in target) {
        return present.has(target.documentId)
          ? { present: present.get(target.documentId) }
          : undefined;
      }
      return undefined;
    },
  });
  return { sessions, relayed, cards };
}

test("a frame's form shows the card in the tab's top frame", async () => {
  const { sessions, relayed, cards } = signInCards();

  await cards.show(frame("login-frame"), "sign-in", accounts, false);

  assert.deepEqual(relayed, [
    [
      { tabId, frameId: 0 },
      { kind: "card-show", token: "token-1", reopen: false },
    ],
  ]);
  const card = await sessions.cardOf(tabId);
  assert.deepEqual(card?.page, frame("login-frame"));
  assert.deepEqual(card?.content, {
    state: "sign-in-card",
    purpose: "sign-in",
    listing: accounts,
    requested: false,
  });
});

test("the first frame keeps the card while its form is on its page", async () => {
  const { sessions, relayed, cards } = signInCards({
    present: new Map([["first-frame", true]]),
  });
  await cards.show(frame("first-frame"), "sign-in", accounts, false);

  await cards.show(frame("second-frame"), "sign-in", accounts, false);

  assert.equal((await sessions.cardOf(tabId))?.page.documentId, "first-frame");
  assert.deepEqual(relayed.at(-1), [
    frame("first-frame"),
    { kind: "form-present" },
  ]);
});

test("a later frame takes the card once the first frame's form left its page", async () => {
  for (const present of [
    new Map([["first-frame", false]]),
    new Map<string, boolean>(),
  ]) {
    const { sessions, relayed, cards } = signInCards({ present });
    await cards.show(frame("first-frame"), "sign-in", accounts, false);

    await cards.show(frame("second-frame"), "sign-in", accounts, false);

    assert.equal(
      (await sessions.cardOf(tabId))?.page.documentId,
      "second-frame",
    );
    assert.deepEqual(relayed.slice(-2), [
      [
        { tabId, frameId: 0 },
        { kind: "card-hide", token: "token-1", dismissed: false },
      ],
      [
        { tabId, frameId: 0 },
        { kind: "card-show", token: "token-2", reopen: false },
      ],
    ]);
  }
});

test("a focused field takes the card for its frame at once and reopens it", async () => {
  const { sessions, relayed, cards } = signInCards({
    present: new Map([["first-frame", true]]),
  });
  await cards.show(frame("first-frame"), "sign-in", accounts, false);

  await cards.show(frame("second-frame"), "sign-in", accounts, true);

  assert.equal((await sessions.cardOf(tabId))?.page.documentId, "second-frame");
  assert.deepEqual(relayed.at(-1), [
    { tabId, frameId: 0 },
    { kind: "card-show", token: "token-2", reopen: true },
  ]);
});

test("the card already shown for a frame's form stays, requested once a field asks for it", async () => {
  const { sessions, relayed, cards } = signInCards();
  await cards.show(frame("login-frame"), "sign-in", accounts, false);

  await cards.show(frame("login-frame"), "sign-in", accounts, true);

  assert.equal(relayed.length, 1);
  const card = await sessions.cardOf(tabId);
  assert.equal(card?.token, "token-1");
  assert.equal(
    card?.content.state === "sign-in-card" && card.content.requested,
    true,
  );
});

test("a code field of the same frame replaces its sign-in card", async () => {
  const { sessions, cards } = signInCards();
  await cards.show(frame("login-frame"), "sign-in", accounts, false);

  const listing = { state: "locked", purpose: "code" } as const;
  await cards.show(frame("login-frame"), "code", listing, false);

  const card = await sessions.cardOf(tabId);
  assert.equal(card?.token, "token-2");
  assert.deepEqual(card?.content, {
    state: "sign-in-card",
    purpose: "code",
    listing,
    requested: false,
  });
});

test("a card the top frame refuses, as one the person closed, ends", async () => {
  const { sessions, cards } = signInCards({ refuses: true });

  await cards.show(frame("login-frame"), "sign-in", accounts, false);

  assert.equal(await sessions.cardOf(tabId), null);
});

test("hiding a card tells the top frame and ends its token", async () => {
  const { sessions, relayed, cards } = signInCards();
  await cards.show(frame("login-frame"), "sign-in", accounts, false);
  const card = await sessions.cardOf(tabId);
  assert.ok(card);

  await cards.hide(card, true);

  assert.deepEqual(relayed.at(-1), [
    { tabId, frameId: 0 },
    { kind: "card-hide", token: "token-1", dismissed: true },
  ]);
  assert.equal(await sessions.cardOf(tabId), null);
});
