import assert from "node:assert/strict";
import test from "node:test";
import type {
  AutofillCredential,
  AutofillPasskey,
  AutofillResults,
  FillOutcome,
  SearchOpening,
  SignInOutcome,
} from "./autofill-api.ts";
import { SearchScreen } from "./search-screen.ts";

const mail: AutofillCredential = {
  id: "a1",
  label: "Mail",
  account: "alex",
  site: "example.com",
  matches: true,
  tags: [],
};
const bank: AutofillCredential = {
  id: "b2",
  label: "Bank",
  account: "alex",
  site: "bank.example",
  matches: false,
  tags: [],
};

const opening: SearchOpening = {
  screen: "search",
  site: "example.com",
  code: false,
  passkey: false,
  scope: "matches",
  credentials: [mail],
  passkeys: [],
};

const passkey: AutofillPasskey = {
  key: "a2V5",
  label: "Mail",
  account: "alex",
  site: "example.com",
};

/** A host whose searches wait until the test settles them. */
class Phone {
  readonly searches = new Map<
    string,
    (results: AutofillResults | null) => void
  >();
  readonly fills: [string, boolean][] = [];
  readonly signIns: string[] = [];
  outcome: FillOutcome = "filled";
  signedIn: SignInOutcome = "signed";

  search = (query: string) =>
    new Promise<AutofillResults>((resolve, reject) => {
      this.searches.set(query, (results) =>
        results ? resolve(results) : reject(new Error("search failed")),
      );
    });

  fill = async (credential: AutofillCredential, add: boolean) => {
    this.fills.push([credential.id, add]);
    return this.outcome;
  };

  signIn = async (chosen: AutofillPasskey) => {
    this.signIns.push(chosen.key);
    return this.signedIn;
  };
}

const vault: AutofillResults = { scope: "vault", credentials: [bank, mail] };

test("typing searches the vault, a late answer is dropped, and clearing shows the opening again", async () => {
  const phone = new Phone();
  const screen = new SearchScreen(opening, phone);
  const first = screen.type("b");
  const second = screen.type("ba");
  phone.searches.get("ba")?.(vault);
  await second;
  phone.searches.get("b")?.({ scope: "vault", credentials: [] });
  await first;
  assert.equal(screen.state.get().query, "ba");
  assert.deepEqual(screen.state.get().results, vault);
  await screen.type("  ");
  assert.deepEqual(screen.state.get().results, opening);
  assert.equal(phone.searches.size, 2);
});

test("a failed search is noted and the next answer clears it", async () => {
  const phone = new Phone();
  const screen = new SearchScreen(opening, phone);
  const failing = screen.type("b");
  phone.searches.get("b")?.(null);
  await failing;
  assert.equal(screen.state.get().searchFailed, true);
  const next = screen.type("ba");
  phone.searches.get("ba")?.(vault);
  await next;
  assert.equal(screen.state.get().searchFailed, false);
});

test("a match fills at once, and another credential asks first", async () => {
  const phone = new Phone();
  const screen = new SearchScreen(opening, phone);
  screen.choose(mail);
  assert.deepEqual(phone.fills, [["a1", false]]);
  assert.equal(screen.state.get().filling, "a1");

  const other = new SearchScreen(opening, phone);
  other.choose(bank);
  assert.deepEqual(other.state.get().asking, {
    credential: bank,
    addition: "add-site",
  });
  other.decline();
  assert.equal(other.state.get().asking, null);
  other.choose(bank);
  other.confirm();
  assert.deepEqual(phone.fills, [
    ["a1", false],
    ["b2", true],
  ]);
});

test("an app is asked to use the credential rather than to add a site", () => {
  const screen = new SearchScreen({ ...opening, site: "" }, new Phone());
  screen.choose(bank);
  assert.equal(screen.state.get().asking?.addition, "link");
  assert.equal(screen.forSite, false);
});

test("the site's passkeys show while no query is typed", async () => {
  const phone = new Phone();
  const screen = new SearchScreen({ ...opening, passkeys: [passkey] }, phone);
  assert.deepEqual(screen.passkeys, [passkey]);
  const typed = screen.type("b");
  assert.deepEqual(screen.passkeys, []);
  phone.searches.get("b")?.(vault);
  await typed;
  await screen.type("");
  assert.deepEqual(screen.passkeys, [passkey]);
});

test("a passkey signs in, and holds every other choice until it ends", async () => {
  const phone = new Phone();
  const screen = new SearchScreen({ ...opening, passkeys: [passkey] }, phone);
  const signing = screen.signIn(passkey);
  assert.equal(screen.state.get().signingIn, "a2V5");
  screen.choose(mail);
  void screen.signIn(passkey);
  await signing;
  assert.deepEqual(phone.signIns, ["a2V5"]);
  assert.deepEqual(phone.fills, []);
});

test("a passkey request narrows its passkeys as the owner types and never searches the vault", async () => {
  const phone = new Phone();
  const work: AutofillPasskey = { ...passkey, key: "d29yaw", account: "sam" };
  const screen = new SearchScreen(
    { ...opening, passkey: true, credentials: [], passkeys: [passkey, work] },
    phone,
  );
  assert.ok(screen.passkeyOnly);
  await screen.type("SAM");
  assert.deepEqual(screen.passkeys, [work]);
  assert.deepEqual(screen.state.get().results.credentials, []);
  await screen.type("bank");
  assert.deepEqual(screen.passkeys, []);
  assert.equal(phone.searches.size, 0);
});

test("a passkey that did not sign in says why and returns to the list", async () => {
  for (const outcome of [
    "not-found",
    "unverifiable",
    "unreachable",
    "outdated",
    "failed",
  ] as const) {
    const phone = new Phone();
    phone.signedIn = outcome;
    const screen = new SearchScreen({ ...opening, passkeys: [passkey] }, phone);
    await screen.signIn(passkey);
    const view = screen.state.get();
    assert.equal(view.signInNote, outcome);
    assert.equal(view.signingIn, null);
    screen.choose(mail);
    assert.equal(screen.state.get().signInNote, null);
  }
});

test("a choice that did not fill says why and returns to the list", async () => {
  for (const outcome of [
    "no-code",
    "not-added",
    "not-found",
    "unreachable",
    "outdated",
    "failed",
  ] as const) {
    const phone = new Phone();
    phone.outcome = outcome;
    const screen = new SearchScreen(opening, phone);
    screen.choose(bank);
    screen.confirm();
    await Promise.resolve();
    await Promise.resolve();
    const view = screen.state.get();
    assert.equal(view.note, outcome);
    assert.equal(view.asking, null);
    assert.equal(view.filling, null);
  }
});
