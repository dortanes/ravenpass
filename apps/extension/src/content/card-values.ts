import type { CardValues } from "../link/client.ts";
import type { FieldDescription } from "./fields.ts";
import type { PaymentPart } from "./payment-fields.ts";

/** One option of a list as the page offers it. */
export interface ListOption {
  readonly value: string;
  readonly text: string;
}

/** `languages` name months and countries, the page's own first. */
export interface Entry {
  readonly part: PaymentPart;
  readonly description: FieldDescription;
  /** Null for an input. */
  readonly options: readonly ListOption[] | null;
  readonly languages: readonly string[];
}

/** What a field takes from the values: an input's text or a list's option value; null for nothing to fill. */
export function entryOf(entry: Entry, values: CardValues): string | null {
  const text = textOf(entry, values);
  if (!text) return null;
  if (!entry.options) return text;
  const [year = "", month = ""] = values.expiry.split("-");
  switch (entry.part) {
    case "exp-month":
      return monthOption(Number(month), entry.options, entry.languages);
    case "exp-year":
      return yearOption(year, entry.options);
    case "country":
      return countryOption(text, entry.options, entry.languages);
    default:
      return namedOption(text, entry.options);
  }
}

function textOf(
  { part, description }: Entry,
  { holder, number, expiry, securityCode, billing }: CardValues,
): string {
  const [year = "", month = ""] = expiry.split("-");
  switch (part) {
    case "number":
      return number;
    case "holder":
      return holder;
    case "given-name":
      return holderParts(holder)[0];
    case "family-name":
      return holderParts(holder)[1];
    case "expiry":
      return expiry && expiryText(month, year, description);
    case "exp-month":
      return month;
    case "exp-year":
      return year && yearText(year, description);
    case "security-code":
      return securityCode;
    case "street":
      return billing?.street ?? "";
    case "city":
      return billing?.city ?? "";
    case "region":
      return billing?.region ?? "";
    case "postal-code":
      return billing?.postalCode ?? "";
    case "country":
      return billing?.country ?? "";
  }
}

/** A holder's given name, and the family name after its last space. */
export function holderParts(holder: string): readonly [string, string] {
  const trimmed = holder.trim();
  const at = trimmed.lastIndexOf(" ");
  return at < 0
    ? [trimmed, ""]
    : [trimmed.slice(0, at).trim(), trimmed.slice(at + 1)];
}

const fourDigitYear = /yyyy|гггг/;
const twoDigitYear = /(^|[^yг])(yy|гг)([^yг]|$)/;

/** `MM/YYYY` where the field asks for four year digits, `MMYY` in four characters, else `MM/YY`. */
export function expiryText(
  month: string,
  year: string,
  { placeholder, maxLength }: FieldDescription,
): string {
  const asked = placeholder.toLowerCase();
  if (fourDigitYear.test(asked) || (maxLength !== null && maxLength >= 7)) {
    return `${month}/${year}`;
  }
  if (maxLength === 4) return `${month}${year.slice(2)}`;
  return `${month}/${year.slice(2)}`;
}

/** The year's last two digits where the field holds two characters or asks for two digits. */
function yearText(
  year: string,
  { placeholder, maxLength }: FieldDescription,
): string {
  const asked = placeholder.toLowerCase();
  return maxLength === 2 ||
    (twoDigitYear.test(asked) && !fourDigitYear.test(asked))
    ? year.slice(2)
    : year;
}

/** The option reading as `month`'s number, else naming it in one of `languages`. */
export function monthOption(
  month: number,
  options: readonly ListOption[],
  languages: readonly string[],
): string | null {
  const numbered = options.find(
    ({ value, text }) =>
      leadingNumber(value) === month || leadingNumber(text) === month,
  );
  if (numbered) return numbered.value;
  const day = new Date(2000, month - 1, 1);
  const names = languages.flatMap((language) =>
    (["long", "short"] as const).map((style) =>
      new Intl.DateTimeFormat(language, { month: style })
        .format(day)
        .replace(/\.$/, "")
        .toLowerCase(),
    ),
  );
  return (
    options.find(({ text }) => {
      const shown = text.trim().toLowerCase();
      return names.some((name) => name && shown.startsWith(name));
    })?.value ?? null
  );
}

function leadingNumber(text: string): number | null {
  const digits = /^\s*(\d{1,2})(?!\d)/.exec(text);
  return digits ? Number(digits[1]) : null;
}

/** The option written as the four-digit year or its last two digits. */
export function yearOption(
  year: string,
  options: readonly ListOption[],
): string | null {
  const forms = [year, year.slice(2)];
  return (
    options.find(
      ({ value, text }) =>
        forms.includes(value.trim()) || forms.includes(text.trim()),
    )?.value ?? null
  );
}

/** The option whose value or text is `text`, ignoring case. */
function namedOption(
  text: string,
  options: readonly ListOption[],
): string | null {
  const wanted = text.trim().toLowerCase();
  return (
    options.find(
      ({ value, text: shown }) =>
        value.trim().toLowerCase() === wanted ||
        shown.trim().toLowerCase() === wanted,
    )?.value ?? null
  );
}

/** The option naming `country`, by its own value or text, or by the region code either side holds. */
export function countryOption(
  country: string,
  options: readonly ListOption[],
  languages: readonly string[],
): string | null {
  const named = namedOption(country, options);
  if (named !== null) return named;
  const regionNames = languages.map(
    (language) => new Intl.DisplayNames(language, { type: "region" }),
  );
  const nameOf = (code: string) =>
    /^[A-Za-z]{2}$/.test(code.trim())
      ? regionNames.map((names) =>
          (names.of(code.trim().toUpperCase()) ?? "").toLowerCase(),
        )
      : [];
  const wanted = country.trim().toLowerCase();
  const storedNames = nameOf(country);
  return (
    options.find(
      ({ value, text }) =>
        storedNames.includes(text.trim().toLowerCase()) ||
        nameOf(value).includes(wanted),
    )?.value ?? null
  );
}
