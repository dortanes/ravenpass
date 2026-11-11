import { tracksMenuVisibility } from "../menu-visibility.ts";
import { isMenuMoved } from "../messages.ts";

// Intersection Observer v2, which Chrome ships and the DOM typings omit.
declare global {
  interface IntersectionObserverInit {
    trackVisibility?: boolean;
    delay?: number;
  }
  interface IntersectionObserverEntry {
    readonly isVisible?: boolean;
  }
}

/** How long the menu must have been shown and kept its place before a click acts. */
export const stillMs = 500;

/** The shortest interval Chrome accepts between visibility reports. */
const visibilityDelayMs = 100;

export interface ClickGateDependencies {
  /** A monotonic clock in milliseconds. */
  readonly now?: () => number;
  readonly nativeVisibility?: boolean;
  readonly active?: () => boolean;
}

export interface GatedEvent {
  readonly isTrusted: boolean;
  preventDefault(): void;
  stopImmediatePropagation(): void;
}

/** Native occlusion tracking, or local visibility followed by an authenticated host proof in the background. */
export class ClickGate {
  private readonly now: () => number;
  private visible = false;
  private readonly nativeVisibility: boolean;
  private readonly active: () => boolean;
  private stillSince = 0;
  private root: Element | null = null;

  constructor({
    now = () => performance.now(),
    nativeVisibility = tracksMenuVisibility(),
    active = () => true,
  }: ClickGateDependencies = {}) {
    this.now = now;
    this.nativeVisibility = nativeVisibility;
    this.active = active;
  }

  visibility(visible: boolean): void {
    if (visible && !this.visible) this.stillSince = this.now();
    this.visible = visible;
  }

  /** The framing content script reports the end of the entrance and each move or resize. */
  moved(): void {
    this.stillSince = this.now();
  }

  accepts(): boolean {
    return (
      this.visible &&
      this.active() &&
      this.documentActive() &&
      this.now() - this.stillSince >= stillMs
    );
  }

  /** Must run in the capture phase, ahead of the menu's own listeners. */
  readonly screen = (event: GatedEvent): void => {
    if (event.isTrusted && this.accepts()) return;
    event.preventDefault();
    event.stopImmediatePropagation();
  };

  report(
    entry: Pick<
      IntersectionObserverEntry,
      "isVisible" | "isIntersecting" | "intersectionRatio"
    >,
  ): void {
    this.visibility(
      this.nativeVisibility
        ? entry.isVisible === true
        : entry.isIntersecting && entry.intersectionRatio === 1,
    );
  }

  private documentActive(): boolean {
    if (!this.root) return true;
    const document = this.root.ownerDocument;
    return (
      document.visibilityState === "visible" &&
      (this.nativeVisibility ||
        (document.hasFocus() &&
          this.root.checkVisibility({
            checkOpacity: true,
            checkVisibilityCSS: true,
          })))
    );
  }

  start(root: Element): () => void {
    this.root = root;
    const document = root.ownerDocument;
    const observer = new IntersectionObserver(
      (entries) => {
        const latest = entries.at(-1);
        if (latest) this.report(latest);
      },
      this.nativeVisibility
        ? { trackVisibility: true, delay: visibilityDelayMs }
        : { threshold: [0, 1] },
    );
    observer.observe(root);
    const onVisibility = () => {
      this.visibility(false);
      if (document.visibilityState === "visible") {
        observer.unobserve(root);
        observer.observe(root);
      }
    };
    document.addEventListener("visibilitychange", onVisibility);
    const onMessage = (event: MessageEvent) => {
      if (event.source === parent && isMenuMoved(event.data)) this.moved();
    };
    addEventListener("message", onMessage);
    addEventListener("click", this.screen, true);
    return () => {
      this.root = null;
      this.visibility(false);
      observer.disconnect();
      document.removeEventListener("visibilitychange", onVisibility);
      removeEventListener("message", onMessage);
      removeEventListener("click", this.screen, true);
    };
  }
}
