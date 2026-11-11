import assert from "node:assert/strict";
import test from "node:test";
import { PopupPairing } from "./pairing.ts";

test("valid bounded pairing replies pass through", async () => {
  for (const answer of [
    { ok: true },
    { ok: false, reason: "failed", diagnostic: "RP-11" },
  ] as const) {
    assert.deepEqual(
      await new PopupPairing(async () => answer).link("synthetic-key"),
      answer,
    );
  }
});

test("missing, malformed and unbounded background replies become a fixed reference", async () => {
  for (const answer of [
    null,
    undefined,
    "synthetic-secret",
    { ok: false, reason: "failed", diagnostic: "synthetic-secret" },
  ]) {
    assert.deepEqual(
      await new PopupPairing(async () => answer).link("synthetic-key"),
      { ok: false, reason: "failed", diagnostic: "RP-12" },
    );
  }
});

test("message port exceptions never expose their contents", async () => {
  const pairing = new PopupPairing(async () => {
    throw new Error("synthetic-connection-key");
  });
  assert.deepEqual(await pairing.link("synthetic-key"), {
    ok: false,
    reason: "failed",
    diagnostic: "RP-12",
  });
});

test("a reply from a background without references is identified separately", async () => {
  const pairing = new PopupPairing(async () => ({
    ok: false,
    reason: "failed",
  }));
  assert.deepEqual(await pairing.link("synthetic-key"), {
    ok: false,
    reason: "failed",
    diagnostic: "RP-15",
  });
});
