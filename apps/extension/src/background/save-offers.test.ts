import assert from "node:assert/strict";
import test from "node:test";
import {
  type CapturedPassword,
  type CaptureOffer,
  type PendingCapture,
  type SaveChoice,
  type SavedAs,
  SessionError,
} from "../link/client.ts";
import type { FrameMessage, OfferRequest } from "../messages.ts";
import { RefusedRequest } from "./menus.ts";
import { captureFailure, type OfferClient, SaveOffers } from "./save-offers.ts";
import { MenuSessions, type Sender, type TabDocument } from "./sessions.ts";
import { TabOffers } from "./tab-offers.ts";
import {
  Clock,
  extensionId,
  MemoryArea,
  menuSender,
  pageSender,
} from "./test-doubles.test-support.ts";

const tabId = 7;
const typedPassword = "correct horse";
const currentPassword = "old horse";
const lifetimeMs = 3 * 60_000;

const ready: PendingCapture = {
  state: "ready",
  pending: "p1",
  site: "github.com",
  account: "alex",
  name: "github.com",
  targets: [
    {
      credential: "a1",
      label: "GitHub",
      account: "alex",
      action: "update",
      tags: [],
    },
  ],
  suggested: "a1",
};

const locked: PendingCapture = {
  ...ready,
  state: "locked",
  targets: [],
  suggested: "",
};

/** Answers captures and reviews in turn from `answers`. */
class ScriptedRavenpass implements OfferClient {
  readonly captures: [string, CapturedPassword][] = [];
  readonly reviews: string[] = [];
  readonly saves: [string, SaveChoice][] = [];
  readonly discards: string[] = [];
  answers: (CaptureOffer | SessionError)[] = [];
  saved: SavedAs | SessionError = "updated";
  discardFailure: Error | null = null;

  async capture(
    origin: string,
    { account, password, current }: CapturedPassword,
  ): Promise<CaptureOffer> {
    this.captures.push([
      origin,
      current === undefined
        ? { account, password }
        : { account, password, current },
    ]);
    return this.answer();
  }

  async review(pending: string): Promise<CaptureOffer> {
    this.reviews.push(pending);
    return this.answer();
  }

  async save(
    pending: string,
    { target, account, name }: SaveChoice,
  ): Promise<SavedAs> {
    this.saves.push([pending, { target, account, name }]);
    if (this.saved instanceof SessionError) throw this.saved;
    return this.saved;
  }

  async discard(pending: string): Promise<void> {
    this.discards.push(pending);
    if (this.discardFailure) throw this.discardFailure;
  }

  private answer(): CaptureOffer {
    const answer = this.answers.shift();
    if (!answer) throw new Error("The script has no answer left.");
    if (answer instanceof SessionError) throw answer;
    return answer;
  }
}

function saveOffers() {
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
  const relayed: [TabDocument, FrameMessage][] = [];
  const shown: number[] = [];
  const router = new SaveOffers({
    client: ravenpass,
    sessions,
    offers: new TabOffers({ area, sessions, now: clock.now }),
    relay: async (target, message) => {
      relayed.push([target, message]);
    },
    show: async (tab) => {
      shown.push(tab);
    },
  });
  /** Defaults to a top-frame document loaded after the sign-in page that shows no sign-in form. */
  const offerOpen = async (
    documentId = "next-document",
    signInForm = false,
    sender: Partial<Sender> = {},
  ) => {
    const { token } = (await router.serve(
      { kind: "offer-open", signInForm },
      pageSender({ documentId, ...sender }),
    )) as { token: string | null };
    return token;
  };
  const openMenu = async (documentId = "next-document") => {
    const token = await offerOpen(documentId);
    assert.ok(token);
    await sessions.bind(token, menuSender({ documentId: `menu-${token}` }));
    return token;
  };
  const fromMenu = (request: OfferRequest & { token: string }) =>
    router.serve(request, menuSender({ documentId: `menu-${request.token}` }));
  return {
    area,
    clock,
    sessions,
    ravenpass,
    relayed,
    shown,
    router,
    offerOpen,
    openMenu,
    fromMenu,
  };
}

const capture: OfferRequest = {
  kind: "capture",
  account: "alex",
  password: typedPassword,
};

function stored(area: MemoryArea): string {
  return JSON.stringify([...area.items]);
}

test("a capture goes to Ravenpass with the origin Chrome reports for its frame, and no password stays", async () => {
  const { area, clock, ravenpass, shown, router, sessions, openMenu } =
    saveOffers();
  ravenpass.answers = [ready];

  const answer = await router.serve(
    { ...capture, current: currentPassword },
    pageSender({
      frameId: 2,
      url: "https://accounts.github.com/sign-in",
      origin: "https://accounts.github.com",
    }),
  );

  assert.deepEqual(answer, { ok: true });
  assert.deepEqual(ravenpass.captures, [
    [
      "https://accounts.github.com",
      { account: "alex", password: typedPassword, current: currentPassword },
    ],
  ]);
  assert.deepEqual(shown, []);
  const token = await openMenu();
  const { pending: _, ...offer } = ready;
  assert.deepEqual(
    (await sessions.forMenu(token, menuSender({ documentId: `menu-${token}` })))
      ?.content,
    { state: "offer", offer: { ...offer, expiresAt: clock.time + lifetimeMs } },
  );
  assert.equal(stored(area).includes(typedPassword), false);
  assert.equal(stored(area).includes(currentPassword), false);
});

test("a capture Ravenpass does not answer is dropped and its failure logged by name", async (t) => {
  for (const reason of ["unlinked", "not-open", "invalid-origin"] as const) {
    const errors = t.mock.method(console, "error", () => {});
    const { area, ravenpass, shown, router } = saveOffers();
    ravenpass.answers = [new SessionError(reason)];

    assert.deepEqual(await router.serve(capture, pageSender()), { ok: true });
    assert.deepEqual(shown, []);
    assert.equal(area.items.size, 0);
    assert.deepEqual(
      errors.mock.calls.map((call) => call.arguments),
      [[captureFailure, "SessionError"]],
    );
    errors.mock.restore();
  }
});

test("a capture with nothing to offer shows nothing", async () => {
  const { area, ravenpass, shown, router } = saveOffers();
  ravenpass.answers = [{ state: "none" }];

  await router.serve(capture, pageSender());

  assert.deepEqual(shown, []);
  assert.equal(area.items.size, 0);
});

test("a capture from anything but a web page's frame is refused", async () => {
  const { ravenpass, router } = saveOffers();

  await assert.rejects(router.serve(capture, menuSender()), RefusedRequest);
  assert.deepEqual(ravenpass.captures, []);
});

test("a newer capture discards the older one in Ravenpass and closes its offer", async () => {
  const { ravenpass, relayed, router, offerOpen, openMenu } = saveOffers();
  ravenpass.answers = [ready, { ...ready, pending: "p2", account: "sam" }];
  await router.serve(capture, pageSender());
  const first = await openMenu();

  await router.serve(
    { ...capture, account: "sam" },
    pageSender({ documentId: "next-document" }),
  );

  assert.deepEqual(ravenpass.discards, ["p1"]);
  assert.deepEqual(relayed, [
    [
      { tabId, documentId: "next-document" },
      { kind: "menu-close", token: first },
    ],
  ]);
  assert.equal(await offerOpen("next-document"), null);
  assert.notEqual(await openMenu("third-document"), first);
});

test("a discard Ravenpass cannot serve still replaces the offer, and any other failure surfaces", async () => {
  const { ravenpass, router, openMenu } = saveOffers();
  ravenpass.answers = [ready, { ...ready, pending: "p2" }];
  ravenpass.discardFailure = new SessionError("not-open");
  await router.serve(capture, pageSender());
  await openMenu();
  await router.serve(capture, pageSender({ documentId: "next-document" }));
  assert.deepEqual(ravenpass.discards, ["p1"]);

  const broken = saveOffers();
  broken.ravenpass.answers = [ready, { ...ready, pending: "p2" }];
  broken.ravenpass.discardFailure = new TypeError("A bug.");
  await broken.router.serve(capture, pageSender());
  await assert.rejects(
    broken.router.serve(capture, pageSender({ documentId: "next-document" })),
    TypeError,
  );
});

test("a newer capture with nothing to offer ends the older offer", async () => {
  const { ravenpass, relayed, router, offerOpen, openMenu } = saveOffers();
  ravenpass.answers = [ready, { state: "none" }];
  await router.serve(capture, pageSender());
  await openMenu();

  await router.serve(capture, pageSender());

  assert.deepEqual(ravenpass.discards, ["p1"]);
  assert.equal(relayed.length, 1);
  assert.equal(await offerOpen("third-document"), null);
});

test("only the tab's top frame gets a token for its offer", async () => {
  const { ravenpass, router, offerOpen } = saveOffers();
  ravenpass.answers = [ready];
  await router.serve(
    capture,
    pageSender({ frameId: 2, documentId: "login-frame" }),
  );

  assert.equal(await offerOpen("login-frame", false, { frameId: 2 }), null);
  assert.equal(await offerOpen("page-document"), "token-1");
});

test("an offer waits while the form it was captured from stays on its page", async () => {
  const { ravenpass, shown, router, offerOpen } = saveOffers();
  ravenpass.answers = [ready];

  await router.serve(capture, pageSender());

  assert.deepEqual(shown, []);
  assert.equal(await offerOpen("page-document"), null);
});

test("the capturing frame reporting its form gone readies the offer and shows it", async () => {
  const { ravenpass, shown, router, offerOpen } = saveOffers();
  ravenpass.answers = [ready];
  await router.serve(capture, pageSender());

  assert.deepEqual(await router.serve({ kind: "form-gone" }, pageSender()), {
    ok: true,
  });

  assert.deepEqual(shown, [tabId]);
  assert.equal(await offerOpen("page-document"), "token-1");
});

test("a form gone from another document readies nothing", async () => {
  const { ravenpass, shown, router, offerOpen } = saveOffers();
  ravenpass.answers = [ready];
  await router.serve(capture, pageSender());

  await router.serve(
    { kind: "form-gone" },
    pageSender({ frameId: 2, documentId: "ad-frame" }),
  );

  assert.deepEqual(shown, []);
  assert.equal(await offerOpen("page-document"), null);
});

test("a later document without a sign-in form readies the offer, and it stays ready", async () => {
  const { ravenpass, router, offerOpen } = saveOffers();
  ravenpass.answers = [ready];
  await router.serve(capture, pageSender());

  assert.equal(await offerOpen("next-document", false), "token-1");
  assert.equal(await offerOpen("third-document", true), "token-2");
});

test("a later document showing a sign-in form for the site keeps the offer waiting", async () => {
  const { ravenpass, shown, router, offerOpen } = saveOffers();
  ravenpass.answers = [ready, { ...ready, pending: "p2" }];
  await router.serve(capture, pageSender());

  assert.equal(await offerOpen("next-document", true), null);
  assert.equal(await offerOpen("next-document", true), null);

  await router.serve(capture, pageSender({ documentId: "next-document" }));
  assert.deepEqual(ravenpass.discards, ["p1"]);
  assert.deepEqual(shown, []);
});

test("a sign-in form on another site readies the offer", async () => {
  const { ravenpass, router, offerOpen } = saveOffers();
  ravenpass.answers = [ready];
  await router.serve(capture, pageSender());

  const token = await offerOpen("next-document", true, {
    url: "https://example.com/login",
    origin: "https://example.com",
  });

  assert.equal(token, "token-1");
});

test("the offer's menu saves where the person chose, and the tab forgets the offer", async () => {
  const { ravenpass, router, offerOpen, openMenu, fromMenu } = saveOffers();
  ravenpass.answers = [ready];
  await router.serve(capture, pageSender());
  const token = await openMenu();

  const answer = await fromMenu({
    kind: "offer-save",
    token,
    target: "",
    account: "alex@example.com",
    name: "GitHub",
  });

  assert.deepEqual(answer, { ok: true, saved: "updated" });
  assert.deepEqual(ravenpass.saves, [
    ["p1", { target: "", account: "alex@example.com", name: "GitHub" }],
  ]);
  assert.equal(await offerOpen("third-document"), null);
});

test("a save while Ravenpass is locked keeps the offer, locked, for the next document", async () => {
  const { ravenpass, router, sessions, openMenu, fromMenu } = saveOffers();
  ravenpass.answers = [ready];
  ravenpass.saved = new SessionError("locked");
  await router.serve(capture, pageSender());
  const token = await openMenu();

  const answer = await fromMenu({
    kind: "offer-save",
    token,
    target: "a1",
    account: "alex",
    name: "github.com",
  });

  assert.deepEqual(answer, { ok: false, reason: "locked" });
  const next = await openMenu("next-document");
  const session = await sessions.forMenu(
    next,
    menuSender({ documentId: `menu-${next}` }),
  );
  assert.equal(
    session?.content.state === "offer" && session.content.offer.state,
    "locked",
  );
});

test("a review keeps what Ravenpass offers once it is unlocked, without a password", async () => {
  const { area, clock, ravenpass, router, sessions, openMenu, fromMenu } =
    saveOffers();
  ravenpass.answers = [locked, ready];
  await router.serve({ ...capture, current: currentPassword }, pageSender());
  const token = await openMenu();

  const answer = await fromMenu({ kind: "offer-review", token });

  const { pending: _, ...offer } = ready;
  assert.deepEqual(answer, {
    ok: true,
    offer: { ...offer, expiresAt: clock.time + lifetimeMs },
  });
  assert.deepEqual(ravenpass.reviews, ["p1"]);
  const next = await openMenu("next-document");
  const session = await sessions.forMenu(
    next,
    menuSender({ documentId: `menu-${next}` }),
  );
  assert.equal(
    session?.content.state === "offer" && session.content.offer.state,
    "ready",
  );
  assert.equal(stored(area).includes(typedPassword), false);
  assert.equal(stored(area).includes(currentPassword), false);
});

test("an offer whose capture Ravenpass no longer holds is gone", async () => {
  for (const [answers, saved] of [
    [[ready, new SessionError("not-found")], "updated"],
    [[ready, { state: "none" }], "updated"],
    [[ready], new SessionError("not-found")],
    [[ready], new SessionError("unlinked")],
  ] as const) {
    const { ravenpass, router, offerOpen, openMenu, fromMenu } = saveOffers();
    ravenpass.answers = [...answers];
    ravenpass.saved = saved;
    await router.serve(capture, pageSender());
    const token = await openMenu();

    const answer =
      answers.length > 1
        ? await fromMenu({ kind: "offer-review", token })
        : await fromMenu({
            kind: "offer-save",
            token,
            target: "",
            account: "alex",
            name: "github.com",
          });

    assert.deepEqual(answer, { ok: false, reason: "gone" });
    assert.deepEqual(ravenpass.discards, ["p1"]);
    assert.equal(await offerOpen("third-document"), null);
  }
});

test("a save refusing the account or the name keeps the offer", async () => {
  for (const reason of ["invalid-account", "invalid-name"] as const) {
    const { ravenpass, router, openMenu, fromMenu } = saveOffers();
    ravenpass.answers = [ready];
    ravenpass.saved = new SessionError(reason);
    await router.serve(capture, pageSender());
    const token = await openMenu();

    const answer = await fromMenu({
      kind: "offer-save",
      token,
      target: "",
      account: " ",
      name: "",
    });

    assert.deepEqual(answer, { ok: false, reason });
    assert.deepEqual(ravenpass.discards, []);
    assert.ok(await openMenu("next-document"));
  }
});

test("not now discards the capture in Ravenpass and forgets the offer", async () => {
  const { ravenpass, router, offerOpen, openMenu, fromMenu } = saveOffers();
  ravenpass.answers = [ready];
  await router.serve(capture, pageSender());
  const token = await openMenu();

  assert.deepEqual(await fromMenu({ kind: "offer-discard", token }), {
    ok: true,
  });
  assert.deepEqual(ravenpass.discards, ["p1"]);
  assert.equal(await offerOpen("third-document"), null);
});

test("only the menu under the offer's latest token may answer it", async () => {
  const { ravenpass, router, sessions, openMenu, fromMenu } = saveOffers();
  ravenpass.answers = [ready];
  await router.serve(capture, pageSender());
  const first = await openMenu("first-document");
  await openMenu("second-document");

  await assert.rejects(
    fromMenu({ kind: "offer-discard", token: first }),
    RefusedRequest,
  );
  const fieldMenu = await sessions.open(
    { tabId, documentId: "page-document", origin: "https://github.com" },
    { state: "locked", purpose: "sign-in" },
  );
  await sessions.bind(
    fieldMenu,
    menuSender({ documentId: `menu-${fieldMenu}` }),
  );
  await assert.rejects(
    fromMenu({ kind: "offer-review", token: fieldMenu }),
    RefusedRequest,
  );
  assert.deepEqual(ravenpass.discards, []);
  assert.deepEqual(ravenpass.reviews, []);
});

test("closing a tab discards its capture in Ravenpass", async () => {
  const { ravenpass, router, offerOpen } = saveOffers();
  ravenpass.answers = [ready];
  await router.serve(capture, pageSender());

  await router.tabClosed(tabId);

  assert.deepEqual(ravenpass.discards, ["p1"]);
  assert.equal(await offerOpen("third-document"), null);
});
