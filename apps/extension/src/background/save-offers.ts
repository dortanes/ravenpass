import { logFailure } from "../failures.ts";
import {
  type CaptureOffer,
  type LinkClient,
  type SaveChoice,
  SessionError,
} from "../link/client.ts";
import type { Answers, OfferFailure, OfferRequest } from "../messages.ts";
import {
  type FrameMessenger,
  menuFailure,
  RefusedRequest,
  siteOf,
} from "./menus.ts";
import type { MenuSessions, PageFrame, Sender } from "./sessions.ts";
import { shownOffer, type TabOffer, type TabOffers } from "./tab-offers.ts";

export type OfferClient = Pick<
  LinkClient,
  "capture" | "captureCard" | "review" | "save" | "discard"
>;

export interface SaveOffersDependencies {
  readonly client: OfferClient;
  readonly sessions: MenuSessions;
  readonly offers: TabOffers;
  readonly relay: FrameMessenger;
  /** Tells a tab's top frame that the tab's offer is ready to show. */
  readonly show: (tabId: number) => Promise<void>;
}

export const captureFailure =
  "Ravenpass could not offer to save a password or a card.";

type TokenRequest = Extract<OfferRequest, { token: string }>;

/** `offer` is null once the offer is gone or the menu's token is not the one last issued. */
interface MenuOffer {
  readonly tabId: number;
  readonly offer: TabOffer | null;
}

type Refusal = { ok: false; reason: OfferFailure };

/** Keeps no password: a tab holds only the offer the desktop app returned for its capture. */
export class SaveOffers {
  private readonly client: OfferClient;
  private readonly sessions: MenuSessions;
  private readonly offers: TabOffers;
  private readonly relay: FrameMessenger;
  private readonly show: (tabId: number) => Promise<void>;

  constructor({
    client,
    sessions,
    offers,
    relay,
    show,
  }: SaveOffersDependencies) {
    this.client = client;
    this.sessions = sessions;
    this.offers = offers;
    this.relay = relay;
    this.show = show;
  }

  async serve(
    request: OfferRequest,
    sender: Sender,
  ): Promise<Answers[OfferRequest["kind"]]> {
    switch (request.kind) {
      case "capture": {
        const { kind: _, ...captured } = request;
        return this.capture(sender, (origin) =>
          this.client.capture(origin, captured),
        );
      }
      case "card-capture": {
        const { kind: _, ...card } = request;
        return this.capture(sender, (origin) =>
          this.client.captureCard(origin, card),
        );
      }
      case "form-gone":
        return this.formGone(sender);
      case "offer-open":
        return this.open(sender, request.signInForm);
      case "offer-review":
        return this.review(await this.fromMenu(request, sender));
      case "offer-save":
        return this.save(await this.fromMenu(request, sender), request);
      case "offer-discard":
        return this.discard(await this.fromMenu(request, sender));
    }
  }

  async tabClosed(tabId: number): Promise<void> {
    await this.end(tabId);
  }

  /** `send` sends the capture for the frame's origin as Chrome reports it. */
  private async capture(
    sender: Sender,
    send: (origin: string) => Promise<CaptureOffer>,
  ): Promise<Answers["capture"]> {
    const page = this.sessions.pageOf(sender);
    if (!page) throw new RefusedRequest("capture");
    let answer: CaptureOffer;
    try {
      answer = await send(page.origin);
    } catch (error) {
      logFailure(captureFailure, error);
      return { ok: true };
    }
    const offered = answer.state === "none" ? null : answer;
    const earlier = await this.offers.replace(page, offered);
    if (earlier) await this.retire(page.tabId, earlier);
    return { ok: true };
  }

  private async formGone(sender: Sender): Promise<Answers["form-gone"]> {
    const page = this.sessions.pageOf(sender);
    if (!page) throw new RefusedRequest("form-gone");
    const offer = await this.offers.current(page.tabId);
    if (offer && !offer.ready && offer.capturedIn === page.documentId) {
      await this.offers.markReady(page.tabId, offer.pending);
      await this.show(page.tabId);
    }
    return { ok: true };
  }

  /** The top frame asks on every document it loads and whenever told the offer is ready. */
  private async open(
    sender: Sender,
    signInForm: boolean,
  ): Promise<Answers["offer-open"]> {
    const page = this.sessions.pageOf(sender);
    if (!page || sender.frameId !== 0) return { token: null };
    const offer = await this.offers.current(page.tabId);
    if (offer && !offer.ready && signedIn(offer, page, signInForm)) {
      await this.offers.markReady(page.tabId, offer.pending);
    }
    return { token: await this.offers.issue(page) };
  }

  private async review({
    tabId,
    offer,
  }: MenuOffer): Promise<Answers["offer-review"]> {
    if (!offer) return { ok: false, reason: "gone" };
    let answer: CaptureOffer;
    try {
      answer = await this.client.review(offer.pending);
    } catch (error) {
      return this.refused(tabId, offer, error);
    }
    if (answer.state === "none") {
      await this.end(tabId, offer.pending);
      return { ok: false, reason: "gone" };
    }
    const revised = await this.offers.revise(tabId, answer);
    return revised
      ? { ok: true, offer: shownOffer(revised) }
      : { ok: false, reason: "gone" };
  }

  private async save(
    { tabId, offer }: MenuOffer,
    choice: SaveChoice,
  ): Promise<Answers["offer-save"]> {
    if (!offer) return { ok: false, reason: "gone" };
    try {
      const saved = await this.client.save(offer.pending, choice);
      await this.offers.take(tabId, offer.pending);
      return { ok: true, saved };
    } catch (error) {
      if (error instanceof SessionError && error.reason === "locked") {
        await this.offers.revise(tabId, { ...offer, state: "locked" });
      }
      return this.refused(tabId, offer, error);
    }
  }

  /** The offer's menu closes itself. */
  private async discard({
    tabId,
    offer,
  }: MenuOffer): Promise<Answers["offer-discard"]> {
    if (offer) await this.end(tabId, offer.pending);
    return { ok: true };
  }

  private async fromMenu(
    request: TokenRequest,
    sender: Sender,
  ): Promise<MenuOffer> {
    const session = await this.sessions.forMenu(request.token, sender);
    if (session?.content.state !== "offer") {
      throw new RefusedRequest(request.kind);
    }
    const { tabId } = session.page;
    return { tabId, offer: await this.offers.shownBy(tabId, session.token) };
  }

  /** A gone offer still discards its capture, which the desktop app may hold after a failed save. */
  private async refused(
    tabId: number,
    offer: TabOffer,
    error: unknown,
  ): Promise<Refusal> {
    const reason = offerFailure(error);
    if (reason === "gone") await this.end(tabId, offer.pending);
    return { ok: false, reason };
  }

  /** Without `pending`, ends whatever offer the tab holds. */
  private async end(tabId: number, pending?: string): Promise<void> {
    const offer = await this.offers.take(tabId, pending);
    if (offer) await this.forget(offer);
  }

  private async retire(tabId: number, offer: TabOffer): Promise<void> {
    await this.forget(offer);
    if (!offer.shown) return;
    const { token, documentId } = offer.shown;
    await this.relay({ tabId, documentId }, { kind: "menu-close", token });
    await this.sessions.end(token);
  }

  /** Asks Ravenpass to forget a capture; one it cannot reach forgets it after `captureLifetimeMs`. */
  private async forget(offer: TabOffer): Promise<void> {
    try {
      await this.client.discard(offer.pending);
    } catch (error) {
      if (!(error instanceof SessionError)) throw error;
    }
  }
}

/** A later document still showing a sign-in form for the capture's site means the sign-in likely failed; any later
 * document follows a card's checkout. */
function signedIn(
  offer: TabOffer,
  page: PageFrame,
  signInForm: boolean,
): boolean {
  if (page.documentId === offer.capturedIn) return false;
  return (
    offer.kind === "card" || !signInForm || siteOf(page.origin) !== offer.site
  );
}

function offerFailure(error: unknown): OfferFailure {
  if (error instanceof SessionError) {
    switch (error.reason) {
      case "not-found":
      case "unlinked":
        return "gone";
      case "invalid-account":
      case "invalid-name":
        return error.reason;
    }
  }
  return menuFailure(error);
}
