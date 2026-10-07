import type { FillValues } from "../link/client.ts";
import {
  classify,
  codeEntries,
  describe,
  type FieldKind,
  fillTargets,
  formInputs,
  loginValue,
} from "./fields.ts";

// The isolated world's setter writes the element's value past any setter the page defines on it.
const setNativeValue = Object.getOwnPropertyDescriptor(
  HTMLInputElement.prototype,
  "value",
)?.set;

export interface FormFill {
  /** Whether a field was found and every field found was filled. */
  readonly complete: boolean;
  readonly password: boolean;
}

/** Fills the password and the login or email into the form of a field taken as `kind`, without submitting it. */
export function fillForm(
  field: HTMLInputElement,
  kind: FieldKind,
  values: FillValues,
): FormFill {
  const inputs = formInputs(field);
  const kinds = inputs.map((input) => classify(describe(input)));
  const targets = fillTargets(kinds, inputs.indexOf(field), kind);
  const password = inputs[targets.password];
  const login = inputs[targets.login];
  let complete = Boolean(login || password);
  if (login) {
    const value = loginValue(describe(login), values);
    if (value) setValue(login, value);
    else complete = false;
  }
  if (password) {
    if (values.password) setValue(password, values.password);
    else complete = false;
  }
  return { complete, password: password !== undefined };
}

/** Types a one-time code into a code field's inputs, box by box, without submitting the form. */
export function typeCode(
  inputs: readonly HTMLInputElement[],
  code: string,
): void {
  codeEntries(code, inputs.length).forEach((text, index) => {
    const input = inputs[index];
    if (input) typeInto(input, text);
  });
}

function typeInto(input: HTMLInputElement, text: string): void {
  input.focus();
  input.select();
  const before = input.value;
  // Deprecated but the only way to raise trusted `beforeinput` and `input`, which split code boxes follow.
  input.ownerDocument.execCommand("insertText", false, text);
  if (input.value === before) setValue(input, text);
}

export function setValue(input: HTMLInputElement, value: string): void {
  setNativeValue?.call(input, value);
  announceChange(input);
}

export function announceChange(
  input: HTMLInputElement | HTMLSelectElement,
): void {
  input.dispatchEvent(new Event("input", { bubbles: true, composed: true }));
  input.dispatchEvent(new Event("change", { bubbles: true }));
}
