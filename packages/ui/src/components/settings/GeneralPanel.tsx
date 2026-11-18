import { cn } from "cn";
import { type Appearance, isAppearance } from "../../host/appearance.ts";
import { useCapabilities } from "../../host/capabilities.tsx";
import { naturalInterfaceSize } from "../../host/interface-size.ts";
import { formatPercent } from "../../i18n/language.ts";
import type { MessageKey } from "../../i18n/messages.ts";
import { useLanguageChoice, useTranslator } from "../../i18n/translator.tsx";
import type {
  AppearanceSetting,
  DockIcon,
  InterfaceSize,
  LaunchAtLoginSettings,
} from "../../vault-api.ts";
import { Block, BlockHeading, BlockRow, blockControl } from "../Block.tsx";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "../ui/select.tsx";
import { Switch } from "../ui/switch.tsx";
import { LaunchAtLoginSetting } from "./LaunchAtLoginSetting.tsx";

const appearanceLabels: Record<Appearance, MessageKey> = {
  system: "settings.appearance.system",
  light: "settings.appearance.light",
  dark: "settings.appearance.dark",
};

export function GeneralPanel({
  interfaceSize,
  onInterfaceSize,
  appearance,
  onAppearance,
  dockIcon,
  onDockIcon,
  launchAtLogin,
  busy,
}: {
  interfaceSize: InterfaceSize | null;
  onInterfaceSize: (percent: number) => void;
  appearance: AppearanceSetting | null;
  onAppearance: (appearance: Appearance) => void;
  dockIcon: DockIcon | null;
  onDockIcon: (hideWithWindow: boolean) => void;
  launchAtLogin: LaunchAtLoginSettings;
  busy: boolean;
}) {
  const { t } = useTranslator();
  const { language, options, choose } = useLanguageChoice();
  const offers = useCapabilities();

  return (
    <>
      <BlockHeading>{t("settings.general.interface")}</BlockHeading>
      <Block>
        <BlockRow title={t("settings.general.language")} htmlFor="language">
          <Select value={language} onValueChange={choose}>
            <SelectTrigger
              id="language"
              size="sm"
              className={cn(blockControl, "max-w-[170px]")}
              aria-label={t("settings.general.language")}
            >
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                {options.map((option) => (
                  <SelectItem key={option.value} value={option.value}>
                    {option.label}
                  </SelectItem>
                ))}
              </SelectGroup>
            </SelectContent>
          </Select>
        </BlockRow>
        {offers.appearance && (
          <BlockRow title={t("settings.appearance")} htmlFor="appearance">
            <Select
              value={appearance?.appearance}
              disabled={busy || !appearance}
              onValueChange={(next) => {
                if (isAppearance(next)) onAppearance(next);
              }}
            >
              <SelectTrigger
                id="appearance"
                size="sm"
                className={cn(blockControl, "max-w-[170px]")}
                aria-label={t("settings.appearance")}
              >
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  {appearance?.offered.map((option) => (
                    <SelectItem key={option} value={option}>
                      {t(appearanceLabels[option])}
                    </SelectItem>
                  ))}
                </SelectGroup>
              </SelectContent>
            </Select>
          </BlockRow>
        )}
        {offers.interfaceSize && (
          <BlockRow
            title={t("settings.interface-size")}
            htmlFor="interface-size"
          >
            <Select
              value={interfaceSize ? String(interfaceSize.percent) : undefined}
              disabled={busy || !interfaceSize}
              onValueChange={(next) => onInterfaceSize(Number(next))}
            >
              <SelectTrigger
                id="interface-size"
                size="sm"
                className={cn(blockControl, "max-w-[170px]")}
                aria-label={t("settings.interface-size")}
              >
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  {interfaceSize?.offered.map((percent) => {
                    const size = formatPercent(percent, language);
                    return (
                      <SelectItem key={percent} value={String(percent)}>
                        {percent === naturalInterfaceSize
                          ? t("settings.interface-size.natural", { size })
                          : size}
                      </SelectItem>
                    );
                  })}
                </SelectGroup>
              </SelectContent>
            </Select>
          </BlockRow>
        )}
      </Block>

      {(offers.launchAtLogin || offers.dockIcon) && (
        <>
          <BlockHeading>{t("settings.general.macos")}</BlockHeading>
          <Block>
            {offers.launchAtLogin && (
              <LaunchAtLoginSetting settings={launchAtLogin} busy={busy} />
            )}
            {offers.dockIcon && (
              <BlockRow
                title={t("settings.dock-icon")}
                detail={t("settings.dock-icon.detail")}
                htmlFor="dock-icon"
              >
                <Switch
                  id="dock-icon"
                  checked={Boolean(dockIcon?.hideWithWindow)}
                  disabled={busy || !dockIcon}
                  onCheckedChange={onDockIcon}
                />
              </BlockRow>
            )}
          </Block>
        </>
      )}
    </>
  );
}
