import type { CardValues } from "../link/client.ts";
import type { PageFrame, SessionArea, TabDocument } from "./sessions.ts";

const keyPrefix = "card-frames:";

/** A tab keeps the documents that reported payment fields most recently, up to this many. */
const maxFrames = 16;

/** A document of a tab that reported payment fields, with the origin Chrome reported for it. */
interface CardFrame {
  readonly documentId: string;
  readonly origin: string;
}

/** What one frame receives of a card chosen in another. */
export interface CardDelivery {
  readonly frame: TabDocument;
  readonly values: CardValues;
}

export interface CardFramesDependencies {
  readonly area: SessionArea;
}

/** Remembers, per tab, the frames holding payment fields, so a card fill reaches a payment provider's other frames. */
export class CardFrames {
  private readonly area: SessionArea;

  constructor({ area }: CardFramesDependencies) {
    this.area = area;
  }

  async report(page: PageFrame): Promise<void> {
    const kept = await this.framesOf(page.tabId);
    const frames = [
      ...kept.filter(({ documentId }) => documentId !== page.documentId),
      { documentId: page.documentId, origin: page.origin },
    ].slice(-maxFrames);
    await this.area.set({ [keyPrefix + page.tabId]: frames });
  }

  /** The tab's other payment frames and what each may receive of `values`, chosen in `chosenIn` under `topOrigin`. */
  async deliveries(
    chosenIn: PageFrame,
    topOrigin: string,
    values: CardValues,
  ): Promise<CardDelivery[]> {
    return (await this.framesOf(chosenIn.tabId)).flatMap(
      ({ documentId, origin }) => {
        if (documentId === chosenIn.documentId) return [];
        const shared = sharedWith(origin, chosenIn.origin, topOrigin, values);
        return shared
          ? [{ frame: { tabId: chosenIn.tabId, documentId }, values: shared }]
          : [];
      },
    );
  }

  async forget(tabId: number): Promise<void> {
    await this.area.remove(keyPrefix + tabId);
  }

  private async framesOf(tabId: number): Promise<readonly CardFrame[]> {
    const key = keyPrefix + tabId;
    return ((await this.area.get(key))[key] as CardFrame[] | undefined) ?? [];
  }
}

/** A frame of the chosen frame's origin takes every value; one of the top frame's origin all but the number and the
 * security code; any other frame nothing. */
export function sharedWith(
  origin: string,
  chosenOrigin: string,
  topOrigin: string,
  values: CardValues,
): CardValues | null {
  if (origin === chosenOrigin) return values;
  if (origin === topOrigin) return { ...values, number: "", securityCode: "" };
  return null;
}
