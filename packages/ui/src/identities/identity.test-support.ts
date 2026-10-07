import type { IdentityInput } from "../vault-api.ts";

export const emptyIdentity: IdentityInput = {
  label: "",
  fullName: "",
  birthday: "",
  emails: [],
  phones: [],
  addresses: [],
  documents: [],
  notes: "",
  photo: "",
  tags: [],
};
