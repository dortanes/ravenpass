import type { CredentialInput, CredentialSummary } from "../vault-api.ts";

export const emptyCredential: CredentialInput = {
  label: "",
  websites: [],
  login: "",
  email: "",
  password: "",
  notes: "",
  totp: "",
  apps: [],
  tags: [],
};

export interface CredentialDraft {
  input: CredentialInput;
  /** `PasskeyView` IDs the same save removes. */
  removedPasskeys: string[];
}

/** Drops blank website rows but keeps others as typed, so the vault can refuse them with a reason. */
export function readyToSave(input: CredentialInput): CredentialInput {
  return {
    ...input,
    label: input.label.trim(),
    websites: input.websites.filter((website) => website.trim()),
  };
}

export function accountOf(
  credential: Pick<CredentialSummary, "login" | "email">,
): string {
  return credential.login || credential.email;
}
