import { browserStorage } from "./browser-storage.ts";

/** A hint shown once on the device. */
export type Hint = "card-bank-site" | "credential-site";

const prefix = "ravenpass.hint.";

/** hintSeen tells whether a hint was already shown. A storage that cannot be read shows it. */
export function hintSeen(hint: Hint, store = browserStorage()): boolean {
  try {
    return store?.getItem(prefix + hint) === "seen";
  } catch {
    return false;
  }
}

/** markHintSeen records that a hint was shown. A storage that refuses the write shows it again. */
export function markHintSeen(hint: Hint, store = browserStorage()): void {
  try {
    store?.setItem(prefix + hint, "seen");
  } catch {
    // The hint is shown again next time.
  }
}
