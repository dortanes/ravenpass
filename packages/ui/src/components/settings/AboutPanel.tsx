import { useQuery } from "@tanstack/react-query";
import { ExternalLink } from "lucide-react";
import { useState } from "react";
import appIcon from "../../../../../apps/desktop/build/appicon.png";
import { useCapabilities } from "../../host/capabilities.tsx";
import { useTranslator } from "../../i18n/translator.tsx";
import {
  donationAddress,
  problemReportAddress,
  releaseNotesAddress,
  sourceRepository,
  vulnerabilityReportAddress,
} from "../../project.ts";
import { queryKeys } from "../../query/keys.ts";
import type { AboutApi } from "../../vault-api.ts";
import { Block, BlockRow, BlockRowButton } from "../Block.tsx";
import { Button } from "../ui/button.tsx";
import { LicenseDialog } from "./LicenseDialog.tsx";

export function AboutPanel({
  about,
  onOpenWebsite,
}: {
  about: AboutApi;
  onOpenWebsite: (address: string) => void;
}) {
  const { t } = useTranslator();
  const offers = useCapabilities();
  const [licenseOpen, setLicenseOpen] = useState(false);
  // A report opened before the details arrive, or without them, carries the version alone.
  const details =
    useQuery({
      queryKey: queryKeys.supportDetails,
      queryFn: () => about.supportDetails(),
      staleTime: Number.POSITIVE_INFINITY,
    }).data ?? null;

  return (
    <>
      <div className="mt-0.5 mb-6 flex items-center gap-3">
        <img
          src={appIcon}
          alt=""
          draggable={false}
          className="size-11 shrink-0 object-contain"
        />
        <span className="text-xl font-medium">Ravenpass</span>
      </div>
      <Block>
        <BlockRow title={t("settings.about.version")}>
          <span className="select-text text-[11px] text-muted-foreground">
            {RAVENPASS_BUILD.version}
          </span>
        </BlockRow>
        <BlockRow title={t("settings.about.build")}>
          <span className="select-text text-[11px] text-muted-foreground">
            {RAVENPASS_BUILD.build}
          </span>
        </BlockRow>
      </Block>
      <Block className="mt-5">
        <BlockRowButton
          external
          title={t("settings.about.release-notes")}
          onClick={() => onOpenWebsite(releaseNotesAddress(RAVENPASS_BUILD))}
        />
        <BlockRowButton
          external
          title={t("settings.about.report-problem")}
          detail={t("settings.about.report-problem.detail")}
          onClick={() =>
            onOpenWebsite(problemReportAddress(RAVENPASS_BUILD, details))
          }
        />
        <BlockRowButton
          external
          title={t("settings.about.report-vulnerability")}
          detail={t("settings.about.report-vulnerability.detail")}
          onClick={() => onOpenWebsite(vulnerabilityReportAddress)}
        />
      </Block>
      <Block className="mt-5">
        <BlockRowButton
          title={t("settings.about.license")}
          detail={t("settings.about.license.detail")}
          onClick={() => setLicenseOpen(true)}
        />
        {offers.donations && (
          <BlockRowButton
            external
            title={t("settings.about.donate")}
            onClick={() => onOpenWebsite(donationAddress)}
          />
        )}
      </Block>
      <Button
        type="button"
        variant="quiet"
        size="pill-sm"
        className="mt-[18px] h-auto min-h-[26px] bg-tile py-[5px] font-normal hover:bg-field-hover"
        onClick={() => onOpenWebsite(sourceRepository)}
      >
        <ExternalLink data-icon="inline-start" />
        {t("settings.about.source")}
      </Button>
      <LicenseDialog
        open={licenseOpen}
        onOpenChange={setLicenseOpen}
        about={about}
        onOpenWebsite={onOpenWebsite}
      />
    </>
  );
}
