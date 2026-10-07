import type { PendingCapture } from "../link/client.ts";
import {
  captureLifetimeMs,
  type OfferMenuContent,
  type SaveOffer,
} from "../messages.ts";
import type {
  MenuSessions,
  PageFrame,
  SessionArea,
  TabDocument,
} from "./sessions.ts";

const keyPrefix = "offer:";

/** `documentId` is the top-frame document the token was issued to. */
export interface OfferShown {
  readonly token: string;
  readonly documentId: string;
}

/** Never holds the captured password or card; `expiresAt` is when the desktop app forgets it, in Unix milliseconds. */
export interface TabOffer extends PendingCapture {
  readonly expiresAt: number;
  readonly capturedIn: string;
  readonly ready: boolean;
  readonly shown: OfferShown | null;
}

export interface TabOffersDependencies {
  readonly area: SessionArea;
  readonly sessions: MenuSessions;
  readonly now?: () => number;
}

/** One offer per tab; each token issued for it ends the token issued before. */
export class TabOffers {
  private readonly area: SessionArea;
  private readonly sessions: MenuSessions;
  private readonly now: () => number;

  constructor({ area, sessions, now = Date.now }: TabOffersDependencies) {
    this.area = area;
    this.sessions = sessions;
    this.now = now;
  }

  /** The returned earlier offer's capture and menu are the caller's to end. */
  async replace(
    capturedIn: TabDocument,
    capture: PendingCapture | null,
  ): Promise<TabOffer | null> {
    const earlier = await this.take(capturedIn.tabId);
    if (capture) {
      const offer: TabOffer = {
        kind: capture.kind,
        state: capture.state,
        pending: capture.pending,
        site: capture.site,
        account: capture.account,
        name: capture.name,
        card: capture.card,
        targets: capture.targets,
        suggested: capture.suggested,
        expiresAt: this.now() + captureLifetimeMs,
        capturedIn: capturedIn.documentId,
        ready: false,
        shown: null,
      };
      await this.area.set({ [keyOf(capturedIn.tabId)]: offer });
    }
    return earlier;
  }

  async markReady(tabId: number, pending: string): Promise<TabOffer | null> {
    const offer = await this.current(tabId);
    if (!offer || offer.pending !== pending) return null;
    const ready: TabOffer = { ...offer, ready: true };
    await this.area.set({ [keyOf(tabId)]: ready });
    return ready;
  }

  async issue(page: PageFrame): Promise<string | null> {
    const offer = await this.current(page.tabId);
    if (!offer?.ready) return null;
    if (offer.shown) await this.sessions.end(offer.shown.token);
    const token = await this.sessions.open(page, contentOf(offer));
    const shown: TabOffer = {
      ...offer,
      shown: { token, documentId: page.documentId },
    };
    await this.area.set({ [keyOf(page.tabId)]: shown });
    return token;
  }

  async shownBy(tabId: number, token: string): Promise<TabOffer | null> {
    const offer = await this.current(tabId);
    return offer?.shown?.token === token ? offer : null;
  }

  async revise(
    tabId: number,
    capture: PendingCapture,
  ): Promise<TabOffer | null> {
    const offer = await this.current(tabId);
    if (!offer || offer.pending !== capture.pending) return null;
    const revised: TabOffer = {
      ...offer,
      kind: capture.kind,
      state: capture.state,
      site: capture.site,
      account: capture.account,
      name: capture.name,
      card: capture.card,
      targets: capture.targets,
      suggested: capture.suggested,
    };
    await this.area.set({ [keyOf(tabId)]: revised });
    return revised;
  }

  /** The offer's menu token stays valid until its menu closes. */
  async take(tabId: number, pending?: string): Promise<TabOffer | null> {
    const offer = await this.current(tabId);
    if (!offer || (pending !== undefined && offer.pending !== pending)) {
      return null;
    }
    await this.area.remove(keyOf(tabId));
    return offer;
  }

  async current(tabId: number): Promise<TabOffer | null> {
    const key = keyOf(tabId);
    const offer = (await this.area.get(key))[key] as TabOffer | undefined;
    if (!offer) return null;
    if (offer.expiresAt <= this.now()) {
      await this.area.remove(key);
      return null;
    }
    return offer;
  }
}

/** Leaves out the pending capture id and the menu token. */
export function shownOffer(offer: TabOffer): SaveOffer {
  return {
    kind: offer.kind,
    state: offer.state,
    site: offer.site,
    account: offer.account,
    name: offer.name,
    card: offer.card,
    targets: offer.targets,
    suggested: offer.suggested,
    expiresAt: offer.expiresAt,
  };
}

function contentOf(offer: TabOffer): OfferMenuContent {
  return { state: "offer", offer: shownOffer(offer) };
}

function keyOf(tabId: number): string {
  return keyPrefix + tabId;
}
