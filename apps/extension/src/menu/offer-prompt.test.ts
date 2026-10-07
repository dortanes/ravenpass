import assert from "node:assert/strict";
import { afterEach, beforeEach, mock, test } from "node:test";
import type { SaveChoice } from "../link/client.ts";
import type { Answers, SaveOffer } from "../messages.ts";
import { OfferPrompt } from "./offer-prompt.ts";
import { FakeVault, settle } from "./vault-state.test-support.ts";

const now = 1_750_000_000_000;
const lifetimeMs = 3 * 60_000;

beforeEach(() => {
  mock.timers.enable({ apis: ["setTimeout", "Date"], now });
});

afterEach(() => {
  mock.timers.reset();
});

const ready: SaveOffer = {
  kind: "password",
  state: "ready",
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
    {
      credential: "b2",
      label: "GitLab",
      account: "alex",
      action: "add-site",
      tags: [],
    },
  ],
  suggested: "a1",
  expiresAt: now + lifetimeMs,
};

const locked: SaveOffer = {
  ...ready,
  state: "locked",
  targets: [],
  suggested: "",
};

/** Answers reviews and saves in turn from `script`, and records what the offer asked. */
function offerPrompt(
  offer: SaveOffer,
  script: {
    reviews?: Answers["offer-review"][];
    saves?: Answers["offer-save"][];
    unlock?: Answers["menu-unlock"];
    discard?: Error;
  } = {},
) {
  const asked: string[] = [];
  const choices: SaveChoice[] = [];
  const reviews = [...(script.reviews ?? [])];
  const saves = [...(script.saves ?? [])];
  const vault = new FakeVault();
  const prompt = new OfferPrompt(offer, {
    review: async () => {
      asked.push("review");
      const answer = reviews.shift();
      if (!answer) throw new Error("The script has no review left.");
      return answer;
    },
    save: async (choice) => {
      asked.push("save");
      choices.push(choice);
      const answer = saves.shift();
      if (!answer) throw new Error("The script has no save left.");
      return answer;
    },
    discard: async () => {
      asked.push("discard");
      if (script.discard) throw script.discard;
      return { ok: true };
    },
    unlock: async () => {
      asked.push("unlock");
      return script.unlock ?? { ok: true };
    },
    watchVault: vault.watch,
    close: () => {
      asked.push("close");
    },
  });
  prompt.start();
  return { prompt, asked, choices, vault };
}

test("an offer starts on the suggested target, with the suggested name and the captured account", () => {
  const { prompt } = offerPrompt(ready);

  assert.deepEqual(prompt.view(), {
    stage: "asking",
    offer: ready,
    choice: { target: "a1", account: "alex", name: "github.com" },
    saving: false,
    failed: false,
    refused: null,
    unlockFailed: false,
  });
});

test("an account or a name the vault refused is marked until it is changed", async () => {
  const { prompt, choices } = offerPrompt(ready, {
    saves: [
      { ok: false, reason: "invalid-account" },
      { ok: false, reason: "invalid-name" },
    ],
  });
  prompt.edit({ target: "" });

  await prompt.save();
  const account = prompt.view();
  assert.equal(account.stage === "asking" && account.refused, "account");
  prompt.edit({ name: "GitHub" });
  const kept = prompt.view();
  assert.equal(kept.stage === "asking" && kept.refused, "account");
  prompt.edit({ account: "alex@example.com" });
  const cleared = prompt.view();
  assert.equal(cleared.stage === "asking" && cleared.refused, null);

  await prompt.save();
  const name = prompt.view();
  assert.equal(name.stage === "asking" && name.refused, "name");
  assert.equal(name.stage === "asking" && name.saving, false);
  prompt.edit({ target: "a1" });
  const moved = prompt.view();
  assert.equal(moved.stage === "asking" && moved.refused, null);
  assert.equal(choices.length, 2);
});

test("an update says so and closes 1.6 seconds after the save", async () => {
  const { prompt, asked } = offerPrompt(ready, {
    saves: [{ ok: true, saved: "updated" }],
  });

  await prompt.save();

  assert.deepEqual(prompt.view(), {
    stage: "saved",
    offer: ready,
    choice: { target: "a1", account: "alex", name: "github.com" },
    result: "updated",
  });
  mock.timers.tick(1599);
  assert.equal(prompt.view().stage, "saved");
  mock.timers.tick(1);
  assert.equal(prompt.view().stage, "closed");
  assert.deepEqual(asked, ["save", "close"]);
});

test("a new credential takes the name and the account the person entered", async () => {
  const { prompt, choices } = offerPrompt(ready, {
    saves: [{ ok: true, saved: "created" }],
  });

  prompt.edit({ target: "" });
  prompt.edit({ account: "alex@example.com", name: "GitHub" });
  await prompt.save();

  assert.deepEqual(choices, [
    { target: "", account: "alex@example.com", name: "GitHub" },
  ]);
  const view = prompt.view();
  assert.equal(view.stage === "saved" && view.result, "created");
});

test("adding the site to a credential says the site was added", async () => {
  const { prompt, choices } = offerPrompt(ready, {
    saves: [{ ok: true, saved: "updated" }],
  });

  prompt.edit({ target: "b2" });
  await prompt.save();

  assert.equal(choices[0]?.target, "b2");
  const view = prompt.view();
  assert.equal(view.stage === "saved" && view.result, "site-added");
});

test("a save Ravenpass could not serve can be tried again", async () => {
  const { prompt, asked } = offerPrompt(ready, {
    saves: [
      { ok: false, reason: "failed" },
      { ok: true, saved: "updated" },
    ],
  });

  await prompt.save();
  const failed = prompt.view();
  assert.equal(failed.stage === "asking" && failed.failed, true);
  assert.equal(failed.stage === "asking" && failed.saving, false);

  await prompt.save();
  assert.equal(prompt.view().stage, "saved");
  assert.deepEqual(asked, ["save", "save"]);
});

test("Not now discards the capture, then closes", async () => {
  const { prompt, asked } = offerPrompt(ready);

  await prompt.dismiss();

  assert.equal(prompt.view().stage, "closed");
  assert.deepEqual(asked, ["discard", "close"]);
});

test("Not now closes when the service worker is gone, and still closes when the discard fails", async (t) => {
  const errors = t.mock.method(console, "error", () => {});
  const gone = offerPrompt(ready, {
    discard: new Error(
      "Could not establish connection. Receiving end does not exist.",
    ),
  });
  await gone.prompt.dismiss();
  assert.equal(gone.prompt.view().stage, "closed");
  assert.equal(errors.mock.callCount(), 0);

  const refused = new Error("The service worker could not serve the request.");
  const failing = offerPrompt(ready, { discard: refused });
  await failing.prompt.dismiss();
  assert.equal(failing.prompt.view().stage, "closed");
  assert.deepEqual(failing.asked, ["discard", "close"]);
  assert.equal(errors.mock.callCount(), 1);
});

test("a locked Ravenpass is reviewed on each vault change until it is unlocked", async () => {
  const { prompt, asked, vault } = offerPrompt(locked, {
    reviews: [
      { ok: true, offer: locked },
      { ok: false, reason: "failed" },
      { ok: true, offer: ready },
    ],
  });

  vault.push("locked");
  await settle();
  assert.deepEqual(asked, []);
  for (let review = 1; review <= 3; review += 1) {
    vault.push("unlocked");
    await settle();
    assert.equal(asked.length, review);
  }

  assert.deepEqual(prompt.view(), {
    stage: "asking",
    offer: ready,
    choice: { target: "a1", account: "alex", name: "github.com" },
    saving: false,
    failed: false,
    refused: null,
    unlockFailed: false,
  });
  assert.equal(vault.watching(), false);
});

test("Unlock brings Ravenpass forward, and the vault change reviews the offer", async () => {
  const { prompt, asked, vault } = offerPrompt(locked, {
    reviews: [{ ok: true, offer: ready }],
  });

  await prompt.unlock();
  vault.push("unlocked");
  await settle();

  assert.deepEqual(asked, ["unlock", "review"]);
  const view = prompt.view();
  assert.equal(view.stage === "asking" && view.offer.state, "ready");
});

test("an unlock Ravenpass could not serve is noted until Unlock is tried again", async () => {
  const { prompt, asked } = offerPrompt(locked, {
    unlock: { ok: false, reason: "failed" },
  });

  await prompt.unlock();
  const failed = prompt.view();
  assert.equal(failed.stage === "asking" && failed.unlockFailed, true);
  assert.equal(failed.stage === "asking" && failed.offer.state, "locked");

  const retrying = prompt.unlock();
  const cleared = prompt.view();
  assert.equal(cleared.stage === "asking" && cleared.unlockFailed, false);
  await retrying;
  assert.deepEqual(asked, ["unlock", "unlock"]);
});

test("a save that finds Ravenpass locked waits for it to be unlocked", async () => {
  const { prompt, asked, vault } = offerPrompt(ready, {
    saves: [{ ok: false, reason: "locked" }],
    reviews: [{ ok: true, offer: ready }],
  });
  assert.equal(vault.watching(), false);

  await prompt.save();
  const waiting = prompt.view();
  assert.equal(waiting.stage === "asking" && waiting.offer.state, "locked");

  vault.push("unlocked");
  await settle();
  const view = prompt.view();
  assert.equal(view.stage === "asking" && view.offer.state, "ready");
  assert.deepEqual(asked, ["save", "review"]);
});

test("a Ravenpass that closed shows as not open", async () => {
  const saving = offerPrompt(ready, {
    saves: [{ ok: false, reason: "not-open" }],
  });
  await saving.prompt.save();
  assert.equal(saving.prompt.view().stage, "not-open");

  const reviewing = offerPrompt(locked, {
    reviews: [{ ok: false, reason: "not-open" }],
  });
  reviewing.vault.push("not-open");
  await settle();
  assert.equal(reviewing.prompt.view().stage, "not-open");

  const unlocking = offerPrompt(locked, {
    unlock: { ok: false, reason: "not-open" },
  });
  await unlocking.prompt.unlock();
  assert.equal(unlocking.prompt.view().stage, "not-open");
});

test("an offer whose capture is gone closes", async () => {
  const saving = offerPrompt(ready, { saves: [{ ok: false, reason: "gone" }] });
  await saving.prompt.save();
  assert.deepEqual(saving.asked, ["save", "close"]);

  const reviewing = offerPrompt(locked, {
    reviews: [{ ok: false, reason: "gone" }],
  });
  reviewing.vault.push("unlinked");
  await settle();
  assert.deepEqual(reviewing.asked, ["review", "close"]);
});

test("an offer closes when its capture expires", () => {
  const { prompt, asked } = offerPrompt(ready);

  mock.timers.tick(lifetimeMs - 1);
  assert.equal(prompt.view().stage, "asking");
  mock.timers.tick(1);

  const view = prompt.view();
  assert.equal(view.stage, "closed");
  assert.equal(view.stage === "closed" && view.shown.stage, "asking");
  assert.deepEqual(asked, ["close"]);
});

test("a stopped offer asks nothing until it starts again", async () => {
  const { prompt, asked, vault } = offerPrompt(locked, {
    reviews: [{ ok: true, offer: locked }],
  });

  prompt.stop();
  assert.equal(vault.watching(), false);
  vault.push("unlocked");
  await settle();
  assert.deepEqual(asked, []);

  prompt.start();
  vault.push("unlocked");
  await settle();
  assert.deepEqual(asked, ["review"]);
});
