import type { MessageKey } from "../i18n/messages.ts";
import type { Translator } from "../i18n/translator.tsx";

/** `update` replaces a matching credential's password; `add-site` adds the site to one holding the same account and password. */
export interface SaveTarget {
  readonly credential: string;
  readonly label: string;
  /** The credential's own account. */
  readonly account: string;
  readonly action: "update" | "add-site";
  /** What tells two accounts on one site apart. */
  readonly tags: readonly string[];
}

/** An empty `target` is a new credential, which takes `account` and `name`. */
export interface SaveChoice {
  readonly target: string;
  readonly account: string;
  readonly name: string;
}

/** `suggested` is empty for a new credential. */
export interface SaveDestinations {
  readonly targets: readonly SaveTarget[];
  readonly suggested: string;
}

export type RefusedField = "account" | "name";

export type SaveResult = "created" | "updated" | "site-added";

export const savedTitles: Record<SaveResult, MessageKey> = {
  created: "saving.saved.created",
  updated: "saving.saved.updated",
  "site-added": "saving.saved.site-added",
};

export function suggestedChoice(
  offer: SaveDestinations & { readonly account: string; readonly name: string },
): SaveChoice {
  return { target: offer.suggested, account: offer.account, name: offer.name };
}

/** Null for a new credential. */
export function targetOf(
  offer: SaveDestinations,
  target: string,
): SaveTarget | null {
  return offer.targets.find(({ credential }) => credential === target) ?? null;
}

/** The suggestion first, then a new credential (empty), then targets in vault order. */
export function destinationsOf(offer: SaveDestinations): string[] {
  const all = ["", ...offer.targets.map(({ credential }) => credential)];
  return [offer.suggested, ...all.filter((id) => id !== offer.suggested)];
}

export type DestinationAction = SaveTarget["action"] | "add-passkey";

const destinationTitles: Record<DestinationAction, MessageKey> = {
  update: "saving.target.update",
  "add-site": "saving.target.add-site",
  "add-passkey": "extension.passkey.target.add",
};

export function destinationTitle(
  offer: SaveDestinations,
  destination: string,
  t: Translator["t"],
): string {
  return targetTitle(targetOf(offer, destination), t);
}

/** A null target is titled as a new item. */
export function targetTitle(
  target: {
    readonly label: string;
    readonly action: DestinationAction;
  } | null,
  t: Translator["t"],
): string {
  if (!target) return t("saving.target.new");
  return t(destinationTitles[target.action], {
    label: target.label || t("credential.untitled"),
  });
}

export function saveActionOf(target: SaveTarget | null): MessageKey {
  if (!target) return "saving.save";
  return target.action === "update" ? "saving.update" : "saving.add-site";
}

export function saveResultOf(
  created: boolean,
  offer: SaveDestinations,
  choice: SaveChoice,
): SaveResult {
  if (created) return "created";
  return targetOf(offer, choice.target)?.action === "add-site"
    ? "site-added"
    : "updated";
}

/** Changing the target or the refused field clears the refusal. */
export function reviseChoice(
  choice: SaveChoice,
  refused: RefusedField | null,
  change: Partial<SaveChoice>,
): { choice: SaveChoice; refused: RefusedField | null } {
  const cleared =
    change.target !== undefined ||
    (refused !== null && change[refused] !== undefined);
  return {
    choice: { ...choice, ...change },
    refused: cleared ? null : refused,
  };
}
