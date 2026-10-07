import type { Appearance } from "../host/appearance.ts";
import type { SaveChoice, SaveDestinations } from "../saving/offer.ts";
import type { LanguageSource, SiteIcon, UnlockMethods } from "../vault-api.ts";

/** A credential as the vault's index lists it, never its secrets; `matches` means it already fills where the owner signs in. */
export interface AutofillCredential {
  readonly id: string;
  readonly label: string;
  readonly account: string;
  /** The site its icon is kept under. */
  readonly site: string;
  readonly matches: boolean;
  /** What tells two accounts on one site apart. */
  readonly tags: readonly string[];
}

/** A passkey for the site asking; `key` names it to the host and `label` is the item holding it. */
export interface AutofillPasskey {
  readonly key: string;
  readonly label: string;
  readonly account: string;
  /** The site its icon is kept under. */
  readonly site: string;
}

export interface AutofillResults {
  readonly scope: "matches" | "vault";
  readonly credentials: readonly AutofillCredential[];
}

export interface SearchOpening extends AutofillResults {
  readonly screen: "search";
  /** Empty for an app. */
  readonly site: string;
  /** Whether the field takes a one-time code. */
  readonly code: boolean;
  /** Whether the system asks for a passkey alone, which a password cannot answer. */
  readonly passkey: boolean;
  /** Empty when the site's sign-in asks for none. */
  readonly passkeys: readonly AutofillPasskey[];
}

export interface UnlockOpening {
  readonly screen: "unlock";
  readonly methods: UnlockMethods;
  /** The vault is already open and the screen only confirms the owner. */
  readonly verify: boolean;
}

export interface SaveReviewOffer extends SaveDestinations {
  /** Empty for an app no site verified. */
  readonly site: string;
  readonly account: string;
  /** The name a new credential takes. */
  readonly name: string;
}

export interface SaveOpening {
  readonly screen: "save";
  readonly offer: SaveReviewOffer;
}

export type AutofillOpening = SearchOpening | UnlockOpening | SaveOpening;

/** `filled` ends the screen; `outdated` means Ravenpass answers only once restarted after an update. */
export type FillOutcome =
  | "filled"
  | "no-code"
  | "not-added"
  | "not-found"
  | "unreachable"
  | "outdated"
  | "failed";

/** `signed` ends the screen; `unverifiable` means the site requires a check the vault cannot offer. */
export type SignInOutcome =
  | "signed"
  | "not-found"
  | "unverifiable"
  | "unreachable"
  | "outdated"
  | "failed";

/** Why Ravenpass stopped a request it worked on outside the page. */
export type AutofillFailure =
  | "unreachable"
  | "outdated"
  | "fill-failed"
  | "code-failed"
  | "sign-in-failed"
  | "save-failed"
  | "unverifiable"
  | "unsupported"
  | "failed";

/** What the host shows while Ravenpass works on the request outside the page. */
export type AutofillWait =
  | { readonly kind: "opening" }
  | { readonly kind: "unlocking" }
  | { readonly kind: "verifying" }
  | { readonly kind: "failed"; readonly failure: AutofillFailure };

/** How an unlock ended; `too-soon` refuses a PIN tried before the delay earned by wrong ones passed. */
export type UnlockOutcome =
  | { readonly kind: "opened" }
  | { readonly kind: "wrong-pin"; readonly attemptsLeft: number }
  | { readonly kind: "pin-removed" }
  | { readonly kind: "too-soon" }
  | { readonly kind: "canceled" }
  | { readonly kind: "failed" };

export type SaveOutcome =
  | { readonly kind: "saved"; readonly created: boolean }
  | { readonly kind: "name-refused" }
  | { readonly kind: "account-refused" }
  | { readonly kind: "failed" };

/** AutofillApi is the autofill host; it keeps who signs in and the values it fills, and the page names items by id or key only. */
export interface AutofillApi extends LanguageSource {
  /** Waits for the vault to open when the screen needs it. */
  open(): Promise<AutofillOpening>;
  /** Reports the page drawn. */
  ready(): void;
  /** Ends the screen without an answer. */
  cancel(): void;
  /** Brings up the keyboard for the focused field. */
  showKeyboard(): void;
  /** Reports each wait, and null once it ends, until the returned function is called. */
  watchWait(onWait: (wait: AutofillWait | null) => void): () => void;
  /** In percent; the natural size where the host keeps none. */
  interfaceSize(): Promise<number>;
  /** `system` where the host keeps none. */
  appearance(): Promise<Appearance>;
  siteIcon(site: string): Promise<SiteIcon>;
  /** An empty query lists what the screen opened on; any other searches the whole vault. */
  search(query: string): Promise<AutofillResults>;
  /** `add` first adds the site or app to the credential. */
  fill(credential: AutofillCredential, add: boolean): Promise<FillOutcome>;
  signIn(passkey: AutofillPasskey): Promise<SignInOutcome>;
  /** An empty `pin` asks for the device's own unlock. */
  unlock(pin: string): Promise<UnlockOutcome>;
  save(choice: SaveChoice): Promise<SaveOutcome>;
}
