import {
  cardNumberFits,
  detectNetwork,
  digitsOf,
} from "@ravenpass/ui/cards/networks.ts";
import type { CapturedCard } from "../link/client.ts";
import type { PaymentPart } from "./payment-fields.ts";

/** A payment field as submitted; a list's `text` is its chosen option's. */
export interface SubmittedPayment {
  readonly part: PaymentPart;
  readonly value: string;
  readonly text: string;
  /** Whether the person typed `value`, not a fill or a page script. */
  readonly typed: boolean;
}

/** Captures only a card number the person typed; the other values go along however they were filled. */
export function cardCaptureOf(
  fields: readonly SubmittedPayment[],
): CapturedCard | null {
  const number = filled(fields, "number");
  const digits = digitsOf(number?.value ?? "");
  if (!number?.typed || !cardNumberFits(digits)) return null;
  const securityCode = digitsOf(filled(fields, "security-code")?.value ?? "");
  return {
    holder: holderOf(fields),
    number: digits,
    expiry: expiryOf(fields),
    securityCode: /^\d{3,4}$/.test(securityCode) ? securityCode : "",
    network: detectNetwork(digits),
  };
}

/** The first field of `part` holding a value. */
function filled(
  fields: readonly SubmittedPayment[],
  part: PaymentPart,
): SubmittedPayment | undefined {
  return fields.find((field) => field.part === part && field.value.trim());
}

function holderOf(fields: readonly SubmittedPayment[]): string {
  const text = (part: PaymentPart) => filled(fields, part)?.value.trim() ?? "";
  return (
    text("holder") ||
    [text("given-name"), text("family-name")].filter(Boolean).join(" ")
  );
}

/** `YYYY-MM` from one expiry field, or from a month and a year; empty when unreadable. */
function expiryOf(fields: readonly SubmittedPayment[]): string {
  const field = (part: PaymentPart) =>
    fields.find((candidate) => candidate.part === part);
  const whole = field("expiry");
  if (whole) {
    const [, month, year] =
      /^(\d{1,2})\s*[/\-.]?\s*(\d{4}|\d{2})$/.exec(whole.value.trim()) ?? [];
    return storedExpiry(month, year);
  }
  const month = field("exp-month");
  const year = field("exp-year");
  return storedExpiry(
    month && (leadingDigits(month.value) ?? leadingDigits(month.text)),
    year && (fullYear(year.value) ?? fullYear(year.text)),
  );
}

function storedExpiry(
  month: string | undefined | null,
  year: string | undefined | null,
): string {
  if (!month || !year) return "";
  const number = Number(month);
  if (number < 1 || number > 12) return "";
  const full = year.length === 2 ? `20${year}` : year;
  return `${full}-${String(number).padStart(2, "0")}`;
}

function leadingDigits(text: string): string | null {
  return /^\s*(\d{1,2})(?!\d)/.exec(text)?.[1] ?? null;
}

function fullYear(text: string): string | null {
  return /^\s*(\d{4}|\d{2})\s*$/.exec(text)?.[1] ?? null;
}
