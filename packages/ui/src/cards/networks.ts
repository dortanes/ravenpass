import creditCardType from "credit-card-type";
import type { CardNetwork } from "../vault-api.ts";

/** In the vault's order. */
export const cardNetworks: readonly Exclude<CardNetwork, "">[] = [
  "visa",
  "mastercard",
  "american-express",
  "discover",
  "diners-club",
  "jcb",
  "unionpay",
  "maestro",
  "mir",
  "elo",
  "hiper",
  "hipercard",
  "troy",
  "verve",
  "naranja",
];

export function isCardNetwork(value: string): value is CardNetwork {
  return value === "" || (cardNetworks as readonly string[]).includes(value);
}

export function digitsOf(text: string): string {
  return text.replace(/\D/g, "");
}

/** Empty while more than one network, or none, fits the digits. */
export function detectNetwork(number: string): CardNetwork {
  const digits = digitsOf(number);
  if (!digits) return "";
  const [match, ...others] = creditCardType(digits);
  if (!match || others.length > 0) return "";
  return isCardNetwork(match.type) ? match.type : "";
}

/** Whether `digits` are 12 to 19 digits as long as a number of their network, when one fits them. */
export function cardNumberFits(digits: string): boolean {
  if (!/^\d{12,19}$/.test(digits)) return false;
  const network = detectNetwork(digits);
  return (
    !network ||
    creditCardType.getTypeInfo(network).lengths.includes(digits.length)
  );
}

/** The network's own name, such as `American Express`; empty for none. */
export function networkName(network: CardNetwork): string {
  return network ? creditCardType.getTypeInfo(network).niceType : "";
}
