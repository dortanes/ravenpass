import assert from "node:assert/strict";
import test from "node:test";
import type { FieldDescription } from "./fields.ts";
import {
  classifyPayment,
  type PaymentShape,
  paymentFieldsAmong,
} from "./payment-fields.ts";
import { field } from "./test-fields.test-support.ts";

const shapes: readonly [
  string,
  Partial<FieldDescription>,
  PaymentShape | null,
][] = [
  // Autocomplete decides when present
  [
    "declared number",
    { autocomplete: ["cc-number"] },
    { part: "number", section: null, contextual: false },
  ],
  [
    "declared billing city",
    { autocomplete: ["billing", "address-level2"] },
    { part: "city", section: "billing", contextual: false },
  ],
  [
    "declared shipping street",
    { autocomplete: ["section-ship", "shipping", "street-address"] },
    { part: "street", section: "shipping", contextual: false },
  ],
  [
    "declared month list",
    { type: "select-one", autocomplete: ["cc-exp-month"] },
    { part: "exp-month", section: null, contextual: false },
  ],
  [
    "a field declared for something else",
    { autocomplete: ["email"], name: "card-number" },
    null,
  ],
  ["cc-type is left alone", { autocomplete: ["cc-type"] }, null],
  // Naming
  [
    "card number by its label",
    { type: "tel", label: "Card number" },
    { part: "number", section: null, contextual: false },
  ],
  [
    "security code named as a code",
    { type: "tel", maxLength: 4, label: "Security code" },
    { part: "security-code", section: null, contextual: false },
  ],
  [
    "hidden security code",
    { type: "password", name: "cvv" },
    { part: "security-code", section: null, contextual: false },
  ],
  [
    "expiry by its placeholder",
    { placeholder: "MM / YY" },
    { part: "expiry", section: null, contextual: false },
  ],
  [
    "expiry month list",
    { type: "select-one", label: "Expiration month" },
    { part: "exp-month", section: null, contextual: false },
  ],
  [
    "expiry year in an input",
    { name: "expYear" },
    { part: "exp-year", section: null, contextual: false },
  ],
  [
    "holder",
    { label: "Name on card" },
    { part: "holder", section: null, contextual: false },
  ],
  [
    "Russian number",
    { label: "Номер карты" },
    { part: "number", section: null, contextual: false },
  ],
  [
    "a month alone counts only beside a number",
    { name: "month" },
    { part: "exp-month", section: null, contextual: true },
  ],
  [
    "postal code",
    { label: "ZIP code" },
    { part: "postal-code", section: null, contextual: true },
  ],
  [
    "country list",
    { type: "select-one", name: "country" },
    { part: "country", section: null, contextual: true },
  ],
  ["second address line", { label: "Address line 2" }, null],
  ["an email field", { type: "email", label: "Email address" }, null],
  ["a search box", { type: "search", label: "Card number" }, null],
  ["hidden field", { visible: false, autocomplete: ["cc-number"] }, null],
  ["read-only field", { editable: false, autocomplete: ["cc-csc"] }, null],
];

for (const [name, shape, expected] of shapes) {
  test(`payment field: ${name}`, () => {
    assert.deepEqual(classifyPayment(field(shape)), expected);
  });
}

test("a birth month without a card number is no card field", () => {
  const signUp = [
    classifyPayment(field({ name: "month" })),
    classifyPayment(field({ name: "year" })),
    classifyPayment(field({ label: "City" })),
  ];
  assert.deepEqual(paymentFieldsAmong(signUp), []);
});

test("a checkout's month, year and address count beside its number", () => {
  const checkout = [
    classifyPayment(field({ label: "Card number" })),
    classifyPayment(field({ type: "select-one", name: "month" })),
    classifyPayment(field({ type: "select-one", name: "year" })),
    classifyPayment(field({ label: "City" })),
  ];
  assert.deepEqual(
    paymentFieldsAmong(checkout).map(({ part }) => part),
    ["number", "exp-month", "exp-year", "city"],
  );
});

test("a payment provider's lone expiry or security code frame counts", () => {
  for (const lone of [
    field({ autocomplete: ["cc-exp"] }),
    field({ type: "tel", label: "CVC" }),
  ]) {
    assert.equal(paymentFieldsAmong([classifyPayment(lone)]).length, 1);
  }
});
