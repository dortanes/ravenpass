import { useCapabilities } from "../../host/capabilities.tsx";
import { useTranslator } from "../../i18n/translator.tsx";
import type {
  BankDetails,
  BreachChecks,
  ClipboardClearing,
  ScreenshotSettings,
  SiteIcons,
} from "../../vault-api.ts";
import { Block, BlockHeading, BlockNote, BlockRow } from "../Block.tsx";
import { Switch } from "../ui/switch.tsx";
import { DelaySetting } from "./DelaySetting.tsx";
import { ScreenshotsSetting } from "./ScreenshotsSetting.tsx";

export function PrivacyPanel({
  busy,
  siteIcons,
  onSiteIcons,
  bankDetails,
  onBankDetails,
  breachChecks,
  onBreachChecks,
  clipboard,
  onClipboard,
  screenshots,
}: {
  busy: boolean;
  siteIcons: SiteIcons | null;
  onSiteIcons: (enabled: boolean) => void;
  bankDetails: BankDetails | null;
  onBankDetails: (enabled: boolean) => void;
  breachChecks: BreachChecks | null;
  onBreachChecks: (enabled: boolean) => void;
  clipboard: ClipboardClearing | null;
  onClipboard: (enabled: boolean, seconds: number) => void;
  screenshots: ScreenshotSettings;
}) {
  const { t } = useTranslator();
  const offers = useCapabilities();

  return (
    <>
      <BlockHeading>{t("settings.privacy.clipboard")}</BlockHeading>
      <DelaySetting
        className=""
        id="clipboard-clear"
        setting={clipboard}
        busy={busy}
        onChange={onClipboard}
        title="settings.clipboard.clear"
        detail="settings.clipboard.clear.detail"
        delay="settings.clipboard.delay"
        note={
          offers.lockWhenHidden
            ? "settings.clipboard.note.hidden"
            : "settings.clipboard.note"
        }
      />
      <BlockHeading>{t("settings.privacy.websites")}</BlockHeading>
      <Block>
        <BlockRow
          title={t("settings.site-icons")}
          detail={t("settings.site-icons.detail")}
          htmlFor="site-icons"
        >
          <Switch
            id="site-icons"
            checked={Boolean(siteIcons?.enabled)}
            disabled={busy || !siteIcons}
            onCheckedChange={onSiteIcons}
          />
        </BlockRow>
        <BlockRow
          title={t("settings.bank-details")}
          detail={t("settings.bank-details.detail")}
          htmlFor="bank-details"
        >
          <Switch
            id="bank-details"
            checked={Boolean(bankDetails?.enabled)}
            disabled={busy || !bankDetails}
            onCheckedChange={onBankDetails}
          />
        </BlockRow>
      </Block>
      <BlockNote>{t("settings.site-icons.note")}</BlockNote>
      <BlockHeading>{t("settings.privacy.passwords")}</BlockHeading>
      <Block>
        <BlockRow
          title={t("settings.breach-checks")}
          detail={t("settings.breach-checks.detail")}
          htmlFor="breach-checks"
          wrap
        >
          <Switch
            id="breach-checks"
            checked={Boolean(breachChecks?.enabled)}
            disabled={busy || !breachChecks}
            onCheckedChange={onBreachChecks}
          />
        </BlockRow>
      </Block>
      {offers.screenshots && (
        <ScreenshotsSetting settings={screenshots} busy={busy} />
      )}
    </>
  );
}
