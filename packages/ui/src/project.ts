import type { SupportDetails } from "./vault-api.ts";

export const sourceRepository = "https://github.com/dortanes/ravenpass";

export const licenseAddress = `${sourceRepository}/blob/main/LICENSE`;

/** GitHub keeps a report sent here private between the reporter and the maintainers. */
export const vulnerabilityReportAddress = `${sourceRepository}/security/advisories/new`;

export const donationAddress = "https://anthy.net/donate";

export interface BuildInfo {
  version: string;
  build: string;
  released: boolean;
}

/** Release tags are `v<version>`; a build no release names opens the list of releases. */
export function releaseNotesAddress(build: BuildInfo): string {
  if (!build.released) return `${sourceRepository}/releases`;
  return `${sourceRepository}/releases/tag/v${encodeURIComponent(build.version)}`;
}

/** The `app` options of .github/ISSUE_TEMPLATE/bug_report.yml by platform. */
const reportedApps: ReadonlyMap<string, string> = new Map([
  ["macOS", "macOS app"],
  ["Android", "Android app"],
]);

/** Fills the bug report form's technical fields, by their ids, and nothing else. */
export function problemReportAddress(
  build: BuildInfo,
  details: SupportDetails | null,
): string {
  const address = new URL(`${sourceRepository}/issues/new`);
  const fields = address.searchParams;
  fields.set("template", "bug_report.yml");
  fields.set("version", build.version);
  fields.set("build", build.build);
  const app = details && reportedApps.get(details.platform);
  if (app) fields.set("app", app);
  const system = [details?.platform, details?.osVersion]
    .filter(Boolean)
    .join(" ");
  if (system) fields.set("platform", system);
  return address.href;
}
