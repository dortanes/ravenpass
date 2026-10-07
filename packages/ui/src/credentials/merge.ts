import type {
  Credential,
  CredentialInput,
  CredentialSummary,
} from "../vault-api.ts";
import {
  credentialSearchValues,
  selectEntries,
} from "../workspace/sections.ts";
import { appKey } from "./apps.ts";

/** A field holding one value, which a merge takes from one side. */
export type MergeField = "label" | "login" | "email" | "password" | "totp";

/** Which value a differing field keeps: the open password's, the other's, or for notes both. */
export type MergeChoice = "kept" | "other" | "both";

/** Fields in the order the merge dialog asks about them. */
export const mergeFields: readonly (MergeField | "notes")[] = [
  "label",
  "login",
  "email",
  "password",
  "totp",
  "notes",
];

/** The bounds a merged password must fit, in values and characters. */
export interface MergeBounds {
  websites: number;
  tags: number;
  notes: number;
}

export type MergeChoices = Partial<Record<MergeField | "notes", MergeChoice>>;

/** Notes kept from both sides are joined by an empty line. */
const notesJoint = "\n\n";

/**
 * CredentialMerge plans keeping one password with what another holds: a field equal on both sides or empty on one
 * needs no choice; lists are combined, the kept password's values first, without repeats and within bounds.
 */
export class CredentialMerge {
  readonly kept: Credential;
  readonly other: Credential;
  /** The fields that differ, in the order the dialog asks about them. */
  readonly conflicts: readonly (MergeField | "notes")[];
  readonly #websites: string[];
  readonly #tags: string[];
  /** Values left out to fit the bounds. */
  readonly dropped: { websites: number; tags: number };
  readonly #bounds: MergeBounds;

  constructor(kept: Credential, other: Credential, bounds: MergeBounds) {
    this.kept = kept;
    this.other = other;
    this.#bounds = bounds;
    this.conflicts = mergeFields.filter((field) => {
      const ours = kept[field];
      const theirs = other[field];
      return ours !== "" && theirs !== "" && ours !== theirs;
    });
    const websites = combined(kept.websites, other.websites, (website) =>
      website.trim().toLocaleLowerCase(),
    );
    const tags = combined(kept.tags, other.tags, (tag) =>
      tag.toLocaleLowerCase(),
    );
    this.#websites = websites.slice(0, bounds.websites);
    this.#tags = tags.slice(0, bounds.tags);
    this.dropped = {
      websites: websites.length - this.#websites.length,
      tags: tags.length - this.#tags.length,
    };
  }

  /** Whether notes from both sides fit together. */
  get notesFitTogether(): boolean {
    return (
      [...(this.kept.notes + notesJoint + this.other.notes)].length <=
      this.#bounds.notes
    );
  }

  /** The kept password's input once merged; an unanswered conflict keeps the kept password's value. */
  input(choices: MergeChoices): CredentialInput {
    const pick = (field: MergeField): string => {
      const ours = this.kept[field];
      const theirs = this.other[field];
      if (!ours) return theirs;
      if (!theirs) return ours;
      return choices[field] === "other" ? theirs : ours;
    };
    return {
      label: pick("label"),
      login: pick("login"),
      email: pick("email"),
      password: pick("password"),
      totp: pick("totp"),
      notes: this.#notes(choices.notes),
      websites: this.#websites,
      tags: this.#tags,
      apps: combined(this.kept.apps, this.other.apps, appKey),
    };
  }

  /** Every group either password belongs to. */
  groups(): string[] {
    return combined(this.kept.groups, this.other.groups, (id) => id);
  }

  #notes(choice: MergeChoice | undefined): string {
    const ours = this.kept.notes;
    const theirs = this.other.notes;
    if (!ours || ours === theirs) return theirs || ours;
    if (!theirs) return ours;
    if (choice === "other") return theirs;
    if (choice === "both" && this.notesFitTogether) {
      return ours + notesJoint + theirs;
    }
    return ours;
  }
}

/** combined is first followed by the values of second it lacks, compared by key. */
function combined<Value>(
  first: readonly Value[],
  second: readonly Value[],
  key: (value: Value) => string,
): Value[] {
  const seen = new Set<string>();
  const result: Value[] = [];
  for (const value of [...first, ...second]) {
    const name = key(value);
    if (seen.has(name)) continue;
    seen.add(name);
    result.push(value);
  }
  return result;
}

type Closeness = 0 | 1 | 2;

/** How close another password is to the open one: a shared site, then a shared login or email, then neither. */
function closeness(
  open: CredentialSummary,
  other: CredentialSummary,
): Closeness {
  if (other.sites.some((site) => open.sites.includes(site))) return 0;
  const accounts = [open.login, open.email]
    .filter(Boolean)
    .map((value) => value.toLocaleLowerCase());
  const shares = [other.login, other.email].some(
    (value) => value && accounts.includes(value.toLocaleLowerCase()),
  );
  return shares ? 1 : 2;
}

/** mergeCandidates lists the passwords the open one can merge with: matching a search, else closest first. */
export function mergeCandidates(
  open: CredentialSummary,
  credentials: CredentialSummary[],
  query: string,
): CredentialSummary[] {
  const others = selectEntries(
    credentials.filter((credential) => credential.id !== open.id),
    credentialSearchValues,
    "all",
    query,
  );
  if (query.trim()) return others;
  return others
    .map((credential, order) => ({
      credential,
      order,
      close: closeness(open, credential),
    }))
    .sort((a, b) => a.close - b.close || a.order - b.order)
    .map(({ credential }) => credential);
}
