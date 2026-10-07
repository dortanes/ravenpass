import assert from "node:assert/strict";
import test from "node:test";
import { breachedFirst } from "./breaches.ts";

test("breached passwords come first, each part in its own order", () => {
  const entries = ["a", "b", "c", "d"].map((id) => ({ id }));
  const counts = new Map([
    ["d", 3],
    ["b", 120],
  ]);
  assert.deepEqual(
    breachedFirst(entries, counts).map((entry) => entry.id),
    ["b", "d", "a", "c"],
  );
  assert.equal(breachedFirst(entries, new Map()), entries);
});
