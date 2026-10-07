import assert from "node:assert/strict";
import test from "node:test";
import creditCardType from "credit-card-type";
import { emptyAddress } from "../identities/identity.ts";
import type { Card } from "../vault-api.ts";
import { emptyCard } from "./card.test-support.ts";
import {
  billingChoiceOf,
  billingOf,
  cardColor,
  cardLine,
  concealedDigits,
  digitsWithin,
  expiryDisplay,
  expiryEnd,
  expiryStart,
  expiryStored,
  failsCheck,
  logoInColour,
  maskedGroups,
  networkLogo,
  neutralCardColor,
  numberGroups,
  readableOn,
} from "./card.ts";
import {
  cardNetworks,
  detectNetwork,
  digitsOf,
  isCardNetwork,
  networkName,
} from "./networks.ts";

test("the network is detected once exactly one fits the digits", () => {
  assert.equal(detectNetwork("4111111111111111"), "visa");
  assert.equal(detectNetwork("4111 1111 1111 1111"), "visa");
  assert.equal(detectNetwork("5555555555554444"), "mastercard");
  assert.equal(detectNetwork("378282246310005"), "american-express");
  assert.equal(detectNetwork("2200000000000004"), "mir");
  assert.equal(detectNetwork("6011111111111117"), "discover");
  assert.equal(detectNetwork("3530111333300000"), "jcb");
});

// Imports name networks from the eight-digit issuer identification number (ISO/IEC 7812-1:2017).
test("every network is known from the first eight digits of a number", () => {
  const issuerDigits = 8;
  for (const network of cardNetworks) {
    for (const pattern of creditCardType.getTypeInfo(network).patterns) {
      const leading = Array.isArray(pattern) ? pattern[0] : pattern;
      assert.ok(
        String(leading).length <= issuerDigits,
        `${network} matches ${leading}`,
      );
    }
  }
  assert.equal(detectNetwork("41111111"), "visa");
  assert.equal(detectNetwork("22000000"), "mir");
  assert.equal(detectNetwork("37828224"), "american-express");
});

test("no network is named while several or none fit", () => {
  assert.equal(detectNetwork(""), "");
  assert.equal(detectNetwork("3"), "");
  assert.equal(detectNetwork("9999999999999999"), "");
});

test("the networks are the names credit-card-type uses", () => {
  assert.equal(cardNetworks.length, 15);
  for (const network of cardNetworks) {
    assert.ok(isCardNetwork(network));
    assert.ok(networkName(network), `${network} has no name`);
  }
  assert.equal(networkLogo("mastercard"), "Mastercard");
  assert.equal(networkLogo("american-express"), "Amex");
  assert.equal(networkLogo("troy"), null);
  assert.equal(networkLogo(""), null);
  assert.ok(logoInColour("mastercard"));
  assert.ok(!logoInColour("visa"));
  assert.ok(isCardNetwork(""));
  assert.ok(!isCardNetwork("amex"));
  assert.equal(networkName("american-express"), "American Express");
  assert.equal(networkName(""), "");
});

test("a number is grouped as its network spaces it", () => {
  assert.deepEqual(numberGroups("4111111111111111", "visa"), [
    "4111",
    "1111",
    "1111",
    "1111",
  ]);
  assert.deepEqual(numberGroups("378282246310005", "american-express"), [
    "3782",
    "822463",
    "10005",
  ]);
  assert.deepEqual(numberGroups("123456", ""), ["1234", "56"]);
  assert.deepEqual(numberGroups("1234567890123456789", ""), [
    "1234",
    "5678",
    "9012",
    "3456789",
  ]);
  assert.deepEqual(numberGroups("", "visa"), []);
});

test("only the last four digits are shown of a masked number", () => {
  assert.deepEqual(maskedGroups("4111111111111111", "visa"), [
    "••••",
    "••••",
    "••••",
    "1111",
  ]);
  assert.deepEqual(maskedGroups("378282246310005", "american-express"), [
    "••••",
    "••••••",
    "•0005",
  ]);
  assert.equal(cardLine("Monzo", "", "1234"), "Monzo •• 1234");
  assert.equal(
    cardLine("Example Bank", "visa", "4321"),
    "Example Bank · Visa •• 4321",
  );
  assert.equal(cardLine("", "mastercard", "2038"), "Mastercard •• 2038");
  assert.equal(cardLine("", "", "1234"), "•• 1234");
  assert.equal(cardLine("Monzo", "", ""), "Monzo");
});

test("digits are kept from typed or pasted text", () => {
  assert.equal(digitsOf("4111-1111 1111.1111"), "4111111111111111");
});

test("a value of digits fits its bounds, and an optional one may be empty", () => {
  assert.ok(digitsWithin("123", 3, 4, true));
  assert.ok(digitsWithin("1234", 3, 4, false));
  assert.ok(digitsWithin("", 3, 4, true));
  assert.ok(!digitsWithin("", 3, 4, false));
  assert.ok(!digitsWithin("12", 3, 4, true));
  assert.ok(!digitsWithin("12345", 3, 4, true));
  assert.ok(!digitsWithin("12a", 3, 4, true));
});

test("an expiry is typed as MM/YY and stored as YYYY-MM", () => {
  assert.equal(expiryStored("08/29"), "2029-08");
  assert.equal(expiryStored("12/00"), "2000-12");
  assert.equal(expiryStored(""), "");
  for (const typed of ["08/2", "0829", "13/29", "00/29", "8/29"]) {
    assert.equal(expiryStored(typed), null, typed);
  }
  assert.equal(expiryDisplay("2029-08"), "08/29");
  assert.equal(expiryDisplay(""), "");
  assert.equal(expiryDisplay("2029-13"), "");
  assert.equal(expiryDisplay("29-08"), "");
});

test("a card is valid through the last day of its expiry month", () => {
  assert.equal(expiryEnd("2029-08"), "2029-08-31");
  assert.equal(expiryEnd("2028-02"), "2028-02-29");
  assert.equal(expiryEnd("2027-02"), "2027-02-28");
  assert.equal(expiryStart("2029-08"), "2024-09-01");
  assert.equal(expiryStart("2028-02"), "2023-03-01");
  assert.equal(expiryStart(""), "");
  assert.equal(expiryEnd("2026-11"), "2026-11-30");
  assert.equal(expiryEnd(""), "");
  assert.equal(expiryEnd("2026-00"), "");
});

test("a card without a colour of its own takes its network's", () => {
  assert.equal(cardColor("#123456", "visa"), "#123456");
  assert.equal(cardColor("", "visa"), "#1a1f71");
  assert.equal(cardColor("", ""), neutralCardColor);
  for (const network of cardNetworks) {
    assert.match(cardColor("", network), /^#[0-9a-f]{6}$/);
  }
});

test("text on a card is light on a dark colour and dark on a light one", () => {
  assert.equal(readableOn("#1a1f71"), "#fff");
  assert.equal(readableOn(neutralCardColor), "#fff");
  assert.equal(readableOn("#f7e9a0"), "#000");
});

test("a whole number that fails its check digit is flagged", () => {
  assert.equal(failsCheck("4242424242424242", "visa"), false);
  assert.equal(failsCheck("4242 4242 4242 4241", "visa"), true);
  assert.equal(failsCheck("424242424242424", "visa"), false);
  assert.equal(failsCheck("6212345678901231", "unionpay"), false);
  assert.equal(failsCheck("4242424242424241", ""), false);
});

test("a short secret is concealed one dot per digit", () => {
  assert.equal(concealedDigits("212"), "•••");
  assert.equal(concealedDigits("1234"), "••••");
  assert.equal(concealedDigits(""), "");
});

const read: Card = {
  ...emptyCard,
  id: "c1",
  groups: [],
  site: "",
  linked: null,
};
const link = { identityId: "i1", addressId: "a1" };

test("a card read from the vault bills to its link, its own address or nothing", () => {
  assert.deepEqual(billingChoiceOf(undefined), { kind: "none" });
  assert.deepEqual(billingChoiceOf(read), { kind: "none" });
  assert.deepEqual(
    billingChoiceOf({
      ...read,
      billingLink: link,
      linked: { identityLabel: "Me", address: { ...emptyAddress, id: "a1" } },
    }),
    { kind: "linked", link },
  );
  assert.deepEqual(billingChoiceOf({ ...read, billingLink: link }), {
    kind: "none",
  });
  const own = { ...emptyAddress, city: "Almaty" };
  assert.deepEqual(billingChoiceOf({ ...read, billing: own }), {
    kind: "own",
    address: own,
  });
});

test("a card saves one billing address at most, and its own without id or label", () => {
  assert.deepEqual(billingOf({ kind: "none" }), {
    billing: null,
    billingLink: null,
  });
  assert.deepEqual(billingOf({ kind: "linked", link }), {
    billing: null,
    billingLink: link,
  });
  assert.deepEqual(billingOf({ kind: "own", address: emptyAddress }), {
    billing: null,
    billingLink: null,
  });
  assert.deepEqual(
    billingOf({
      kind: "own",
      address: { ...emptyAddress, id: "x", label: "Home", city: "Almaty" },
    }),
    { billing: { ...emptyAddress, city: "Almaty" }, billingLink: null },
  );
});
