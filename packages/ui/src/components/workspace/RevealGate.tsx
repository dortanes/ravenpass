import { useQuery, useQueryClient } from "@tanstack/react-query";
import { type ReactNode, useRef, useState } from "react";
import { failureCode } from "../../failures.ts";
import type { MessageKey } from "../../i18n/messages.ts";
import { useTranslator } from "../../i18n/translator.tsx";
import { queryKeys } from "../../query/keys.ts";
import { unlockMethodsQuery } from "../../query/unlock-methods.ts";
import type { VaultApi } from "../../vault-api.ts";
import { CurrentPinDialog } from "../UnlockOptions.tsx";
import { ownerCheck } from "../unlock-change.ts";

/** What the gate asks of the host. */
export type RevealHost = Pick<VaultApi, "confirmReveal" | "unlockMethods">;

/** A reveal gate holds an item's secrets back until the owner confirms it is them, once per opened item. */
export interface RevealGate {
  confirmed: boolean;
  /** Runs `action` once the owner confirms, at once when they already did. */
  after: (action: () => void) => void;
  /** The PIN dialog, for a vault the device cannot confirm the owner of by itself. */
  dialog: ReactNode;
}

/** useRevealGate asks device authentication where the vault opens with it, else the PIN, else nothing. */
export function useRevealGate(
  host: RevealHost,
  item: string,
  report: (cause: unknown, message: MessageKey) => void,
): RevealGate {
  const { t } = useTranslator();
  const client = useQueryClient();
  const methods = useQuery(unlockMethodsQuery(host)).data ?? null;
  const [confirmed, setConfirmed] = useState(false);
  const [asking, setAsking] = useState(false);
  const pending = useRef<(() => void) | null>(null);
  const checking = useRef(false);

  function pass() {
    setConfirmed(true);
    const action = pending.current;
    pending.current = null;
    action?.();
  }

  function refused(cause: unknown) {
    // A prompt the owner declined says nothing more.
    if (failureCode(cause) !== "owner-unverified") {
      report(cause, "workspace.reveal.error");
    }
  }

  async function after(action: () => void) {
    if (confirmed) {
      action();
      return;
    }
    if (checking.current) return;
    pending.current = action;
    checking.current = true;
    try {
      const known =
        methods ?? (await client.fetchQuery(unlockMethodsQuery(host)));
      if (ownerCheck(known) === "pin") {
        setAsking(true);
        return;
      }
      await host.confirmReveal(item, "");
      pass();
    } catch (cause) {
      pending.current = null;
      refused(cause);
    } finally {
      checking.current = false;
    }
  }

  async function confirmPin(pin: string): Promise<boolean> {
    try {
      await host.confirmReveal(item, pin);
      pass();
      return true;
    } catch (cause) {
      refused(cause);
      await client.invalidateQueries({ queryKey: queryKeys.unlockMethods });
      return false;
    }
  }

  const dialog = (
    <CurrentPinDialog
      open={asking}
      description={t("workspace.reveal.pin", { item })}
      attemptsLeft={methods?.pinAttemptsLeft}
      minimum={methods?.pinMinLength ?? 6}
      maximum={methods?.pinMaxLength ?? 12}
      onConfirm={confirmPin}
      onClose={() => {
        setAsking(false);
        pending.current = null;
      }}
    />
  );

  return { confirmed, after: (action) => void after(action), dialog };
}
