import { useQueryClient } from "@tanstack/react-query";
import { useEffect } from "react";
import { useTranslator } from "../../i18n/translator.tsx";
import { useHostSetting } from "../../query/host-setting.ts";
import { queryKeys } from "../../query/keys.ts";
import type { LaunchAtLoginSettings } from "../../vault-api.ts";
import { BlockRow, BlockRowButton } from "../Block.tsx";
import { Switch } from "../ui/switch.tsx";

/** LaunchAtLoginSetting chooses whether the system opens Ravenpass at login. */
export function LaunchAtLoginSetting({
  settings,
  busy,
}: {
  settings: LaunchAtLoginSettings;
  busy: boolean;
}) {
  const { t } = useTranslator();
  const client = useQueryClient();
  const launch = useHostSetting({
    key: queryKeys.launchAtLogin,
    read: () => settings.launchAtLogin(),
    write: (enabled: boolean) => settings.setLaunchAtLogin(enabled),
    readFailure: "app.error.setting-read",
    writeFailure: "workspace.error.launch-at-login",
  });
  const hidden = useHostSetting({
    key: queryKeys.hiddenAtLogin,
    read: () => settings.hiddenAtLogin(),
    write: (next: boolean) => settings.setHiddenAtLogin(next),
    readFailure: "app.error.setting-read",
    writeFailure: "workspace.error.hidden-at-login",
  });
  // The owner can change it in System Settings, which leaves this window visible behind it.
  useEffect(() => {
    const read = () => {
      void client.invalidateQueries({ queryKey: queryKeys.launchAtLogin });
    };
    window.addEventListener("focus", read);
    return () => window.removeEventListener("focus", read);
  }, [client]);
  const blocked = Boolean(launch.value?.needsApproval);

  return (
    <>
      <BlockRow
        title={t("settings.launch-at-login")}
        detail={blocked ? t("settings.launch-at-login.blocked") : undefined}
        htmlFor="launch-at-login"
      >
        <Switch
          id="launch-at-login"
          checked={Boolean(launch.value?.enabled)}
          disabled={busy || launch.changing || !launch.value}
          onCheckedChange={(enabled) => {
            void launch.change(enabled);
          }}
        />
      </BlockRow>
      {blocked && (
        <BlockRowButton
          title={t("settings.launch-at-login.open")}
          onClick={() => {
            void settings.openLoginItems();
          }}
        />
      )}
      {launch.value?.enabled && (
        <BlockRow
          title={t("settings.launch-at-login.hidden")}
          htmlFor="hidden-at-login"
        >
          <Switch
            id="hidden-at-login"
            checked={Boolean(hidden.value?.hidden)}
            disabled={busy || hidden.changing || !hidden.value}
            onCheckedChange={(next) => {
              void hidden.change(next);
            }}
          />
        </BlockRow>
      )}
    </>
  );
}
