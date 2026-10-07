import { ask, isCardFillFrame } from "../messages.ts";
import { sendIgnoringClosedPort } from "../messaging/send.ts";
import { fillOtherFrame } from "./card-fill.ts";
import { changesAround } from "./page-changes.ts";
import { holdsPaymentFields } from "./payment-fields.ts";

/** How often at most a page that keeps changing is searched for payment fields again. */
const scanIntervalMs = 500;

/** Reports once that the document holds payment fields, and fills what a card chosen in another frame sends it. */
export class CardFrames {
  private readonly document: Document;
  private reported = false;
  private scheduled: ReturnType<typeof setTimeout> | undefined;
  private unobserve: (() => void) | null = null;

  constructor(document: Document) {
    this.document = document;
  }

  start(): void {
    this.unobserve = changesAround(this.document)(this.schedule);
    chrome.runtime.onMessage.addListener(this.onMessage);
    this.scan();
  }

  stop(): void {
    this.unobserve?.();
    this.unobserve = null;
    clearTimeout(this.scheduled);
    this.scheduled = undefined;
    chrome.runtime.onMessage.removeListener(this.onMessage);
  }

  private readonly schedule = (): void => {
    if (this.reported || this.scheduled !== undefined) return;
    this.scheduled = setTimeout(() => {
      this.scheduled = undefined;
      this.scan();
    }, scanIntervalMs);
  };

  private scan(): void {
    if (this.reported || !holdsPaymentFields(this.document)) return;
    this.reported = true;
    this.unobserve?.();
    this.unobserve = null;
    void sendIgnoringClosedPort(ask({ kind: "card-frame" }));
  }

  private readonly onMessage = (
    message: unknown,
    sender: chrome.runtime.MessageSender,
  ): undefined => {
    if (sender.id === chrome.runtime.id && isCardFillFrame(message)) {
      fillOtherFrame(this.document, message.values);
    }
  };
}
