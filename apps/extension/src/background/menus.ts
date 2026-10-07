import type { SignInStyle } from "@ravenpass/ui/extensions/sign-in-style.ts";
import type { OneTimeCode } from "@ravenpass/ui/vault-api.ts";
import { base64 } from "@scure/base";
import type { FieldKind } from "../content/fields.ts";
import { logRejection } from "../failures.ts";
import {
  type CardValues,
  type FillValues,
  type LinkClient,
  type PasskeyOption,
  SessionError,
  type SharedFile,
  type ShareProgress,
  type Suggestion,
  type SuggestPurpose,
} from "../link/client.ts";
import {
  isSecure,
  type MatchStrength,
  matchStrength,
} from "../match-strength.ts";
import {
  type Answers,
  type CardFillFrame,
  type CardListing,
  type CardShow,
  type CredentialFailure,
  type CredentialMenuContent,
  type FileDestination,
  type FileFailure,
  type FormMessage,
  type FrameMessage,
  isPlaced,
  isSubmission,
  type Listed,
  listsNothing,
  type MenuContent,
  type MenuFailure,
  type MenuRequest,
  type PasskeyAnswerMessage,
  type PasskeyMenuRequest,
  type Request,
  type ShareFailure,
} from "../messages.ts";
import type { CardFrames } from "./card-frames.ts";
import type { PagePasskeys } from "./page-passkeys.ts";
import type { PendingSignIns } from "./pending-sign-ins.ts";
import type { RecentFills } from "./recent-fills.ts";
import { sameSite } from "./same-site.ts";
import {
  hostOf,
  type MenuSession,
  type MenuSessions,
  menuDocumentOf,
  type PageFrame,
  type Sender,
  type TabDocument,
  type TabFrame,
  topOriginOf,
} from "./sessions.ts";
import type { SignInCards } from "./sign-in-cards.ts";

/** Resolves undefined for a document that is gone or answered nothing. */
export type FrameMessenger = (
  target: TabDocument,
  message: FrameMessage,
) => Promise<unknown>;

/** A FrameMessenger that also reaches whatever document a tab's frame currently holds. */
export type TabMessenger = (
  target: TabDocument | TabFrame,
  message:
    | FrameMessage
    | FormMessage
    | CardShow
    | CardFillFrame
    | PasskeyAnswerMessage,
) => Promise<unknown>;

export type MenuClient = Pick<
  LinkClient,
  | "suggest"
  | "fill"
  | "code"
  | "cards"
  | "fillCard"
  | "addWebsite"
  | "icon"
  | "unlock"
  | "identities"
  | "share"
>;

const addWebsiteFailure = "Ravenpass could not remember the page for the item.";

export interface MenuRouterDependencies {
  readonly client: MenuClient;
  readonly sessions: MenuSessions;
  readonly cards: SignInCards;
  readonly passkeys: PagePasskeys;
  readonly recentFills: RecentFills;
  readonly pendingSignIns: PendingSignIns;
  readonly cardFrames: CardFrames;
  /** The sign-in style the desktop app reported last. */
  readonly signInStyle: () => Promise<SignInStyle>;
  readonly relay: TabMessenger;
}

type CredentialRequest = Extract<MenuRequest, { id: string }>;

type FillRequest = Extract<MenuRequest, { confirmed: boolean }>;

type ShareRequest = Extract<MenuRequest, { kind: "menu-share" }>;

type PasskeySignRequest = Extract<MenuRequest, { kind: "passkey-sign" }>;

type CardFillRequest = Extract<MenuRequest, { kind: "menu-fill-card" }>;

interface CardSession extends MenuSession {
  readonly content: Extract<MenuContent, { state: "cards" }>;
}

interface Listing {
  readonly session: MenuSession;
  readonly id: string;
}

interface Chosen extends Listing {
  /** Whether the filling frame's origin, as Chrome reports it, joins the credential's websites once it fills. */
  readonly remember: boolean;
}

/** A request from a sender that may not make it, or about a menu that has ended. */
export class RefusedRequest extends Error {
  constructor(kind: Request["kind"]) {
    super(`The ${kind} request was refused.`);
    this.name = "RefusedRequest";
  }
}

/** Refuses a sender that may not ask, an item its menu does not list, an unconfirmed non-strong fill, and files without a destination. */
export class MenuRouter {
  private readonly client: MenuClient;
  private readonly sessions: MenuSessions;
  private readonly cards: SignInCards;
  private readonly passkeys: PagePasskeys;
  private readonly recentFills: RecentFills;
  private readonly pendingSignIns: PendingSignIns;
  private readonly cardFrames: CardFrames;
  private readonly signInStyle: () => Promise<SignInStyle>;
  private readonly relay: TabMessenger;

  constructor({
    client,
    sessions,
    cards,
    passkeys,
    recentFills,
    pendingSignIns,
    cardFrames,
    signInStyle,
    relay,
  }: MenuRouterDependencies) {
    this.client = client;
    this.sessions = sessions;
    this.cards = cards;
    this.passkeys = passkeys;
    this.recentFills = recentFills;
    this.pendingSignIns = pendingSignIns;
    this.cardFrames = cardFrames;
    this.signInStyle = signInStyle;
    this.relay = relay;
  }

  async serve(
    request: MenuRequest,
    sender: Sender,
  ): Promise<Answers[MenuRequest["kind"]]> {
    switch (request.kind) {
      case "menu-open":
        return request.field === "card"
          ? this.openCards(sender, request.requested)
          : this.open(sender, request.field, request.requested);
      case "menu-fill-card":
        return this.fillCard(request, sender);
      case "card-review":
        return this.reviewCards(await this.fromCardMenu(request, sender));
      case "card-frame": {
        const page = this.sessions.pageOf(sender);
        if (page) await this.cardFrames.report(page);
        return { ok: true };
      }
      case "file-menu":
        return this.openFileMenu(sender, request.destination);
      case "menu": {
        const session = await this.sessions.bind(request.token, sender);
        if (!session) throw new RefusedRequest(request.kind);
        return {
          site: siteOf(session.page.origin),
          host: new URL(session.page.origin).host,
          content: session.content,
        };
      }
      case "menu-icon": {
        const session = await this.fromMenu(request, sender);
        if (!sitesOf(session).includes(request.site)) {
          throw new RefusedRequest(request.kind);
        }
        return this.client.icon(request.site);
      }
      case "menu-fill": {
        const chosen = await this.chosen(request, sender, "sign-in");
        return this.remembered(chosen, await this.fill(chosen));
      }
      case "menu-code":
        return this.code(await this.listing(request, sender, "code"));
      case "menu-fill-code": {
        const chosen = await this.chosen(request, sender, "code");
        return this.remembered(chosen, await this.fillCode(chosen));
      }
      case "menu-unlock":
        await this.fromMenu(request, sender);
        return this.unlock();
      case "menu-identities":
        return this.identities(await this.fromFileMenu(request, sender));
      case "menu-share":
        return this.share(
          await this.fromFileMenu(request, sender),
          request,
          sender,
        );
      case "menu-size": {
        const session = await this.fromMenu(request, sender);
        await this.relay(hostOf(session), {
          kind: "menu-size",
          token: session.token,
          height: request.height,
        });
        return { ok: true };
      }
      case "menu-close":
        return this.close(request, sender);
      case "menu-focus":
        return this.focus(request, sender);
      case "menu-option":
        return this.option(request, sender);
      case "sign-in-form":
        return this.reportForm(sender, request.field);
      case "menu-review":
        return this.review(await this.fromMenu(request, sender));
      case "passkey-sign":
        return this.signPasskey(request, sender);
      case "passkey-save": {
        const session = await this.fromPasskeyCard(request, sender);
        const { content } = session;
        const offered =
          content.state === "passkey" &&
          content.listing.state === "save" &&
          (request.target === "new" ||
            content.listing.targets.some(
              ({ credential }) => credential === request.target,
            ));
        if (!offered) throw new RefusedRequest(request.kind);
        return this.passkeys.save(session, request.target);
      }
      case "passkey-elsewhere":
        return this.passkeys.elsewhere(
          await this.fromPasskeyCard(request, sender),
        );
      case "passkey-review":
        return this.passkeys.review(
          await this.fromPasskeyCard(request, sender),
        );
    }
  }

  async tabClosed(tabId: number): Promise<void> {
    await Promise.all([
      this.sessions.endTab(tabId),
      this.recentFills.forget(tabId),
      this.pendingSignIns.forget(tabId),
      this.cardFrames.forget(tabId),
    ]);
  }

  /** A card menu opens only where both the field's frame and the top frame are secure, and replaces the sign-in card.
   * One the person requested from the context menu opens even with no card to list. */
  private async openCards(
    sender: Sender,
    requested: boolean,
  ): Promise<Answers["menu-open"]> {
    const page = this.sessions.pageOf(sender);
    const top = topOriginOf(sender);
    if (!page || !top || !isSecure(page.origin) || !isSecure(top)) {
      return { token: null };
    }
    const listing = await this.cardListing(page.origin);
    if (
      !listing ||
      (!requested && listing.state === "list" && listing.cards.length === 0)
    ) {
      return { token: null };
    }
    if ((await this.signInStyle()) === "card") {
      await this.cards.withdraw(page.tabId);
    }
    return {
      token: await this.sessions.open(page, { state: "cards", listing }),
    };
  }

  private async cardListing(origin: string): Promise<CardListing | null> {
    try {
      return { state: "list", cards: await this.client.cards(origin) };
    } catch (error) {
      if (error instanceof SessionError) {
        if (error.reason === "locked") return { state: "locked" };
        if (error.reason === "not-open") return { state: "not-open" };
      }
      return null;
    }
  }

  /** Fills the card into the chosen field's form, then into the tab's other payment frames what each may receive. */
  private async fillCard(
    request: CardFillRequest,
    sender: Sender,
  ): Promise<Answers["menu-fill-card"]> {
    const session = await this.fromCardMenu(request, sender);
    const { listing } = session.content;
    const top = topOriginOf(sender);
    if (
      listing.state !== "list" ||
      !listing.cards.some(({ id }) => id === request.id) ||
      !top
    ) {
      throw new RefusedRequest(request.kind);
    }
    let values: CardValues;
    try {
      values = await this.client.fillCard(
        request.id,
        session.page.origin,
        this.progressTo(session, "fill-progress"),
      );
    } catch (error) {
      return { ok: false, reason: credentialFailure(error) };
    }
    await this.relay(session.page, {
      kind: "fill-card",
      token: session.token,
      values,
    });
    for (const delivery of await this.cardFrames.deliveries(
      session.page,
      top,
      values,
    )) {
      await this.relay(delivery.frame, {
        kind: "card-fill-frame",
        values: delivery.values,
      });
    }
    await this.sessions.end(session.token);
    return { ok: true };
  }

  /** A card menu closes only when Ravenpass cannot answer. */
  private async reviewCards(
    session: CardSession,
  ): Promise<Answers["card-review"]> {
    const listing = await this.cardListing(session.page.origin);
    if (!listing) {
      await this.endFromMenu(session);
      return { listing: null };
    }
    await this.sessions.revise(session.token, { state: "cards", listing });
    return { listing };
  }

  private async fromCardMenu(
    request: Extract<MenuRequest, { token: string }>,
    sender: Sender,
  ): Promise<CardSession> {
    const session = await this.fromMenu(request, sender);
    if (!isCardSession(session)) throw new RefusedRequest(request.kind);
    return session;
  }

  /** In the card style, shows the sign-in card even when the person closed it; a field menu replaces the card. A menu
   * the person requested from the context menu opens as a field menu, even with nothing to list. */
  private async open(
    sender: Sender,
    field: FieldKind,
    requested: boolean,
  ): Promise<Answers["menu-open"]> {
    const page = this.sessions.pageOf(sender);
    if (!page) return { token: null };
    const purpose = purposeOf(field);
    const content = await this.contentFor(page, purpose);
    if (!content || (!requested && listsNothing(content))) {
      return { token: null };
    }
    if ((await this.signInStyle()) === "card") {
      if (!requested && this.servesCard(sender, page)) {
        if (content.state !== "not-open") {
          await this.cards.show(page, purpose, content, true);
        }
        return { token: null };
      }
      await this.cards.withdraw(page.tabId);
    }
    return { token: await this.sessions.open(page, content) };
  }

  /** The top frame and frames of its origin or site; both origins are the ones Chrome reports, never the page's own claim. */
  private servesCard(sender: Sender, page: PageFrame): boolean {
    if (sender.frameId === 0) return true;
    const top = topOriginOf(sender);
    return top !== null && (top === page.origin || sameSite(page.origin, top));
  }

  /** A password field after a card sign-in's account step, in the same tab and origin, takes that credential. */
  private async reportForm(
    sender: Sender,
    field: FieldKind,
  ): Promise<Answers["sign-in-form"]> {
    const page = this.sessions.pageOf(sender);
    if (!page) return { fill: null };
    if (field === "password") {
      const credential = await this.pendingSignIns.take(
        page.tabId,
        page.origin,
      );
      if (credential) {
        // A fill that fails leaves the field to the menu or card below.
        const fill = await this.client
          .fill(credential, page.origin)
          .catch(() => null);
        if (fill) return { fill };
      }
    }
    const purpose = purposeOf(field);
    const listing = await this.contentFor(page, purpose);
    if (
      listing &&
      showsOnItsOwn(listing) &&
      (await this.signInStyle()) === "card" &&
      this.servesCard(sender, page)
    ) {
      await this.cards.show(page, purpose, listing, false);
    }
    return { fill: null };
  }

  /** A field's menu closes only when Ravenpass cannot answer; a card closes once it lists nothing, and one the form
   * showed on its own, not one the person requested, once it lists nothing strong. */
  private async review(session: MenuSession): Promise<Answers["menu-review"]> {
    const { content } = session;
    if (content.state !== "sign-in-card") {
      const shown = listingOf(content);
      if (!shown) throw new RefusedRequest("menu-review");
      const listing = await this.contentFor(session.page, shown.purpose);
      if (!listing) {
        await this.endFromMenu(session);
        return { listing: null };
      }
      await this.sessions.revise(session.token, listing);
      return { listing };
    }
    const listing = await this.contentFor(session.page, content.purpose);
    if (
      !listing ||
      listing.state === "not-open" ||
      listsNothing(listing) ||
      (!content.requested && !showsOnItsOwn(listing))
    ) {
      await this.cards.hide(session, false);
      return { listing: null };
    }
    await this.sessions.revise(session.token, { ...content, listing });
    return { listing };
  }

  private async openFileMenu(
    sender: Sender,
    destination: FileDestination | null,
  ): Promise<Answers["file-menu"]> {
    const page = this.sessions.pageOf(sender);
    if (!page) return { token: null };
    const content: MenuContent = destination
      ? { state: "files", accept: destination.accept }
      : { state: "no-destination" };
    return { token: await this.sessions.open(page, content) };
  }

  /** Answers the identities whose files Ravenpass can share with the window's frame. */
  private async identities(
    session: MenuSession,
  ): Promise<Answers["menu-identities"]> {
    try {
      return {
        ok: true,
        identities: await this.client.identities(session.page.origin),
      };
    } catch (error) {
      return { ok: false, reason: fileFailure(error) };
    }
  }

  /** Places the file only while its window is still open; the file is never kept once it is sent. */
  private async share(
    session: MenuSession,
    request: ShareRequest,
    sender: Sender,
  ): Promise<Answers["menu-share"]> {
    let file: SharedFile;
    try {
      file = await this.client.share(
        request.identity,
        request.file,
        session.page.origin,
        this.progressTo(session, "share-progress"),
      );
    } catch (error) {
      return { ok: false, reason: shareFailure(error) };
    }
    if (!(await this.sessions.forMenu(session.token, sender))) {
      throw new RefusedRequest(request.kind);
    }
    const placement = await this.relay(session.page, {
      kind: "place-file",
      token: session.token,
      name: file.name,
      mediaType: file.mediaType,
      content: base64.encode(file.content),
    });
    if (!isPlaced(placement)) return { ok: false, reason: "not-taken" };
    await this.sessions.end(session.token);
    return { ok: true };
  }

  private async contentFor(
    page: PageFrame,
    purpose: SuggestPurpose,
  ): Promise<CredentialMenuContent | null> {
    const withStrength = <Credential extends Suggestion>(
      credentials: readonly Credential[],
    ): Listed<Credential>[] =>
      credentials.map((credential) => ({
        ...credential,
        strength: matchStrength(credential.exact, credential.site, page.origin),
      }));
    try {
      const content: CredentialMenuContent =
        purpose === "code"
          ? {
              state: "list",
              purpose,
              credentials: withStrength(
                await this.recentFills.order(
                  page.tabId,
                  await this.client.suggest(page.origin, purpose),
                ),
              ),
            }
          : {
              state: "list",
              purpose,
              credentials: withStrength(
                await this.client.suggest(page.origin, purpose),
              ),
              passkeys: await this.passkeys.optionsFor(page),
            };
      return content;
    } catch (error) {
      if (error instanceof SessionError) {
        if (error.reason === "locked") return { state: "locked", purpose };
        if (error.reason === "not-open") return { state: "not-open", purpose };
      }
      return null;
    }
  }

  /** A sign-in card whose form asked for the account alone keeps the credential pending for the next step. */
  private async fill({ session, id }: Listing): Promise<Answers["menu-fill"]> {
    let values: FillValues;
    try {
      values = await this.client.fill(
        id,
        session.page.origin,
        this.progressTo(session, "fill-progress"),
      );
    } catch (error) {
      return { ok: false, reason: credentialFailure(error) };
    }
    if (session.content.state === "sign-in-card") {
      const submission = await this.relay(session.page, {
        kind: "form-sign-in",
        ...values,
      });
      await this.recentFills.remember(session.page.tabId, id);
      if (
        isSubmission(submission) &&
        submission.submitted &&
        !submission.password
      ) {
        await this.pendingSignIns.record(
          session.page.tabId,
          id,
          session.page.origin,
        );
      }
      await this.cards.hide(session, false);
      return { ok: true };
    }
    await this.relay(session.page, {
      kind: "fill",
      token: session.token,
      ...values,
    });
    await this.recentFills.remember(session.page.tabId, id);
    await this.sessions.end(session.token);
    return { ok: true };
  }

  private async code({ session, id }: Listing): Promise<Answers["menu-code"]> {
    try {
      return {
        ok: true,
        code: await this.client.code(
          id,
          session.page.origin,
          this.progressTo(session, "fill-progress"),
        ),
      };
    } catch (error) {
      return { ok: false, reason: credentialFailure(error) };
    }
  }

  /** A sign-in card also submits the form. */
  private async fillCode({
    session,
    id,
  }: Listing): Promise<Answers["menu-fill-code"]> {
    let code: OneTimeCode;
    try {
      code = await this.client.code(
        id,
        session.page.origin,
        this.progressTo(session, "fill-progress"),
      );
    } catch (error) {
      return { ok: false, reason: credentialFailure(error) };
    }
    if (session.content.state === "sign-in-card") {
      await this.relay(session.page, { kind: "form-code", code: code.code });
      await this.cards.hide(session, false);
      return { ok: true };
    }
    await this.relay(session.page, {
      kind: "fill-code",
      token: session.token,
      code: code.code,
    });
    await this.sessions.end(session.token);
    return { ok: true };
  }

  /** Adds the filled frame's origin to the credential after a fill; a failed addition leaves the fill as it is. */
  private async remembered<Answer extends { readonly ok: boolean }>(
    { session, id, remember }: Chosen,
    answer: Answer,
  ): Promise<Answer> {
    if (answer.ok && remember) {
      await logRejection(
        this.client.addWebsite(id, session.page.origin),
        addWebsiteFailure,
      );
    }
    return answer;
  }

  /** Relays what Ravenpass asks of the person to the menu or card waiting on the request. */
  private progressTo(
    session: MenuSession,
    kind: "share-progress" | "fill-progress",
  ): (progress: ShareProgress) => void {
    const menu = menuDocumentOf(session);
    return (progress) => {
      if (menu) {
        void this.relay(menu, { kind, token: session.token, progress });
      }
    };
  }

  /** A menu or card stays open and asks again once unlocked. */
  private async unlock(): Promise<Answers["menu-unlock"]> {
    try {
      await this.client.unlock();
    } catch (error) {
      return { ok: false, reason: menuFailure(error) };
    }
    return { ok: true };
  }

  /** Closed from its own page, a menu also closes its frame; closed from its host frame, it only ends. */
  private async close(
    request: Extract<MenuRequest, { kind: "menu-close" }>,
    sender: Sender,
  ): Promise<Answers["menu-close"]> {
    const fromMenu = await this.sessions.forMenu(request.token, sender);
    if (fromMenu) {
      await this.endFromMenu(fromMenu);
      return { ok: true };
    }
    const fromHost = await this.sessions.forHost(request.token, sender);
    if (!fromHost) throw new RefusedRequest(request.kind);
    if (fromHost.content.state === "passkey") {
      await this.passkeys.dismiss(fromHost);
    } else {
      await this.sessions.end(fromHost.token);
    }
    return { ok: true };
  }

  /** Moves focus from the field into its menu, or from the menu back to the field. */
  private async focus(
    request: Extract<MenuRequest, { kind: "menu-focus" }>,
    sender: Sender,
  ): Promise<Answers["menu-focus"]> {
    const message = { kind: "menu-focus", token: request.token } as const;
    const fromMenu = await this.sessions.forMenu(request.token, sender);
    if (fromMenu) {
      await this.relay(fromMenu.page, message);
      return { ok: true };
    }
    const fromPage = await this.sessions.forPage(request.token, sender);
    if (!fromPage) throw new RefusedRequest(request.kind);
    const menu = menuDocumentOf(fromPage);
    if (menu) await this.relay(menu, message);
    return { ok: true };
  }

  /** Relays Option pressed or released in a code field to the field's code menu. */
  private async option(
    request: Extract<MenuRequest, { kind: "menu-option" }>,
    sender: Sender,
  ): Promise<Answers["menu-option"]> {
    const session = await this.sessions.forPage(request.token, sender);
    if (
      session?.content.state !== "list" ||
      session.content.purpose !== "code"
    ) {
      throw new RefusedRequest(request.kind);
    }
    const menu = menuDocumentOf(session);
    if (menu) {
      await this.relay(menu, {
        kind: "menu-option",
        token: session.token,
        held: request.held,
      });
    }
    return { ok: true };
  }

  /** A closed sign-in card stays away until a sign-in field takes focus; a closed passkey card refuses its request. */
  private async endFromMenu(session: MenuSession): Promise<void> {
    if (session.content.state === "sign-in-card") {
      await this.cards.hide(session, true);
      return;
    }
    if (session.content.state === "passkey") {
      await this.passkeys.dismiss(session);
      return;
    }
    await this.relay(session.page, {
      kind: "menu-close",
      token: session.token,
    });
    await this.sessions.end(session.token);
  }

  private async fromMenu(
    request: Extract<MenuRequest, { token: string }>,
    sender: Sender,
  ): Promise<MenuSession> {
    const session = await this.sessions.forMenu(request.token, sender);
    if (!session) throw new RefusedRequest(request.kind);
    return session;
  }

  /** A passkey card is closed by PagePasskeys, not here. */
  private async signPasskey(
    request: PasskeySignRequest,
    sender: Sender,
  ): Promise<Answers["passkey-sign"]> {
    const session = await this.fromMenu(request, sender);
    const listed = passkeysOf(session.content).some(
      ({ credential, credentialId }) =>
        credential === request.credential &&
        credentialId === request.credentialId,
    );
    if (!listed) throw new RefusedRequest(request.kind);
    const answer = await this.passkeys.sign(session, {
      credential: request.credential,
      credentialId: request.credentialId,
    });
    if (!answer.ok || session.content.state === "passkey") return answer;
    if (session.content.state === "sign-in-card") {
      await this.cards.hide(session, false);
    } else {
      await this.relay(session.page, {
        kind: "menu-close",
        token: session.token,
      });
      await this.sessions.end(session.token);
    }
    return answer;
  }

  private async fromPasskeyCard(
    request: PasskeyMenuRequest,
    sender: Sender,
  ): Promise<MenuSession> {
    const session = await this.fromMenu(request, sender);
    if (session.content.state !== "passkey") {
      throw new RefusedRequest(request.kind);
    }
    return session;
  }

  private async fromFileMenu(
    request: Extract<MenuRequest, { token: string }>,
    sender: Sender,
  ): Promise<MenuSession> {
    const session = await this.fromMenu(request, sender);
    if (session.content.state !== "files") {
      throw new RefusedRequest(request.kind);
    }
    return session;
  }

  private async listing(
    request: CredentialRequest,
    sender: Sender,
    purpose: SuggestPurpose,
  ): Promise<Listing> {
    const { session } = await this.listed(request, sender, purpose);
    return { session, id: request.id };
  }

  private async chosen(
    request: FillRequest,
    sender: Sender,
    purpose: SuggestPurpose,
  ): Promise<Chosen> {
    const { session, credential } = await this.listed(request, sender, purpose);
    if (credential.strength !== "strong" && !request.confirmed) {
      throw new RefusedRequest(request.kind);
    }
    return {
      session,
      id: request.id,
      remember: remembers(credential.strength, request.remember === true),
    };
  }

  private async listed(
    request: CredentialRequest,
    sender: Sender,
    purpose: SuggestPurpose,
  ): Promise<{ session: MenuSession; credential: Listed<Suggestion> }> {
    const session = await this.fromMenu(request, sender);
    const content = listingOf(session.content);
    const credential =
      content?.state === "list" && content.purpose === purpose
        ? content.credentials.find(({ id }) => id === request.id)
        : undefined;
    if (!credential) throw new RefusedRequest(request.kind);
    return { session, credential };
  }
}

/** A confirmed fill on the saved site's `https` page adds it; another site's page only when asked; an insecure page never. */
function remembers(strength: MatchStrength, asked: boolean): boolean {
  return strength === "same-site" || (strength === "other-site" && asked);
}

function showsOnItsOwn(listing: CredentialMenuContent): boolean {
  if (listing.state !== "list") return listing.state === "locked";
  return (
    listing.credentials.some(({ strength }) => strength === "strong") ||
    (listing.purpose === "sign-in" && listing.passkeys.length > 0)
  );
}

function listingOf(content: MenuContent): CredentialMenuContent | null {
  switch (content.state) {
    case "list":
    case "locked":
    case "not-open":
      return content;
    case "sign-in-card":
      return content.listing;
    default:
      return null;
  }
}

function isCardSession(session: MenuSession): session is CardSession {
  return session.content.state === "cards";
}

/** A save offer's new credential and its update targets all take the offer site's icon; a card its bank's. */
function sitesOf({ content, page }: MenuSession): string[] {
  if (content.state === "offer") return [content.offer.site];
  if (content.state === "passkey") return [siteOf(page.origin)];
  if (content.state === "cards") {
    return content.listing.state === "list"
      ? content.listing.cards.flatMap(({ site }) => (site ? [site] : []))
      : [];
  }
  const listing = listingOf(content);
  return listing?.state === "list"
    ? listing.credentials.map(({ site }) => site)
    : [];
}

function passkeysOf(content: MenuContent): readonly PasskeyOption[] {
  if (content.state === "passkey") {
    return content.listing.state === "sign-in" ? content.listing.passkeys : [];
  }
  const listing = listingOf(content);
  return listing?.state === "list" && listing.purpose === "sign-in"
    ? listing.passkeys
    : [];
}

function purposeOf(field: FieldKind): SuggestPurpose {
  return field === "code" ? "code" : "sign-in";
}

/** The vault writes sites without a leading `www.`. */
export function siteOf(origin: string): string {
  return new URL(origin).hostname.replace(/^www\./, "");
}

export function menuFailure(error: unknown): MenuFailure {
  if (error instanceof SessionError) {
    if (error.reason === "locked") return "locked";
    if (error.reason === "not-open") return "not-open";
  }
  return "failed";
}

function fileFailure(error: unknown): FileFailure {
  if (error instanceof SessionError && error.reason === "unlinked") {
    return "unlinked";
  }
  return menuFailure(error);
}

/** A request Ravenpass refused because the person did not, or could not, confirm it. */
function unconfirmed(error: unknown): "declined" | "unverifiable" | null {
  if (
    error instanceof SessionError &&
    (error.reason === "declined" || error.reason === "unverifiable")
  ) {
    return error.reason;
  }
  return null;
}

function credentialFailure(error: unknown): CredentialFailure {
  return unconfirmed(error) ?? menuFailure(error);
}

function shareFailure(error: unknown): ShareFailure {
  return unconfirmed(error) ?? fileFailure(error);
}
