import assert from "node:assert/strict";
import test from "node:test";
import { cardCaptureOf, type SubmittedPayment } from "./card-captures.ts";
import type { PaymentPart } from "./payment-fields.ts";

function typed(part: PaymentPart, value: string): SubmittedPayment {
  return { part, value, text: "", typed: true };
}

function chosen(
  part: PaymentPart,
  value: string,
  text: string,
): SubmittedPayment {
  return { part, value, text, typed: false };
}

test("a typed card is captured with its holder, expiry and security code", () => {
  assert.deepEqual(
    cardCaptureOf([
      typed("number", "4111 1111 1111 1111"),
      typed("holder", " Alex Example "),
      typed("expiry", "08 / 29"),
      typed("security-code", "739"),
    ]),
    {
      holder: "Alex Example",
      number: "4111111111111111",
      expiry: "2029-08",
      securityCode: "739",
      network: "visa",
    },
  );
});

test("the expiry is read from a month and a year, and the holder from two names", () => {
  assert.deepEqual(
    cardCaptureOf([
      typed("number", "5555555555554444"),
      typed("given-name", "Alex"),
      typed("family-name", "Example"),
      chosen("exp-month", "8", "08 - August"),
      chosen("exp-year", "y29", "2029"),
    ]),
    {
      holder: "Alex Example",
      number: "5555555555554444",
      expiry: "2029-08",
      securityCode: "",
      network: "mastercard",
    },
  );
});

test("a number the person did not type, or that no card has, is not captured", () => {
  assert.equal(
    cardCaptureOf([{ ...typed("number", "4111111111111111"), typed: false }]),
    null,
  );
  for (const number of [
    "41111111111",
    "4111 1111 1111 11112",
    "411111111111111",
  ]) {
    assert.equal(cardCaptureOf([typed("number", number)]), null, number);
  }
});

test("an unreadable expiry or security code is left out", () => {
  const card = cardCaptureOf([
    typed("number", "4111111111111111"),
    typed("expiry", "13/29"),
    typed("security-code", "73"),
  ]);
  assert.equal(card?.expiry, "");
  assert.equal(card?.securityCode, "");
  assert.equal(
    cardCaptureOf([
      typed("number", "4111111111111111"),
      typed("expiry", "0829"),
    ])?.expiry,
    "2029-08",
  );
  assert.equal(
    cardCaptureOf([
      typed("number", "4111111111111111"),
      typed("expiry", "8/2031"),
    ])?.expiry,
    "2031-08",
  );
});
