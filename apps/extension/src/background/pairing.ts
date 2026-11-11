import { type LinkClient, LinkError } from "../link/client.ts";
import { isPairingDiagnostic } from "../link/diagnostics.ts";
import { ConnectionKey } from "../link/key.ts";
import type { Answers } from "../messages.ts";

import type { ExtensionPageSender, ExtensionPages } from "./extension-pages.ts";

/** Keeps pairing authorization and bounded failure reporting at the background boundary. */
export class ExtensionPairing {
  private readonly client: Pick<LinkClient, "link">;
  private readonly name: () => string;
  private readonly pages: ExtensionPages;

  constructor(
    client: Pick<LinkClient, "link">,
    name: () => string,
    pages: ExtensionPages,
  ) {
    this.client = client;
    this.name = name;
    this.pages = pages;
  }

  async link(
    text: string,
    sender: ExtensionPageSender,
  ): Promise<Answers["link"]> {
    if (!this.pages.allows(sender)) {
      return { ok: false, reason: "failed", diagnostic: "RP-11" };
    }
    let key: ConnectionKey;
    try {
      key = await ConnectionKey.parse(text);
    } catch {
      return { ok: false, reason: "not-key", diagnostic: "RP-00" };
    }
    try {
      await this.client.link(key, this.name());
      return { ok: true };
    } catch (error) {
      return error instanceof LinkError
        ? {
            ok: false,
            reason: error.reason,
            diagnostic: isPairingDiagnostic(error.diagnostic)
              ? error.diagnostic
              : "RP-14",
          }
        : { ok: false, reason: "failed", diagnostic: "RP-14" };
    }
  }
}
