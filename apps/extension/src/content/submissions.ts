import type { CapturedCard } from "../link/client.ts";
import { ask, captureLifetimeMs } from "../messages.ts";
import { sendIgnoringClosedPort } from "../messaging/send.ts";
import { captureOf, type SubmittedInput, TypedValues } from "./captures.ts";
import { cardCaptureOf } from "./card-captures.ts";
import {
  boxInputs,
  boxOf,
  classify,
  describe,
  type FieldDescription,
  type FormBox,
  isVisible,
} from "./fields.ts";
import { Arming, Leaving } from "./leaving.ts";
import { changesAround } from "./page-changes.ts";
import { paymentFieldsIn } from "./payment-fields.ts";
import { controlEffect, controlSelector, readControls } from "./submit.ts";

interface ArmedForm {
  readonly box: FormBox;
  /** Described at arming: an input that left the page no longer reports its size or visibility. */
  readonly fields: readonly {
    readonly input: HTMLInputElement;
    readonly description: FieldDescription;
  }[];
  /** The password input the person typed into, whose leaving shows the form left. */
  readonly password: HTMLInputElement;
}

/** `ready` for a form that already left the page; `none` for a page being hidden. */
type AfterCapture = "watch" | "ready" | "none";

interface TypedCard {
  readonly card: CapturedCard;
  /** The number input, whose leaving shows the checkout moved on. */
  readonly number: HTMLInputElement;
}

/** Remembers of what the person typed only the value each input holds, which the page holds too. */
export class SubmissionWatcher {
  private readonly document: Document;
  private readonly typed = new TypedValues<HTMLInputElement>();
  /** Password inputs the person typed into; a show-password control turns them into text inputs. */
  private readonly passwordInputs = new WeakSet<HTMLInputElement>();
  /** Classified once, at the input's first keystroke. */
  private readonly logins = new WeakMap<HTMLInputElement, boolean>();
  private typedAccount = "";
  /** Trusted `input` events seen; tells a new submission of a box from a repeat of the captured one. */
  private keystrokes = 0;
  private readonly captured = new WeakMap<FormBox, number>();
  /** The card number last captured from each box. */
  private readonly capturedCards = new WeakMap<FormBox, string>();
  /** The form last captured, watched until it leaves the page. */
  private leaving: Leaving | null = null;
  private readonly arming = new Arming<ArmedForm>({
    left: ({ box, password }) => formLeft(box, password),
    capture: (form, left) => this.captureArmed(form, left),
    changes: ({ password }) => changesAround(password),
  });

  constructor(document: Document) {
    this.document = document;
  }

  start(): void {
    this.document.addEventListener("input", this.onInput, true);
    this.document.addEventListener("submit", this.onSubmit, true);
    this.document.addEventListener("click", this.onClick, true);
    this.document.addEventListener("keydown", this.onKeyDown, true);
    addEventListener("pagehide", this.onPageHide);
  }

  stop(): void {
    this.document.removeEventListener("input", this.onInput, true);
    this.document.removeEventListener("submit", this.onSubmit, true);
    this.document.removeEventListener("click", this.onClick, true);
    this.document.removeEventListener("keydown", this.onKeyDown, true);
    removeEventListener("pagehide", this.onPageHide);
    this.arming.disarm();
    this.leaving?.stop();
    this.leaving = null;
  }

  private readonly onInput = (event: Event): void => {
    const input = event.composedPath()[0];
    if (!(input instanceof HTMLInputElement)) return;
    this.typed.input(input, input.value, event.isTrusted);
    if (!event.isTrusted) return;
    this.keystrokes += 1;
    if (this.isPassword(input)) {
      this.passwordInputs.add(input);
    } else if (this.isLogin(input)) {
      this.typedAccount = input.value;
    }
  };

  private readonly onSubmit = (event: SubmitEvent): void => {
    if (event.isTrusted && event.target instanceof HTMLFormElement) {
      this.capture(event.target);
    }
  };

  private readonly onClick = (event: MouseEvent): void => {
    if (!event.isTrusted) return;
    const control = clickedControl(event.composedPath());
    if (!control) return;
    const box = boxOf(control);
    const card = this.typedCard(box);
    if (!card && !this.typedPassword(box)) return;
    const { controls, descriptions } = readControls(box);
    const clicked = controls.indexOf(control);
    if (clicked < 0) return;
    const effect = controlEffect(descriptions, clicked);
    if (card) {
      if (effect !== "unsent") this.captureCard(box, card);
      return;
    }
    switch (effect) {
      case "sends":
        this.capture(box);
        break;
      case "may-send":
        this.arm(box);
        break;
      case "unsent":
        this.arming.disarm((armed) => armed.box === box);
        break;
    }
  };

  private readonly onKeyDown = (event: KeyboardEvent): void => {
    const input = event.composedPath()[0];
    if (
      event.isTrusted &&
      event.key === "Enter" &&
      !event.isComposing &&
      input instanceof HTMLInputElement
    ) {
      this.arm(boxOf(input));
    }
  };

  private readonly onPageHide = (): void => {
    this.arming.pageHidden();
  };

  private capture(box: FormBox): void {
    this.arming.disarm((armed) => armed.box === box);
    const card = this.typedCard(box);
    if (card) {
      this.captureCard(box, card);
      return;
    }
    if (this.captured.get(box) === this.keystrokes) return;
    const inputs = boxInputs(box);
    if (!inputs.some((input) => this.isPassword(input))) return;
    this.send(
      box,
      inputs,
      inputs.map((input) => this.submitted(input, this.described(input))),
      "watch",
    );
  }

  private arm(box: FormBox): void {
    if (this.captured.get(box) === this.keystrokes) return;
    const password = this.typedPassword(box);
    if (!password) return;
    this.arming.arm({
      box,
      fields: boxInputs(box).map((input) => ({
        input,
        description: this.described(input),
      })),
      password,
    });
  }

  /** Inputs keep their values after leaving the page. */
  private captureArmed(form: ArmedForm, left: boolean): void {
    if (this.captured.get(form.box) === this.keystrokes) return;
    this.send(
      form.box,
      form.fields.map(({ input }) => input),
      form.fields.map(({ input, description }) =>
        this.submitted(input, description),
      ),
      left ? "ready" : "none",
    );
  }

  private send(
    box: FormBox,
    inputs: readonly HTMLInputElement[],
    submitted: readonly SubmittedInput[],
    after: AfterCapture,
  ): void {
    const captured = captureOf(submitted, this.typedAccount);
    if (!captured) return;
    this.captured.set(box, this.keystrokes);
    const sent = sendIgnoringClosedPort(ask({ kind: "capture", ...captured }));
    const gone = () => {
      void sent.then(() => sendIgnoringClosedPort(ask({ kind: "form-gone" })));
    };
    this.leaving?.stop();
    this.leaving = null;
    if (after === "ready") {
      gone();
    } else if (after === "watch") {
      const password = inputs.find(
        (input) => this.isPassword(input) && input.value === captured.password,
      );
      if (password) {
        this.leaving = new Leaving(
          () => formLeft(box, password),
          gone,
          captureLifetimeMs,
          changesAround(password),
        );
      }
    }
  }

  /** The box's payment fields as submitted, when the person typed its card number. */
  private typedCard(box: FormBox): TypedCard | null {
    const fields = paymentFieldsIn(box);
    const number = fields.find(({ part }) => part === "number")?.control;
    if (
      !(number instanceof HTMLInputElement) ||
      !this.typed.typedDigits(number, number.value)
    ) {
      return null;
    }
    const submitted = fields.map(({ part, control }) => ({
      part,
      value: control.value,
      text:
        control instanceof HTMLSelectElement
          ? (control.selectedOptions[0]?.text ?? "")
          : "",
      typed: control === number,
    }));
    const card = cardCaptureOf(submitted);
    return card && { card, number };
  }

  /** Sends a typed card once per number and box; the offer shows once the number field leaves the page. */
  private captureCard(box: FormBox, { card, number }: TypedCard): void {
    if (this.capturedCards.get(box) === card.number) return;
    this.capturedCards.set(box, card.number);
    const sent = sendIgnoringClosedPort(ask({ kind: "card-capture", ...card }));
    const gone = () => {
      void sent.then(() => sendIgnoringClosedPort(ask({ kind: "form-gone" })));
    };
    this.leaving?.stop();
    this.leaving = new Leaving(
      () => formLeft(box, number),
      gone,
      captureLifetimeMs,
      changesAround(number),
    );
  }

  private typedPassword(box: FormBox): HTMLInputElement | undefined {
    return boxInputs(box).find(
      (input) =>
        this.isPassword(input) &&
        input.value !== "" &&
        this.typed.typed(input, input.value),
    );
  }

  /** A password input a show-password control turned into text still reads as a password. */
  private described(input: HTMLInputElement): FieldDescription {
    const description = describe(input);
    return this.isPassword(input)
      ? { ...description, type: "password" }
      : description;
  }

  private submitted(
    input: HTMLInputElement,
    description: FieldDescription,
  ): SubmittedInput {
    return {
      ...description,
      value: input.value,
      typed: this.typed.typed(input, input.value),
    };
  }

  private isPassword(input: HTMLInputElement): boolean {
    return input.type === "password" || this.passwordInputs.has(input);
  }

  private isLogin(input: HTMLInputElement): boolean {
    let login = this.logins.get(input);
    if (login === undefined) {
      login = classify(describe(input)) === "login";
      this.logins.set(input, login);
    }
    return login;
  }
}

/** A sign-in that does not navigate removes or hides its password field once done. */
function formLeft(box: FormBox, password: HTMLInputElement): boolean {
  return (
    !password.isConnected ||
    !isVisible(password) ||
    (box instanceof HTMLFormElement && !box.isConnected)
  );
}

/** The nearest control on the click's path; null when the click went into a field first. */
function clickedControl(path: readonly EventTarget[]): HTMLElement | null {
  for (const target of path) {
    if (!(target instanceof HTMLElement)) continue;
    if (target.matches(controlSelector)) return target;
    if (
      target instanceof HTMLInputElement ||
      target instanceof HTMLTextAreaElement ||
      target instanceof HTMLSelectElement
    ) {
      return null;
    }
  }
  return null;
}
