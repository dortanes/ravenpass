import { useState } from "react";
import { toast } from "sonner";
import { useTranslator } from "../../i18n/translator.tsx";
import type {
  PasswordHistoryCount,
  PasswordHistorySetting as Setting,
} from "../../vault-api.ts";
import { Block, BlockRow, blockControl } from "../Block.tsx";
import { ConfirmDialog } from "../ConfirmDialog.tsx";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "../ui/select.tsx";
import { Switch } from "../ui/switch.tsx";

/** A change that deletes passwords already kept, waiting for confirmation. */
interface Pending {
  enabled: boolean;
  days: number;
  count: number;
}

/** PasswordHistorySetting turns the generator's history on or off and sets its period, confirming what that deletes. */
export function PasswordHistorySetting({
  setting,
  busy,
  history,
  onChange,
}: {
  setting: Setting | null;
  busy: boolean;
  history: PasswordHistoryCount;
  onChange: (enabled: boolean, days: number) => void;
}) {
  const { t, failure } = useTranslator();
  const [pending, setPending] = useState<Pending | null>(null);

  // Off deletes every password kept, which a period of zero days counts.
  async function choose(enabled: boolean, days: number) {
    if (!setting) return;
    const shortens = !enabled || (setting.enabled && days < setting.days);
    if (!shortens) {
      onChange(enabled, days);
      return;
    }
    try {
      const count = await history.countPasswordHistoryPast(enabled ? days : 0);
      if (count > 0) setPending({ enabled, days, count });
      else onChange(enabled, days);
    } catch (cause) {
      toast.error(failure(cause, "settings.password-history.error"));
    }
  }

  function confirm() {
    if (!pending) return;
    onChange(pending.enabled, pending.days);
    setPending(null);
  }

  return (
    <>
      <Block>
        <BlockRow
          title={t("settings.password-history")}
          detail={t("settings.password-history.detail")}
          htmlFor="password-history"
          wrap
        >
          <Switch
            id="password-history"
            checked={Boolean(setting?.enabled)}
            disabled={busy || !setting}
            onCheckedChange={(enabled) =>
              void choose(enabled, setting?.days ?? 30)
            }
          />
        </BlockRow>
        {setting?.enabled && (
          <BlockRow
            title={t("settings.password-history.period")}
            htmlFor="password-history-period"
          >
            <Select
              value={String(setting.days)}
              onValueChange={(next) => void choose(true, Number(next))}
              disabled={busy}
            >
              <SelectTrigger
                id="password-history-period"
                size="sm"
                className={blockControl}
              >
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  {setting.offered.map((days) => (
                    <SelectItem key={days} value={String(days)}>
                      {t("settings.password-history.days", { count: days })}
                    </SelectItem>
                  ))}
                </SelectGroup>
              </SelectContent>
            </Select>
          </BlockRow>
        )}
      </Block>
      <ConfirmDialog
        open={pending !== null}
        title={t(
          pending?.enabled
            ? "settings.password-history.shorten.title"
            : "settings.password-history.off.title",
          { count: pending?.count ?? 0 },
        )}
        detail={t("settings.password-history.delete.detail")}
        confirm={t(
          pending?.enabled
            ? "settings.password-history.shorten.confirm"
            : "settings.password-history.off.confirm",
        )}
        cancel={t("workspace.delete.cancel")}
        destructive
        busy={busy}
        onConfirm={confirm}
        onCancel={() => setPending(null)}
      />
    </>
  );
}
