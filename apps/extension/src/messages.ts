import type { OneTimeCode, SiteIcon } from "@ravenpass/ui/vault-api.ts";
import * as v from "valibot";
import type { FieldKind } from "./content/fields.ts";
import type {
  CapturedCard,
  CapturedPassword,
  CardOption,
  CardValues,
  CodeSuggestion,
  DesktopState,
  FillValues,
  IdentityFiles,
  LinkFailure,
  LinkStatus,
  PasskeyChoice,
  PasskeyOption,
  PasskeyTarget,
  PendingCapture,
  SaveChoice,
  SavedAs,
  ShareProgress,
  Suggestion,
  SuggestPurpose,
} from "./link/client.ts";
import { cardValues } from "./link/results.ts";
import type { MatchStrength } from "./match-strength.ts";
import type { PageAnswer } from "./passkeys/page-channel.ts";
import type { PageRequest } from "./passkeys/requests.ts";
import type { ColorScheme } from "./toolbar/icon.ts";

export const menuPage = "pages/menu.html";

/** The desktop app forgets a pending capture this long after it arrives. */
export const captureLifetimeMs = 3 * 60_000;

/** What the service worker is asked, over `chrome.runtime.sendMessage`. */
export type Request =
  | { readonly kind: "link"; readonly key: string }
  | { readonly kind: "status" }
  | { readonly kind: "unlink" }
  | { readonly kind: "color-scheme"; readonly scheme: ColorScheme }
  | MenuRequest
  | OfferRequest
  | PasskeyPageRequest;

/** What the isolated script of a tab's top frame relays from the page, under the page's own ids. */
export type PasskeyPageRequest =
  | {
      readonly kind: "passkey-request";
      readonly id: number;
      readonly request: PageRequest;
    }
  | { readonly kind: "passkey-withdraw"; readonly id: number }
  | { readonly kind: "passkey-linked" };

/** `accept` is the destination's `accept` attribute, empty when it takes any file. */
export interface FileDestination {
  readonly accept: string;
}

/** What a field's menu lists: credentials for a sign-in or code field, cards for a card field. */
export type MenuField = FieldKind | "card";

/**
 * `requested` is set when the person asked for the field's menu from the context menu. `confirmed` is set once the
 * person confirmed a fill of a suggestion that is not strong; `remember` once they also asked to remember the page for
 * an `other-site` suggestion.
 */
export type MenuRequest =
  | {
      readonly kind: "menu-open";
      readonly field: MenuField;
      readonly requested: boolean;
    }
  | {
      readonly kind: "file-menu";
      readonly destination: FileDestination | null;
    }
  | { readonly kind: "menu"; readonly token: string }
  | {
      readonly kind: "menu-icon";
      readonly token: string;
      readonly site: string;
    }
  | {
      readonly kind: "menu-fill";
      readonly token: string;
      readonly id: string;
      readonly confirmed: boolean;
      readonly remember?: boolean;
    }
  | { readonly kind: "menu-code"; readonly token: string; readonly id: string }
  | {
      readonly kind: "menu-fill-code";
      readonly token: string;
      readonly id: string;
      readonly confirmed: boolean;
      readonly remember?: boolean;
    }
  | {
      readonly kind: "menu-fill-card";
      readonly token: string;
      readonly id: string;
    }
  | { readonly kind: "card-review"; readonly token: string }
  | { readonly kind: "card-frame" }
  | { readonly kind: "menu-unlock"; readonly token: string }
  | { readonly kind: "menu-identities"; readonly token: string }
  | {
      readonly kind: "menu-share";
      readonly token: string;
      readonly identity: string;
      readonly file: string;
    }
  | {
      readonly kind: "menu-size";
      readonly token: string;
      readonly height: number;
    }
  | { readonly kind: "menu-close"; readonly token: string }
  | { readonly kind: "menu-focus"; readonly token: string }
  | OptionMessage
  | SignInRequest
  | PasskeyMenuRequest;

/** `target` is a credential the passkey card offers, or "new". */
export type PasskeyMenuRequest =
  | ({ readonly kind: "passkey-sign"; readonly token: string } & PasskeyChoice)
  | {
      readonly kind: "passkey-save";
      readonly token: string;
      readonly target: string;
    }
  | { readonly kind: "passkey-elsewhere"; readonly token: string }
  | { readonly kind: "passkey-review"; readonly token: string };

type SignInRequest =
  | { readonly kind: "sign-in-form"; readonly field: FieldKind }
  | { readonly kind: "menu-review"; readonly token: string };

/** Key events reach only the focused frame, so a code field's frame relays Option to its menu. */
interface OptionMessage {
  readonly kind: "menu-option";
  readonly token: string;
  readonly held: boolean;
}

export type OfferRequest =
  | ({ readonly kind: "capture" } & CapturedPassword)
  | ({ readonly kind: "card-capture" } & CapturedCard)
  | { readonly kind: "form-gone" }
  | { readonly kind: "offer-open"; readonly signInForm: boolean }
  | { readonly kind: "offer-review"; readonly token: string }
  | ({ readonly kind: "offer-save"; readonly token: string } & SaveChoice)
  | { readonly kind: "offer-discard"; readonly token: string };

export type LinkRefusal = "not-key" | LinkFailure;

export type MenuContent =
  | CredentialMenuContent
  | CardMenuContent
  | FileMenuContent
  | OfferMenuContent
  | SignInCardContent
  | PasskeyCardContent;

/** `request` is the id the page's script gave the request. */
export interface PasskeyCardContent {
  readonly state: "passkey";
  readonly request: number;
  readonly mode: "get" | "create";
  readonly listing: PasskeyListing;
}

export type PasskeyListing =
  | { readonly state: "sign-in"; readonly passkeys: readonly PasskeyOption[] }
  | {
      readonly state: "save";
      readonly account: string;
      readonly targets: readonly PasskeyTarget[];
    }
  | { readonly state: "excluded" }
  | { readonly state: "locked" }
  | { readonly state: "not-open" };

/** `requested` is set once the person's input on a sign-in field asked for the card. */
export interface SignInCardContent {
  readonly state: "sign-in-card";
  readonly purpose: SuggestPurpose;
  readonly listing: CredentialMenuContent;
  readonly requested: boolean;
}

/** Whether a menu shows as the card in the corner of the tab's top frame. */
export function isCard(content: MenuContent): boolean {
  return (
    content.state === "sign-in-card" ||
    content.state === "offer" ||
    content.state === "passkey"
  );
}

/** Whether a listing holds no credential and no passkey to choose. */
export function listsNothing(listing: CredentialMenuContent): boolean {
  return (
    listing.state === "list" &&
    listing.credentials.length === 0 &&
    (listing.purpose === "code" || listing.passkeys.length === 0)
  );
}

export type Listed<Credential extends Suggestion> = Credential & {
  readonly strength: MatchStrength;
};

export type CredentialMenuContent =
  | {
      readonly state: "list";
      readonly purpose: "sign-in";
      readonly credentials: readonly Listed<Suggestion>[];
      readonly passkeys: readonly PasskeyOption[];
    }
  | {
      readonly state: "list";
      readonly purpose: "code";
      readonly credentials: readonly Listed<CodeSuggestion>[];
    }
  | { readonly state: "locked"; readonly purpose: SuggestPurpose }
  | { readonly state: "not-open"; readonly purpose: SuggestPurpose };

export type CardListing =
  | { readonly state: "list"; readonly cards: readonly CardOption[] }
  | { readonly state: "locked" }
  | { readonly state: "not-open" };

/** A card field's menu. */
export interface CardMenuContent {
  readonly state: "cards";
  readonly listing: CardListing;
}

export type FileMenuContent =
  | ({ readonly state: "files" } & FileDestination)
  | { readonly state: "no-destination" };

/** `expiresAt` is in Unix milliseconds; the pending id never reaches the menu. */
export type SaveOffer = Omit<PendingCapture, "pending"> & {
  readonly expiresAt: number;
};

export interface OfferMenuContent {
  readonly state: "offer";
  readonly offer: SaveOffer;
}

export type MenuFailure = "locked" | "not-open" | "failed";

/** `declined` and `unverifiable` are fills the owner's choice made wait for a confirmation that did not come. */
export type CredentialFailure = MenuFailure | "declined" | "unverifiable";

/** `gone` is a capture that expired, was answered, or lost its link. */
export type OfferFailure =
  | MenuFailure
  | "gone"
  | "invalid-account"
  | "invalid-name";

export type FileFailure = MenuFailure | "unlinked";

export type ShareFailure =
  | FileFailure
  | "declined"
  | "unverifiable"
  | "not-taken";

/** `full` is a credential that holds as many passkeys as it can. */
export type PasskeyFailure =
  | MenuFailure
  | "declined"
  | "unverifiable"
  | "full"
  | "excluded";

export interface Answers {
  link: { ok: true } | { ok: false; reason: LinkRefusal };
  status: LinkStatus;
  unlink: { ok: true };
  "color-scheme": { ok: true };
  "menu-open": { token: string | null };
  "file-menu": { token: string | null };
  /** `site` is the frame's host as the vault writes sites; `host` keeps the origin's port. */
  menu: { site: string; host: string; content: MenuContent };
  "menu-icon": SiteIcon;
  "menu-fill": { ok: true } | { ok: false; reason: CredentialFailure };
  "menu-code":
    | { ok: true; code: OneTimeCode }
    | { ok: false; reason: CredentialFailure };
  "menu-fill-code": { ok: true } | { ok: false; reason: CredentialFailure };
  "menu-fill-card": { ok: true } | { ok: false; reason: CredentialFailure };
  /** Null once the menu closed. */
  "card-review": { listing: CardListing | null };
  "card-frame": { ok: true };
  "menu-unlock": { ok: true } | { ok: false; reason: MenuFailure };
  "menu-identities":
    | { ok: true; identities: readonly IdentityFiles[] }
    | { ok: false; reason: FileFailure };
  "menu-share": { ok: true } | { ok: false; reason: ShareFailure };
  "menu-size": { ok: true };
  "menu-close": { ok: true };
  "menu-focus": { ok: true };
  "menu-option": { ok: true };
  "sign-in-form": { fill: FillValues | null };
  /** Null once the card closed. */
  "menu-review": { listing: CredentialMenuContent | null };
  capture: { ok: true };
  "card-capture": { ok: true };
  "form-gone": { ok: true };
  "offer-open": { token: string | null };
  "offer-review":
    | { ok: true; offer: SaveOffer }
    | { ok: false; reason: OfferFailure };
  "offer-save":
    | { ok: true; saved: SavedAs }
    | { ok: false; reason: OfferFailure };
  "offer-discard": { ok: true };
  /** Null while the request waits; its answer arrives later as a `PasskeyAnswerMessage`. */
  "passkey-request": { answer: PageAnswer | null };
  "passkey-withdraw": { ok: true };
  "passkey-linked": { linked: boolean };
  "passkey-sign": { ok: true } | { ok: false; reason: PasskeyFailure };
  "passkey-save": { ok: true } | { ok: false; reason: PasskeyFailure };
  "passkey-elsewhere": { ok: true };
  /** Null once the card closed. */
  "passkey-review": { listing: PasskeyListing | null };
}

export type Answer = Answers[Request["kind"]];

/** What the service worker relays to one document of a tab; `place-file` is answered with a `Placement`. */
export type FrameMessage =
  | ({ readonly kind: "fill"; readonly token: string } & FillValues)
  | {
      readonly kind: "fill-code";
      readonly token: string;
      readonly code: string;
    }
  | {
      readonly kind: "fill-card";
      readonly token: string;
      readonly values: CardValues;
    }
  | {
      readonly kind: "place-file";
      readonly token: string;
      readonly name: string;
      readonly mediaType: string;
      /** Base64. */
      readonly content: string;
    }
  | {
      readonly kind: "share-progress";
      readonly token: string;
      readonly progress: ShareProgress;
    }
  | {
      readonly kind: "fill-progress";
      readonly token: string;
      readonly progress: ShareProgress;
    }
  | {
      readonly kind: "passkey-progress";
      readonly token: string;
      readonly progress: ShareProgress;
    }
  | {
      readonly kind: "menu-size";
      readonly token: string;
      readonly height: number;
    }
  | { readonly kind: "menu-close"; readonly token: string }
  | { readonly kind: "menu-focus"; readonly token: string }
  | OptionMessage
  | CardHide;

/** `dismissed` keeps the card away from the document until a sign-in field takes focus. */
interface CardHide {
  readonly kind: "card-hide";
  readonly token: string;
  readonly dismissed: boolean;
}

/** Answered with a `CardShown`; `reopen` shows a card the person closed. */
export interface CardShow {
  readonly kind: "card-show";
  readonly token: string;
  readonly reopen: boolean;
}

export interface CardShown {
  readonly shown: boolean;
}

/** Sent to a frame of the tab that reported payment fields, other than the one a card was chosen in, with what that
 * frame may receive; the number and security code are empty for a frame of another origin. */
export interface CardFillFrame {
  readonly kind: "card-fill-frame";
  readonly values: CardValues;
}

/** Sent to the frame a sign-in card opened for; answered with a `Submission` or a `FormPresence`. */
export type FormMessage =
  | ({ readonly kind: "form-sign-in" } & FillValues)
  | { readonly kind: "form-code"; readonly code: string }
  | { readonly kind: "form-present" };

/** `submitted` is never set after a fill that left a field empty. */
export type Submission = v.InferOutput<typeof submission>;

export interface FormPresence {
  readonly present: boolean;
}

export interface Placement {
  readonly placed: boolean;
}

export interface FileMenuOpen {
  readonly kind: "file-menu-open";
}

/** Asks the frame to open the field menu for the input it last saw right-clicked. */
export interface FieldMenuOpen {
  readonly kind: "field-menu-open";
  readonly field: Extract<MenuField, "login" | "code" | "card">;
}

/** Posted by the framing content script to the menu's window; a page can post it too. */
export interface MenuMoved {
  readonly kind: "menu-moved";
}

export const menuMoved: MenuMoved = { kind: "menu-moved" };

export interface OfferShow {
  readonly kind: "offer-show";
}

export interface PasskeyAnswerMessage {
  readonly kind: "passkey-answer";
  readonly id: number;
  readonly answer: PageAnswer;
}

/** The desktop app's state as a menu port watches it. */
export type VaultState = DesktopState | "unlinked";

/** Pushed over a `vaultStatePort` port: once connected, then on every change. */
export interface VaultStateMessage {
  readonly kind: "vault-state";
  readonly state: VaultState;
}

export const vaultStatePort = "vault-state";

const frameMessageKinds = [
  "fill",
  "fill-code",
  "fill-card",
  "place-file",
  "share-progress",
  "fill-progress",
  "passkey-progress",
  "menu-size",
  "menu-close",
  "menu-focus",
  "menu-option",
  "card-hide",
] as const satisfies readonly FrameMessage["kind"][];

const formMessageKinds = [
  "form-sign-in",
  "form-code",
  "form-present",
] as const satisfies readonly FormMessage["kind"][];

const kind = <Kind extends string>(name: Kind) =>
  v.object({ kind: v.literal(name) });

const cardShow = kind("card-show");
const fileMenuOpen = kind("file-menu-open");
const fieldMenuOpen = v.object({
  kind: v.literal("field-menu-open"),
  field: v.picklist(["login", "code", "card"]),
});
const cardFillFrame = v.object({
  kind: v.literal("card-fill-frame"),
  values: cardValues,
}) satisfies v.GenericSchema<unknown, CardFillFrame>;
const menuMovedMessage = kind("menu-moved");
const offerShow = kind("offer-show");
const passkeyAnswer = kind("passkey-answer");

const frameMessage = v.object({
  kind: v.picklist(frameMessageKinds),
  token: v.string(),
});

const formMessage = v.object({ kind: v.picklist(formMessageKinds) });

const vaultStateMessage = v.object({
  kind: v.literal("vault-state"),
  state: v.picklist(["unlocked", "locked", "not-open", "unlinked"]),
}) satisfies v.GenericSchema<unknown, VaultStateMessage>;

const submission = v.pipe(
  v.object({ submitted: v.boolean(), password: v.boolean() }),
  v.readonly(),
);

const shown = v.object({ shown: v.literal(true) });
const present = v.object({ present: v.literal(true) });
const placed = v.object({ placed: v.literal(true) });

export function isCardShow(message: unknown): message is CardShow {
  return v.is(cardShow, message);
}

export function isFormMessage(message: unknown): message is FormMessage {
  return v.is(formMessage, message);
}

export function isFileMenuOpen(message: unknown): message is FileMenuOpen {
  return v.is(fileMenuOpen, message);
}

export function isFieldMenuOpen(message: unknown): message is FieldMenuOpen {
  return v.is(fieldMenuOpen, message);
}

export function isCardFillFrame(message: unknown): message is CardFillFrame {
  return v.is(cardFillFrame, message);
}

export function isMenuMoved(message: unknown): message is MenuMoved {
  return v.is(menuMovedMessage, message);
}

export function isOfferShow(message: unknown): message is OfferShow {
  return v.is(offerShow, message);
}

/** Checks only the kind; the passkey bridge reads the id and the answer. */
export function isPasskeyAnswer(message: unknown): message is {
  readonly kind: "passkey-answer";
  readonly id: unknown;
  readonly answer: unknown;
} {
  return v.is(passkeyAnswer, message);
}

/** Extension pages also receive every request other pages send, which the token tells apart. */
export function isFrameMessage(
  message: unknown,
  token: string,
): message is FrameMessage {
  return v.is(frameMessage, message) && message.token === token;
}

export function isVaultStateMessage(
  message: unknown,
): message is VaultStateMessage {
  return v.is(vaultStateMessage, message);
}

export function isSubmission(answer: unknown): answer is Submission {
  return v.is(submission, answer);
}

/** Whether the top frame answered a `CardShow` by showing the card. */
export function isShown(answer: unknown): boolean {
  return v.is(shown, answer);
}

/** Whether a frame answered `form-present` with its form still on the page. */
export function isPresent(answer: unknown): boolean {
  return v.is(present, answer);
}

/** Whether a frame answered `place-file` by placing the file. */
export function isPlaced(answer: unknown): boolean {
  return v.is(placed, answer);
}

/** Asks the service worker; its `null` answer to a request it could not serve rejects. */
export async function ask<Kind extends Request["kind"]>(
  request: Extract<Request, { kind: Kind }>,
): Promise<Answers[Kind]> {
  const answer: Answers[Kind] | null =
    await chrome.runtime.sendMessage(request);
  if (answer === null) {
    throw new Error(
      `The service worker could not serve the ${request.kind} request.`,
    );
  }
  return answer;
}
