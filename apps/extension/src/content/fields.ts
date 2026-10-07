import fieldWords from "../../../../packages/app/autofill/fieldwords/words.json" with {
  type: "json",
};

/** `password` is a current password; a new-password field has no kind. */
export type FieldKind = "login" | "password" | "code";

export interface FieldDescription {
  /** The input's type as the DOM normalises it: lowercase, `text` for none or an unknown one. */
  readonly type: string;
  /** The lowercase tokens of its `autocomplete` attribute. */
  readonly autocomplete: readonly string[];
  /** Its `inputmode` as the DOM reflects it: a known mode in lowercase, empty for none. */
  readonly inputMode: string;
  readonly maxLength: number | null;
  readonly name: string;
  readonly id: string;
  readonly placeholder: string;
  /** The text of its labels, its `aria-label`, and the elements its `aria-labelledby` names. */
  readonly label: string;
  /** Whether the next field of its form that a person types in is a password input. */
  readonly precedesPassword: boolean;
  /** Whether it takes space on the page and its style does not hide it. */
  readonly visible: boolean;
  /** Whether it is neither disabled nor read-only. */
  readonly editable: boolean;
}

export interface AccountValues {
  readonly login: string;
  readonly email: string;
}

/** A field as positions among its neighbouring inputs; a split code field spans its boxes. */
export interface FieldShape {
  readonly kind: FieldKind;
  readonly inputs: readonly number[];
}

export interface Field {
  readonly kind: FieldKind;
  readonly inputs: readonly [HTMLInputElement, ...HTMLInputElement[]];
}

const typedTypes: ReadonlySet<string> = new Set([
  "text",
  "email",
  "password",
  "tel",
  "url",
]);
const codeTypes: ReadonlySet<string> = new Set(["text", "tel", "number"]);
const codeLengths = { min: 4, max: 10 };
const splitRunLength = 4;

/** `autocomplete` overrides naming; search and new-password fields are never classified. */
export function classify(field: FieldDescription): FieldKind | null {
  if (
    !field.visible ||
    !field.editable ||
    !(typedTypes.has(field.type) || codeTypes.has(field.type))
  ) {
    return null;
  }
  const words = wordsOf(field);
  if (mentions(words, fieldWords.search)) return null;
  const declared = declaredKind(field.autocomplete);
  if (declared !== undefined) return declared;
  if (field.type === "password") {
    return mentions(words, fieldWords.newPassword) ? null : "password";
  }
  if (field.type === "email") return "login";
  // The code check precedes the login check: a code field's label may name the email the code went to.
  if (
    codeTypes.has(field.type) &&
    (mentions(words, fieldWords.oneTime) ||
      ((asksDigits(field) || takesCode(field.maxLength)) &&
        mentions(words, fieldWords.code)))
  ) {
    return "code";
  }
  if (
    field.type === "text" &&
    (field.precedesPassword || mentions(words, fieldWords.login))
  ) {
    return "login";
  }
  return null;
}

function asksDigits(field: FieldDescription): boolean {
  return (
    field.type === "tel" ||
    field.type === "number" ||
    field.inputMode === "numeric"
  );
}

/** `siblings` are in document order; a login or password field never joins a split code field. */
export function classifyAmong(
  siblings: readonly FieldDescription[],
  index: number,
): FieldShape | null {
  const field = siblings[index];
  if (!field) return null;
  const kind = classify(field);
  if (kind !== "login" && kind !== "password") {
    const run = splitRun(siblings, index);
    if (run) return { kind: "code", inputs: run };
  }
  return kind ? { kind, inputs: [index] } : null;
}

export function codeEntries(code: string, boxes: number): string[] {
  return boxes === code.length ? Array.from(code) : [code];
}

function splitRun(
  siblings: readonly FieldDescription[],
  index: number,
): number[] | null {
  if (!isBox(siblings[index])) return null;
  let start = index;
  while (start > 0 && isBox(siblings[start - 1])) start -= 1;
  let end = index + 1;
  while (end < siblings.length && isBox(siblings[end])) end += 1;
  if (end - start < splitRunLength) return null;
  return Array.from({ length: end - start }, (_, offset) => start + offset);
}

function isBox(field: FieldDescription | undefined): boolean {
  return (
    field?.visible === true &&
    field.editable &&
    codeTypes.has(field.type) &&
    field.maxLength === 1
  );
}

function takesCode(maxLength: number | null): boolean {
  return (
    maxLength !== null &&
    maxLength >= codeLengths.min &&
    maxLength <= codeLengths.max
  );
}

/** `password` is -1 for a form without one; the result is -1 for no login field. */
/** Where a fill from the input at `at` puts the login and the password among a form's input kinds, -1 for none;
 * `chosen` is what the menu took that input for, which counts where detection read the input as no field. */
export function fillTargets(
  kinds: readonly (FieldKind | null)[],
  at: number,
  chosen: FieldKind,
): { readonly login: number; readonly password: number } {
  const own = at < 0 ? null : (kinds[at] ?? chosen);
  const password = own === "password" ? at : kinds.indexOf("password");
  const login = own === "login" ? at : pairedLogin(kinds, password);
  return { login, password };
}

export function pairedLogin(
  kinds: readonly (FieldKind | null)[],
  password: number,
): number {
  if (password < 0) return kinds.indexOf("login");
  return kinds.lastIndexOf("login", password);
}

export function loginValue(
  field: FieldDescription,
  values: AccountValues,
): string {
  const wantsEmail =
    field.type === "email" || field.autocomplete.includes("email");
  if (wantsEmail && values.email) return values.email;
  return values.login || values.email;
}

/** null for a field left alone, undefined when `autocomplete` declares no kind. */
function declaredKind(tokens: readonly string[]): FieldKind | null | undefined {
  for (const token of tokens) {
    switch (token) {
      case "current-password":
        return "password";
      case "username":
      case "email":
        return "login";
      case "one-time-code":
        return "code";
      case "new-password":
        return null;
    }
  }
  return undefined;
}

function wordsOf(field: FieldDescription): string[] {
  return wordsIn(
    [field.name, field.id, field.placeholder, field.label].join(" "),
  );
}

/** Lowercase words, split at case changes and at anything but letters and digits. */
export function wordsIn(text: string): string[] {
  return text
    .replace(/(\p{Ll}|\p{N})(\p{Lu})/gu, "$1 $2")
    .toLowerCase()
    .split(/[^\p{L}\p{N}]+/u)
    .filter(Boolean);
}

/** A multi-word term matches its words in a row, only the last one as a prefix. */
export function mentions(
  words: readonly string[],
  terms: readonly string[],
): boolean {
  return terms.some((term) => {
    const parts = term.split(" ");
    const last = parts.length - 1;
    return words.some((_, start) =>
      parts.every((part, offset) => {
        const word = words[start + offset] ?? "";
        return offset === last ? word.startsWith(part) : word === part;
      }),
    );
  });
}

export function fieldOf(input: HTMLInputElement): Field | null {
  const siblings = input.maxLength === 1 ? neighbourInputs(input) : [input];
  const shape = classifyAmong(siblings.map(describe), siblings.indexOf(input));
  if (!shape) return null;
  const [first, ...rest] = shape.inputs.flatMap(
    (index) => siblings[index] ?? [],
  );
  return first ? { kind: shape.kind, inputs: [first, ...rest] } : null;
}

/** Code components wrap each box in one or two elements of its own. */
const boxAncestorLevels = 3;

function neighbourInputs(input: HTMLInputElement): HTMLInputElement[] {
  let ancestor = input.parentElement;
  for (let level = 0; ancestor && level < boxAncestorLevels; level += 1) {
    const inputs = Array.from(
      ancestor.querySelectorAll<HTMLInputElement>("input:not([type=hidden])"),
    );
    if (inputs.length > 1) return inputs;
    ancestor = ancestor.parentElement;
  }
  return [input];
}

export function describe(input: HTMLInputElement): FieldDescription {
  const typed = formInputs(input).filter(
    (other) => typedTypes.has(other.type) && isVisible(other),
  );
  const next = typed[typed.indexOf(input) + 1];
  return {
    type: input.type,
    autocomplete:
      input
        .getAttribute("autocomplete")
        ?.toLowerCase()
        .split(/\s+/)
        .filter(Boolean) ?? [],
    inputMode: input.inputMode,
    maxLength: input.maxLength < 0 ? null : input.maxLength,
    name: input.name,
    id: input.id,
    placeholder: input.placeholder,
    label: labelOf(input),
    precedesPassword: typed.includes(input) && next?.type === "password",
    visible: isVisible(input),
    editable: !input.disabled && !input.readOnly,
  };
}

/** A form, or for inputs outside every form, their document or shadow root. */
export type FormBox = HTMLFormElement | Document | ShadowRoot;

export function boxOf(element: Element): FormBox {
  const form =
    element instanceof HTMLInputElement ||
    element instanceof HTMLSelectElement ||
    element instanceof HTMLButtonElement
      ? element.form
      : element.closest("form");
  return form ?? scopeOf(element);
}

export function boxInputs(box: FormBox): HTMLInputElement[] {
  if (box instanceof HTMLFormElement) {
    return Array.from(box.elements).filter(
      (element) => element instanceof HTMLInputElement,
    );
  }
  return Array.from(box.querySelectorAll("input")).filter(
    (other) => other.form === null,
  );
}

export function formInputs(input: HTMLInputElement): HTMLInputElement[] {
  return boxInputs(boxOf(input));
}

export function isVisible(
  element: Pick<Element, "getBoundingClientRect" | "checkVisibility">,
): boolean {
  const { width, height } = element.getBoundingClientRect();
  return (
    width > 0 &&
    height > 0 &&
    element.checkVisibility({ visibilityProperty: true, opacityProperty: true })
  );
}

export function labelOf(input: HTMLInputElement | HTMLSelectElement): string {
  const parts = [input.getAttribute("aria-label") ?? ""];
  for (const label of input.labels ?? []) parts.push(label.textContent ?? "");
  const scope = scopeOf(input);
  for (const id of input.getAttribute("aria-labelledby")?.split(/\s+/) ?? []) {
    if (id) parts.push(scope.getElementById(id)?.textContent ?? "");
  }
  return parts.join(" ").replace(/\s+/g, " ").trim();
}

function scopeOf(element: Element): Document | ShadowRoot {
  const root = element.getRootNode();
  return root instanceof ShadowRoot ? root : element.ownerDocument;
}
