import assert from "node:assert/strict";
import test from "node:test";
import type { SignInStyle } from "@ravenpass/ui/extensions/sign-in-style.ts";
import type { OneTimeCode, SiteIcon } from "@ravenpass/ui/vault-api.ts";
import {
  type CodeSuggestion,
  type FillValues,
  type IdentityFiles,
  SessionError,
  type SharedFile,
  type ShareProgress,
  type Suggestion,
  type SuggestPurpose,
} from "../link/client.ts";
import type { Answers, MenuRequest, Submission } from "../messages.ts";
import { type MenuClient, MenuRouter, type TabMessenger } from "./menus.ts";
import { PagePasskeys } from "./page-passkeys.ts";
import { PendingPasskeys } from "./pending-passkeys.ts";
import { PendingSignIns } from "./pending-sign-ins.ts";
import { RecentFills } from "./recent-fills.ts";
import { MenuSessions, type Sender } from "./sessions.ts";
import { SignInCards } from "./sign-in-cards.ts";
import {
  Clock,
  extensionId,
  MemoryArea,
  menuSender,
  pageSender,
  ScriptedPasskeys,
} from "./test-doubles.test-support.ts";

const tabId = 7;
const origin = "https://github.com";

const account: CodeSuggestion = {
  id: "a1",
  label: "GitHub",
  account: "alex",
  site: "github.com",
  exact: true,
  tags: [],
  digits: 6,
  period: 30,
};

const values: FillValues = {
  login: "alex",
  email: "alex@example.com",
  password: "correct horse",
};

/** The content script of a sign-in form in a frame of the tab. */
const loginFrame = pageSender({ frameId: 2, documentId: "login-frame" });

/**
 * Lists `accounts` unless `refusal` is set, each exact for an origin added to it; fills and codes report `progress`
 * before `verification` may refuse; adding a website fails with `addRefusal` when set.
 */
class ScriptedRavenpass implements MenuClient {
  accounts: readonly CodeSuggestion[] = [account];
  refusal: SessionError | null = null;
  progress: ShareProgress[] = [];
  verification: SessionError | null = null;
  addRefusal: SessionError | null = null;
  readonly fills: [string, string][] = [];
  readonly codes: [string, string][] = [];
  readonly added: [string, string][] = [];

  suggest(origin: string, purpose: "sign-in"): Promise<Suggestion[]>;
  suggest(origin: string, purpose: "code"): Promise<CodeSuggestion[]>;
  async suggest(
    origin: string,
    _purpose: SuggestPurpose,
  ): Promise<CodeSuggestion[]> {
    if (this.refusal) throw this.refusal;
    return this.accounts.map((listed) =>
      this.added.some(([id, added]) => id === listed.id && added === origin)
        ? { ...listed, exact: true }
        : listed,
    );
  }

  async addWebsite(id: string, origin: string): Promise<void> {
    if (this.addRefusal) throw this.addRefusal;
    this.added.push([id, origin]);
  }

  async fill(
    id: string,
    origin: string,
    onProgress?: (progress: ShareProgress) => void,
  ): Promise<FillValues> {
    this.fills.push([id, origin]);
    if (this.refusal) throw this.refusal;
    this.verify(onProgress);
    return values;
  }

  async code(
    id: string,
    origin: string,
    onProgress?: (progress: ShareProgress) => void,
  ): Promise<OneTimeCode> {
    this.codes.push([id, origin]);
    this.verify(onProgress);
    return { code: "287082", digits: 6, period: 30, expiresAt: 0 };
  }

  private verify(onProgress?: (progress: ShareProgress) => void): void {
    for (const step of this.progress) onProgress?.(step);
    if (this.verification) throw this.verification;
  }

  async icon(): Promise<SiteIcon> {
    return { image: "", tint: "" };
  }

  async unlock(): Promise<void> {}

  async identities(): Promise<IdentityFiles[]> {
    return [];
  }

  share(): Promise<SharedFile> {
    return Promise.reject(new Error("Sign-in shares no file."));
  }
}

type Relayed = [Parameters<TabMessenger>[0], Parameters<TabMessenger>[1]];

/** Frames answer a fill with `submission`. */
function router(style: SignInStyle = "card") {
  const area = new MemoryArea();
  const clock = new Clock();
  let issued = 0;
  const sessions = new MenuSessions({
    area,
    extensionId,
    now: clock.now,
    newToken: () => `token-${++issued}`,
  });
  const ravenpass = new ScriptedRavenpass();
  const relayed: Relayed[] = [];
  const frames: { submission: Submission } = {
    submission: { submitted: true, password: true },
  };
  const relay: TabMessenger = async (target, message) => {
    relayed.push([target, message]);
    if (message.kind === "card-show") return { shown: true };
    if (message.kind === "form-sign-in") return frames.submission;
    return undefined;
  };
  const cards = new SignInCards({ sessions, relay });
  const menus = new MenuRouter({
    client: ravenpass,
    sessions,
    cards,
    passkeys: new PagePasskeys({
      client: new ScriptedPasskeys(),
      sessions,
      cards,
      pending: new PendingPasskeys({ area }),
      relay,
    }),
    recentFills: new RecentFills({ area, now: clock.now }),
    pendingSignIns: new PendingSignIns({ area, now: clock.now }),
    signInStyle: async () => style,
    relay,
  });
  const serve = (request: MenuRequest, sender: Sender) =>
    menus.serve(request, sender);
  /** Binds the card's token to its menu page, as the page does when it loads. */
  const bindCard = async () => {
    const card = await sessions.cardOf(tabId);
    assert.ok(card);
    await serve({ kind: "menu", token: card.token }, menuSender());
    return card.token;
  };
  return { clock, sessions, ravenpass, relayed, frames, serve, bindCard };
}

const kinds = (relayed: readonly Relayed[]) =>
  relayed.map(([, message]) => message.kind);

test("in the card style a frame's sign-in form shows the card in the top frame", async () => {
  const { sessions, relayed, serve } = router("card");

  const answer = await serve(
    { kind: "sign-in-form", field: "login" },
    loginFrame,
  );

  assert.deepEqual(answer, { fill: null });
  assert.deepEqual(relayed, [
    [
      { tabId, frameId: 0 },
      { kind: "card-show", token: "token-1", reopen: false },
    ],
  ]);
  const card = await sessions.cardOf(tabId);
  assert.equal(card?.page.documentId, "login-frame");
  assert.equal(card?.page.origin, origin);
});

test("in the card style a focused field shows the card again, and no menu", async () => {
  const { relayed, serve } = router("card");

  const answer = await serve(
    { kind: "menu-open", field: "password", requested: false },
    loginFrame,
  );

  assert.deepEqual(answer, { token: null });
  assert.deepEqual(relayed, [
    [
      { tabId, frameId: 0 },
      { kind: "card-show", token: "token-1", reopen: true },
    ],
  ]);
});

test("in the field style a found form shows no card, and a focused field opens its menu", async () => {
  const { sessions, relayed, serve } = router("field");

  assert.deepEqual(
    await serve({ kind: "sign-in-form", field: "login" }, loginFrame),
    { fill: null },
  );
  const { token } = (await serve(
    { kind: "menu-open", field: "login", requested: false },
    loginFrame,
  )) as { token: string | null };

  assert.deepEqual(relayed, []);
  assert.equal(token, "token-1");
  assert.equal(await sessions.cardOf(tabId), null);
});

test("a site without accounts, and Ravenpass not open, show no card", async () => {
  for (const refusal of [null, new SessionError("not-open")]) {
    const { ravenpass, relayed, serve } = router("card");
    ravenpass.accounts = [];
    ravenpass.refusal = refusal;

    await serve({ kind: "sign-in-form", field: "password" }, loginFrame);
    await serve(
      { kind: "menu-open", field: "password", requested: false },
      loginFrame,
    );

    assert.deepEqual(relayed, []);
  }
});

test("a locked Ravenpass shows the card locked, which lists the accounts once Ravenpass is unlocked", async () => {
  const { sessions, ravenpass, serve, bindCard } = router("card");
  ravenpass.refusal = new SessionError("locked");
  await serve({ kind: "sign-in-form", field: "login" }, loginFrame);
  const token = await bindCard();
  assert.deepEqual(await serve({ kind: "menu-review", token }, menuSender()), {
    listing: { state: "locked", purpose: "sign-in" },
  });

  ravenpass.refusal = null;
  const answer = await serve({ kind: "menu-review", token }, menuSender());

  const listing = {
    state: "list",
    purpose: "sign-in",
    credentials: [{ ...account, strength: "strong" }],
    passkeys: [],
  };
  assert.deepEqual(answer, { listing });
  assert.deepEqual((await sessions.cardOf(tabId))?.content, {
    state: "sign-in-card",
    purpose: "sign-in",
    listing,
    requested: false,
  });
});

test("a locked card closes once Ravenpass is unlocked without accounts for the site", async () => {
  const { sessions, ravenpass, relayed, serve, bindCard } = router("card");
  ravenpass.refusal = new SessionError("locked");
  await serve({ kind: "sign-in-form", field: "login" }, loginFrame);
  const token = await bindCard();

  ravenpass.refusal = null;
  ravenpass.accounts = [];
  const answer = await serve({ kind: "menu-review", token }, menuSender());

  assert.deepEqual(answer, { listing: null });
  assert.deepEqual(relayed.at(-1), [
    { tabId, frameId: 0 },
    { kind: "card-hide", token, dismissed: false },
  ]);
  assert.equal(await sessions.cardOf(tabId), null);
});

test("choosing an account fills and submits the card's form, then closes the card", async () => {
  const { sessions, ravenpass, relayed, serve, bindCard } = router("card");
  await serve({ kind: "sign-in-form", field: "password" }, loginFrame);
  const token = await bindCard();

  const answer = await serve(
    { kind: "menu-fill", token, id: "a1", confirmed: false },
    menuSender(),
  );

  assert.deepEqual(answer, { ok: true });
  assert.deepEqual(ravenpass.fills, [["a1", origin]]);
  assert.deepEqual(relayed.slice(-2), [
    [
      { tabId, documentId: "login-frame", origin },
      { kind: "form-sign-in", ...values },
    ],
    [
      { tabId, frameId: 0 },
      { kind: "card-hide", token, dismissed: false },
    ],
  ]);
  assert.equal(await sessions.cardOf(tabId), null);
});

test("an account-only form fills the chosen password into the next step of the same site", async () => {
  const { ravenpass, relayed, frames, serve, bindCard } = router("card");
  frames.submission = { submitted: true, password: false };
  await serve({ kind: "sign-in-form", field: "login" }, loginFrame);
  await serve(
    { kind: "menu-fill", token: await bindCard(), id: "a1", confirmed: false },
    menuSender(),
  );
  const shown = relayed.length;
  const nextStep = pageSender({ frameId: 2, documentId: "password-frame" });

  const answer = await serve(
    { kind: "sign-in-form", field: "password" },
    nextStep,
  );

  assert.deepEqual(answer, { fill: values });
  assert.deepEqual(ravenpass.fills.at(-1), ["a1", origin]);
  assert.deepEqual(
    kinds(relayed.slice(shown)),
    [],
    "the next step shows no card",
  );
  const again = await serve(
    { kind: "sign-in-form", field: "password" },
    pageSender({ frameId: 2, documentId: "another-frame" }),
  );
  assert.deepEqual(again, { fill: null });
});

test("the next step takes nothing after 60 seconds, from another origin, or after a partial fill", async () => {
  const late = router("card");
  late.frames.submission = { submitted: true, password: false };
  await late.serve({ kind: "sign-in-form", field: "login" }, loginFrame);
  await late.serve(
    {
      kind: "menu-fill",
      token: await late.bindCard(),
      id: "a1",
      confirmed: false,
    },
    menuSender(),
  );
  late.clock.time += 60_000;
  assert.deepEqual(
    await late.serve({ kind: "sign-in-form", field: "password" }, loginFrame),
    { fill: null },
  );

  const elsewhere = router("card");
  elsewhere.frames.submission = { submitted: true, password: false };
  await elsewhere.serve({ kind: "sign-in-form", field: "login" }, loginFrame);
  await elsewhere.serve(
    {
      kind: "menu-fill",
      token: await elsewhere.bindCard(),
      id: "a1",
      confirmed: false,
    },
    menuSender(),
  );
  for (const other of ["https://example.com", "http://github.com"]) {
    assert.deepEqual(
      await elsewhere.serve(
        { kind: "sign-in-form", field: "password" },
        pageSender({ origin: other, documentId: "other-origin" }),
      ),
      { fill: null },
      other,
    );
  }
  assert.deepEqual(elsewhere.ravenpass.fills, [["a1", origin]]);

  const partial = router("card");
  partial.frames.submission = { submitted: false, password: false };
  await partial.serve({ kind: "sign-in-form", field: "login" }, loginFrame);
  await partial.serve(
    {
      kind: "menu-fill",
      token: await partial.bindCard(),
      id: "a1",
      confirmed: false,
    },
    menuSender(),
  );
  assert.deepEqual(
    await partial.serve(
      { kind: "sign-in-form", field: "password" },
      loginFrame,
    ),
    { fill: null },
  );
});

test("a code chosen in the card is filled and submitted, then the card closes", async () => {
  const { ravenpass, relayed, serve, bindCard } = router("card");
  await serve({ kind: "sign-in-form", field: "code" }, loginFrame);
  const token = await bindCard();

  await serve(
    { kind: "menu-fill-code", token, id: "a1", confirmed: false },
    menuSender(),
  );

  assert.deepEqual(ravenpass.codes, [["a1", origin]]);
  assert.deepEqual(relayed.slice(-2), [
    [
      { tabId, documentId: "login-frame", origin },
      { kind: "form-code", code: "287082" },
    ],
    [
      { tabId, frameId: 0 },
      { kind: "card-hide", token, dismissed: false },
    ],
  ]);
});

test("the person closing the card keeps it away from the document", async () => {
  const { sessions, relayed, serve, bindCard } = router("card");
  await serve({ kind: "sign-in-form", field: "login" }, loginFrame);
  const token = await bindCard();

  await serve({ kind: "menu-close", token }, menuSender());

  assert.deepEqual(relayed.at(-1), [
    { tabId, frameId: 0 },
    { kind: "card-hide", token, dismissed: true },
  ]);
  assert.equal(await sessions.cardOf(tabId), null);
});

test("the top frame framing the card may end it; the form's frame may not", async () => {
  const { sessions, serve } = router("card");
  await serve({ kind: "sign-in-form", field: "login" }, loginFrame);
  const card = await sessions.cardOf(tabId);
  assert.ok(card);

  await assert.rejects(
    serve({ kind: "menu-close", token: card.token }, loginFrame),
  );
  await serve({ kind: "menu-close", token: card.token }, pageSender());

  assert.equal(await sessions.cardOf(tabId), null);
});

/** Opens a save offer's menu in the tab's top frame and binds it to its menu page. */
async function openOffer(
  sessions: MenuSessions,
  serve: (request: MenuRequest, sender: Sender) => Promise<unknown>,
) {
  const token = await sessions.open(
    { tabId, documentId: "page-document", origin },
    {
      state: "offer",
      offer: {
        state: "locked",
        site: "github.com",
        account: "alex",
        name: "github.com",
        targets: [],
        suggested: "",
        expiresAt: Date.now() + 180_000,
      },
    },
  );
  await serve({ kind: "menu", token }, menuSender());
  return token;
}

test("an offer's menu shows the icon of the offer's site and no other", async () => {
  const { sessions, serve } = router("card");
  const token = await openOffer(sessions, serve);

  assert.deepEqual(
    await serve({ kind: "menu-icon", token, site: "github.com" }, menuSender()),
    { image: "", tint: "" },
  );
  await assert.rejects(
    serve({ kind: "menu-icon", token, site: "example.com" }, menuSender()),
  );
});

test("unlocking from an offer keeps it open, and closing it closes it in the top frame", async () => {
  const { sessions, relayed, serve } = router("card");
  const token = await openOffer(sessions, serve);

  await serve({ kind: "menu-unlock", token }, menuSender());
  assert.deepEqual(relayed, []);

  await serve({ kind: "menu-close", token }, menuSender());
  assert.deepEqual(relayed, [
    [
      { tabId, documentId: "page-document", origin },
      { kind: "menu-close", token },
    ],
  ]);
});

test("unlocking from a card keeps the card open", async () => {
  const { sessions, ravenpass, serve, bindCard } = router("card");
  ravenpass.refusal = new SessionError("locked");
  await serve({ kind: "sign-in-form", field: "login" }, loginFrame);
  const token = await bindCard();

  assert.deepEqual(await serve({ kind: "menu-unlock", token }, menuSender()), {
    ok: true,
  });
  assert.equal((await sessions.cardOf(tabId))?.token, token);
});

test("a frame of another origin than the tab's top document, or of a tab Chrome reports no address for, shows no card and keeps its field menu", async () => {
  for (const tab of [
    { id: tabId, url: "https://embedder.example/page" },
    { id: tabId, url: "http://github.com/login" },
    { id: tabId, url: "about:blank" },
    { id: tabId },
  ]) {
    const { sessions, relayed, serve } = router("card");
    const frame = pageSender({ frameId: 2, documentId: "login-frame", tab });

    const found = await serve({ kind: "sign-in-form", field: "login" }, frame);
    const focused = await serve(
      { kind: "menu-open", field: "login", requested: false },
      frame,
    );

    assert.deepEqual(found, { fill: null });
    assert.deepEqual(focused, { token: "token-1" });
    assert.deepEqual(relayed, []);
    assert.equal(await sessions.cardOf(tabId), null);
  }
});

test("the top frame's form and a form of a frame of its origin show the card", async () => {
  const cases: Sender[] = [pageSender({ tab: { id: tabId } }), loginFrame];
  for (const sender of cases) {
    const { sessions, serve } = router("card");

    await serve({ kind: "sign-in-form", field: "login" }, sender);

    assert.equal(
      (await sessions.cardOf(tabId))?.page.documentId,
      sender.documentId,
    );
  }
});

/** A sign-in form in a frame of another host of the top page's registrable domain. */
const sameSiteOrigin = "https://secure.github.com";
const sameSiteFrame = pageSender({
  url: `${sameSiteOrigin}/sign-in`,
  origin: sameSiteOrigin,
  frameId: 2,
  documentId: "login-frame",
});

/** A sign-in form in a frame of another site. */
const crossSiteFrame = pageSender({
  url: "https://accounts.example/sign-in",
  origin: "https://accounts.example",
  frameId: 3,
  documentId: "other-site-frame",
});

test("a frame of the top page's site shows the card, which names the frame and fills it for its own origin", async () => {
  const { sessions, ravenpass, relayed, serve } = router("card");

  assert.deepEqual(
    await serve(
      { kind: "menu-open", field: "login", requested: false },
      sameSiteFrame,
    ),
    { token: null },
  );
  const card = await sessions.cardOf(tabId);
  assert.ok(card);
  assert.equal(card.page.origin, sameSiteOrigin);
  const menu = (await serve(
    { kind: "menu", token: card.token },
    menuSender(),
  )) as Answers["menu"];
  assert.equal(menu.host, "secure.github.com");
  await serve(
    { kind: "menu-fill", token: card.token, id: "a1", confirmed: false },
    menuSender(),
  );

  assert.deepEqual(ravenpass.fills, [["a1", sameSiteOrigin]]);
  assert.deepEqual(
    relayed.find(([, message]) => message.kind === "form-sign-in")?.[0],
    { tabId, documentId: "login-frame", origin: sameSiteOrigin },
  );
});

test("a field menu in another site's frame closes the card, and an open field menu keeps a found form's card away", async () => {
  const { sessions, relayed, serve } = router("card");
  await serve({ kind: "sign-in-form", field: "login" }, pageSender());
  const card = await sessions.cardOf(tabId);
  assert.ok(card);

  const { token } = (await serve(
    { kind: "menu-open", field: "login", requested: false },
    crossSiteFrame,
  )) as { token: string | null };

  assert.ok(token);
  assert.deepEqual(relayed.at(-1), [
    { tabId, frameId: 0 },
    { kind: "card-hide", token: card.token, dismissed: false },
  ]);
  assert.equal(await sessions.cardOf(tabId), null);

  const shown = relayed.length;
  await serve({ kind: "sign-in-form", field: "password" }, sameSiteFrame);
  assert.deepEqual(kinds(relayed.slice(shown)), []);
  assert.equal(await sessions.cardOf(tabId), null);

  await serve({ kind: "menu-close", token }, crossSiteFrame);
  await serve(
    { kind: "menu-open", field: "password", requested: false },
    sameSiteFrame,
  );
  assert.equal((await sessions.cardOf(tabId))?.page.origin, sameSiteOrigin);
});

test("the top page's form and a same-site frame's field share one card and open no field menu", async () => {
  const { sessions, relayed, serve } = router("card");
  await serve({ kind: "sign-in-form", field: "login" }, pageSender());

  const focused = await serve(
    { kind: "menu-open", field: "password", requested: false },
    sameSiteFrame,
  );

  assert.deepEqual(focused, { token: null });
  assert.deepEqual(kinds(relayed), ["card-show", "card-hide", "card-show"]);
  assert.equal((await sessions.cardOf(tabId))?.page.origin, sameSiteOrigin);
});

test("a domain-only match and a plain-http page show no card on their own", async () => {
  const sibling = router("card");
  sibling.ravenpass.accounts = [{ ...account, exact: false }];
  await sibling.serve({ kind: "sign-in-form", field: "password" }, loginFrame);

  const insecure = router("card");
  await insecure.serve(
    { kind: "sign-in-form", field: "password" },
    pageSender({ origin: "http://github.com" }),
  );

  assert.deepEqual(sibling.relayed, []);
  assert.deepEqual(insecure.relayed, []);
});

test("a field asking for the card shows it for a weak match", async () => {
  const { sessions, ravenpass, serve } = router("card");
  ravenpass.accounts = [{ ...account, exact: false }];

  await serve(
    { kind: "menu-open", field: "password", requested: false },
    loginFrame,
  );

  const card = await sessions.cardOf(tabId);
  assert.equal(card?.content.state, "sign-in-card");
});

test("a weak match fills a password or a code only once the person confirmed it", async () => {
  const fills = [
    ["login", "menu-fill"],
    ["code", "menu-fill-code"],
  ] as const;
  for (const [field, kind] of fills) {
    const { ravenpass, relayed, serve } = router("field");
    ravenpass.accounts = [
      { ...account, site: "alice.github.io", exact: false },
    ];
    const { token } = (await serve(
      { kind: "menu-open", field, requested: false },
      loginFrame,
    )) as {
      token: string;
    };
    const menu = (await serve(
      { kind: "menu", token },
      menuSender(),
    )) as Answers["menu"];
    assert.equal(menu.host, "github.com");
    assert.equal(
      menu.content.state === "list" && menu.content.credentials[0]?.strength,
      "other-site",
    );

    await assert.rejects(
      serve({ kind, token, id: "a1", confirmed: false }, menuSender()),
    );
    assert.deepEqual([...ravenpass.fills, ...ravenpass.codes], []);
    assert.deepEqual(relayed, []);

    assert.deepEqual(
      await serve({ kind, token, id: "a1", confirmed: true }, menuSender()),
      { ok: true },
    );
    assert.deepEqual(
      [...ravenpass.fills, ...ravenpass.codes],
      [["a1", origin]],
    );
  }
});

test("an exact match on a plain-http page fills only once the person confirmed it", async () => {
  const { ravenpass, serve } = router("field");
  const insecure = pageSender({ origin: "http://github.com" });
  const { token } = (await serve(
    { kind: "menu-open", field: "password", requested: false },
    insecure,
  )) as { token: string };
  await serve({ kind: "menu", token }, menuSender());

  await assert.rejects(
    serve(
      { kind: "menu-fill", token, id: "a1", confirmed: false },
      menuSender(),
    ),
  );
  assert.deepEqual(ravenpass.fills, []);
});

test("a confirmed fill in a frame of the saved site remembers the frame's origin, which then fills without confirmation", async () => {
  const fills = [
    ["login", "menu-fill"],
    ["code", "menu-fill-code"],
  ] as const;
  for (const [field, kind] of fills) {
    const { sessions, ravenpass, serve, bindCard } = router("card");
    ravenpass.accounts = [{ ...account, exact: false }];
    await serve({ kind: "menu-open", field, requested: false }, sameSiteFrame);
    const token = await bindCard();
    const card = await sessions.cardOf(tabId);
    assert.equal(
      card?.content.state === "sign-in-card" &&
        card.content.listing.state === "list" &&
        card.content.listing.credentials[0]?.strength,
      "same-site",
    );

    assert.deepEqual(
      await serve({ kind, token, id: "a1", confirmed: true }, menuSender()),
      { ok: true },
    );
    assert.deepEqual(ravenpass.added, [["a1", sameSiteOrigin]]);

    await serve({ kind: "menu-open", field, requested: false }, sameSiteFrame);
    const again = await bindCard();
    assert.deepEqual(
      await serve(
        { kind, token: again, id: "a1", confirmed: false },
        menuSender(),
      ),
      { ok: true },
    );
    assert.deepEqual(ravenpass.added, [["a1", sameSiteOrigin]]);
  }
});

test("a fill that does not go through remembers nothing", async () => {
  const { ravenpass, serve } = router("field");
  ravenpass.accounts = [{ ...account, exact: false }];
  ravenpass.verification = new SessionError("declined");
  const { token } = (await serve(
    { kind: "menu-open", field: "login", requested: false },
    sameSiteFrame,
  )) as { token: string };
  await serve({ kind: "menu", token }, menuSender());

  assert.deepEqual(
    await serve(
      { kind: "menu-fill", token, id: "a1", confirmed: true },
      menuSender(),
    ),
    { ok: false, reason: "declined" },
  );
  assert.deepEqual(ravenpass.added, []);
});

test("another site's page is remembered only when the person asks", async () => {
  const otherSite = pageSender({
    url: "https://bob.github.io/login",
    origin: "https://bob.github.io",
  });
  for (const remember of [false, true]) {
    const { ravenpass, serve } = router("field");
    ravenpass.accounts = [
      { ...account, site: "alice.github.io", exact: false },
    ];
    const { token } = (await serve(
      { kind: "menu-open", field: "login", requested: false },
      otherSite,
    )) as { token: string };
    const menu = (await serve(
      { kind: "menu", token },
      menuSender(),
    )) as Answers["menu"];
    assert.equal(
      menu.content.state === "list" && menu.content.credentials[0]?.strength,
      "other-site",
    );

    assert.deepEqual(
      await serve(
        { kind: "menu-fill", token, id: "a1", confirmed: true, remember },
        menuSender(),
      ),
      { ok: true },
    );
    assert.deepEqual(
      ravenpass.added,
      remember ? [["a1", "https://bob.github.io"]] : [],
    );
  }
});

test("a plain-http page is never remembered, even when asked", async () => {
  for (const exact of [true, false]) {
    const { ravenpass, serve } = router("field");
    ravenpass.accounts = [{ ...account, exact }];
    const { token } = (await serve(
      { kind: "menu-open", field: "password", requested: false },
      pageSender({ origin: "http://github.com" }),
    )) as { token: string };
    await serve({ kind: "menu", token }, menuSender());

    assert.deepEqual(
      await serve(
        { kind: "menu-fill", token, id: "a1", confirmed: true, remember: true },
        menuSender(),
      ),
      { ok: true },
    );
    assert.deepEqual(ravenpass.added, []);
  }
});

test("a page Ravenpass does not remember still fills, and only the error's name is logged", async (t) => {
  const logged = t.mock.method(console, "error", () => {});
  const { ravenpass, relayed, serve } = router("field");
  ravenpass.accounts = [{ ...account, exact: false }];
  ravenpass.addRefusal = new SessionError("no-match");
  const { token } = (await serve(
    { kind: "menu-open", field: "login", requested: false },
    sameSiteFrame,
  )) as { token: string };
  await serve({ kind: "menu", token }, menuSender());

  assert.deepEqual(
    await serve(
      { kind: "menu-fill", token, id: "a1", confirmed: true },
      menuSender(),
    ),
    { ok: true },
  );
  assert.deepEqual(kinds(relayed), ["fill"]);
  assert.deepEqual(ravenpass.added, []);
  assert.deepEqual(
    logged.mock.calls.map(({ arguments: logArguments }) => logArguments[1]),
    ["SessionError"],
  );
});

test("a card the form showed while Ravenpass was locked closes once Ravenpass lists nothing strong", async () => {
  const { sessions, ravenpass, serve, bindCard } = router("card");
  ravenpass.refusal = new SessionError("locked");
  await serve({ kind: "sign-in-form", field: "login" }, loginFrame);
  const token = await bindCard();

  ravenpass.refusal = null;
  ravenpass.accounts = [{ ...account, exact: false }];

  assert.deepEqual(await serve({ kind: "menu-review", token }, menuSender()), {
    listing: null,
  });
  assert.equal(await sessions.cardOf(tabId), null);
});

test("a card a field asked for while Ravenpass was locked lists a weak match once Ravenpass is unlocked", async () => {
  const { sessions, ravenpass, serve, bindCard } = router("card");
  ravenpass.refusal = new SessionError("locked");
  await serve(
    { kind: "menu-open", field: "login", requested: false },
    loginFrame,
  );
  const token = await bindCard();

  ravenpass.refusal = null;
  ravenpass.accounts = [{ ...account, exact: false }];
  await serve({ kind: "menu-review", token }, menuSender());

  assert.equal((await sessions.cardOf(tabId))?.token, token);
});

/** Opens a field's menu of passwords or codes and binds it to its menu page. */
async function openFieldMenu(
  serve: (request: MenuRequest, sender: Sender) => Promise<unknown>,
  field: "login" | "code",
) {
  const { token } = (await serve(
    { kind: "menu-open", field, requested: false },
    loginFrame,
  )) as {
    token: string;
  };
  await serve({ kind: "menu", token }, menuSender());
  return token;
}

const menuDocument = { tabId, documentId: "menu-document" };
const fieldFrame = { tabId, documentId: "login-frame", origin };

test("unlocking from a field's menu keeps it open, and it lists the accounts once Ravenpass is unlocked", async () => {
  for (const field of ["login", "code"] as const) {
    const { ravenpass, relayed, serve } = router("field");
    ravenpass.refusal = new SessionError("locked");
    const token = await openFieldMenu(serve, field);

    assert.deepEqual(
      await serve({ kind: "menu-unlock", token }, menuSender()),
      {
        ok: true,
      },
    );
    assert.deepEqual(kinds(relayed), []);

    ravenpass.refusal = null;
    const answer = (await serve(
      { kind: "menu-review", token },
      menuSender(),
    )) as {
      listing: { state: string; purpose: string } | null;
    };

    assert.equal(answer.listing?.state, "list");
    assert.equal(
      answer.listing?.purpose,
      field === "code" ? "code" : "sign-in",
    );
    assert.deepEqual(kinds(relayed), []);
  }
});

const nothingListed = {
  state: "list",
  purpose: "sign-in",
  credentials: [],
  passkeys: [],
};

test("a field's menu unlocked from the menu says the site has nothing once Ravenpass lists nothing", async () => {
  const { ravenpass, relayed, serve } = router("field");
  ravenpass.refusal = new SessionError("locked");
  const token = await openFieldMenu(serve, "login");

  ravenpass.refusal = null;
  ravenpass.accounts = [];

  assert.deepEqual(await serve({ kind: "menu-review", token }, menuSender()), {
    listing: nothingListed,
  });
  assert.deepEqual(kinds(relayed), []);
});

test("a field's menu closes when Ravenpass cannot list", async () => {
  const { ravenpass, relayed, serve } = router("field");
  ravenpass.refusal = new SessionError("locked");
  const token = await openFieldMenu(serve, "login");

  ravenpass.refusal = new SessionError("failed");

  assert.deepEqual(await serve({ kind: "menu-review", token }, menuSender()), {
    listing: null,
  });
  assert.deepEqual(relayed.at(-1), [fieldFrame, { kind: "menu-close", token }]);
});

test("a menu asked for from the context menu opens beneath the field with nothing to list, in either style", async () => {
  for (const style of ["field", "card"] as const) {
    const { sessions, ravenpass, serve } = router(style);
    ravenpass.accounts = [];

    assert.deepEqual(
      await serve(
        { kind: "menu-open", field: "login", requested: false },
        loginFrame,
      ),
      { token: null },
    );
    const { token } = (await serve(
      { kind: "menu-open", field: "login", requested: true },
      loginFrame,
    )) as { token: string | null };

    assert.ok(token);
    const menu = (await serve({ kind: "menu", token }, menuSender())) as {
      content: unknown;
    };
    assert.deepEqual(menu.content, nothingListed);
    assert.equal(await sessions.cardOf(tabId), null);
  }
});

test("a menu asked for from the context menu in the card style replaces the card", async () => {
  const { sessions, serve } = router("card");
  await serve({ kind: "sign-in-form", field: "login" }, loginFrame);
  assert.ok(await sessions.cardOf(tabId));

  const { token } = (await serve(
    { kind: "menu-open", field: "login", requested: true },
    loginFrame,
  )) as { token: string | null };

  assert.ok(token);
  assert.equal(await sessions.cardOf(tabId), null);
});

const waitingFills = [
  ["login", "menu-fill", "fill"],
  ["code", "menu-fill-code", "fill-code"],
] as const;

test("a fill waiting for the owner shows each step in its menu, then fills the field", async () => {
  for (const [field, kind, filled] of waitingFills) {
    const { ravenpass, relayed, serve } = router("field");
    ravenpass.progress = ["confirm-on-device", "confirm-in-ravenpass"];
    const token = await openFieldMenu(serve, field);

    const answer = await serve(
      { kind, token, id: "a1", confirmed: false },
      menuSender(),
    );

    assert.deepEqual(answer, { ok: true });
    assert.deepEqual(relayed.slice(0, 2), [
      [
        menuDocument,
        { kind: "fill-progress", token, progress: "confirm-on-device" },
      ],
      [
        menuDocument,
        { kind: "fill-progress", token, progress: "confirm-in-ravenpass" },
      ],
    ]);
    assert.deepEqual(kinds(relayed.slice(2)), [filled]);
    assert.deepEqual(relayed[2]?.[0], fieldFrame);
  }
});

test("a code shown while the owner confirms it reports each step to its menu", async () => {
  const { ravenpass, relayed, serve } = router("field");
  ravenpass.progress = ["confirm-on-device"];
  const token = await openFieldMenu(serve, "code");

  const answer = await serve(
    { kind: "menu-code", token, id: "a1" },
    menuSender(),
  );

  assert.deepEqual(answer, {
    ok: true,
    code: { code: "287082", digits: 6, period: 30, expiresAt: 0 },
  });
  assert.deepEqual(relayed, [
    [
      menuDocument,
      { kind: "fill-progress", token, progress: "confirm-on-device" },
    ],
  ]);
});

test("a fill or a code the owner declines or cannot confirm answers why and fills nothing", async () => {
  for (const reason of ["declined", "unverifiable"] as const) {
    for (const [field, kind] of waitingFills) {
      const { ravenpass, relayed, serve } = router("field");
      ravenpass.progress = ["confirm-on-device"];
      ravenpass.verification = new SessionError(reason);
      const token = await openFieldMenu(serve, field);

      const answer = await serve(
        { kind, token, id: "a1", confirmed: false },
        menuSender(),
      );

      assert.deepEqual(answer, { ok: false, reason });
      assert.deepEqual(kinds(relayed), ["fill-progress"]);
    }
    const { ravenpass, serve } = router("field");
    ravenpass.verification = new SessionError(reason);
    const token = await openFieldMenu(serve, "code");
    assert.deepEqual(
      await serve({ kind: "menu-code", token, id: "a1" }, menuSender()),
      { ok: false, reason },
    );
  }
});

test("a card's fill waiting for the owner shows each step in the card, then fills the form", async () => {
  const { ravenpass, relayed, serve, bindCard } = router("card");
  ravenpass.progress = ["confirm-on-device"];
  await serve({ kind: "sign-in-form", field: "password" }, loginFrame);
  const token = await bindCard();
  const shown = relayed.length;

  await serve(
    { kind: "menu-fill", token, id: "a1", confirmed: false },
    menuSender(),
  );

  assert.deepEqual(relayed[shown], [
    menuDocument,
    { kind: "fill-progress", token, progress: "confirm-on-device" },
  ]);
  assert.deepEqual(kinds(relayed.slice(shown + 1)), [
    "form-sign-in",
    "card-hide",
  ]);
});

test("a fill the owner's choice does not hold shows no step in its menu", async () => {
  for (const [field, kind, filled] of waitingFills) {
    const { relayed, serve } = router("field");
    const token = await openFieldMenu(serve, field);

    await serve({ kind, token, id: "a1", confirmed: false }, menuSender());

    assert.deepEqual(kinds(relayed), [filled]);
  }
});
