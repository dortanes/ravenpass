import type { Language } from "../i18n/language.ts";

export const hostScope = ["host"] as const;

export const vaultScope = ["vault"] as const;

export type LimitsKind =
  | "credentials"
  | "identities"
  | "cards"
  | "notes"
  | "seeds"
  | "tags";

export const queryKeys = {
  opening: [...hostScope, "opening"],
  capabilities: [...hostScope, "capabilities"],
  interfaceSize: [...hostScope, "interface-size"],
  appearance: [...hostScope, "appearance"],
  shortcuts: [...hostScope, "shortcuts"],
  clipboardClearing: [...hostScope, "clipboard-clearing"],
  autoLock: [...hostScope, "auto-lock"],
  dockIcon: [...hostScope, "dock-icon"],
  siteIcons: [...hostScope, "site-icons"],
  bankDetails: [...hostScope, "bank-details"],
  screenshots: [...hostScope, "screenshots"],
  systemAutofill: [...hostScope, "system-autofill"],
  identityList: [...hostScope, "identity-list"],
  autoBackup: [...hostScope, "auto-backup"],
  signInStyle: [...hostScope, "sign-in-style"],
  extensionFillConfirmation: [...hostScope, "extension-fill-confirmation"],
  seedWordlist: [...hostScope, "seed-wordlist"],
  supportDetails: [...hostScope, "support-details"],
  thirdPartyNotices: [...hostScope, "third-party-notices"],
  storage: (language: Language) => [...hostScope, "storage", language],
  unlockMethods: [...hostScope, "unlock-methods"],
  credentials: [...vaultScope, "credentials"],
  identities: [...vaultScope, "identities"],
  cards: [...vaultScope, "cards"],
  notes: [...vaultScope, "notes"],
  seeds: [...vaultScope, "seeds"],
  groups: [...vaultScope, "groups"],
  exportStatus: [...vaultScope, "export-status"],
  codeSetup: [...vaultScope, "code-setup"],
  limits: (kind: LimitsKind) => [...vaultScope, "limits", kind],
  identityAddresses: [...vaultScope, "identity-addresses"],
  promptsUnlock: [...hostScope, "prompts-unlock"],
  extensionLinks: [...hostScope, "extension-links"],
  knowsVault: [...hostScope, "knows-vault"],
  confirmationMethods: (request: string) => [
    ...hostScope,
    "unlock-methods",
    request,
  ],
} as const;
