import { isCardNetwork } from "@ravenpass/ui/cards/networks.ts";
import {
  type SignInStyle,
  signInStyleOf,
} from "@ravenpass/ui/extensions/sign-in-style.ts";
import { matchLanguage } from "@ravenpass/ui/i18n/language.ts";
import { isDocumentType } from "@ravenpass/ui/identities/identity.ts";
import type { SaveTarget } from "@ravenpass/ui/saving/offer.ts";
import type { OneTimeCode, SiteIcon } from "@ravenpass/ui/vault-api.ts";
import * as v from "valibot";
import { base64url } from "../passkeys/page-wire.ts";

// A session carries one request; every frame names this id.
export const requestId = 1;

const greetingEntries = {
  language: v.pipe(v.string(), v.transform(matchLanguage)),
  signIn: v.pipe(v.string(), v.transform<string, SignInStyle>(signInStyleOf)),
};

export const sessionGreeting = v.pipe(v.object(greetingEntries), v.readonly());

export const linkConfirmation = v.pipe(
  v.object({ type: v.literal("linked"), ...greetingEntries }),
  v.readonly(),
);

const frameId = v.literal(requestId);

// Tried in order; a head's raw parts carry a file, or the UTF-8 JSON result when it has none.
export const sessionFrame = v.union([
  v.object({ id: frameId, progress: v.unknown() }),
  v.object({ id: frameId, error: v.string() }),
  v.object({
    id: frameId,
    parts: v.unknown(),
    result: v.exactOptional(v.unknown()),
  }),
  v.object({ id: frameId, result: v.unknown() }),
]);

export const partCount = v.pipe(v.number(), v.safeInteger(), v.minValue(0));

export const shareProgress = v.picklist([
  "confirm-on-device",
  "confirm-in-ravenpass",
]);

export const desktopStatus = v.pipe(
  v.object({ vault: v.picklist(["unlocked", "locked"]) }),
  v.transform(({ vault }) => vault),
);

const count = v.pipe(v.number(), v.safeInteger(), v.gtValue(0));

const suggestionEntries = {
  id: v.string(),
  label: v.string(),
  account: v.string(),
  site: v.string(),
  // Whether the origin matched the host, not only its registrable domain.
  exact: v.boolean(),
  // What tells two accounts on one site apart; an older Ravenpass sends none.
  tags: v.optional(v.array(v.string()), []),
};

export const suggestion = v.pipe(v.object(suggestionEntries), v.readonly());

export const codeSuggestion = v.pipe(
  v.object({ ...suggestionEntries, digits: count, period: count }),
  v.readonly(),
);

export const suggestions = v.pipe(
  v.object({ credentials: v.array(suggestion) }),
  v.transform(({ credentials }) => credentials),
);

export const codeSuggestions = v.pipe(
  v.object({ credentials: v.array(codeSuggestion) }),
  v.transform(({ credentials }) => credentials),
);

export const fillValues = v.pipe(
  v.object({ login: v.string(), email: v.string(), password: v.string() }),
  v.readonly(),
);

export const oneTimeCode = v.object({
  code: v.string(),
  digits: v.number(),
  period: v.number(),
  expiresAt: v.number(),
}) satisfies v.GenericSchema<unknown, OneTimeCode>;

export const siteIcon = v.object({
  image: v.string(),
  tint: v.string(),
}) satisfies v.GenericSchema<unknown, SiteIcon>;

const saveTarget = v.object({
  credential: v.string(),
  label: v.string(),
  account: v.string(),
  action: v.picklist(["update", "add-site"]),
  // An older Ravenpass sends none.
  tags: v.optional(v.array(v.string()), []),
}) satisfies v.GenericSchema<unknown, SaveTarget>;

const cardNetwork = v.pipe(v.string(), v.guard(isCardNetwork));

const cardFace = v.pipe(
  v.object({ network: cardNetwork, lastFour: v.string() }),
  v.readonly(),
);

const offerEntries = {
  // An older Ravenpass offers passwords alone and names no kind.
  kind: v.optional(v.picklist(["password", "card"]), "password"),
  site: v.string(),
  account: v.string(),
  name: v.string(),
  // The typed card of a card's offer.
  card: v.optional(v.nullable(cardFace), null),
  targets: v.pipe(v.array(saveTarget), v.readonly()),
  suggested: v.string(),
};

export const pendingCapture = v.pipe(
  v.object({
    state: v.picklist(["ready", "locked"]),
    pending: v.string(),
    ...offerEntries,
  }),
  // `suggested` is one of the targets' credentials, or empty for a new credential.
  v.check(
    ({ suggested, targets }) =>
      suggested === "" ||
      targets.some(({ credential }) => credential === suggested),
  ),
  v.readonly(),
);

export const captureOffer = v.union([
  v.pipe(
    v.object({ state: v.literal("none"), ...offerEntries }),
    v.transform(() => ({ state: "none" }) as const),
  ),
  pendingCapture,
]);

export const cardOption = v.pipe(
  v.object({
    id: v.string(),
    label: v.string(),
    bankName: v.string(),
    // The bank's host, empty for none.
    site: v.string(),
    network: cardNetwork,
    lastFour: v.string(),
    color: v.string(),
    // The last day of the expiry month, YYYY-MM-DD, empty for none.
    expiresOn: v.string(),
  }),
  v.readonly(),
);

export const cardOptions = v.pipe(
  v.object({ cards: v.array(cardOption) }),
  v.transform(({ cards }) => cards),
);

const billingAddress = v.pipe(
  v.object({
    street: v.string(),
    city: v.string(),
    region: v.string(),
    postalCode: v.string(),
    country: v.string(),
  }),
  v.readonly(),
);

export const cardValues = v.pipe(
  v.object({
    holder: v.string(),
    number: v.string(),
    // YYYY-MM, empty for none.
    expiry: v.string(),
    securityCode: v.string(),
    billing: v.nullable(billingAddress),
  }),
  v.readonly(),
);

export const savedAs = v.pipe(
  v.object({ saved: v.picklist(["created", "updated"]) }),
  v.transform(({ saved }) => saved),
);

const passkeyChoiceEntries = {
  credential: v.string(),
  credentialId: base64url,
};

export const passkeyOption = v.pipe(
  v.object({
    ...passkeyChoiceEntries,
    account: v.string(),
    label: v.string(),
  }),
  v.readonly(),
);

export const passkeyOptions = v.pipe(
  v.object({ passkeys: v.array(passkeyOption) }),
  v.transform(({ passkeys }) => passkeys),
);

export const passkeyTarget = v.pipe(
  v.object({
    credential: v.string(),
    label: v.string(),
    // The credential's login, else its email.
    account: v.string(),
    // An older Ravenpass sends none.
    tags: v.optional(v.array(v.string()), []),
  }),
  v.readonly(),
);

export const passkeyTargets = v.pipe(
  v.object({
    targets: v.pipe(v.array(passkeyTarget), v.readonly()),
    excluded: v.boolean(),
  }),
  v.readonly(),
);

const fileDocument = v.pipe(
  v.object({
    type: v.pipe(v.string(), v.guard(isDocumentType)),
    label: v.string(),
  }),
  v.readonly(),
);

// A JPEG in base64, empty for none.
const thumbnail = v.string();

const fileEntries = {
  id: v.string(),
  name: v.string(),
  mediaType: v.string(),
  thumbnail,
};

export const identityFile = v.union([
  // A photo's id is "photo".
  v.pipe(
    v.object({
      ...fileEntries,
      kind: v.literal("photo"),
      document: v.optional(v.undefined()),
    }),
    v.transform(({ document: _document, ...photo }) => photo),
    v.readonly(),
  ),
  v.pipe(
    v.object({
      ...fileEntries,
      kind: v.literal("scan"),
      document: fileDocument,
    }),
    v.readonly(),
  ),
]);

export const identity = v.pipe(
  v.object({
    id: v.string(),
    label: v.string(),
    thumbnail,
    files: v.pipe(v.array(identityFile), v.readonly()),
  }),
  v.readonly(),
);

export const identityFiles = v.pipe(
  v.object({ identities: v.array(identity) }),
  v.transform(({ identities }) => identities),
);

export const sharedFileHead = v.object({
  name: v.string(),
  mediaType: v.string(),
  size: v.number(),
});
