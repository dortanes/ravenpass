import { logFailure } from "../failures.ts";
import { RequestFailure } from "./request-failure.ts";

// Chrome's messaging errors when the receiving document, frame, tab or extension context is gone.
const closedPortErrors = [
  "Receiving end does not exist.",
  "The message port closed before a response was received.",
  "message channel closed before a response was received",
  "No tab with id",
  "Extension context invalidated.",
];

export const sendFailure = "Ravenpass could not deliver an extension message.";

export function isClosedPortError(error: unknown): boolean {
  if (error instanceof RequestFailure && error.reason === "transport") {
    return isClosedPortError(error.cause);
  }
  return (
    error instanceof Error &&
    closedPortErrors.some((text) => error.message.includes(text))
  );
}

/** Settles without rejection; closed receivers and refused menu housekeeping stay quiet. */
export async function sendIgnoringClosedPort<Answer>(
  sent: Promise<Answer>,
): Promise<Answer | undefined> {
  try {
    return await sent;
  } catch (error) {
    if (
      !isClosedPortError(error) &&
      !(error instanceof RequestFailure && error.isHousekeepingRefusal)
    ) {
      logFailure(sendFailure, error);
    }
    return undefined;
  }
}
