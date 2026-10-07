import assert from "node:assert/strict";
import test from "node:test";
import type { Suggestion } from "../link/client.ts";
import type { MatchStrength } from "../match-strength.ts";
import type { Listed } from "../messages.ts";
import { FillConfirmation } from "./fill-confirmation.ts";

function suggestion(id: string, strength: MatchStrength): Listed<Suggestion> {
  return {
    id,
    label: "Example",
    account: "alex",
    site: "example.com",
    exact: strength === "strong" || strength === "insecure-page",
    tags: [],
    strength,
  };
}

test("a strong suggestion fills at once, unconfirmed", () => {
  const confirmation = new FillConfirmation();
  const filled: string[] = [];

  confirmation.choose(suggestion("a1", "strong"), () => filled.push("a1"));

  assert.deepEqual(filled, ["a1"]);
  assert.equal(confirmation.view(), null);
  assert.equal(confirmation.confirmed("a1"), false);
});

test("a weak password or code suggestion fills only once the person confirms it", () => {
  for (const strength of [
    "insecure-page",
    "same-site",
    "other-site",
  ] as const) {
    const confirmation = new FillConfirmation();
    const filled: string[] = [];
    const weak = suggestion("a1", strength);

    confirmation.choose(weak, () => filled.push("a1"));
    assert.deepEqual(filled, []);
    assert.equal(confirmation.view(), weak);
    assert.equal(confirmation.confirmed("a1"), false);

    confirmation.confirm(false);
    assert.deepEqual(filled, ["a1"]);
    assert.equal(confirmation.view(), null);
    assert.equal(confirmation.confirmed("a1"), true);
    assert.equal(confirmation.remembers("a1"), false);
  }
});

test("only another site's suggestion is remembered, and only when the person asks", () => {
  const confirmation = new FillConfirmation();
  for (const strength of ["insecure-page", "same-site"] as const) {
    confirmation.choose(suggestion(strength, strength), () => {});
    confirmation.confirm(true);
    assert.equal(confirmation.remembers(strength), false, strength);
  }

  confirmation.choose(suggestion("a1", "other-site"), () => {});
  confirmation.confirm(true);
  assert.equal(confirmation.remembers("a1"), true);

  confirmation.choose(suggestion("a1", "other-site"), () => {});
  confirmation.confirm(false);
  assert.equal(confirmation.remembers("a1"), false);
});

test("cancelling drops the choice, and a later confirmation fills nothing", () => {
  const confirmation = new FillConfirmation();
  const filled: string[] = [];
  confirmation.choose(suggestion("a1", "other-site"), () => filled.push("a1"));

  confirmation.cancel();
  confirmation.confirm(true);

  assert.deepEqual(filled, []);
  assert.equal(confirmation.confirmed("a1"), false);
  assert.equal(confirmation.remembers("a1"), false);
});

test("a newer weak choice replaces the one waiting", () => {
  const confirmation = new FillConfirmation();
  const filled: string[] = [];
  confirmation.choose(suggestion("a1", "other-site"), () => filled.push("a1"));
  confirmation.choose(suggestion("b2", "insecure-page"), () =>
    filled.push("b2"),
  );

  confirmation.confirm(false);

  assert.deepEqual(filled, ["b2"]);
  assert.equal(confirmation.confirmed("a1"), false);
});

test("listeners hear a choice waiting and its end", () => {
  const confirmation = new FillConfirmation();
  let heard = 0;
  confirmation.subscribe(() => {
    heard += 1;
  });

  confirmation.choose(suggestion("a1", "other-site"), () => {});
  confirmation.cancel();

  assert.equal(heard, 2);
});
