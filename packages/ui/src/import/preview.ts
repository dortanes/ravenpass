import type { ImportFormat, ImportOrigin, ImportReason } from "../vault-api.ts";

const importFormats: readonly ImportFormat[] = [
  "json",
  "encrypted-json",
  "zip",
  "encrypted-zip",
  "csv",
];

const encryptedFormats: readonly ImportFormat[] = [
  "encrypted-json",
  "encrypted-zip",
];

const importOrigins: readonly ImportOrigin[] = [
  "login",
  "alias",
  "card",
  "identity",
  "passport",
  "drivers-license",
  "note",
  "ssh-key",
  "bank-account",
];

const importReasons: readonly ImportReason[] = [
  "too-long",
  "unnamed",
  "unsupported",
];

export function isImportFormat(value: string): value is ImportFormat {
  return (importFormats as readonly string[]).includes(value);
}

export function isEncryptedFormat(format: ImportFormat): boolean {
  return encryptedFormats.includes(format);
}

export function isImportOrigin(value: string): value is ImportOrigin {
  return (importOrigins as readonly string[]).includes(value);
}

export function isImportReason(value: string): value is ImportReason {
  return (importReasons as readonly string[]).includes(value);
}
