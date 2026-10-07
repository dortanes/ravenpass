import type { MaskitoOptions, MaskitoPreprocessor } from "@maskito/core";
import { readableColor } from "color2k";
import creditCardType from "credit-card-type";
import { addDays } from "date-fns/addDays";
import { format } from "date-fns/format";
import { isValid } from "date-fns/isValid";
import { lastDayOfMonth } from "date-fns/lastDayOfMonth";
import { parse } from "date-fns/parse";
import { subYears } from "date-fns/subYears";
import {
  type PaymentType,
  validateCardNumber,
} from "react-svg-credit-card-payment-icons";
import { blankAddress } from "../identities/identity.ts";
import type {
  Address,
  AddressLink,
  Card,
  CardInput,
  CardNetwork,
} from "../vault-api.ts";
import { digitsOf, networkName } from "./networks.ts";

const defaultGaps = [4, 8, 12];
const longestNumber = 19;

function gapsOf(network: CardNetwork): number[] {
  return network ? creditCardType.getTypeInfo(network).gaps : defaultGaps;
}

const networkLogos: Partial<Record<CardNetwork, PaymentType>> = {
  visa: "Visa",
  mastercard: "Mastercard",
  "american-express": "Amex",
  discover: "Discover",
  "diners-club": "Diners",
  jcb: "JCB",
  unionpay: "UnionPay",
  maestro: "Maestro",
  mir: "Mir",
  elo: "Elo",
  hiper: "Hiper",
  hipercard: "Hipercard",
};

/** Null for a network `react-svg-credit-card-payment-icons` draws no logo for. */
export function networkLogo(network: CardNetwork): PaymentType | null {
  return networkLogos[network] ?? null;
}

/** Mastercard and Maestro circles merge into one shape in a single colour; other logos print white. */
export function logoInColour(network: CardNetwork): boolean {
  return network === "mastercard" || network === "maestro";
}

/** Splits where the network spaces digits, four by four for an unknown network. */
export function numberGroups(number: string, network: CardNetwork): string[] {
  const digits = digitsOf(number);
  const groups: string[] = [];
  let start = 0;
  for (const gap of gapsOf(network)) {
    if (gap >= digits.length) break;
    groups.push(digits.slice(start, gap));
    start = gap;
  }
  if (start < digits.length) groups.push(digits.slice(start));
  return groups;
}

/** Hides every digit but the last four. */
export function maskedGroups(number: string, network: CardNetwork): string[] {
  const digits = digitsOf(number);
  const shownFrom = digits.length - 4;
  let position = 0;
  return numberGroups(digits, network).map((group) =>
    Array.from(group, (digit) => (position++ < shownFrom ? "•" : digit)).join(
      "",
    ),
  );
}

/** cardLine is what a card's row says under its name: the bank, the payment network and the last four digits. */
export function cardLine(
  bankName: string,
  network: CardNetwork,
  lastFour: string,
): string {
  const issuer = [bankName, networkName(network)].filter(Boolean).join(" · ");
  return [issuer, lastFour ? `•• ${lastFour}` : ""].filter(Boolean).join(" ");
}

/** Only a complete number of a known network is checked; UnionPay numbers carry no Luhn digit. */
export function failsCheck(number: string, network: CardNetwork): boolean {
  const digits = digitsOf(number);
  if (!network || network === "unionpay") return false;
  if (!creditCardType.getTypeInfo(network).lengths.includes(digits.length)) {
    return false;
  }
  return !validateCardNumber(digits);
}

export function digitsWithin(
  value: string,
  fewest: number,
  most: number,
  optional: boolean,
): boolean {
  if (!value) return optional;
  return /^\d+$/.test(value) && value.length >= fewest && value.length <= most;
}

const onlyDigits: MaskitoPreprocessor = ({ elementState, data }) => ({
  elementState,
  data: digitsOf(data),
});

/** `longest` applies only to an unknown network; pasted separators are dropped. */
export function numberMask(
  network: CardNetwork,
  longest = longestNumber,
): MaskitoOptions {
  const gaps = new Set(gapsOf(network));
  const length = network
    ? Math.max(...creditCardType.getTypeInfo(network).lengths)
    : longest;
  const mask: (string | RegExp)[] = [];
  for (let index = 0; index < length; index++) {
    if (index > 0 && gaps.has(index)) mask.push(" ");
    mask.push(/\d/);
  }
  return { mask, preprocessors: [onlyDigits] };
}

export function digitsMask(longest: number): MaskitoOptions {
  return {
    mask: new RegExp(`^\\d{0,${longest}}$`),
    preprocessors: [onlyDigits],
  };
}

const storedExpiry = /^\d{4}-\d{2}$/;
const typedExpiry = /^(\d{2})\/(\d{2})$/;

function expiryMonth(stored: string): Date | null {
  if (!storedExpiry.test(stored)) return null;
  const month = parse(stored, "yyyy-MM", new Date(0));
  return isValid(month) ? month : null;
}

/** Stored `YYYY-MM` as printed `MM/YY`; empty when unreadable. */
export function expiryDisplay(stored: string): string {
  const month = expiryMonth(stored);
  return month ? format(month, "MM/yy") : "";
}

/** Typed `MM/YY` as stored `YYYY-MM` in the 2000s; empty text is no expiry, other text is null. */
export function expiryStored(typed: string): string | null {
  if (!typed) return "";
  const [, month, year] = typedExpiry.exec(typed) ?? [];
  if (!month || !year) return null;
  const number = Number(month);
  if (number < 1 || number > 12) return null;
  return `20${year}-${month}`;
}

/** The last valid day, `YYYY-MM-DD`, or empty. */
export function expiryEnd(stored: string): string {
  const month = expiryMonth(stored);
  return month ? format(lastDayOfMonth(month), "yyyy-MM-dd") : "";
}

/** Cards print no issue date; banks issue them for three to five years. */
const cardTermYears = 5;

/** The issue day of a card of the longest usual term, `YYYY-MM-DD`, or empty. */
export function expiryStart(stored: string): string {
  const month = expiryMonth(stored);
  return month
    ? format(
        addDays(subYears(lastDayOfMonth(month), cardTermYears), 1),
        "yyyy-MM-dd",
      )
    : "";
}

/** For a card with no network and no colour of its own. */
export const neutralCardColor = "#3a3a3c";

const networkColors: Record<Exclude<CardNetwork, "">, string> = {
  visa: "#1a1f71",
  mastercard: "#eb001b",
  "american-express": "#006fcf",
  discover: "#ff6000",
  "diners-club": "#0079be",
  jcb: "#0b4ea2",
  unionpay: "#d7282f",
  maestro: "#0099df",
  mir: "#0f754e",
  elo: "#00a4e0",
  hiper: "#f37421",
  hipercard: "#b3131b",
  troy: "#00a3b4",
  verve: "#00425f",
  naranja: "#fe5000",
};

export function cardColor(color: string, network: CardNetwork): string {
  if (color) return color;
  return network ? networkColors[network] : neutralCardColor;
}

export function readableOn(color: string): string {
  return readableColor(color);
}

export function concealedDigits(value: string): string {
  return "•".repeat(value.length);
}

export type BillingChoice =
  | { kind: "none" }
  | { kind: "linked"; link: AddressLink }
  | { kind: "own"; address: Address };

/** A link whose address is gone reads as none, and the next save drops it. */
export function billingChoiceOf(card: Card | undefined): BillingChoice {
  if (card?.billingLink && card.linked) {
    return { kind: "linked", link: card.billingLink };
  }
  if (card?.billing) return { kind: "own", address: card.billing };
  return { kind: "none" };
}

/** A blank own address saves as none; an own address never carries an id or a label. */
export function billingOf(
  choice: BillingChoice,
): Pick<CardInput, "billing" | "billingLink"> {
  switch (choice.kind) {
    case "linked":
      return { billing: null, billingLink: choice.link };
    case "own":
      return blankAddress(choice.address)
        ? { billing: null, billingLink: null }
        : {
            billing: { ...choice.address, id: "", label: "" },
            billingLink: null,
          };
    case "none":
      return { billing: null, billingLink: null };
  }
}

export const cardSwatches: readonly string[] = [
  "#2c2c2e",
  "#1e3a8a",
  "#0f766e",
  "#15803d",
  "#b45309",
  "#b91c1c",
  "#7e22ce",
  "#be185d",
];
