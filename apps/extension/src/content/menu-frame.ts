import { enterTiming, leaveTiming } from "@ravenpass/ui/motion/timings.ts";
import { menuMoved } from "../messages.ts";
import {
  backdropFilter,
  type Frosting,
  frostings,
  pageLuminanceBehind,
  preferredScheme,
} from "./backdrop.ts";
import { Frost } from "./frost.ts";
import { guardedAttributes, HostGuard } from "./host-guard.ts";
import { HostVisibility } from "./host-visibility.ts";

const gapPx = 4;
const minWidthPx = 280;
const maxWidthPx = 360;
const pointWidthPx = 320;
const cornerInsetPx = 16;
const cornerWidthPx = 300;
const maxHeightPx = 480;

/** Matches the menu surface's `rounded-row`. */
const surfaceRadius = "14px";

/** Important inline declarations win over page rules and also outrank animations:
 * animate the surface inside the shadow root, never the host. */
const hostStyle: Readonly<Record<string, string>> = {
  all: "initial",
  display: "block",
  position: "fixed",
  visibility: "hidden",
  margin: "0",
  padding: "0",
  border: "0",
  overflow: "visible",
  background: "transparent",
};

/** At rest nothing on the host, surface or frame may carry opacity below 1, a filter, a backdrop
 * filter or a transform: Chrome reports a frame under any of them as not visible. */
const surfaceStyle: Readonly<Record<string, string>> = {
  display: "block",
  position: "relative",
  width: "100%",
  height: "100%",
  "border-radius": surfaceRadius,
  "transform-origin": "top left",
};

/** Painted beneath the frame; carries the shadow and, at rest, the blur. */
function frostStyle({ ring }: Frosting): Readonly<Record<string, string>> {
  return {
    position: "absolute",
    inset: "0",
    "border-radius": surfaceRadius,
    // The dark ring sets the menu off a dark page, where the drop shadows disappear.
    "box-shadow": `0 0 0 1px rgb(0 0 0 / ${ring}), 0 24px 56px -14px rgb(0 0 0 / 0.6), 0 8px 18px -8px rgb(0 0 0 / 0.45)`,
    "pointer-events": "none",
  };
}

const heightTransition = `height ${enterTiming.duration}ms ${enterTiming.easing}`;

const reducedMotion = matchMedia("(prefers-reduced-motion: reduce)");

const shown: Keyframe = { opacity: 1, transform: "none" };
const entering: Keyframe = {
  opacity: 0,
  transform: "translateY(-6px) scale(0.96)",
};
const leaving: Keyframe = {
  opacity: 0,
  transform: "translateY(-3px) scale(0.98)",
};
const still: Keyframe = { opacity: 0 };

// Chrome paints a frame opaque when its colour scheme differs from its document's, which follows Chrome's mode.
const frameStyle: Readonly<Record<string, string>> = {
  all: "initial",
  display: "block",
  // Positioned after the frost layer, the frame paints above it.
  position: "relative",
  width: "100%",
  height: "100%",
  border: "0",
  margin: "0",
  padding: "0",
  background: "transparent",
  "color-scheme": "light dark",
};

/** Where a menu frame goes, in the viewport's coordinates. */
export interface FrameBox {
  readonly top: number;
  readonly left: number;
  readonly width: number;
}

export interface MenuAnchor {
  readonly document: Document;
  /** Where the frame goes at a height, or null once the anchor has left the page. */
  box(height: number): FrameBox | null;
  /** Reports the anchor moving or leaving the page until the returned function is called. */
  watch(events: { moved: () => void; gone: () => void }): () => void;
}

/** Gone when the field leaves the page or stops taking space. */
export class FieldAnchor implements MenuAnchor {
  readonly document: Document;
  private readonly field: HTMLInputElement;

  constructor(field: HTMLInputElement) {
    this.field = field;
    this.document = field.ownerDocument;
  }

  box(): FrameBox | null {
    const rect = this.field.getBoundingClientRect();
    if (!this.field.isConnected || (rect.width === 0 && rect.height === 0)) {
      return null;
    }
    const viewport = this.document.documentElement.clientWidth;
    const width = Math.min(Math.max(rect.width, minWidthPx), maxWidthPx);
    const left = Math.min(
      Math.max(rect.left, 0),
      Math.max(viewport - width, 0),
    );
    return { top: rect.bottom + gapPx, left, width };
  }

  watch({ moved, gone }: { moved: () => void; gone: () => void }): () => void {
    const resizes = new ResizeObserver(moved);
    resizes.observe(this.field);
    const removals = new MutationObserver(() => {
      if (!this.field.isConnected) gone();
    });
    removals.observe(this.document, { childList: true, subtree: true });
    const root = this.field.getRootNode();
    if (root instanceof ShadowRoot) {
      removals.observe(root, { childList: true, subtree: true });
    }
    return () => {
      resizes.disconnect();
      removals.disconnect();
    };
  }
}

/** Keeps the window in the viewport, moving it up and left from the point. */
export class PointAnchor implements MenuAnchor {
  readonly document: Document;
  private readonly x: number;
  private readonly y: number;

  constructor(document: Document, point: { x: number; y: number }) {
    this.document = document;
    this.x = point.x;
    this.y = point.y;
  }

  box(height: number): FrameBox {
    const { clientWidth, clientHeight } = this.document.documentElement;
    return {
      top: Math.max(Math.min(this.y, clientHeight - height), 0),
      left: Math.max(Math.min(this.x, clientWidth - pointWidthPx), 0),
      width: pointWidthPx,
    };
  }

  watch(): () => void {
    return () => {};
  }
}

export class CornerAnchor implements MenuAnchor {
  readonly document: Document;

  constructor(document: Document) {
    this.document = document;
  }

  box(): FrameBox {
    const { clientWidth } = this.document.documentElement;
    return {
      top: cornerInsetPx,
      left: Math.max(clientWidth - cornerWidthPx - cornerInsetPx, 0),
      width: cornerWidthPx,
    };
  }

  watch(): () => void {
    return () => {};
  }
}

/** A modal dialog makes the rest of the document inert: the host must go inside the last one. */
function hostContainer(document: Document): Element {
  const dialogs = document.querySelectorAll("dialog:modal");
  return dialogs[dialogs.length - 1] ?? document.documentElement;
}

/** The closed shadow root keeps the page's styles and scripts from reaching the frame. */
export class MenuFrame {
  private static readonly frames = new Map<string, MenuFrame>();
  private readonly token: string;

  static visible(token: string): boolean {
    const frame = MenuFrame.frames.get(token);
    if (!frame || frame.removed || !frame.shown) return false;
    frame.guard.review();
    return (
      !frame.removed &&
      new HostVisibility(frame.host, [frame.surface, frame.frame]).visible()
    );
  }

  readonly host: HTMLElement;
  private readonly frame: HTMLIFrameElement;
  private readonly anchor: MenuAnchor;
  private readonly surface: HTMLElement;
  private readonly frostLayer: HTMLElement;
  /** Chrome's mode as the menu opens; the menu page reads the same. */
  private readonly frosting = frostings[preferredScheme()];
  private frost: Frost | null = null;
  private readonly onGone: () => void;
  private readonly unwatch: () => void;
  private readonly guard: HostGuard;
  private readonly restyles: MutationObserver;
  private height = 0;
  private scheduled = 0;
  /** The animation frame of the next look at the host's box. */
  private measured = 0;
  /** The host's box on screen when last looked at. */
  private box = "";
  private shown = false;
  private removed = false;
  /** A hidden frame cannot take focus; focus asked for early moves in once shown. */
  private focusWhenShown = false;

  constructor(anchor: MenuAnchor, url: string, onGone: () => void) {
    this.token = new URL(url).hash.slice(1);
    this.anchor = anchor;
    this.onGone = onGone;
    const { document } = anchor;
    this.host = document.createElement("div");
    applyStyle(this.host, hostStyle);
    this.frame = document.createElement("iframe");
    this.frame.src = url;
    this.frame.title = "Ravenpass";
    applyStyle(this.frame, frameStyle);
    this.surface = document.createElement("div");
    applyStyle(this.surface, surfaceStyle);
    this.frostLayer = document.createElement("div");
    applyStyle(this.frostLayer, frostStyle(this.frosting));
    this.surface.append(this.frostLayer, this.frame);
    this.host.attachShadow({ mode: "closed" }).append(this.surface);
    this.host.popover = "manual";
    hostContainer(document).append(this.host);
    this.host.showPopover();
    this.guard = new HostGuard(this.host, onGone);
    this.restyles = new MutationObserver(() => this.guard.review());
    this.restyles.observe(this.host, {
      attributes: true,
      attributeFilter: [...guardedAttributes],
    });

    addEventListener("scroll", this.schedule, { capture: true, passive: true });
    addEventListener("resize", this.schedule, { passive: true });
    this.unwatch = anchor.watch({ moved: this.schedule, gone: onGone });
    this.schedule();
    this.measure();
    MenuFrame.frames.set(this.token, this);
  }

  focus(): void {
    if (this.shown) this.frame.focus();
    else this.focusWhenShown = true;
  }

  resize(height: number): void {
    if (this.removed) return;
    this.height = Math.min(Math.max(height, 0), maxHeightPx);
    this.place();
    if (!this.shown && this.height > 0) this.show();
  }

  remove(): void {
    if (this.removed) return;
    this.removed = true;
    if (MenuFrame.frames.get(this.token) === this)
      MenuFrame.frames.delete(this.token);
    this.restyles.disconnect();
    removeEventListener("scroll", this.schedule, { capture: true });
    removeEventListener("resize", this.schedule);
    this.unwatch();
    cancelAnimationFrame(this.scheduled);
    cancelAnimationFrame(this.measured);
    if (!this.shown || !this.host.isConnected) {
      this.host.remove();
      return;
    }
    applyStyle(this.host, { "pointer-events": "none" });
    this.frost?.moving();
    const exit = this.surface.animate(
      [shown, reducedMotion.matches ? still : leaving],
      { ...leaveTiming, fill: "forwards" },
    );
    const detach = () => this.host.remove();
    exit.finished.then(detach, detach);
  }

  private show(): void {
    this.shown = true;
    const frost = new Frost(
      this.surface,
      this.frostLayer,
      backdropFilter(pageLuminanceBehind(this.host), this.frosting),
    );
    this.frost = frost;
    frost.moving();
    this.styleHost({ visibility: "visible" });
    const entrance = this.surface.animate(
      [reducedMotion.matches ? still : entering, shown],
      enterTiming,
    );
    // The menu's still period starts once the surface has no opacity, transform or blur left.
    entrance.finished.then(
      () => {
        if (this.removed) return;
        frost.settled();
        this.tellMoved();
      },
      () => {},
    );
    // Lays out the first height before the transition exists: otherwise it animates from 0.
    this.host.getBoundingClientRect();
    this.styleHost({ transition: heightTransition });
    if (this.focusWhenShown) this.frame.focus();
  }

  /** Polls every frame: the page can move the host in ways no event reports. */
  private readonly measure = (): void => {
    const { top, left, width, height } = this.host.getBoundingClientRect();
    const box = `${top} ${left} ${width} ${height}`;
    if (box !== this.box) {
      this.box = box;
      this.tellMoved();
    }
    this.measured = requestAnimationFrame(this.measure);
  };

  private tellMoved(): void {
    this.frame.contentWindow?.postMessage(menuMoved, "*");
  }

  private styleHost(style: Readonly<Record<string, string>>): void {
    applyStyle(this.host, style);
    this.guard.settle();
  }

  private readonly schedule = (): void => {
    if (this.scheduled) return;
    this.scheduled = requestAnimationFrame(() => {
      this.scheduled = 0;
      this.place();
    });
  };

  private place(): void {
    const box = this.anchor.box(this.height);
    if (!box) {
      this.onGone();
      return;
    }
    this.styleHost({
      top: `${box.top}px`,
      left: `${box.left}px`,
      width: `${box.width}px`,
      height: `${this.height}px`,
    });
  }
}

function applyStyle(
  element: HTMLElement,
  style: Readonly<Record<string, string>>,
): void {
  for (const [property, value] of Object.entries(style)) {
    element.style.setProperty(property, value, "important");
  }
}
