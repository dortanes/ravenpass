import type { Request } from "../messages.ts";

/** A fixed reply, never an exception or request payload. */
export const refusedReply = { requestFailure: "refused" } as const;

export function isRefusedReply(reply: unknown): reply is typeof refusedReply {
  return (
    typeof reply === "object" &&
    reply !== null &&
    "requestFailure" in reply &&
    reply.requestFailure === "refused"
  );
}

/** A background authorization refusal, including a menu that has ended. */
export class RefusedRequest extends Error {
  constructor(kind: Request["kind"]) {
    super(`The ${kind} request was refused.`);
    this.name = "RefusedRequest";
  }
}

/** Unknown exceptions remain failures; never serialize their messages. */
export function requestFailureReply(
  error: unknown,
): typeof refusedReply | null {
  return error instanceof RefusedRequest ? refusedReply : null;
}

/** Only the caller's fixed kind and a known stage enter diagnostics. */
export class RequestFailure extends Error {
  readonly kind: Request["kind"];
  readonly reason: "refused" | "failed" | "transport";

  constructor(
    kind: Request["kind"],
    reason: "refused" | "failed" | "transport",
    options?: ErrorOptions,
  ) {
    super(`The ${kind} request ${reason}.`, options);
    this.kind = kind;
    this.reason = reason;
    this.name = `RequestFailure:${kind}:${reason}`;
  }

  /** Layout and dismissal need no retry once their menu refuses them. */
  get isHousekeepingRefusal(): boolean {
    return (
      this.reason === "refused" &&
      (this.kind === "menu-size" ||
        this.kind === "menu-close" ||
        this.kind === "menu-focus")
    );
  }
}
