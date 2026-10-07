import assert from "node:assert/strict";
import test from "node:test";
import type { OneTimeCode, SiteIcon } from "@ravenpass/ui/vault-api.ts";
import {
  type CardOption,
  type CardValues,
  type CodeSuggestion,
  type FillValues,
  type IdentityFiles,
  type PasskeyOption,
  SessionError,
  type SessionFailure,
  type SharedFile,
} from "../link/client.ts";
import type { Answers, MenuRequest, PasskeyPageRequest } from "../messages.ts";
import {
  alreadyPending,
  alreadyRegistered,
  native,
  notAllowed,
  type PageAnswer,
} from "../passkeys/page-channel.ts";
import type {
  CreateOptions,
  GetOptions,
  PageRequest,
} from "../passkeys/requests.ts";
import { CardFrames } from "./card-frames.ts";
import { type MenuClient, MenuRouter, type TabMessenger } from "./menus.ts";
import { PagePasskeys } from "./page-passkeys.ts";
import { PendingPasskeys } from "./pending-passkeys.ts";
import { PendingSignIns } from "./pending-sign-ins.ts";
import { RecentFills } from "./recent-fills.ts";
import { MenuSessions, type Sender } from "./sessions.ts";
import { SignInCards } from "./sign-in-cards.ts";
import {
  Clock,
  createdPasskey,
  extensionId,
  MemoryArea,
  menuSender,
  pageSender,
  ScriptedPasskeys,
  signedPasskey,
} from "./test-doubles.test-support.ts";

const tabId = 7;
const origin = "https://github.com";
const page = { tabId, documentId: "page-document", origin };
const topFrame = { tabId, frameId: 0 };

const getOptions: GetOptions = {
  rpId: "github.com",
  challenge: "AQI",
  timeout: null,
  allow: [],
  userVerification: "required",
  hints: [],
};

const createOptions: CreateOptions = {
  rpId: "github.com",
  rpName: "GitHub",
  user: { id: "dXNlcg", name: "alex@example.com", displayName: "Alex" },
  challenge: "AQI",
  algorithms: [-7, -257],
  timeout: null,
  exclude: [],
  attachment: null,
  userVerification: "discouraged",
  hints: [],
  credProps: true,
};

const signIn: PageRequest = {
  mode: "get",
  options: getOptions,
  conditional: false,
};
const creation: PageRequest = { mode: "create", options: createOptions };
const conditional: PageRequest = {
  mode: "get",
  options: getOptions,
  conditional: true,
};

const passkey: PasskeyOption = {
  credential: "c1",
  credentialId: "AQID",
  account: "alex@example.com",
  label: "GitHub",
};

/** A desktop app without passwords for any site. */
class NoPasswords implements MenuClient {
  async suggest(): Promise<CodeSuggestion[]> {
    return [];
  }

  fill(): Promise<FillValues> {
    return Promise.reject(new Error("Passkey tests fill nothing."));
  }

  code(): Promise<OneTimeCode> {
    return Promise.reject(new Error("Passkey tests show no code."));
  }

  addWebsite(): Promise<void> {
    return Promise.reject(new Error("Passkey tests add no website."));
  }

  async icon(): Promise<SiteIcon> {
    return { image: "", tint: "" };
  }

  async unlock(): Promise<void> {}

  async identities(): Promise<IdentityFiles[]> {
    return [];
  }

  share(): Promise<SharedFile> {
    return Promise.reject(new Error("Passkey tests share no file."));
  }

  async cards(): Promise<CardOption[]> {
    return [];
  }

  fillCard(): Promise<CardValues> {
    return Promise.reject(new Error("Passkey tests fill no card."));
  }
}

type Relayed = [Parameters<TabMessenger>[0], Parameters<TabMessenger>[1]];

/** The top frame shows every passkey card unless `refuses`; the sign-in style is field. */
function worker({ refuses = false } = {}) {
  const area = new MemoryArea();
  const clock = new Clock();
  let issued = 0;
  const sessions = new MenuSessions({
    area,
    extensionId,
    now: clock.now,
    newToken: () => `token-${++issued}`,
  });
  const ravenpass = new ScriptedPasskeys();
  const relayed: Relayed[] = [];
  const relay: TabMessenger = async (target, message) => {
    relayed.push([target, message]);
    if (message.kind === "card-show") return { shown: !refuses };
    return undefined;
  };
  const cards = new SignInCards({ sessions, relay });
  const passkeys = new PagePasskeys({
    client: ravenpass,
    sessions,
    cards,
    pending: new PendingPasskeys({ area }),
    relay,
  });
  const menus = new MenuRouter({
    client: new NoPasswords(),
    sessions,
    cards,
    passkeys,
    recentFills: new RecentFills({ area, now: clock.now }),
    pendingSignIns: new PendingSignIns({ area, now: clock.now }),
    cardFrames: new CardFrames({ area }),
    signInStyle: async () => "field",
    relay,
  });
  const ask = <Kind extends PasskeyPageRequest["kind"]>(
    request: Extract<PasskeyPageRequest, { kind: Kind }>,
    sender: Sender = pageSender(),
  ) => passkeys.serve(request, sender) as Promise<Answers[Kind]>;
  const request = (pageRequest: PageRequest, id = 1, sender?: Sender) =>
    ask({ kind: "passkey-request", id, request: pageRequest }, sender);
  const menu = <Kind extends MenuRequest["kind"]>(
    request: Extract<MenuRequest, { kind: Kind }>,
    sender: Sender = menuSender(),
  ) => menus.serve(request, sender) as Promise<Answers[Kind]>;
  /** Binds the tab's card to its menu page, as the page does when it loads. */
  const bindCard = async () => {
    const card = await sessions.cardOf(tabId);
    assert.ok(card);
    await menu({ kind: "menu", token: card.token });
    return card.token;
  };
  /** The answers relayed to pages' documents. */
  const answers = () =>
    relayed
      .filter(([, message]) => message.kind === "passkey-answer")
      .map(([target, message]) => [target, message]);
  return {
    sessions,
    ravenpass,
    relayed,
    passkeys,
    ask,
    request,
    menu,
    bindCard,
    answers,
  };
}

const answered = (id: number, answer: PageAnswer): Relayed => [
  page,
  { kind: "passkey-answer", id, answer },
];

const refusal = (reason: SessionFailure) => new SessionError(reason);

test("while unlinked, every request goes to Chrome", async () => {
  const { ravenpass, request, relayed } = worker();
  ravenpass.linkedNow = false;
  ravenpass.refusal = refusal("unlinked");

  for (const pageRequest of [signIn, creation, conditional]) {
    assert.deepEqual(await request(pageRequest), { answer: native });
  }
  assert.deepEqual(relayed, []);
});

test("while Ravenpass is not open, a sign-in and a creation show the card saying so", async () => {
  for (const pageRequest of [signIn, creation]) {
    const { sessions, ravenpass, request, relayed } = worker();
    ravenpass.refusal = refusal("not-open");

    assert.deepEqual(await request(pageRequest), { answer: null });

    assert.deepEqual(relayed, [
      [topFrame, { kind: "card-show", token: "token-1", reopen: true }],
    ]);
    assert.deepEqual((await sessions.cardOf(tabId))?.content, {
      state: "passkey",
      request: 1,
      mode: pageRequest.mode,
      listing: { state: "not-open" },
    });
  }
});

test("while Ravenpass is locked, the card offers Unlock and lists the passkeys once Ravenpass is unlocked", async () => {
  const { sessions, ravenpass, request, menu, bindCard } = worker();
  ravenpass.refusal = refusal("locked");
  await request(signIn);
  const token = await bindCard();
  assert.deepEqual(await menu({ kind: "passkey-review", token }), {
    listing: { state: "locked" },
  });
  assert.deepEqual(await menu({ kind: "menu-unlock", token }), { ok: true });

  ravenpass.refusal = null;
  ravenpass.listed = [passkey];
  const listing = { state: "sign-in", passkeys: [passkey] };

  assert.deepEqual(await menu({ kind: "passkey-review", token }), { listing });
  assert.deepEqual((await sessions.cardOf(tabId))?.content, {
    state: "passkey",
    request: 1,
    mode: "get",
    listing,
  });
});

test("a locked sign-in goes to Chrome once Ravenpass is unlocked without a passkey for the site", async () => {
  const { sessions, ravenpass, request, menu, bindCard, answers } = worker();
  ravenpass.refusal = refusal("locked");
  await request(signIn);
  const token = await bindCard();

  ravenpass.refusal = null;
  assert.deepEqual(await menu({ kind: "passkey-review", token }), {
    listing: null,
  });
  assert.deepEqual(answers(), [answered(1, native)]);
  assert.equal(await sessions.cardOf(tabId), null);
});

test("while unlocked, a sign-in without a matching passkey goes to Chrome, and a creation shows the save card", async () => {
  const { sessions, ravenpass, request } = worker();

  assert.deepEqual(await request(signIn, 1), { answer: native });
  assert.deepEqual(ravenpass.asked, [["passkeys", origin, getOptions]]);

  ravenpass.targets = {
    targets: [
      { credential: "c0", label: "Old GitHub", account: "alexander", tags: [] },
      {
        credential: "c1",
        label: "GitHub",
        account: "alex@example.com",
        tags: [],
      },
    ],
    excluded: false,
  };
  assert.deepEqual(await request(creation, 2), { answer: null });
  assert.deepEqual(ravenpass.asked.at(-1), ["targets", origin, createOptions]);
  assert.deepEqual((await sessions.cardOf(tabId))?.content, {
    state: "passkey",
    request: 2,
    mode: "create",
    listing: {
      state: "save",
      account: "alex@example.com",
      targets: ravenpass.targets.targets,
    },
  });
});

test("choosing a listed passkey signs with the origin Chrome reported, answers the document and closes the card", async () => {
  const { ravenpass, request, menu, bindCard, relayed, answers } = worker();
  ravenpass.listed = [passkey];
  ravenpass.progress = ["confirm-on-device"];
  await request(signIn, 3);
  const token = await bindCard();

  const answer = await menu({
    kind: "passkey-sign",
    token,
    credential: "c1",
    credentialId: "AQID",
  });

  assert.deepEqual(answer, { ok: true });
  assert.deepEqual(ravenpass.signed, [
    [origin, getOptions, { credential: "c1", credentialId: "AQID" }],
  ]);
  assert.deepEqual(relayed.slice(-3), [
    [
      { tabId, documentId: "menu-document" },
      { kind: "passkey-progress", token, progress: "confirm-on-device" },
    ],
    answered(3, { kind: "signed", passkey: signedPasskey }),
    [topFrame, { kind: "card-hide", token, dismissed: false }],
  ]);
  assert.equal(answers().length, 1);
});

test("a card signs only with a passkey it lists", async () => {
  const { ravenpass, request, menu, bindCard } = worker();
  ravenpass.listed = [passkey];
  await request(signIn);
  const token = await bindCard();

  for (const [credential, credentialId] of [
    ["c2", "AQID"],
    ["c1", "BBBB"],
  ] as const) {
    await assert.rejects(
      menu({ kind: "passkey-sign", token, credential, credentialId }),
      { name: "RefusedRequest" },
    );
  }
  assert.deepEqual(ravenpass.signed, []);
});

test("saving answers the page with the new passkey kept where the person chose", async () => {
  const { ravenpass, request, menu, bindCard, relayed } = worker();
  ravenpass.targets = {
    targets: [{ credential: "c1", label: "GitHub", account: "alex", tags: [] }],
    excluded: false,
  };
  await request(creation, 5);
  const token = await bindCard();

  await assert.rejects(menu({ kind: "passkey-save", token, target: "c9" }), {
    name: "RefusedRequest",
  });
  assert.deepEqual(await menu({ kind: "passkey-save", token, target: "c1" }), {
    ok: true,
  });

  assert.deepEqual(ravenpass.created, [[origin, createOptions, "c1"]]);
  assert.deepEqual(relayed.slice(-2), [
    answered(5, { kind: "created", passkey: createdPasskey }),
    [topFrame, { kind: "card-hide", token, dismissed: false }],
  ]);
});

test("a creation Ravenpass already holds a passkey for shows it, and closing the card refuses as an existing credential", async () => {
  const { sessions, ravenpass, request, menu, bindCard, answers } = worker();
  ravenpass.targets = { targets: [], excluded: true };
  await request(creation);
  assert.deepEqual((await sessions.cardOf(tabId))?.content, {
    state: "passkey",
    request: 1,
    mode: "create",
    listing: { state: "excluded" },
  });
  const token = await bindCard();

  assert.deepEqual(await menu({ kind: "menu-close", token }), { ok: true });

  assert.deepEqual(answers(), [answered(1, alreadyRegistered)]);
  assert.equal(await sessions.cardOf(tabId), null);
});

test("a save Ravenpass refuses as excluded, locked or closed shows that on the card", async () => {
  for (const reason of ["excluded", "locked", "not-open"] as const) {
    const { sessions, ravenpass, request, menu, bindCard, answers } = worker();
    await request(creation);
    const token = await bindCard();
    ravenpass.verification = refusal(reason);

    assert.deepEqual(
      await menu({ kind: "passkey-save", token, target: "new" }),
      {
        ok: false,
        reason,
      },
    );
    const card = await sessions.cardOf(tabId);
    assert.deepEqual(
      card?.content.state === "passkey" && card.content.listing,
      { state: reason },
    );
    assert.deepEqual(answers(), []);
  }
});

test("a declined or unverifiable sign-in, and a full target, keep the card open with the reason", async () => {
  for (const reason of ["declined", "unverifiable", "full"] as const) {
    const { sessions, ravenpass, request, menu, bindCard, answers } = worker();
    ravenpass.listed = [passkey];
    ravenpass.targets = {
      targets: [
        { credential: "c1", label: "GitHub", account: "alex", tags: [] },
      ],
      excluded: false,
    };
    await request(reason === "full" ? creation : signIn);
    const token = await bindCard();
    ravenpass.verification = refusal(reason);

    const answer =
      reason === "full"
        ? await menu({ kind: "passkey-save", token, target: "c1" })
        : await menu({
            kind: "passkey-sign",
            token,
            credential: "c1",
            credentialId: "AQID",
          });

    assert.deepEqual(answer, { ok: false, reason });
    assert.equal((await sessions.cardOf(tabId))?.token, token);
    assert.deepEqual(answers(), []);
  }
});

test("a relying party ID or a field Ravenpass refuses hands the request to Chrome", async () => {
  for (const reason of ["invalid-rp", "invalid-request"] as const) {
    const { ravenpass, request } = worker();
    ravenpass.refusal = refusal(reason);
    assert.deepEqual(await request(signIn), { answer: native });
    assert.deepEqual(await request(creation, 2), { answer: native });
  }

  const { sessions, ravenpass, request, menu, bindCard, answers } = worker();
  ravenpass.listed = [passkey];
  await request(signIn);
  const token = await bindCard();
  ravenpass.verification = refusal("invalid-rp");

  assert.deepEqual(
    await menu({
      kind: "passkey-sign",
      token,
      credential: "c1",
      credentialId: "AQID",
    }),
    { ok: true },
  );
  assert.deepEqual(answers(), [answered(1, native)]);
  assert.equal(await sessions.cardOf(tabId), null);
});

test("requests Ravenpass cannot serve go to Chrome without asking Ravenpass", async () => {
  const { ravenpass, request } = worker();
  const requests: PageRequest[] = [
    { mode: "create", options: { ...createOptions, algorithms: [-257] } },
    {
      mode: "create",
      options: { ...createOptions, attachment: "cross-platform" },
    },
    { mode: "create", options: { ...createOptions, hints: ["security-key"] } },
    {
      mode: "get",
      options: { ...getOptions, hints: ["security-key"] },
      conditional: false,
    },
  ];
  for (const pageRequest of requests) {
    assert.deepEqual(await request(pageRequest), { answer: native });
  }
  assert.deepEqual(ravenpass.asked, []);
});

test("only the top frame of a web page on https or http://localhost is served", async () => {
  const { ravenpass, request, ask } = worker();
  ravenpass.listed = [passkey];
  for (const sender of [
    pageSender({ frameId: 2 }),
    pageSender({ origin: "http://github.com" }),
    pageSender({ origin: "http://localhost.example.com" }),
    pageSender({ documentId: undefined }),
    pageSender({ id: "another-extension" }),
    pageSender({ tab: undefined }),
    menuSender({ frameId: 0 }),
  ]) {
    assert.deepEqual(await request(signIn, 1, sender), { answer: native });
    assert.deepEqual(await ask({ kind: "passkey-linked" }, sender), {
      linked: false,
    });
  }
  assert.deepEqual(ravenpass.asked, []);

  const local = pageSender({ origin: "http://localhost:8080" });
  assert.deepEqual(await request(signIn, 1, local), { answer: null });
  assert.deepEqual(ravenpass.asked, [
    ["passkeys", "http://localhost:8080", getOptions],
  ]);
  assert.deepEqual(await ask({ kind: "passkey-linked" }, local), {
    linked: true,
  });
});

test("a malformed request goes to Chrome", async () => {
  const { ravenpass, ask } = worker();
  for (const [id, pageRequest] of [
    [1, { ...signIn, options: { ...getOptions, challenge: 7 } }],
    [1.5, signIn],
  ] as const) {
    assert.deepEqual(
      await ask({
        kind: "passkey-request",
        id,
        request: pageRequest as PageRequest,
      }),
      { answer: native },
    );
  }
  assert.deepEqual(ravenpass.asked, []);
});

test("a second modal request while one waits is refused as Chrome refuses it", async () => {
  const { ravenpass, request } = worker();
  ravenpass.listed = [passkey];
  await request(signIn, 1);
  assert.deepEqual(await request(creation, 2), { answer: alreadyPending });
});

test("closing the card refuses the request as not allowed", async () => {
  const { sessions, ravenpass, request, menu, bindCard, answers } = worker();
  ravenpass.listed = [passkey];
  await request(signIn);
  const token = await bindCard();

  await menu({ kind: "menu-close", token });

  assert.deepEqual(answers(), [answered(1, notAllowed)]);
  assert.equal(await sessions.cardOf(tabId), null);
});

test("the top frame closing the card, as when the page is hidden, refuses the request", async () => {
  const { ravenpass, request, menu, answers } = worker();
  ravenpass.listed = [passkey];
  await request(signIn);

  await menu({ kind: "menu-close", token: "token-1" }, pageSender());

  assert.deepEqual(answers(), [answered(1, notAllowed)]);
});

test("Other options hands the request to Chrome and closes the card", async () => {
  const { sessions, ravenpass, request, menu, bindCard, relayed } = worker();
  ravenpass.refusal = refusal("not-open");
  await request(creation);
  const token = await bindCard();

  assert.deepEqual(await menu({ kind: "passkey-elsewhere", token }), {
    ok: true,
  });

  assert.deepEqual(relayed.slice(-2), [
    answered(1, native),
    [topFrame, { kind: "card-hide", token, dismissed: false }],
  ]);
  assert.equal(await sessions.cardOf(tabId), null);
});

test("a request the page withdraws, aborted or timed out, closes its card and takes no answer", async () => {
  const {
    sessions,
    ravenpass,
    request,
    ask,
    menu,
    bindCard,
    relayed,
    answers,
  } = worker();
  ravenpass.listed = [passkey];
  await request(signIn, 4);
  const token = await bindCard();

  assert.deepEqual(await ask({ kind: "passkey-withdraw", id: 4 }), {
    ok: true,
  });

  assert.deepEqual(relayed.at(-1), [
    topFrame,
    { kind: "card-hide", token, dismissed: false },
  ]);
  assert.equal(await sessions.cardOf(tabId), null);
  await assert.rejects(
    menu({
      kind: "passkey-sign",
      token,
      credential: "c1",
      credentialId: "AQID",
    }),
    { name: "RefusedRequest" },
  );
  assert.deepEqual(answers(), []);
  await assert.rejects(
    ask({ kind: "passkey-withdraw", id: 4 }, pageSender({ frameId: 1 })),
    { name: "RefusedRequest" },
  );
});

test("a card the top frame does not show, as while a save offer shows, hands the request to Chrome", async () => {
  const { sessions, ravenpass, request } = worker({ refuses: true });
  ravenpass.listed = [passkey];
  assert.deepEqual(await request(signIn), { answer: native });
  assert.equal(await sessions.cardOf(tabId), null);
  assert.deepEqual(await request(signIn, 2), { answer: native });
});

test("a passkey card keeps the corner from sign-in cards until it is answered", async () => {
  const { sessions, ravenpass, request, menu, relayed } = worker();
  ravenpass.listed = [passkey];
  await request(signIn);

  await menu({ kind: "sign-in-form", field: "login" }, pageSender());

  assert.equal(
    relayed.filter(([, { kind }]) => kind === "card-show").length,
    1,
  );
  assert.equal((await sessions.cardOf(tabId))?.content.state, "passkey");
});

test("a conditional request waits for a sign-in field of its document, whose menu lists its passkeys first and answers it", async () => {
  const { sessions, ravenpass, request, menu, relayed, answers } = worker();
  ravenpass.listed = [passkey];

  assert.deepEqual(await request(conditional, 6), { answer: null });
  assert.deepEqual(ravenpass.asked, []);
  assert.deepEqual(relayed, []);

  const { token } = await menu(
    { kind: "menu-open", field: "login", requested: false },
    pageSender(),
  );
  assert.ok(token);
  const { content } = await menu({ kind: "menu", token });
  assert.deepEqual(content, {
    state: "list",
    purpose: "sign-in",
    credentials: [],
    passkeys: [passkey],
  });

  assert.deepEqual(
    await menu({
      kind: "passkey-sign",
      token,
      credential: "c1",
      credentialId: "AQID",
    }),
    { ok: true },
  );
  assert.deepEqual(ravenpass.signed, [
    [origin, getOptions, { credential: "c1", credentialId: "AQID" }],
  ]);
  assert.deepEqual(answers(), [
    answered(6, { kind: "signed", passkey: signedPasskey }),
  ]);
  assert.deepEqual(relayed.at(-1), [page, { kind: "menu-close", token }]);
  assert.equal(await sessions.forMenu(token, menuSender()), null);
});

test("a sign-in field of another document, or after the conditional request ends, lists no passkey", async () => {
  const { ravenpass, request, ask, menu } = worker();
  ravenpass.listed = [passkey];
  await request(conditional, 6);

  const frame = pageSender({ frameId: 2, documentId: "login-frame" });
  assert.deepEqual(
    await menu({ kind: "menu-open", field: "login", requested: false }, frame),
    {
      token: null,
    },
  );

  await ask({ kind: "passkey-withdraw", id: 6 });
  assert.deepEqual(
    await menu(
      { kind: "menu-open", field: "login", requested: false },
      pageSender(),
    ),
    { token: null },
  );
});

test("a later document of the tab, and a closed tab, forget the requests of the one before", async () => {
  const { ravenpass, passkeys, request, menu } = worker();
  ravenpass.listed = [passkey];
  await request(conditional, 6);

  const next = pageSender({ documentId: "next-document" });
  await request(conditional, 1, next);
  assert.deepEqual(
    await menu(
      { kind: "menu-open", field: "login", requested: false },
      pageSender(),
    ),
    { token: null },
  );

  await passkeys.tabClosed(tabId);
  assert.deepEqual(
    await menu({ kind: "menu-open", field: "login", requested: false }, next),
    {
      token: null,
    },
  );
});
