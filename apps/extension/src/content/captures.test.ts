import assert from "node:assert/strict";
import test from "node:test";
import type { CapturedPassword } from "../link/client.ts";
import { captureOf, type SubmittedInput, TypedValues } from "./captures.ts";
import { field } from "./test-fields.test-support.ts";

function input(shape: Partial<SubmittedInput>): SubmittedInput {
  return { ...field(shape), value: "", typed: false, ...shape };
}

/** A password input holding what the person typed. */
function typed(value: string, shape: Partial<SubmittedInput> = {}) {
  return input({ type: "password", value, typed: true, ...shape });
}

/** A password input holding a value Ravenpass filled or a script set. */
function filled(value: string, shape: Partial<SubmittedInput> = {}) {
  return input({ type: "password", value, typed: false, ...shape });
}

function email(value: string) {
  return input({ type: "email", name: "email", value });
}

const forms: readonly [
  string,
  readonly SubmittedInput[],
  string,
  CapturedPassword | null,
][] = [
  [
    "a sign-in",
    [email("alex@example.com"), typed("correct horse")],
    "",
    { account: "alex@example.com", password: "correct horse" },
  ],
  [
    "a sign-in whose account field says nothing but precedes the password",
    [input({ name: "q7", precedesPassword: true, value: "alex" }), typed("pw")],
    "",
    { account: "alex", password: "pw" },
  ],
  [
    "a sign-in that declares its password current",
    [email("alex"), typed("pw", { autocomplete: ["current-password"] })],
    "",
    { account: "alex", password: "pw" },
  ],
  [
    "a sign-up with a confirmation",
    [email("alex"), typed("new pw"), typed("new pw")],
    "",
    { account: "alex", password: "new pw" },
  ],
  [
    "a sign-up declaring new passwords",
    [
      email("alex"),
      typed("new pw", { autocomplete: ["new-password"] }),
      typed("new pw", { autocomplete: ["new-password"] }),
    ],
    "",
    { account: "alex", password: "new pw" },
  ],
  [
    "a change of password with a confirmation",
    [typed("old pw"), typed("new pw"), typed("new pw")],
    "",
    { account: "", password: "new pw", current: "old pw" },
  ],
  [
    "a change of password without a confirmation",
    [typed("old pw"), typed("new pw")],
    "",
    { account: "", password: "new pw", current: "old pw" },
  ],
  [
    "a change of password whose declarations disagree with the order",
    [
      typed("new pw", { autocomplete: ["new-password"] }),
      typed("old pw", { autocomplete: ["current-password"] }),
    ],
    "",
    { account: "", password: "new pw", current: "old pw" },
  ],
  [
    "a change of password whose current password Ravenpass filled",
    [filled("old pw"), typed("new pw"), typed("new pw")],
    "",
    { account: "", password: "new pw", current: "old pw" },
  ],
  [
    "a change of password naming the account in a hidden username field",
    [
      input({
        type: "hidden",
        autocomplete: ["username"],
        visible: false,
        value: "alex",
      }),
      typed("old pw"),
      typed("new pw"),
      typed("new pw"),
    ],
    "",
    { account: "alex", password: "new pw", current: "old pw" },
  ],
  [
    "a password step after an account step",
    [typed("pw")],
    "alex@example.com",
    { account: "alex@example.com", password: "pw" },
  ],
  [
    "a form's own account over the one typed earlier",
    [email("sam@example.com"), typed("pw")],
    "alex@example.com",
    { account: "sam@example.com", password: "pw" },
  ],
  [
    "an empty account field falls back to the one typed earlier",
    [email(""), typed("pw")],
    "alex@example.com",
    { account: "alex@example.com", password: "pw" },
  ],
  [
    "a login field after the password is not its account",
    [typed("pw"), email("alex")],
    "",
    { account: "", password: "pw" },
  ],
  [
    "a hidden password field and an empty one do not count",
    [
      email("alex"),
      typed("pw"),
      filled("decoy", { visible: false }),
      typed(""),
    ],
    "",
    { account: "alex", password: "pw" },
  ],
  ["a password Ravenpass filled", [email("alex"), filled("pw")], "", null],
  [
    "a new password a script set beside a typed current one",
    [typed("old pw"), filled("new pw"), filled("new pw")],
    "",
    null,
  ],
  ["an empty password", [email("alex"), typed("")], "", null],
  ["a form without a password", [email("alex")], "alex", null],
  [
    "four password fields",
    [typed("a"), typed("b"), typed("c"), typed("d")],
    "",
    null,
  ],
];

for (const [name, inputs, typedAccount, expected] of forms) {
  test(`captureOf: ${name}`, () => {
    assert.deepEqual(captureOf(inputs, typedAccount), expected);
  });
}

test("a value the person typed counts as typed", () => {
  const values = new TypedValues<object>();
  const password = {};

  values.input(password, "correct", true);
  values.input(password, "correct horse", true);

  assert.equal(values.typed(password, "correct horse"), true);
});

test("digits the person typed stay typed when the page spaces them", () => {
  const values = new TypedValues<object>();
  const number = {};

  values.input(number, "41111", true);
  assert.equal(values.typedDigits(number, "4111 1"), true);
  assert.equal(values.typedDigits(number, "4111 2"), false);

  values.input(number, "4111 1111", false);
  assert.equal(values.typedDigits(number, "4111 1111"), false);
  assert.equal(values.typedDigits({}, ""), false);
});

test("a value set with an untrusted event is not typed until the person types again", () => {
  const values = new TypedValues<object>();
  const password = {};

  values.input(password, "correct horse", true);
  values.input(password, "filled", false);
  assert.equal(values.typed(password, "filled"), false);
  assert.equal(values.typed(password, "correct horse"), false);

  values.input(password, "filled!", true);
  assert.equal(values.typed(password, "filled!"), true);
});

test("a value set without any event is not typed", () => {
  const values = new TypedValues<object>();
  const password = {};

  values.input(password, "correct horse", true);

  assert.equal(values.typed(password, "scripted"), false);
});

test("typing counts only for the field typed into", () => {
  const values = new TypedValues<object>();
  const password = {};
  const confirmation = {};

  values.input(password, "correct horse", true);

  assert.equal(values.typed(confirmation, "correct horse"), false);
});
