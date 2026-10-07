import assert from "node:assert/strict";
import test from "node:test";
import { MessageFormatter, type MessageValues } from "../i18n/format.ts";
import type { MessageKey } from "../i18n/messages.ts";
import {
  destinationsOf,
  destinationTitle,
  reviseChoice,
  type SaveDestinations,
  saveActionOf,
  saveResultOf,
  suggestedChoice,
  targetOf,
  targetTitle,
} from "./offer.ts";

const offer: SaveDestinations = {
  targets: [
    {
      credential: "a1",
      label: "GitHub",
      account: "alex",
      action: "update",
      tags: [],
    },
    {
      credential: "b2",
      label: "",
      account: "alex",
      action: "add-site",
      tags: [],
    },
  ],
  suggested: "a1",
};

const formatter = new MessageFormatter();
const t = (key: MessageKey, values?: MessageValues) =>
  formatter.format("en", key, values);

test("an offer lists its suggestion first, then a new credential and the targets in order", () => {
  assert.deepEqual(destinationsOf(offer), ["a1", "", "b2"]);
  assert.deepEqual(destinationsOf({ ...offer, suggested: "b2" }), [
    "b2",
    "",
    "a1",
  ]);
  assert.deepEqual(destinationsOf({ ...offer, suggested: "" }), [
    "",
    "a1",
    "b2",
  ]);
});

test("a destination is named for what saving there does", () => {
  assert.equal(destinationTitle(offer, "", t), "New item");
  assert.equal(destinationTitle(offer, "a1", t), "Update “GitHub”");
  assert.equal(
    destinationTitle(offer, "b2", t),
    "Add site to “Untitled password”",
  );
  assert.equal(saveActionOf(null), "saving.save");
  assert.equal(saveActionOf(targetOf(offer, "a1")), "saving.update");
  assert.equal(saveActionOf(targetOf(offer, "b2")), "saving.add-site");
  assert.equal(targetOf(offer, "c3"), null);
});

test("a passkey destination is named for adding the passkey", () => {
  assert.equal(targetTitle(null, t), "New item");
  assert.equal(
    targetTitle({ label: "GitHub", action: "add-passkey" }, t),
    "Add to “GitHub”",
  );
  assert.equal(
    targetTitle({ label: "", action: "add-passkey" }, t),
    "Add to “Untitled password”",
  );
});

test("a save reports what it did", () => {
  const choice = { target: "b2", account: "alex", name: "" };
  assert.equal(saveResultOf(true, offer, { ...choice, target: "" }), "created");
  assert.equal(saveResultOf(false, offer, choice), "site-added");
  assert.equal(
    saveResultOf(false, offer, { ...choice, target: "a1" }),
    "updated",
  );
});

test("an offer starts on its suggestion with the submitted account and the suggested name", () => {
  assert.deepEqual(
    suggestedChoice({ ...offer, account: "alex", name: "github.com" }),
    { target: "a1", account: "alex", name: "github.com" },
  );
});

test("a refusal stands until its field or the target changes", () => {
  const choice = { target: "", account: "alex", name: "" };
  assert.deepEqual(reviseChoice(choice, "name", { account: "sam" }), {
    choice: { target: "", account: "sam", name: "" },
    refused: "name",
  });
  assert.equal(reviseChoice(choice, "name", { name: "Mail" }).refused, null);
  assert.equal(reviseChoice(choice, "account", { target: "a1" }).refused, null);
  assert.equal(reviseChoice(choice, null, { name: "Mail" }).refused, null);
});
