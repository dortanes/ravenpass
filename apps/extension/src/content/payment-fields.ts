import {
  boxOf,
  type FieldDescription,
  type FormBox,
  isVisible,
  labelOf,
  mentions,
  wordsIn,
} from "./fields.ts";
import paymentWords from "./payment-words.json" with { type: "json" };

export type CardPart =
  | "number"
  | "holder"
  | "given-name"
  | "family-name"
  | "expiry"
  | "exp-month"
  | "exp-year"
  | "security-code";

export type AddressPart =
  | "street"
  | "city"
  | "region"
  | "postal-code"
  | "country";

export type PaymentPart = CardPart | AddressPart;

export type Section = "billing" | "shipping" | null;

/** `contextual` is a part named only by words that also name other fields, which counts only beside a card number. */
export interface PaymentShape {
  readonly part: PaymentPart;
  readonly section: Section;
  readonly contextual: boolean;
}

export type PaymentControl = HTMLInputElement | HTMLSelectElement;

export interface PaymentField extends PaymentShape {
  readonly control: PaymentControl;
  readonly description: FieldDescription;
}

const declaredParts: Readonly<Record<string, PaymentPart>> = {
  "cc-number": "number",
  "cc-name": "holder",
  "cc-given-name": "given-name",
  "cc-family-name": "family-name",
  "cc-exp": "expiry",
  "cc-exp-month": "exp-month",
  "cc-exp-year": "exp-year",
  "cc-csc": "security-code",
  "street-address": "street",
  "address-line1": "street",
  "address-level2": "city",
  "address-level1": "region",
  "postal-code": "postal-code",
  country: "country",
  "country-name": "country",
};

const cardParts: ReadonlySet<PaymentPart> = new Set<CardPart>([
  "number",
  "holder",
  "given-name",
  "family-name",
  "expiry",
  "exp-month",
  "exp-year",
  "security-code",
]);

/** Tokens that place a field without naming what it holds. */
const placingTokens: ReadonlySet<string> = new Set([
  "on",
  "off",
  "billing",
  "shipping",
]);

/** A security code field may hide its digits. */
const typedTypes: ReadonlySet<string> = new Set([
  "text",
  "tel",
  "number",
  "password",
]);
const listType = "select-one";

/** A page with more controls than this is not searched for payment fields. */
const maxControls = 150;

export function isCardPart(part: PaymentPart): part is CardPart {
  return cardParts.has(part);
}

/** `autocomplete` decides when it names a part; a field it gives another purpose is no payment field. */
export function classifyPayment(field: FieldDescription): PaymentShape | null {
  const list = field.type === listType;
  if (
    !field.visible ||
    !field.editable ||
    !(list || typedTypes.has(field.type))
  ) {
    return null;
  }
  const section = sectionOf(field.autocomplete);
  for (const token of field.autocomplete) {
    const part = declaredParts[token];
    if (part) return { part, section, contextual: false };
  }
  if (field.autocomplete.some((token) => !placingTokens.has(token))) {
    return null;
  }
  const part = namedPart(
    wordsIn([field.name, field.id, field.placeholder, field.label].join(" ")),
    list,
  );
  return part && { ...part, section };
}

function namedPart(
  words: readonly string[],
  list: boolean,
): Omit<PaymentShape, "section"> | null {
  const named = (part: PaymentPart, contextual = false) => ({
    part,
    contextual,
  });
  const month = mentions(words, paymentWords.month);
  const year = mentions(words, paymentWords.year);
  if (mentions(words, paymentWords.expiry)) {
    if (month && !year) return named("exp-month");
    if (year && !month) return named("exp-year");
    return list ? null : named("expiry");
  }
  if (!list) {
    if (mentions(words, paymentWords.securityCode)) {
      return named("security-code");
    }
    if (mentions(words, paymentWords.number)) return named("number");
    if (mentions(words, paymentWords.holder)) return named("holder");
  }
  if (month) return named("exp-month", true);
  if (year) return named("exp-year", true);
  if (mentions(words, paymentWords.secondLine)) return null;
  if (mentions(words, paymentWords.postalCode)) {
    return named("postal-code", true);
  }
  if (mentions(words, paymentWords.country)) return named("country", true);
  if (mentions(words, paymentWords.region)) return named("region", true);
  if (mentions(words, paymentWords.city)) return named("city", true);
  if (mentions(words, paymentWords.street)) return named("street", true);
  return null;
}

function sectionOf(tokens: readonly string[]): Section {
  if (tokens.includes("billing")) return "billing";
  if (tokens.includes("shipping")) return "shipping";
  return null;
}

/** Contextual parts count only among controls that hold a card number. */
export function paymentFieldsAmong<Field extends PaymentShape>(
  fields: readonly (Field | null)[],
): Field[] {
  const found = fields.filter((field): field is Field => field !== null);
  const beside = found.some(
    ({ part, contextual }) => part === "number" && !contextual,
  );
  return found.filter(({ contextual }) => beside || !contextual);
}

export function describeControl(control: PaymentControl): FieldDescription {
  const input = control instanceof HTMLInputElement ? control : null;
  return {
    type: control.type,
    autocomplete:
      control
        .getAttribute("autocomplete")
        ?.toLowerCase()
        .split(/\s+/)
        .filter(Boolean) ?? [],
    inputMode: input?.inputMode ?? "",
    maxLength: input && input.maxLength >= 0 ? input.maxLength : null,
    name: control.name,
    id: control.id,
    placeholder: input?.placeholder ?? "",
    label: labelOf(control),
    precedesPassword: false,
    visible: isVisible(control),
    editable: !control.disabled && !input?.readOnly,
  };
}

/** The payment fields of a form, or of the controls outside every form of a document or shadow root. */
export function paymentFieldsIn(box: FormBox): PaymentField[] {
  if (box instanceof HTMLFormElement) {
    return fieldsOf(
      Array.from(box.elements).filter(
        (element): element is PaymentControl =>
          element instanceof HTMLInputElement ||
          element instanceof HTMLSelectElement,
      ),
    );
  }
  return fieldsOf(controlsOf(box).filter((control) => control.form === null));
}

/** The payment fields of a whole document, in and out of forms. */
export function documentPaymentFields(document: Document): PaymentField[] {
  return fieldsOf(controlsOf(document));
}

/** The card field an input is among its form's payment fields; null when it is no card field. */
export function cardFieldOf(input: HTMLInputElement): PaymentField | null {
  const field = paymentFieldsIn(boxOf(input)).find(
    ({ control }) => control === input,
  );
  return field && isCardPart(field.part) ? field : null;
}

/** Whether a document holds a card field, or an address field a page marks as billing. */
export function holdsPaymentFields(document: Document): boolean {
  return documentPaymentFields(document).some(
    ({ part, section }) => isCardPart(part) || section === "billing",
  );
}

function controlsOf(scope: Document | ShadowRoot): PaymentControl[] {
  return Array.from(
    scope.querySelectorAll<PaymentControl>("input:not([type=hidden]), select"),
  );
}

function fieldsOf(controls: readonly PaymentControl[]): PaymentField[] {
  if (controls.length > maxControls) return [];
  return paymentFieldsAmong(
    controls.map((control) => {
      const description = describeControl(control);
      const shape = classifyPayment(description);
      return shape && { ...shape, control, description };
    }),
  );
}
