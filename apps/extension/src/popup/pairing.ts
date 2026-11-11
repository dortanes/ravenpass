import * as v from "valibot";
import { pairingDiagnostics } from "../link/diagnostics.ts";
import type { Answers } from "../messages.ts";

const reply = v.union([
  v.object({ ok: v.literal(true) }),
  v.object({
    ok: v.literal(false),
    reason: v.picklist(["not-key", "expired", "not-open", "failed"]),
    diagnostic: v.optional(v.picklist(pairingDiagnostics)),
  }),
]);

/** Validates the background boundary without exposing rejected reply or exception data. */
export class PopupPairing {
  private readonly send: (key: string) => Promise<unknown>;

  constructor(send: (key: string) => Promise<unknown>) {
    this.send = send;
  }

  async link(key: string): Promise<Answers["link"]> {
    try {
      const answer = v.safeParse(reply, await this.send(key));
      if (answer.success) {
        if (!answer.output.ok && !answer.output.diagnostic) {
          return { ...answer.output, diagnostic: "RP-15" };
        }
        return answer.output;
      }
    } catch {
      // A closed message port and an invalid reply share the same bounded reference.
    }
    return { ok: false, reason: "failed", diagnostic: "RP-12" };
  }
}
