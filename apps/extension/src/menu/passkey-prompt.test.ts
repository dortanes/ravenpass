import assert from "node:assert/strict";
import { test } from "node:test";
import type { PasskeyChoice } from "../link/client.ts";
import type { Answers, PasskeyCardContent } from "../messages.ts";
import { PasskeyPrompt, passkeyNote } from "./passkey-prompt.ts";
import { FakeVault, settle } from "./vault-state.test-support.ts";

const passkey = {
  credential: "c1",
  credentialId: "AQID",
  account: "alex",
  label: "GitHub",
};

const signIn: PasskeyCardContent = {
  state: "passkey",
  request: 1,
  mode: "get",
  listing: { state: "sign-in", passkeys: [passkey] },
};

const save: PasskeyCardContent = {
  state: "passkey",
  request: 1,
  mode: "create",
  listing: {
    state: "save",
    account: "alex",
    targets: [
      { credential: "c0", label: "Old", account: "alexander", tags: [] },
      { credential: "c1", label: "GitHub", account: "alex", tags: [] },
    ],
  },
};

const locked: PasskeyCardContent = {
  ...signIn,
  listing: { state: "locked" },
};

/** Requests answered as the test scripts them: each sign-in or save waits for `answer`. */
function prompt(content: PasskeyCardContent) {
  const log: string[] = [];
  let answer: (outcome: Answers["passkey-sign"]) => void = () => {};
  const reviews: Answers["passkey-review"][] = [];
  let unlock: Answers["menu-unlock"] = { ok: true };
  const vault = new FakeVault();
  const card = new PasskeyPrompt(content, {
    sign: (choice: PasskeyChoice) => {
      log.push(`sign ${choice.credential}`);
      return new Promise((resolve) => (answer = resolve));
    },
    save: (target) => {
      log.push(`save ${target}`);
      return new Promise((resolve) => (answer = resolve));
    },
    elsewhere: async () => {
      log.push("elsewhere");
      return { ok: true };
    },
    review: async () => {
      log.push("review");
      return reviews.shift() ?? { listing: { state: "locked" } };
    },
    unlock: async () => {
      log.push("unlock");
      return unlock;
    },
    watchVault: vault.watch,
    close: () => log.push("close"),
  });
  card.start();
  return {
    card,
    log,
    reviews,
    vault,
    answer: (outcome: Answers["passkey-sign"]) => answer(outcome),
    unlocking: (outcome: Answers["menu-unlock"]) => {
      unlock = outcome;
    },
  };
}

test("a save starts on the credential with the same account, and saves where the person chose", async () => {
  const { card, log, answer } = prompt(save);
  assert.equal(card.view().target, "c1");

  card.choose("new");
  const saving = card.save();
  assert.deepEqual(card.view().waiting, { progress: null });
  card.choose("c0");
  card.progress("confirm-on-device");
  assert.equal(card.view().target, "new");
  assert.deepEqual(card.view().waiting, { progress: "confirm-on-device" });
  card.close();

  answer({ ok: true });
  await saving;
  assert.deepEqual(log, ["save new"]);
});

test("a declined sign-in is noted and may be tried again", async () => {
  const { card, log, answer } = prompt(signIn);

  const first = card.sign(passkey);
  answer({ ok: false, reason: "declined" });
  await first;
  assert.equal(card.view().note, "declined");
  assert.equal(card.view().waiting, null);

  const second = card.sign(passkey);
  assert.equal(card.view().note, null);
  answer({ ok: true });
  await second;
  assert.deepEqual(log, ["sign c1", "sign c1"]);
});

test("a save that finds a passkey the site excludes, or Ravenpass locked, shows that", async () => {
  for (const reason of ["excluded", "locked", "not-open"] as const) {
    const { card, answer } = prompt(save);
    const saving = card.save();
    answer({ ok: false, reason });
    await saving;
    assert.deepEqual(card.view().listing, { state: reason });
    assert.equal(card.view().waiting, null);
  }
});

test("while Ravenpass is locked the card asks again on each vault change until it lists something", async () => {
  const { card, log, reviews, vault } = prompt(locked);
  reviews.push({ listing: { state: "locked" } }, { listing: signIn.listing });

  vault.push("locked");
  await settle();
  assert.deepEqual(log, []);
  vault.push("unlocked");
  await settle();
  vault.push("not-open");
  await settle();

  assert.deepEqual(log, ["review", "review"]);
  assert.deepEqual(card.view().listing, signIn.listing);
  assert.equal(vault.watching(), false);
});

test("the card stops watching once it closes or stops", async () => {
  const closed = prompt(locked);
  closed.reviews.push({ listing: null });
  closed.vault.push("unlocked");
  await settle();
  assert.deepEqual(closed.log, ["review"]);
  assert.deepEqual(closed.card.view().listing, { state: "locked" });

  const stopped = prompt(locked);
  assert.equal(stopped.vault.watching(), true);
  stopped.card.stop();
  assert.equal(stopped.vault.watching(), false);
  stopped.vault.push("unlocked");
  await settle();
  assert.deepEqual(stopped.log, []);
});

test("a sign-in that finds Ravenpass locked watches the vault", async () => {
  const { card, answer, vault } = prompt(signIn);
  assert.equal(vault.watching(), false);
  const signing = card.sign(passkey);
  answer({ ok: false, reason: "locked" });
  await signing;
  assert.equal(vault.watching(), true);
});

test("unlocking a Ravenpass that closed meanwhile shows it not open", async () => {
  const { card, log, unlocking } = prompt(locked);
  unlocking({ ok: false, reason: "not-open" });

  await card.unlock();

  assert.deepEqual(log, ["unlock"]);
  assert.deepEqual(card.view().listing, { state: "not-open" });
});

test("an unlock Ravenpass could not serve is noted until Unlock is tried again", async () => {
  const { card, log, unlocking } = prompt(locked);
  unlocking({ ok: false, reason: "failed" });

  await card.unlock();
  assert.equal(card.view().unlockFailed, true);
  assert.deepEqual(card.view().listing, { state: "locked" });

  unlocking({ ok: true });
  const retrying = card.unlock();
  assert.equal(card.view().unlockFailed, false);
  await retrying;
  assert.equal(card.view().unlockFailed, false);
  assert.deepEqual(log, ["unlock", "unlock"]);
});

test("Other options and closing ask the service worker, but not while Ravenpass verifies the person", async () => {
  const { card, log, answer } = prompt(signIn);
  await card.elsewhere();
  card.close();
  assert.deepEqual(log, ["elsewhere", "close"]);

  const signing = card.sign(passkey);
  await card.elsewhere();
  card.close();
  answer({ ok: true });
  await signing;
  assert.deepEqual(log, ["elsewhere", "close", "sign c1"]);
});

test("each failure notes what the person can do", () => {
  assert.equal(passkeyNote("declined", "get"), "extension.passkey.declined");
  assert.equal(
    passkeyNote("unverifiable", "create"),
    "extension.passkey.unverifiable",
  );
  assert.equal(passkeyNote("full", "create"), "extension.passkey.full");
  assert.equal(passkeyNote("failed", "get"), "extension.passkey.sign-in.error");
  assert.equal(passkeyNote("failed", "create"), "extension.passkey.save.error");
});
