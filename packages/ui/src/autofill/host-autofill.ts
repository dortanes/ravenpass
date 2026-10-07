import { type Appearance, appearanceOf } from "../host/appearance.ts";
import { naturalInterfaceSize } from "../host/interface-size.ts";
import type { SaveChoice, SaveTarget } from "../saving/offer.ts";
import type { LanguageSettings, SiteIcon } from "../vault-api.ts";
import type {
  AutofillApi,
  AutofillCredential,
  AutofillFailure,
  AutofillOpening,
  AutofillPasskey,
  AutofillResults,
  AutofillWait,
  FillOutcome,
  SaveOutcome,
  SaveReviewOffer,
  SignInOutcome,
  UnlockOutcome,
} from "./autofill-api.ts";
import type { Answer, AutofillChannel } from "./channel.ts";

const failures: readonly AutofillFailure[] = [
  "unreachable",
  "outdated",
  "fill-failed",
  "code-failed",
  "sign-in-failed",
  "save-failed",
  "unverifiable",
  "unsupported",
  "failed",
];

/** HostAutofill reads host answers defensively: a mistyped field is empty, an unknown screen fails, an unknown wait ends. */
export class HostAutofill implements AutofillApi {
  readonly #channel: AutofillChannel;

  constructor(channel: AutofillChannel) {
    this.#channel = channel;
  }

  async open(): Promise<AutofillOpening> {
    const answer = await this.#channel.request("open");
    if (answer.status !== "ok") throw unanswered();
    switch (answer.screen) {
      case "search":
        return {
          screen: "search",
          site: text(answer.site),
          code: answer.code === true,
          passkey: answer.passkey === true,
          passkeys: list(answer.passkeys).map(passkeyOf),
          ...resultsOf(answer),
        };
      case "unlock":
        return {
          screen: "unlock",
          methods: {
            biometryAvailable: answer.biometry === true,
            biometryEnabled: answer.biometry === true,
            pinSet: answer.pin === true,
            pinAttemptsLeft: count(answer.attemptsLeft),
            pinMinLength: count(answer.pinMin),
            pinMaxLength: count(answer.pinMax),
          },
          verify: answer.open === true,
        };
      case "save":
        return { screen: "save", offer: offerOf(record(answer.offer)) };
      default:
        throw unanswered();
    }
  }

  ready(): void {
    this.#channel.notify("ready");
  }

  cancel(): void {
    this.#channel.notify("cancel");
  }

  showKeyboard(): void {
    this.#channel.notify("keyboard");
  }

  watchWait(onWait: (wait: AutofillWait | null) => void): () => void {
    return this.#channel.watch((notice) => {
      if (notice.status === "wait") onWait(waitOf(notice));
    });
  }

  async interfaceSize(): Promise<number> {
    const answer = await this.#channel.request("interface-size");
    const percent = count(answer.percent);
    return answer.status === "ok" && percent > 0
      ? percent
      : naturalInterfaceSize;
  }

  async appearance(): Promise<Appearance> {
    const answer = await this.#channel.request("appearance");
    return answer.status === "ok" ? appearanceOf(answer.appearance) : "system";
  }

  async siteIcon(site: string): Promise<SiteIcon> {
    const answer = await this.#channel.request("icon", { site });
    if (answer.status !== "ok") return { image: "", tint: "" };
    return { image: text(answer.image), tint: text(answer.tint) };
  }

  async search(query: string): Promise<AutofillResults> {
    const answer = await this.#channel.request("search", { query });
    if (answer.status !== "ok") throw unanswered();
    return resultsOf(answer);
  }

  async fill(
    credential: AutofillCredential,
    add: boolean,
  ): Promise<FillOutcome> {
    const answer = await this.#channel.request("fill", {
      id: credential.id,
      label: credential.label,
      add,
    });
    switch (answer.status) {
      case "ok":
        return "filled";
      case "no-code":
      case "not-added":
      case "not-found":
      case "unreachable":
      case "outdated":
        return answer.status;
      default:
        return "failed";
    }
  }

  async signIn(passkey: AutofillPasskey): Promise<SignInOutcome> {
    const answer = await this.#channel.request("sign-in", { key: passkey.key });
    switch (answer.status) {
      case "ok":
        return "signed";
      case "not-found":
      case "unverifiable":
      case "unreachable":
      case "outdated":
        return answer.status;
      default:
        return "failed";
    }
  }

  async unlock(pin: string): Promise<UnlockOutcome> {
    const answer = await this.#channel.request("unlock", { pin });
    switch (answer.status) {
      case "ok":
        return { kind: "opened" };
      case "wrong-pin":
        return { kind: "wrong-pin", attemptsLeft: count(answer.attemptsLeft) };
      case "pin-removed":
        return { kind: "pin-removed" };
      case "too-soon":
        return { kind: "too-soon" };
      case "canceled":
        return { kind: "canceled" };
      default:
        return { kind: "failed" };
    }
  }

  async save(choice: SaveChoice): Promise<SaveOutcome> {
    const answer = await this.#channel.request("save", { ...choice });
    switch (answer.status) {
      case "ok":
        return { kind: "saved", created: answer.created === true };
      case "name-refused":
        return { kind: "name-refused" };
      case "account-refused":
        return { kind: "account-refused" };
      default:
        return { kind: "failed" };
    }
  }

  async getLanguage(): Promise<LanguageSettings> {
    const answer = await this.#channel.request("language");
    if (answer.status !== "ok") throw unanswered();
    return {
      languages: list(answer.languages).map(text),
      language: text(answer.language),
      chosen: answer.chosen === true,
    };
  }
}

function unanswered(): Error {
  return new Error("Ravenpass could not answer this screen. Try again.");
}

/** The wait a notice reports, or null for none. */
function waitOf(notice: Answer): AutofillWait | null {
  switch (notice.wait) {
    case "opening":
    case "unlocking":
    case "verifying":
      return { kind: notice.wait };
    case "failed": {
      const failure = failures.find((known) => known === notice.failure);
      return { kind: "failed", failure: failure ?? "failed" };
    }
    default:
      return null;
  }
}

function passkeyOf(value: unknown): AutofillPasskey {
  const found = record(value);
  return {
    key: text(found.key),
    label: text(found.label),
    account: text(found.account),
    site: text(found.site),
  };
}

function resultsOf(answer: Answer): AutofillResults {
  return {
    scope: answer.scope === "matches" ? "matches" : "vault",
    credentials: list(answer.results).map((value) => {
      const found = record(value);
      return {
        id: text(found.id),
        label: text(found.label),
        account: text(found.account),
        site: text(found.site),
        matches: found.matches === true,
        tags: list(found.tags).map(text).filter(Boolean),
      };
    }),
  };
}

function offerOf(offer: Record<string, unknown>): SaveReviewOffer {
  return {
    site: text(offer.site),
    account: text(offer.account),
    name: text(offer.name),
    suggested: text(offer.suggested),
    targets: list(offer.targets).flatMap((value): SaveTarget[] => {
      const target = record(value);
      const action = target.action;
      if (action !== "update" && action !== "add-site") return [];
      return [
        {
          credential: text(target.id),
          label: text(target.label),
          account: text(target.account),
          action,
          tags: list(target.tags).map(text).filter(Boolean),
        },
      ];
    }),
  };
}

function text(value: unknown): string {
  return typeof value === "string" ? value : "";
}

function count(value: unknown): number {
  return typeof value === "number" && Number.isInteger(value) ? value : 0;
}

function list(value: unknown): unknown[] {
  return Array.isArray(value) ? value : [];
}

function record(value: unknown): Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value)
    ? (value as Record<string, unknown>)
    : {};
}
