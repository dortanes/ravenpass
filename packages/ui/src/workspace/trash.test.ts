import assert from "node:assert/strict";
import test from "node:test";
import { daysLeft, dueAt, retentionOptions } from "./trash.ts";

const day = 24 * 60 * 60 * 1000;
const deletedAt = Date.UTC(2026, 9, 1, 12);

test("an item has the whole period left when just deleted and none on its last day", () => {
  assert.equal(daysLeft(deletedAt, 30, deletedAt), 30);
  assert.equal(daysLeft(deletedAt, 30, deletedAt + day - 1), 29);
  assert.equal(daysLeft(deletedAt, 30, deletedAt + 29 * day + 1), 0);
  assert.equal(daysLeft(deletedAt, 30, deletedAt + 40 * day), 0);
});

test("a shorter period counts the items it would delete at once", () => {
  const now = deletedAt + 10 * day;
  const deleted = [deletedAt, deletedAt + 5 * day, now];
  assert.equal(dueAt(deleted, 90, now), 0);
  assert.equal(dueAt(deleted, 7, now), 1);
  assert.equal(dueAt(deleted, 5, now), 2);
});

test("a period another device set is offered beside the usual ones", () => {
  assert.deepEqual(retentionOptions(30), [7, 30, 90, 365]);
  assert.deepEqual(retentionOptions(14), [7, 14, 30, 90, 365]);
  assert.deepEqual(retentionOptions(3650), [7, 30, 90, 365, 3650]);
});
