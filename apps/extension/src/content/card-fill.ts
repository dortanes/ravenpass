import type { CardValues } from "../link/client.ts";
import { entryOf } from "./card-values.ts";
import { boxOf } from "./fields.ts";
import { announceChange, setValue } from "./fill.ts";
import {
  describeControl,
  documentPaymentFields,
  isCardPart,
  type PaymentField,
  paymentFieldsIn,
} from "./payment-fields.ts";

/** `form` is the form a card was chosen in, whose address fields fill unless marked shipping; `billing` is any other
 * frame, whose address fields fill only when marked billing. */
type FillReach = "form" | "billing";

const setNativeSelection = Object.getOwnPropertyDescriptor(
  HTMLSelectElement.prototype,
  "value",
)?.set;

/** Fills the form of the input a card was chosen under; an input Ravenpass did not recognise takes the number. */
export function fillChosenForm(
  input: HTMLInputElement,
  values: CardValues,
): void {
  const form = paymentFieldsIn(boxOf(input));
  const fields: PaymentField[] = form.some(({ control }) => control === input)
    ? form
    : [
        ...form,
        {
          part: "number",
          section: null,
          contextual: false,
          control: input,
          description: describeControl(input),
        },
      ];
  fillPayment(input.ownerDocument, fields, values, input, "form");
}

/** Fills a frame other than the one a card was chosen in. */
export function fillOtherFrame(document: Document, values: CardValues): void {
  fillPayment(
    document,
    documentPaymentFields(document),
    values,
    null,
    "billing",
  );
}

/** Fills the field `opened` always, other text fields only when empty, and lists to their matching option. */
function fillPayment(
  document: Document,
  fields: readonly PaymentField[],
  values: CardValues,
  opened: Element | null,
  reach: FillReach,
): void {
  const languages = pageLanguages(document);
  for (const field of fields) {
    if (!reaches(field, reach)) continue;
    const { control, description, part } = field;
    const list = control instanceof HTMLSelectElement ? control : null;
    const entry = entryOf(
      {
        part,
        description,
        options: list
          ? Array.from(list.options, ({ value, text }) => ({ value, text }))
          : null,
        languages,
      },
      values,
    );
    if (entry === null) continue;
    if (list) {
      if (list.value === entry) continue;
      setNativeSelection?.call(list, entry);
      announceChange(list);
    } else if (control instanceof HTMLInputElement) {
      if (control !== opened && control.value !== "") continue;
      setValue(control, entry);
    }
  }
}

function reaches({ part, section }: PaymentField, reach: FillReach): boolean {
  if (isCardPart(part)) return true;
  return reach === "form" ? section !== "shipping" : section === "billing";
}

/** The page's language first, then English, each as `Intl` accepts it. */
function pageLanguages(document: Document): string[] {
  const languages = [document.documentElement.lang, "en"].filter(Boolean);
  return languages.filter((language) => {
    try {
      return Intl.getCanonicalLocales(language).length > 0;
    } catch {
      return false;
    }
  });
}
