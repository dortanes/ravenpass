import assert from "node:assert/strict";
import test from "node:test";
import type { Suggestion } from "../link/client.ts";
import { RecentFills } from "./recent-fills.ts";
import { Clock, MemoryArea } from "./test-doubles.test-support.ts";

const tabId = 7;

function credential(id: string): Suggestion {
  return {
    id,
    label: id,
    account: "alex",
    site: "github.com",
    exact: true,
    tags: [],
  };
}

const listed = [credential("a1"), credential("b2"), credential("c3")];

function recentFills() {
  const area = new MemoryArea();
  const clock = new Clock();
  return { area, clock, fills: new RecentFills({ area, now: clock.now }) };
}

function ids(credentials: readonly Suggestion[]): string[] {
  return credentials.map(({ id }) => id);
}

test("the credential filled in the tab comes first, the others as they came", async () => {
  const { fills } = recentFills();

  await fills.remember(tabId, "c3");

  assert.deepEqual(ids(await fills.order(tabId, listed)), ["c3", "a1", "b2"]);
});

test("without a fill, or with one not listed, the order stays", async () => {
  const { fills } = recentFills();
  assert.deepEqual(ids(await fills.order(tabId, listed)), ["a1", "b2", "c3"]);

  await fills.remember(tabId, "d4");

  assert.deepEqual(ids(await fills.order(tabId, listed)), ["a1", "b2", "c3"]);
});

test("a fill counts only in its own tab", async () => {
  const { fills } = recentFills();

  await fills.remember(tabId + 1, "b2");

  assert.deepEqual(ids(await fills.order(tabId, listed)), ["a1", "b2", "c3"]);
});

test("the latest fill in a tab replaces the one before", async () => {
  const { fills } = recentFills();

  await fills.remember(tabId, "c3");
  await fills.remember(tabId, "b2");

  assert.deepEqual(ids(await fills.order(tabId, listed)), ["b2", "a1", "c3"]);
});

test("a fill counts for ten minutes", async () => {
  const { area, clock, fills } = recentFills();
  await fills.remember(tabId, "c3");

  clock.time += 10 * 60_000 - 1;
  assert.deepEqual(ids(await fills.order(tabId, listed)), ["c3", "a1", "b2"]);
  clock.time += 1;
  assert.deepEqual(ids(await fills.order(tabId, listed)), ["a1", "b2", "c3"]);
  assert.equal(area.items.size, 0);
});

test("closing a tab forgets its fill and only its fill", async () => {
  const { fills } = recentFills();
  await fills.remember(tabId, "c3");
  await fills.remember(tabId + 1, "b2");

  await fills.forget(tabId);

  assert.deepEqual(ids(await fills.order(tabId, listed)), ["a1", "b2", "c3"]);
  assert.deepEqual(ids(await fills.order(tabId + 1, listed)), [
    "b2",
    "a1",
    "c3",
  ]);
});
