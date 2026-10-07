import type { MessageKey } from "../i18n/messages.ts";
import type {
  Confirmation,
  UnlockMethods,
  UnlockRequester,
  VerificationReason,
} from "../vault-api.ts";

/** What a confirmation card shows and offers for one request. */
export interface CardPlan {
  title: MessageKey;
  description: MessageKey;
  values: Record<string, string>;
  /** Null when the card does not take the PIN. */
  pin: { confirm: MessageKey } | null;
  /** Null when the device's own authentication is not offered. */
  biometry: MessageKey | null;
  /** The recovery key is entered in the main window. */
  recovery: boolean;
  /** True when recovery is the only way in on the device. */
  noMethod: boolean;
  /** Shown for a failure without wording of its own. */
  error: MessageKey;
}

interface VerifyCopy {
  title: MessageKey;
  description: MessageKey;
  confirm: MessageKey;
  error: MessageKey;
}

const verifyCopy: Record<VerificationReason, VerifyCopy> = {
  share: {
    title: "sharing.prompt.title",
    description: "sharing.prompt.description",
    confirm: "sharing.prompt.share",
    error: "sharing.prompt.error",
  },
  "save-passkey": {
    title: "sharing.prompt.save-passkey.title",
    description: "sharing.prompt.save-passkey.description",
    confirm: "sharing.prompt.save-passkey.confirm",
    error: "sharing.prompt.passkey.error",
  },
  "sign-in": {
    title: "sharing.prompt.sign-in.title",
    description: "sharing.prompt.sign-in.description",
    confirm: "sharing.prompt.sign-in.confirm",
    error: "sharing.prompt.passkey.error",
  },
  fill: {
    title: "sharing.prompt.fill.title",
    description: "sharing.prompt.fill.description",
    confirm: "sharing.prompt.fill.confirm",
    error: "sharing.prompt.passkey.error",
  },
  "fill-card": {
    title: "sharing.prompt.fill-card.title",
    description: "sharing.prompt.fill-card.description",
    confirm: "sharing.prompt.fill.confirm",
    error: "sharing.prompt.passkey.error",
  },
  "change-unlock": {
    title: "confirmation.change-unlock.title",
    description: "confirmation.change-unlock.description",
    confirm: "confirmation.change-unlock.confirm",
    error: "confirmation.change-unlock.error",
  },
};

const unlockDescriptions: Record<UnlockRequester, MessageKey> = {
  extension: "confirmation.unlock.description",
  autofill: "confirmation.unlock.description.autofill",
};

/** planCard always takes the PIN for a verification; an unlock offers the device's methods once known, plus recovery. */
export function planCard(
  request: Confirmation,
  methods: UnlockMethods | null,
): CardPlan {
  if (request.kind === "verify") {
    const copy = verifyCopy[request.reason];
    const unnamedAccount =
      request.reason === "sign-in" && request.account === "";
    return {
      title: copy.title,
      description: unnamedAccount
        ? "sharing.prompt.sign-in.description.no-account"
        : copy.description,
      values: {
        file: request.file,
        identity: request.identity,
        site: request.site,
        account: request.account,
      },
      pin: { confirm: copy.confirm },
      biometry: null,
      recovery: false,
      noMethod: false,
      error: copy.error,
    };
  }
  const pinSet = methods?.pinSet ?? false;
  const biometry = methods
    ? methods.biometryEnabled && methods.biometryAvailable
    : false;
  return {
    title: "confirmation.unlock.title",
    description: unlockDescriptions[request.requester],
    values: {},
    pin: pinSet ? { confirm: "unlock.pin.action" } : null,
    biometry: biometry
      ? pinSet
        ? "unlock.biometry-action"
        : "unlock.action"
      : null,
    recovery: true,
    noMethod: methods !== null && !pinSet && !biometry,
    error: "unlock.errors.failed",
  };
}

/** Whether pin is long enough to submit. */
export function pinReady(pin: string, methods: UnlockMethods | null): boolean {
  return pin !== "" && pin.length >= (methods?.pinMinLength ?? 0);
}
