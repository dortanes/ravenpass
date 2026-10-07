import {
  ask,
  type FieldMenuOpen,
  isFieldMenuOpen,
  isFrameMessage,
  menuPage,
} from "../messages.ts";
import { sendIgnoringClosedPort } from "../messaging/send.ts";
import { fillChosenForm } from "./card-fill.ts";
import type { Field } from "./fields.ts";
import { fillForm, typeCode } from "./fill.ts";
import { FieldAnchor, MenuFrame } from "./menu-frame.ts";
import { cardFieldOf } from "./payment-fields.ts";
import { PersonInput } from "./person-input.ts";
import { signInFieldOf } from "./sign-in-forms.ts";

// A click into the menu moves focus to its frame only after the field has lost it.
const blurDelayMs = 100;

/** A field whose menu lists cards. */
interface CardField {
  readonly kind: "card";
  readonly inputs: readonly [HTMLInputElement];
}

type MenuTarget = Field | CardField;

interface OpenMenu {
  /** The input the menu opened beneath, which focus returns to. */
  readonly input: HTMLInputElement;
  readonly field: MenuTarget;
  readonly token: string;
  readonly frame: MenuFrame;
}

/** The page never sees what the menu lists: the menu is an extension page in a cross-origin frame. */
export class FieldMenus {
  private readonly document: Document;
  private readonly person = new PersonInput();
  private open: OpenMenu | null = null;
  /** The input whose menu is being asked for. */
  private asking: HTMLInputElement | null = null;
  /** Inputs that open no menu until focus leaves them. */
  private dismissed: readonly HTMLInputElement[] = [];
  /** The input the last right-click in this frame started at, which the context menu's items open a menu for. */
  private rightClicked: HTMLInputElement | null = null;
  private dismissalCheck: ReturnType<typeof setTimeout> | undefined;
  private focusCheck: ReturnType<typeof setTimeout> | undefined;

  constructor(document: Document) {
    this.document = document;
  }

  start(): void {
    this.document.addEventListener("focusin", this.onFocusIn, true);
    this.document.addEventListener("focusout", this.onFocusOut, true);
    this.document.addEventListener("keydown", this.onKeyDown, true);
    this.document.addEventListener("keyup", this.onKeyUp, true);
    this.document.addEventListener("pointerdown", this.onPointerDown, true);
    this.document.addEventListener("contextmenu", this.onContextMenu, true);
    addEventListener("blur", this.onBlur);
    addEventListener("pagehide", this.onPageHide);
    chrome.runtime.onMessage.addListener(this.onMessage);
  }

  stop(): void {
    this.document.removeEventListener("focusin", this.onFocusIn, true);
    this.document.removeEventListener("focusout", this.onFocusOut, true);
    this.document.removeEventListener("keydown", this.onKeyDown, true);
    this.document.removeEventListener("keyup", this.onKeyUp, true);
    this.document.removeEventListener("pointerdown", this.onPointerDown, true);
    this.document.removeEventListener("contextmenu", this.onContextMenu, true);
    removeEventListener("blur", this.onBlur);
    removeEventListener("pagehide", this.onPageHide);
    chrome.runtime.onMessage.removeListener(this.onMessage);
    clearTimeout(this.dismissalCheck);
    clearTimeout(this.focusCheck);
    this.close({ tell: true, refocus: false });
  }

  /** Keeps the inputs from opening a menu until focus leaves them. */
  dismiss(inputs: readonly HTMLInputElement[]): void {
    this.dismissed = inputs;
  }

  private offer(event: KeyboardEvent | PointerEvent | FocusEvent): void {
    const opens = this.person.opensMenu(event);
    const input = inputOf(event);
    if (
      !input ||
      !opens ||
      input === this.asking ||
      holds(this.open?.field.inputs ?? [], input) ||
      holds(this.dismissed, input)
    ) {
      return;
    }
    const field = signInFieldOf(input) ?? cardFieldAt(input);
    if (!field) return;
    this.close({ tell: true, refocus: false });
    void this.ask(input, field);
  }

  private readonly onFocusIn = (event: FocusEvent): void => {
    this.offer(event);
  };

  private readonly onFocusOut = (event: FocusEvent): void => {
    const target = event.composedPath()[0];
    if (target === this.asking) this.asking = null;
    const dismissed = this.dismissed;
    if (holds(dismissed, target)) {
      clearTimeout(this.dismissalCheck);
      this.dismissalCheck = setTimeout(() => {
        if (this.dismissed === dismissed && !dismissed.some(isFocused)) {
          this.dismissed = [];
        }
      }, blurDelayMs);
    }
    const open = this.open;
    if (
      !open ||
      (!holds(open.field.inputs, target) && target !== open.frame.host)
    ) {
      return;
    }
    clearTimeout(this.focusCheck);
    this.focusCheck = setTimeout(() => {
      if (
        this.open === open &&
        !open.field.inputs.some(isFocused) &&
        !this.menuFocused()
      ) {
        this.close({ tell: true, refocus: false });
      }
    }, blurDelayMs);
  };

  private readonly onKeyDown = (event: KeyboardEvent): void => {
    this.person.pressed(event);
    const open = this.open;
    if (!open || !holds(open.field.inputs, event.composedPath()[0])) {
      this.offer(event);
      return;
    }
    if (event.key === "Escape") {
      event.preventDefault();
      event.stopPropagation();
      this.dismissed = open.field.inputs;
      this.close({ tell: true, refocus: false });
    } else if (event.key === "ArrowDown") {
      event.preventDefault();
      event.stopPropagation();
      open.frame.focus();
      void sendIgnoringClosedPort(
        ask({ kind: "menu-focus", token: open.token }),
      );
    } else if (event.key === "Alt" && event.isTrusted && !event.repeat) {
      // A page can dispatch key events of its own; only the person's reveal a code.
      this.reportOption(true);
    }
  };

  private readonly onKeyUp = (event: KeyboardEvent): void => {
    if (event.key === "Alt") this.reportOption(false);
  };

  /** A window that loses focus receives no key release. */
  private readonly onBlur = (): void => {
    this.reportOption(false);
  };

  /** Option's key events stay in the field and never reach the menu's frame. */
  private reportOption(held: boolean): void {
    const open = this.open;
    if (open?.field.kind !== "code") return;
    void sendIgnoringClosedPort(
      ask({ kind: "menu-option", token: open.token, held }),
    );
  }

  private readonly onPointerDown = (event: PointerEvent): void => {
    const open = this.open;
    const target = event.composedPath()[0];
    if (
      open &&
      !holds(open.field.inputs, target) &&
      target !== open.frame.host
    ) {
      this.close({ tell: true, refocus: false });
    }
    this.offer(event);
  };

  private readonly onContextMenu = (event: MouseEvent): void => {
    this.rightClicked = inputOf(event);
  };

  private readonly onPageHide = (): void => {
    this.close({ tell: true, refocus: false });
  };

  private readonly onMessage = (
    message: unknown,
    sender: chrome.runtime.MessageSender,
  ): undefined => {
    if (sender.id === chrome.runtime.id && isFieldMenuOpen(message)) {
      this.openRequested(message.field);
      return;
    }
    const open = this.open;
    if (
      !open ||
      sender.id !== chrome.runtime.id ||
      !isFrameMessage(message, open.token)
    ) {
      return;
    }
    switch (message.kind) {
      case "menu-size":
        open.frame.resize(message.height);
        break;
      case "menu-focus":
        open.input.focus();
        break;
      case "menu-close":
        this.close({ tell: false, refocus: true });
        break;
      case "fill":
        if (open.field.kind !== "card") {
          fillForm(open.input, open.field.kind, message);
        }
        this.close({ tell: false, refocus: true });
        break;
      case "fill-card":
        if (open.field.kind === "card") {
          fillChosenForm(open.input, message.values);
        }
        this.close({ tell: false, refocus: true });
        break;
      case "fill-code":
        // Typing moves focus through the field's inputs, which must not open another menu.
        this.dismissed = open.field.inputs;
        typeCode(open.field.inputs, message.code);
        this.close({ tell: false, refocus: false });
        break;
    }
  };

  /** Opens the menu the person chose from the context menu beneath the input they right-clicked, recognised or not. */
  private openRequested(kind: FieldMenuOpen["field"]): void {
    const input = this.rightClicked;
    this.rightClicked = null;
    if (!input?.isConnected) return;
    const found = kind === "card" ? null : signInFieldOf(input);
    const field: MenuTarget =
      found && (found.kind === "code") === (kind === "code")
        ? found
        : { kind, inputs: [input] };
    this.close({ tell: true, refocus: false });
    void this.ask(input, field, true);
    // Asking first keeps this focus from opening a second menu.
    input.focus({ preventScroll: true });
  }

  private async ask(
    input: HTMLInputElement,
    field: MenuTarget,
    requested = false,
  ): Promise<void> {
    this.asking = input;
    const answer = await sendIgnoringClosedPort(
      ask({ kind: "menu-open", field: field.kind, requested }),
    );
    const token = answer?.token;
    if (!token) return;
    if (this.asking !== input || !isFocused(input)) {
      void sendIgnoringClosedPort(ask({ kind: "menu-close", token }));
      return;
    }
    this.asking = null;
    this.open = {
      input,
      field,
      token,
      frame: new MenuFrame(
        new FieldAnchor(input),
        `${chrome.runtime.getURL(menuPage)}#${token}`,
        () => this.close({ tell: true, refocus: false }),
      ),
    };
  }

  /** `tell` is false when the service worker already ended the token. */
  private close({ tell, refocus }: { tell: boolean; refocus: boolean }): void {
    const open = this.open;
    if (!open) return;
    this.open = null;
    if (refocus && this.menuFocused(open)) {
      this.dismissed = open.field.inputs;
      open.input.focus({ preventScroll: true });
    }
    open.frame.remove();
    if (tell) {
      void sendIgnoringClosedPort(
        ask({ kind: "menu-close", token: open.token }),
      );
    }
  }

  private menuFocused(open = this.open): boolean {
    return open !== null && this.document.activeElement === open.frame.host;
  }
}

function cardFieldAt(input: HTMLInputElement): CardField | null {
  return cardFieldOf(input) ? { kind: "card", inputs: [input] } : null;
}

/** The input an event started at, including one inside an open shadow root. */
function inputOf(event: Event): HTMLInputElement | null {
  const target = event.composedPath()[0];
  return target instanceof HTMLInputElement ? target : null;
}

function holds(
  inputs: readonly HTMLInputElement[],
  target: EventTarget | undefined,
): boolean {
  return inputs.some((input) => input === target);
}

function isFocused(field: HTMLInputElement): boolean {
  const root = field.getRootNode();
  return (
    (root instanceof Document || root instanceof ShadowRoot) &&
    root.activeElement === field
  );
}
