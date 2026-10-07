import assert from "node:assert/strict";
import test from "node:test";
import type { CardValues } from "../link/client.ts";
import {
  countryOption,
  type Entry,
  entryOf,
  expiryText,
  holderParts,
  type ListOption,
  monthOption,
  yearOption,
} from "./card-values.ts";
import type { PaymentPart } from "./payment-fields.ts";
import { field } from "./test-fields.test-support.ts";

const values: CardValues = {
  holder: "Alex  Q Example",
  number: "4111111111111111",
  expiry: "2029-08",
  securityCode: "739",
  billing: {
    street: "1 Example Street",
    city: "Springfield",
    region: "Oregon",
    postalCode: "97477",
    country: "US",
  },
};

function input(part: PaymentPart, shape = {}): Entry {
  return { part, description: field(shape), options: null, languages: ["en"] };
}

function list(part: PaymentPart, options: readonly ListOption[]): Entry {
  return {
    part,
    description: field({ type: "select-one" }),
    options,
    languages: ["en"],
  };
}

const option = (value: string, text = value): ListOption => ({ value, text });

test("an expiry field gets the format it asks for", () => {
  for (const [shape, want] of [
    [{}, "08/29"],
    [{ placeholder: "MM/YY" }, "08/29"],
    [{ placeholder: "MM / YYYY" }, "08/2029"],
    [{ placeholder: "ММ/ГГГГ" }, "08/2029"],
    [{ maxLength: 7 }, "08/2029"],
    [{ maxLength: 5 }, "08/29"],
    [{ maxLength: 4 }, "0829"],
  ] as const) {
    assert.equal(expiryText("08", "2029", field(shape)), want);
  }
});

test("each card field takes its value", () => {
  assert.equal(entryOf(input("number"), values), "4111111111111111");
  assert.equal(entryOf(input("security-code"), values), "739");
  assert.equal(entryOf(input("holder"), values), "Alex  Q Example");
  assert.equal(entryOf(input("given-name"), values), "Alex  Q");
  assert.equal(entryOf(input("family-name"), values), "Example");
  assert.equal(entryOf(input("exp-month"), values), "08");
  assert.equal(entryOf(input("exp-year"), values), "2029");
  assert.equal(entryOf(input("exp-year", { maxLength: 2 }), values), "29");
  assert.equal(entryOf(input("exp-year", { placeholder: "YY" }), values), "29");
  assert.equal(entryOf(input("postal-code"), values), "97477");
  assert.equal(entryOf(input("country"), values), "US");
});

test("a card without a value or a billing address fills nothing there", () => {
  const bare: CardValues = {
    ...values,
    expiry: "",
    securityCode: "",
    billing: null,
  };
  for (const part of [
    "expiry",
    "exp-month",
    "security-code",
    "city",
  ] as const) {
    assert.equal(entryOf(input(part), bare), null, part);
  }
  assert.deepEqual(holderParts("Alex"), ["Alex", ""]);
});

test("a month list is matched by number, then by name", () => {
  assert.equal(
    monthOption(
      8,
      [option("", "Month"), option("07"), option("08"), option("09")],
      ["en"],
    ),
    "08",
  );
  assert.equal(
    monthOption(
      8,
      [option("m7", "7 - July"), option("m8", "8 - August")],
      ["en"],
    ),
    "m8",
  );
  assert.equal(
    monthOption(8, [option("jul", "July"), option("aug", "August")], ["en"]),
    "aug",
  );
  assert.equal(
    monthOption(8, [option("a", "Aug"), option("s", "Sep")], ["en"]),
    "a",
  );
  assert.equal(
    monthOption(8, [option("7", "июль"), option("8", "август")], ["ru", "en"]),
    "8",
  );
  assert.equal(monthOption(8, [option("18", "18")], ["en"]), null);
});

test("a year list takes the year in four or two digits", () => {
  assert.equal(yearOption("2029", [option("2028"), option("2029")]), "2029");
  assert.equal(yearOption("2029", [option("28"), option("29")]), "29");
  assert.equal(yearOption("2029", [option("y", " 2029 ")]), "y");
  assert.equal(yearOption("2029", [option("2030")]), null);
});

test("a country list is matched by its code or its name either way", () => {
  const names = [option("DE", "Germany"), option("US", "United States")];
  assert.equal(countryOption("US", names, ["en"]), "US");
  assert.equal(countryOption("United States", names, ["en"]), "US");
  assert.equal(countryOption("germany", names, ["en"]), "DE");
  const codes = [
    option("de", "Deutschland"),
    option("us", "Vereinigte Staaten"),
  ];
  assert.equal(countryOption("United States", codes, ["de", "en"]), "us");
  assert.equal(countryOption("US", codes, ["de", "en"]), "us");
  assert.equal(countryOption("Atlantis", names, ["en"]), null);
});

test("a list takes the option its part matches", () => {
  const regions = [option("OR", "Oregon"), option("WA", "Washington")];
  assert.equal(entryOf(list("region", regions), values), "OR");
  assert.equal(entryOf(list("exp-month", [option("8", "08")]), values), "8");
  const countries = [option("CA", "Canada"), option("US", "United States")];
  assert.equal(entryOf(list("country", countries), values), "US");
});
