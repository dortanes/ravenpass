import assert from "node:assert/strict";
import test from "node:test";
import type { SaveChoice } from "../saving/offer.ts";
import type { SaveOutcome, SaveReviewOffer } from "./autofill-api.ts";
import { SaveReview } from "./save-review.ts";

const offer: SaveReviewOffer = {
  site: "example.com",
  account: "alex",
  name: "example.com",
  targets: [
    {
      credential: "a1",
      label: "Example",
      account: "alex",
      action: "update",
      tags: [],
    },
  ],
  suggested: "",
};

function reviewAnswering(...outcomes: SaveOutcome[]) {
  const saved: SaveChoice[] = [];
  const review = new SaveReview(offer, async (choice) => {
    saved.push(choice);
    return outcomes.shift() ?? { kind: "failed" };
  });
  return { review, saved };
}

test("a review starts on the suggestion and saves where the owner chose", async () => {
  const { review, saved } = reviewAnswering({ kind: "saved", created: false });
  assert.deepEqual(review.state.get().choice, {
    target: "",
    account: "alex",
    name: "example.com",
  });
  review.edit({ target: "a1" });
  await review.save();
  assert.deepEqual(saved, [
    { target: "a1", account: "alex", name: "example.com" },
  ]);
  assert.equal(review.state.get().saving, true);
});

test("a refused name stays marked until it changes", async () => {
  const { review } = reviewAnswering({ kind: "name-refused" });
  await review.save();
  assert.equal(review.state.get().refused, "name");
  assert.equal(review.state.get().saving, false);
  review.edit({ account: "sam" });
  assert.equal(review.state.get().refused, "name");
  review.edit({ name: "Example" });
  assert.equal(review.state.get().refused, null);
});

test("a failed save is noted and can be tried again", async () => {
  const { review, saved } = reviewAnswering(
    { kind: "failed" },
    { kind: "saved", created: true },
  );
  await review.save();
  assert.equal(review.state.get().failed, true);
  await review.save();
  assert.equal(saved.length, 2);
  assert.equal(review.state.get().failed, false);
});
