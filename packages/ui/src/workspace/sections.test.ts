import assert from "node:assert/strict";
import test from "node:test";
import type {
  CardSummary,
  CredentialSummary,
  IdentitySummary,
  NoteSummary,
  SeedSummary,
} from "../vault-api.ts";
import {
  cardSearchValues,
  countInGroup,
  credentialSearchValues,
  everyGroup,
  identitySearchValues,
  knownTags,
  matchesQuery,
  noteSearchValues,
  paletteEntries,
  paletteLimit,
  recentLimit,
  seedSearchValues,
  selectEntries,
  tagSearch,
} from "./sections.ts";

function entry(
  id: string,
  label: string,
  login = "",
  pinned = false,
  lastUsedAt = 0,
  groups: string[] = [],
  email = "",
): CredentialSummary {
  return {
    id,
    label,
    login,
    pinned,
    lastUsedAt,
    groups,
    tags: [],
    site: "",
    sites: [],
    email,
    oneTimeCode: false,
    passkeys: 0,
  };
}

function identity(
  id: string,
  label: string,
  email = "",
  groups: string[] = [],
): IdentitySummary {
  return {
    id,
    label,
    email,
    pinned: false,
    lastUsedAt: 0,
    groups,
    tags: [],
    expiresOn: "",
    thumbnail: "",
  };
}

const entries = [
  entry("1", "Hetzner", "sam", false, 20, ["work"], "robin@example.com"),
  entry("2", "Fastmail", "alex@mail.example.com", true, 0, ["work", "mail"]),
  entry("3", "GitHub", "morgan", true, 40),
];

const identities = [
  identity("i1", "Personal", "me@example.com", ["work"]),
  identity("i2", "Company", "office@example.org"),
];

function select(
  section: "all" | "recent" | "pinned",
  query: string,
  group?: string,
) {
  return selectEntries(entries, credentialSearchValues, section, query, group);
}

test("all items are listed by title", () => {
  const listed = select("all", "");
  assert.deepEqual(
    listed.map((item) => item.label),
    ["Fastmail", "GitHub", "Hetzner"],
  );
});

test("search matches every website of a credential, not only its first", () => {
  const router = {
    ...entry("4", "Router"),
    site: "console.example.net",
    sites: ["console.example.net", "status.example.org"],
  };
  const found = (query: string) =>
    selectEntries(
      [...entries, router],
      credentialSearchValues,
      "all",
      query,
    ).map((item) => item.id);
  assert.deepEqual(found("EXAMPLE.NET"), ["4"]);
  assert.deepEqual(found("status.example"), ["4"]);
});

test("search matches titles, logins and emails, not other fields", () => {
  assert.deepEqual(
    select("all", "MORG").map((item) => item.id),
    ["3"],
  );
  assert.deepEqual(
    select("all", "mail.example.com").map((item) => item.id),
    ["2"],
  );
  assert.deepEqual(
    select("all", "ROBIN@").map((item) => item.id),
    ["1"],
  );
  assert.deepEqual(select("all", "secret"), []);
});

test("credentials with one title are ordered by login, then email", () => {
  const twins = [
    entry("b", "Mail", "alex", false, 0, [], "z@example.com"),
    entry("a", "Mail", "alex", false, 0, [], "b@example.com"),
    entry("c", "Mail", "", false, 0, [], "y@example.com"),
  ];
  assert.deepEqual(
    selectEntries(twins, credentialSearchValues, "all", "").map(
      (item) => item.id,
    ),
    ["c", "a", "b"],
  );
});

test("identities are found by name and email", () => {
  assert.deepEqual(
    selectEntries(identities, identitySearchValues, "all", "").map(
      (item) => item.id,
    ),
    ["i2", "i1"],
  );
  assert.deepEqual(
    selectEntries(identities, identitySearchValues, "all", "EXAMPLE.ORG").map(
      (item) => item.id,
    ),
    ["i2"],
  );
  assert.deepEqual(
    selectEntries(identities, identitySearchValues, "all", "person").map(
      (item) => item.id,
    ),
    ["i1"],
  );
  assert.deepEqual(
    selectEntries(identities, identitySearchValues, "all", "", "work").map(
      (item) => item.id,
    ),
    ["i1"],
  );
});

test("cards are found by name, bank, last four digits and network", () => {
  const card = (
    id: string,
    label: string,
    bankName: string,
    lastFour: string,
    network: CardSummary["network"],
  ): CardSummary => ({
    id,
    label,
    bankName,
    lastFour,
    network,
    color: "",
    site: "",
    pinned: false,
    lastUsedAt: 0,
    groups: [],
    tags: [],
    expiresOn: "",
  });
  const cards = [
    card("c1", "Daily", "Jyske Bank", "6789", "visa"),
    card("c2", "Travel", "Monzo", "0005", "american-express"),
  ];
  const found = (query: string) =>
    selectEntries(cards, cardSearchValues, "all", query).map((item) => item.id);
  assert.deepEqual(found("jyske"), ["c1"]);
  assert.deepEqual(found("0005"), ["c2"]);
  assert.deepEqual(found("american express"), ["c2"]);
  assert.deepEqual(found("travel"), ["c2"]);
  assert.deepEqual(found("4400"), []);
});

test("notes are found by name and first line, never by hidden text", () => {
  const note = (
    id: string,
    label: string,
    preview: string,
    hidden = false,
  ): NoteSummary => ({
    id,
    label,
    preview,
    hidden,
    pinned: false,
    lastUsedAt: 0,
    groups: [],
    tags: [],
  });
  const notes = [
    note("n1", "Wi-Fi", "Network: Attic 5G"),
    note("n2", "Locker", "", true),
    note("n3", "Recipes", "Grandma's bread"),
  ];
  const found = (query: string) =>
    selectEntries(notes, noteSearchValues, "all", query).map((item) => item.id);
  assert.deepEqual(found("attic"), ["n1"]);
  assert.deepEqual(found("LOCK"), ["n2"]);
  assert.deepEqual(found("bread"), ["n3"]);
  assert.deepEqual(found("5g network"), []);
});

test("seeds are found by name and wallet, ordered by wallet between equal names", () => {
  const seed = (id: string, label: string, wallet: string): SeedSummary => ({
    id,
    label,
    format: "phrase",
    wallet,
    total: 24,
    used: 0,
    pinned: false,
    lastUsedAt: 0,
    groups: [],
    tags: [],
  });
  const seeds = [
    seed("s1", "Savings", "Trezor"),
    seed("s2", "Savings", "Ledger"),
    seed("s3", "Trading", ""),
  ];
  const found = (query: string) =>
    selectEntries(seeds, seedSearchValues, "all", query).map((item) => item.id);
  assert.deepEqual(found(""), ["s2", "s1", "s3"]);
  assert.deepEqual(found("ledger"), ["s2"]);
  assert.deepEqual(found("trad"), ["s3"]);
  assert.deepEqual(found("24"), []);
});

test("recent lists only used entries, newest first, within the limit", () => {
  const used = Array.from({ length: recentLimit + 5 }, (_, index) =>
    entry(`u${index}`, `Item ${index}`, "", false, index + 1),
  );
  const listed = selectEntries(
    [...used, entry("never", "Never")],
    credentialSearchValues,
    "recent",
    "",
  );
  assert.equal(listed.length, recentLimit);
  assert.equal(listed.at(0)?.lastUsedAt, used.length);
  assert.ok(listed.every((item) => item.lastUsedAt > 0));
});

test("pinned lists only pinned entries", () => {
  assert.deepEqual(
    select("pinned", "").map((item) => item.id),
    ["2", "3"],
  );
});

test("a group narrows the list and composes with the section and the search", () => {
  assert.deepEqual(
    select("all", "", "work").map((item) => item.id),
    ["2", "1"],
  );
  assert.deepEqual(
    select("pinned", "", "work").map((item) => item.id),
    ["2"],
  );
  assert.deepEqual(
    select("all", "hetzner", "work").map((item) => item.id),
    ["1"],
  );
  assert.deepEqual(select("all", "", "missing"), []);
});

test("every group leaves the list as the section made it", () => {
  assert.deepEqual(
    select("all", "", everyGroup).map((item) => item.id),
    select("all", "").map((item) => item.id),
  );
});

test("the palette lists recent items before a search", () => {
  assert.deepEqual(
    paletteEntries(entries, credentialSearchValues, " ").map((item) => item.id),
    ["3", "1"],
  );
});

test("the palette searches every section and group, within its limit", () => {
  assert.deepEqual(
    paletteEntries(entries, credentialSearchValues, "FASTMAIL").map(
      (item) => item.id,
    ),
    ["2"],
  );
  const many = Array.from({ length: paletteLimit + 5 }, (_, index) =>
    entry(`m${index}`, `Mail ${index}`),
  );
  assert.equal(
    paletteEntries(many, credentialSearchValues, "mail").length,
    paletteLimit,
  );
});

test("a query matches any value, ignoring case and surrounding spaces", () => {
  assert.ok(matchesQuery(["Groups", "Work"], "  wOr "));
  assert.ok(matchesQuery(["Groups"], ""));
  assert.ok(!matchesQuery(["Groups"], "storage"));
});

test("a group counts the items of each kind it holds", () => {
  assert.equal(countInGroup(entries, "work"), 2);
  assert.equal(countInGroup(identities, "work"), 1);
  assert.equal(countInGroup([...entries, ...identities], "work"), 3);
  assert.equal(countInGroup(identities, "mail"), 0);
});

test("tags find an item, and a tag search finds it by its tags alone", () => {
  const personal = {
    ...entry("1", "Example", "alex@example.com"),
    tags: ["Personal"],
  };
  const work = { ...entry("2", "Example", "sam@example.com"), tags: ["Work"] };
  const personnel = entry("3", "Personnel");
  const found = (query: string) =>
    selectEntries(
      [personal, work, personnel],
      credentialSearchValues,
      "all",
      query,
    ).map((item) => item.id);
  assert.deepEqual(found("work"), ["2"]);
  assert.deepEqual(found("person"), ["1", "3"]);
  assert.deepEqual(found(tagSearch("Personal")), ["1"]);
  assert.deepEqual(found(" #wo "), ["2"]);
  assert.deepEqual(found("#"), ["1", "2"]);
});

test("a tag search naming a whole tag lists that tag alone", () => {
  const work = { ...entry("1", "Example", "alex@example.com"), tags: ["Work"] };
  const workout = {
    ...entry("2", "Gym", "sam@example.com"),
    tags: ["Workout"],
  };
  const found = (query: string) =>
    selectEntries([work, workout], credentialSearchValues, "all", query).map(
      (item) => item.id,
    );
  assert.deepEqual(found(tagSearch("Work")), ["1"]);
  assert.deepEqual(found("#wor"), ["1", "2"]);
});

test("known tags are listed once ignoring case, in alphabetical order", () => {
  assert.deepEqual(
    knownTags([
      { tags: ["work", "Shared"] },
      { tags: ["Work", "дом"] },
      { tags: [] },
    ]),
    ["Shared", "work", "дом"],
  );
});
