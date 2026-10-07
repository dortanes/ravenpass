import { digitsOf } from "@ravenpass/ui/cards/networks.ts";
import type { CapturedPassword } from "../link/client.ts";
import { classify, type FieldDescription, pairedLogin } from "./fields.ts";

export interface SubmittedInput extends FieldDescription {
  readonly value: string;
  /** Whether the person typed `value`, not a fill or a page script. */
  readonly typed: boolean;
}

interface PasswordRoles {
  readonly password: SubmittedInput;
  /** The current password a change-password form asks for beside the new one. */
  readonly current: SubmittedInput | null;
}

/** Captures only a password the person typed; the current password goes along however it was filled. */
export function captureOf(
  inputs: readonly SubmittedInput[],
  typedAccount: string,
): CapturedPassword | null {
  const passwords = inputs.filter(
    (input) => input.type === "password" && input.visible && input.value,
  );
  const roles = passwordRoles(passwords);
  const [firstPassword] = passwords;
  if (!roles?.password.typed || !firstPassword) return null;
  const account = accountOf(inputs, inputs.indexOf(firstPassword));
  const captured = {
    account: account || typedAccount,
    password: roles.password.value,
  };
  return roles.current
    ? { ...captured, current: roles.current.value }
    : captured;
}

function passwordRoles(
  passwords: readonly SubmittedInput[],
): PasswordRoles | null {
  const declared = (token: string) =>
    passwords.find((input) => input.autocomplete.includes(token));
  const fresh = declared("new-password");
  const current = declared("current-password");
  if (fresh) return { password: fresh, current: current ?? null };
  if (current) return { password: current, current: null };
  const [first, second] = passwords;
  if (!first || passwords.length > 3) return null;
  if (!second || (passwords.length === 2 && first.value === second.value)) {
    return { password: first, current: null };
  }
  return { password: second, current: first };
}

function accountOf(
  inputs: readonly SubmittedInput[],
  password: number,
): string {
  const login = inputs[pairedLogin(inputs.map(classify), password)];
  if (login?.value) return login.value;
  return (
    inputs.find(
      (input) => input.autocomplete.includes("username") && input.value,
    )?.value ?? ""
  );
}

/** Ravenpass's fill and page scripts raise untrusted `input` events, which forget a typed value. */
export class TypedValues<Field extends object> {
  private readonly values = new WeakMap<Field, string>();

  input(field: Field, value: string, trusted: boolean): void {
    if (trusted) this.values.set(field, value);
    else this.values.delete(field);
  }

  typed(field: Field, value: string): boolean {
    return this.values.get(field) === value;
  }

  /** Whether the person typed the digits `value` holds; a page's formatting may add separators after the keystroke. */
  typedDigits(field: Field, value: string): boolean {
    const typed = this.values.get(field);
    const digits = digitsOf(value);
    return typed !== undefined && digits !== "" && digitsOf(typed) === digits;
  }
}
