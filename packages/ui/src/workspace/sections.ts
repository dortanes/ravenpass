import { networkName } from "../cards/networks.ts";
import type {
  CardSummary,
  CredentialSummary,
  IdentitySummary,
  NoteSummary,
  SeedSummary,
} from "../vault-api.ts";

export type WorkspaceSection = "all" | "recent" | "pinned";

export const recentLimit = 10;

/** No group chosen: the list is not narrowed to one. */
export const everyGroup = "";

/** What the list needs of an item of any kind. */
export interface ListedItem {
  id: string;
  label: string;
  pinned: boolean;
  lastUsedAt: number;
  groups: string[];
  tags: string[];
}

/** The values besides its label an item is found by, in the order they break label ties. */
export type SearchValues<Item> = (item: Item) => readonly string[];

export const credentialSearchValues: SearchValues<CredentialSummary> = (
  credential,
) => [credential.login, credential.email, ...credential.sites];

export const identitySearchValues: SearchValues<IdentitySummary> = (
  identity,
) => [identity.email];

export const cardSearchValues: SearchValues<CardSummary> = (card) => [
  card.bankName,
  card.lastFour,
  networkName(card.network),
];

export const noteSearchValues: SearchValues<NoteSummary> = (note) => [
  note.preview,
];

export const seedSearchValues: SearchValues<SeedSummary> = (seed) => [
  seed.wallet,
];

export function selectEntries<Item extends ListedItem>(
  entries: Item[],
  values: SearchValues<Item>,
  section: WorkspaceSection,
  query: string,
  group: string = everyGroup,
): Item[] {
  const exact = exactTag(entries, query);
  const matching = entries.filter(
    (entry) =>
      matchesEntry(entry, values, query, exact) && inGroup(entry, group),
  );
  if (section === "recent") {
    return matching
      .filter((entry) => entry.lastUsedAt > 0)
      .sort((left, right) => right.lastUsedAt - left.lastUsedAt)
      .slice(0, recentLimit);
  }
  const scoped =
    section === "pinned" ? matching.filter((entry) => entry.pinned) : matching;
  return scoped.sort((left, right) => byLabel(left, right, values));
}

/** The most items of one kind the command palette lists for a search. */
export const paletteLimit = 20;

/** paletteEntries lists the recent items for an empty query, else the best matches across sections and groups. */
export function paletteEntries<Item extends ListedItem>(
  entries: Item[],
  values: SearchValues<Item>,
  query: string,
): Item[] {
  if (!query.trim()) return selectEntries(entries, values, "recent", "");
  return selectEntries(entries, values, "all", query).slice(0, paletteLimit);
}

/** A search that opens with it finds items by their tags alone. */
export const tagPrefix = "#";

/** tagSearch is the search that lists the items carrying `tag`. */
export function tagSearch(tag: string): string {
  return tagPrefix + tag;
}

/** The tag a tag search names, lowercased; null for any other search. */
function searchedTag(query: string): string | null {
  const trimmed = query.trim();
  if (!trimmed.startsWith(tagPrefix)) return null;
  return trimmed.slice(tagPrefix.length).trim().toLocaleLowerCase();
}

/** exactTag reports a tag search naming a whole tag some item carries, which lists that tag alone. */
function exactTag(entries: readonly ListedItem[], query: string): boolean {
  const term = searchedTag(query);
  return (
    term !== null &&
    term !== "" &&
    entries.some((entry) =>
      entry.tags.some((tag) => tag.toLocaleLowerCase() === term),
    )
  );
}

/** matchesEntry finds an item by its label, its kind's values and its tags, or by its tags alone for a tag search. */
function matchesEntry<Item extends ListedItem>(
  entry: Item,
  values: SearchValues<Item>,
  query: string,
  exact: boolean,
): boolean {
  const term = searchedTag(query);
  if (term !== null) {
    return entry.tags.some((tag) => {
      const key = tag.toLocaleLowerCase();
      return exact ? key === term : key.startsWith(term);
    });
  }
  return matchesQuery([entry.label, ...values(entry), ...entry.tags], query);
}

/** knownTags lists every tag the items carry once, ignoring case, in alphabetical order. */
export function knownTags(items: readonly { tags: string[] }[]): string[] {
  const seen = new Map<string, string>();
  for (const item of items) {
    for (const tag of item.tags) {
      const key = tag.toLocaleLowerCase();
      if (!seen.has(key)) seen.set(key, tag);
    }
  }
  return [...seen.values()].sort((left, right) =>
    left.localeCompare(right, undefined, { sensitivity: "base" }),
  );
}

/** matchesQuery tells whether any value holds the query, ignoring case; an empty query matches all. */
export function matchesQuery(
  values: readonly string[],
  query: string,
): boolean {
  const term = query.trim().toLocaleLowerCase();
  if (!term) return true;
  return values.some((value) => value.toLocaleLowerCase().includes(term));
}

function inGroup(entry: ListedItem, group: string): boolean {
  return group === everyGroup || entry.groups.includes(group);
}

function byLabel<Item extends ListedItem>(
  left: Item,
  right: Item,
  values: SearchValues<Item>,
): number {
  const leftKeys = [left.label, ...values(left)];
  const rightKeys = [right.label, ...values(right)];
  for (const [index, key] of leftKeys.entries()) {
    const order = key.localeCompare(rightKeys[index] ?? "", undefined, {
      sensitivity: "base",
    });
    if (order !== 0) return order;
  }
  return left.id.localeCompare(right.id);
}

/** countInGroup counts the items of any kind a group holds. */
export function countInGroup(
  items: readonly { groups: string[] }[],
  group: string,
): number {
  return items.filter((item) => item.groups.includes(group)).length;
}
