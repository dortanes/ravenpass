import type { FillValues } from "../link/client.ts";
import {
  ask,
  type FormPresence,
  isFormMessage,
  type Submission,
} from "../messages.ts";
import { sendIgnoringClosedPort } from "../messaging/send.ts";
import {
  boxInputs,
  boxOf,
  describe,
  type Field,
  type FieldKind,
  type FormBox,
  fieldOf,
} from "./fields.ts";
import { fillForm, typeCode } from "./fill.ts";
import {
  type FormDescription,
  type FormIntent,
  formIntent,
} from "./form-intent.ts";
import { changesAround } from "./page-changes.ts";
import { cardFieldOf } from "./payment-fields.ts";
import { sendingControlText, submitForm } from "./submit.ts";

/** How often at most a page that keeps changing is searched for a sign-in form again. */
const scanIntervalMs = 500;

/** A page with more inputs than this is not searched; its fields show the card on focus. */
const maxInputs = 40;

const reportedKinds: readonly FieldKind[] = ["password", "login", "code"];

const notSubmitted: Submission = { submitted: false, password: false };

export interface SignInFormsDependencies {
  /** Called before a code is typed into a field's inputs, whose focus must open no menu. */
  readonly typing: (inputs: readonly HTMLInputElement[]) => void;
}

export class SignInForms {
  private readonly document: Document;
  private readonly typing: (inputs: readonly HTMLInputElement[]) => void;
  /** The field last reported or focused, which the sign-in card fills. */
  private target: Field | null = null;
  /** The input the form was last reported by. */
  private reported: HTMLInputElement | null = null;
  private scheduled: ReturnType<typeof setTimeout> | undefined;
  private unobserve: (() => void) | null = null;

  constructor(document: Document, { typing }: SignInFormsDependencies) {
    this.document = document;
    this.typing = typing;
  }

  start(): void {
    this.unobserve = changesAround(this.document)(this.schedule);
    this.document.addEventListener("focusin", this.onFocusIn, true);
    chrome.runtime.onMessage.addListener(this.onMessage);
    this.scan();
  }

  stop(): void {
    this.unobserve?.();
    this.unobserve = null;
    clearTimeout(this.scheduled);
    this.scheduled = undefined;
    this.document.removeEventListener("focusin", this.onFocusIn, true);
    chrome.runtime.onMessage.removeListener(this.onMessage);
  }

  private readonly schedule = (): void => {
    if (this.scheduled !== undefined) return;
    this.scheduled = setTimeout(() => {
      this.scheduled = undefined;
      this.scan();
    }, scanIntervalMs);
  };

  private scan(): void {
    const field = signInField(this.document);
    if (!field || field.inputs[0] === this.reported) return;
    this.reported = field.inputs[0];
    this.target = field;
    void this.report(field);
  }

  /** The service worker answers the values of a sign-in's next step, filled and submitted at once. */
  private async report(field: Field): Promise<void> {
    const answer = await sendIgnoringClosedPort(
      ask({ kind: "sign-in-form", field: field.kind }),
    );
    if (answer?.fill) await this.signIn(field, answer.fill);
  }

  private readonly onFocusIn = (event: FocusEvent): void => {
    const input = event.composedPath()[0];
    if (!(input instanceof HTMLInputElement)) return;
    const field = signInFieldOf(input);
    if (field) this.target = field;
  };

  private readonly onMessage = (
    message: unknown,
    sender: chrome.runtime.MessageSender,
    respond: (answer: Submission | FormPresence) => void,
  ): boolean | undefined => {
    if (sender.id !== chrome.runtime.id || !isFormMessage(message)) {
      return undefined;
    }
    const target = this.present();
    switch (message.kind) {
      case "form-present":
        respond({ present: target !== null });
        return undefined;
      case "form-sign-in":
        if (target?.kind === "code") {
          respond(notSubmitted);
          return undefined;
        }
        void this.signIn(target, message).then(respond);
        return true;
      case "form-code":
        void this.enterCode(target, message.code).then(respond);
        return true;
    }
  };

  /** The target while it is still a sign-in field on the page. */
  private present(): Field | null {
    const target = this.target;
    if (!target?.inputs.every((input) => input.isConnected)) return null;
    return fieldOf(target.inputs[0]) ? target : null;
  }

  /** Submits only when the fill left no field it found empty. */
  private async signIn(
    field: Field | null,
    values: FillValues,
  ): Promise<Submission> {
    if (!field) return notSubmitted;
    const input = field.inputs[0];
    const fill = fillForm(input, field.kind, values);
    if (!fill.complete) return { submitted: false, password: fill.password };
    await afterNextFrame();
    return { submitted: submitForm(input), password: fill.password };
  }

  private async enterCode(
    field: Field | null,
    code: string,
  ): Promise<Submission> {
    if (field?.kind !== "code" || !code) return notSubmitted;
    this.typing(field.inputs);
    typeCode(field.inputs, code);
    await afterNextFrame();
    return { submitted: submitForm(field.inputs[0]), password: false };
  }
}

/** Whether a document shows a form that signs in with an account or current password. */
export function showsSignInForm(document: Document): boolean {
  const field = signInField(document);
  return field !== null && field.kind !== "code";
}

/** An input's field, or null when its form creates an account or the input is a card's field. */
export function signInFieldOf(input: HTMLInputElement): Field | null {
  const field = fieldOf(input);
  return field && intentOf(boxOf(input)) === "sign-in" && !cardFieldOf(input)
    ? field
    : null;
}

function signInField(document: Document): Field | null {
  const inputs = document.querySelectorAll<HTMLInputElement>(
    "input:not([type=hidden])",
  );
  if (inputs.length > maxInputs) return null;
  const intents = new Map<FormBox, FormIntent>();
  const signsIn = (field: Field) => {
    const box = boxOf(field.inputs[0]);
    let intent = intents.get(box);
    if (intent === undefined) {
      intent = intentOf(box);
      intents.set(box, intent);
    }
    return intent === "sign-in";
  };
  const fields = Array.from(inputs, fieldOf).filter(
    (field): field is Field =>
      field !== null && signsIn(field) && !cardFieldOf(field.inputs[0]),
  );
  for (const kind of reportedKinds) {
    const field = fields.find((candidate) => candidate.kind === kind);
    if (field) return field;
  }
  return null;
}

function intentOf(box: FormBox): FormIntent {
  return formIntent(describeForm(box));
}

function describeForm(box: FormBox): FormDescription {
  return {
    fields: boxInputs(box).map(describe),
    control: sendingControlText(box),
    attributes: box instanceof HTMLFormElement ? formAttributes(box) : "",
    path: location.pathname,
    fragment: location.hash,
  };
}

// Read as attributes: an input named `id` or `name` shadows the form's properties.
function formAttributes(form: HTMLFormElement): string {
  const action = form.getAttribute("action");
  return [
    form.getAttribute("id"),
    form.getAttribute("name"),
    form.getAttribute("class"),
    form.getAttribute("aria-label"),
    action && URL.canParse(action, form.baseURI)
      ? new URL(action, form.baseURI).pathname
      : "",
  ].join(" ");
}

/** Pages enable their submit control only once they rendered the fill's input events, some a task later. */
function afterNextFrame(): Promise<void> {
  return new Promise((resolve) =>
    requestAnimationFrame(() => setTimeout(resolve)),
  );
}
