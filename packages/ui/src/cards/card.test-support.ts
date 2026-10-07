import type { CardInput } from "../vault-api.ts";

export const emptyCard: CardInput = {
  label: "",
  holder: "",
  number: "",
  expiry: "",
  securityCode: "",
  pin: "",
  network: "",
  bankName: "",
  bankSite: "",
  tags: [],
  color: "",
  billing: null,
  billingLink: null,
  notes: "",
};
