import assert from "node:assert/strict";
import test from "node:test";
import type { PendingCapture } from "../link/client.ts";
import { MenuSessions, type PageFrame } from "./sessions.ts";
import { TabOffers } from "./tab-offers.ts";
import {
  Clock,
  extensionId,
  MemoryArea,
  menuSender,
  pageSender,
} from "./test-doubles.test-support.ts";

const tabId = 7;
const lifetimeMs = 3 * 60_000;

/** The document of the sign-in form a capture came from. */
const loginPage = { tabId, documentId: "login-document" };

const capture: PendingCapture = {
  kind: "password",
  state: "ready",
  pending: "p1",
  site: "github.com",
  account: "alex",
  name: "github.com",
  card: null,
  targets: [
    {
      credential: "a1",
      label: "GitHub",
      account: "alex",
      action: "update",
      tags: [],
    },
  ],
  suggested: "a1",
};

function tabOffers() {
  const area = new MemoryArea();
  const clock = new Clock();
  let issued = 0;
  const sessions = new MenuSessions({
    area,
    extensionId,
    now: clock.now,
    newToken: () => `token-${++issued}`,
  });
  const offers = new TabOffers({ area, sessions, now: clock.now });
  /** Keeps the offer for a capture and marks it ready, as once the sign-in looks done. */
  const keepReady = async (kept: PendingCapture = capture) => {
    const earlier = await offers.replace(loginPage, kept);
    await offers.markReady(tabId, kept.pending);
    return earlier;
  };
  return { area, clock, sessions, offers, keepReady };
}

function page(documentId = "page-document", tab = tabId): PageFrame {
  return { tabId: tab, documentId, origin: "https://github.com" };
}

test("a new offer waits, remembering the document it was captured in", async () => {
  const { offers } = tabOffers();

  await offers.replace(loginPage, capture);

  const offer = await offers.current(tabId);
  assert.equal(offer?.ready, false);
  assert.equal(offer?.capturedIn, "login-document");
  assert.equal(await offers.issue(page()), null);
});

test("a ready offer reaches its menu without the pending id", async () => {
  const { clock, sessions, offers, keepReady } = tabOffers();
  await keepReady();

  const token = await offers.issue(page());

  assert.equal(token, "token-1");
  const { pending: _, ...shown } = capture;
  assert.deepEqual((await sessions.bind("token-1", menuSender()))?.content, {
    state: "offer",
    offer: { ...shown, expiresAt: clock.time + lifetimeMs },
  });
});

test("only the offer for its own capture is marked ready", async () => {
  const { offers } = tabOffers();
  await offers.replace(loginPage, capture);

  assert.equal(await offers.markReady(tabId, "p9"), null);
  assert.equal((await offers.current(tabId))?.ready, false);
  assert.equal((await offers.markReady(tabId, "p1"))?.ready, true);
});

test("a tab's offer lasts three minutes, as Ravenpass holds its capture", async () => {
  const { area, clock, offers, keepReady } = tabOffers();
  await keepReady();

  clock.time += lifetimeMs - 1;
  assert.equal(await offers.issue(page()), "token-1");
  clock.time += 1;
  assert.equal(await offers.issue(page("next-document")), null);
  assert.equal(area.items.has(`offer:${tabId}`), false);
});

test("a newer capture replaces the tab's offer, waiting, and answers the older one", async () => {
  const { offers, keepReady } = tabOffers();
  await keepReady();
  await offers.issue(page());

  const earlier = await offers.replace(loginPage, {
    ...capture,
    pending: "p2",
  });

  assert.equal(earlier?.pending, "p1");
  assert.deepEqual(earlier?.shown, {
    token: "token-1",
    documentId: "page-document",
  });
  const offer = await offers.current(tabId);
  assert.equal(offer?.pending, "p2");
  assert.equal(offer?.ready, false);
});

test("a capture with nothing to offer leaves the tab without an offer", async () => {
  const { offers, keepReady } = tabOffers();
  await keepReady();

  assert.equal((await offers.replace(loginPage, null))?.pending, "p1");
  assert.equal(await offers.issue(page()), null);
});

test("each top-frame document gets a token of its own, which ends the one before", async () => {
  const { sessions, offers, keepReady } = tabOffers();
  await keepReady();

  const first = await offers.issue(page("first-document"));
  const second = await offers.issue(page("second-document"));

  assert.equal(first, "token-1");
  assert.equal(second, "token-2");
  assert.equal(
    await sessions.forPage(
      "token-1",
      pageSender({ documentId: "first-document" }),
    ),
    null,
  );
  assert.equal(
    (
      await sessions.forPage(
        "token-2",
        pageSender({ documentId: "second-document" }),
      )
    )?.token,
    "token-2",
  );
  assert.equal(await offers.shownBy(tabId, "token-1"), null);
  assert.equal((await offers.shownBy(tabId, "token-2"))?.pending, "p1");
});

test("a tab without an offer, or another tab, gets no token", async () => {
  const { offers, keepReady } = tabOffers();
  assert.equal(await offers.issue(page()), null);

  await keepReady();

  assert.equal(await offers.issue(page("page-document", tabId + 1)), null);
});

test("a review revises the offer for its own capture only, keeping it ready", async () => {
  const { offers, keepReady } = tabOffers();
  await keepReady({ ...capture, state: "locked", targets: [] });

  assert.equal((await offers.revise(tabId, capture))?.state, "ready");
  assert.equal(
    await offers.revise(tabId, { ...capture, pending: "p9", state: "locked" }),
    null,
  );
  const kept = await offers.take(tabId);
  assert.equal(kept?.state, "ready");
  assert.equal(kept?.ready, true);
  assert.deepEqual(kept?.targets, capture.targets);
});

test("forgetting the offer for a capture leaves a newer capture's offer", async () => {
  const { offers } = tabOffers();
  await offers.replace(loginPage, capture);
  await offers.replace(loginPage, { ...capture, pending: "p2" });

  assert.equal(await offers.take(tabId, "p1"), null);
  assert.equal((await offers.take(tabId, "p2"))?.pending, "p2");
  assert.equal(await offers.take(tabId), null);
});
