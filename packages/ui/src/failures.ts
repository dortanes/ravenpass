// The Android bridge prefixes the host's message with its own words, so the code ends the message.
const reportedFailure = /(?:^|\s)ravenpass:([a-z][a-z0-9-]*)$/;

/** Empty when the error does not come from the host service. */
export function failureCode(error: unknown): string {
  const message =
    error instanceof Error
      ? error.message
      : typeof error === "string"
        ? error
        : "";
  return reportedFailure.exec(message.trim())?.[1] ?? "";
}

/** Failures decided before any write, which leave the vault file matching the open vault. */
const refusals = new Set([
  "invalid-item",
  "invalid-code",
  "no-code",
  "code-setup-expired",
  "resource-limit",
  "passkeys-full",
  "breach-checks-off",
  "breach-check-unreachable",
  "item-unreadable",
  "photo-unsupported",
  "photo-too-large",
  "scan-unsupported",
  "scan-too-large",
  "scan-limit",
  "group-unknown",
  "group-name-invalid",
  "vault-locked",
  "vault-unknown",
  "import-not-active",
  "import-empty",
  "import-limit",
]);

/** An unknown failure counts as a write that may have happened. */
export function refusedBeforeWriting(error: unknown): boolean {
  return refusals.has(failureCode(error));
}
