import assert from "node:assert/strict";
import test from "node:test";
import { suggestedTags, withTag } from "./tags.ts";

const limits = { tag: 8, tags: 2 };

test("a typed tag is tidied and added once, within the limits", () => {
  assert.deepEqual(withTag([], "  Work  2 ", limits), ["Work 2"]);
  assert.deepEqual(withTag(["Work"], "work", limits), ["Work"]);
  assert.deepEqual(withTag(["Work"], " ", limits), ["Work"]);
  assert.deepEqual(withTag(["Work"], "far too long", limits), ["Work"]);
  assert.deepEqual(withTag([], "Work\u0007", limits), []);
  assert.deepEqual(withTag([], "😀😀😀😀😀😀😀😀", limits), [
    "😀😀😀😀😀😀😀😀",
  ]);
  assert.deepEqual(withTag(["Work", "Personal"], "Shared", limits), [
    "Work",
    "Personal",
  ]);
  assert.deepEqual(withTag(["Work", "Personal"], "Shared", null), [
    "Work",
    "Personal",
    "Shared",
  ]);
});

test("known tags not chosen are suggested by what is typed, fewer before typing", () => {
  const known = ["Personal", "Pets", "Photos", "Work", "Wallet"];
  assert.deepEqual(suggestedTags(known, ["Pets"], "pe"), ["Personal"]);
  assert.deepEqual(suggestedTags(known, [], "pets"), []);
  assert.deepEqual(suggestedTags(known, [], ""), [
    "Personal",
    "Pets",
    "Photos",
  ]);
  assert.deepEqual(suggestedTags(known, ["personal"], "p"), ["Pets", "Photos"]);
});
