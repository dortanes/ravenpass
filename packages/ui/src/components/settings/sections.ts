import {
  Download,
  Fingerprint,
  Import,
  Info,
  Keyboard,
  type LucideIcon,
  RectangleEllipsis,
  Settings2,
  ShieldCheck,
  Tags,
  Trash2,
  Vault,
} from "lucide-react";
import type { MessageKey } from "../../i18n/messages.ts";
import type { Capabilities } from "../../vault-api.ts";

export type SettingsSection =
  | "general"
  | "security"
  | "privacy"
  | "autofill"
  | "shortcuts"
  | "vaults"
  | "groups"
  | "trash"
  | "import"
  | "backups"
  | "about";

export type VaultSettingsTab =
  | "vaults"
  | "backups"
  | "groups"
  | "trash"
  | "import";

export type SettingsPart = "device" | "vault";

export interface SettingsSectionEntry {
  id: SettingsSection;
  part: SettingsPart;
  icon: LucideIcon;
  label: MessageKey;
  summary: MessageKey;
  /** A host offering any of these capabilities offers the section. */
  requires?: readonly (keyof Capabilities)[];
}

const autofillCapabilities: readonly (keyof Capabilities)[] = [
  "systemAutofill",
  "identityList",
  "extensions",
];

export const settingsSections: readonly SettingsSectionEntry[] = [
  {
    id: "general",
    part: "device",
    icon: Settings2,
    label: "settings.general.heading",
    summary: "settings.general.summary",
  },
  {
    id: "security",
    part: "device",
    icon: Fingerprint,
    label: "settings.security.heading",
    summary: "settings.security.summary",
  },
  {
    id: "privacy",
    part: "device",
    icon: ShieldCheck,
    label: "settings.privacy.heading",
    summary: "settings.privacy.summary",
  },
  {
    id: "autofill",
    part: "device",
    icon: RectangleEllipsis,
    label: "settings.autofill.heading",
    summary: "settings.autofill.summary",
    requires: autofillCapabilities,
  },
  {
    id: "shortcuts",
    part: "device",
    icon: Keyboard,
    label: "settings.shortcuts.heading",
    summary: "settings.shortcuts.summary",
    requires: ["shortcuts"],
  },
  {
    id: "vaults",
    part: "vault",
    icon: Vault,
    label: "settings.vaults.heading",
    summary: "settings.vaults.summary",
  },
  {
    id: "about",
    part: "vault",
    icon: Info,
    label: "settings.about.heading",
    summary: "settings.about.summary",
  },
];

const vaultTabs: readonly (SettingsSectionEntry & { id: VaultSettingsTab })[] =
  [
    {
      id: "vaults",
      part: "vault",
      icon: Vault,
      label: "settings.storage.heading",
      summary: "settings.vaults.summary",
    },
    {
      id: "backups",
      part: "vault",
      icon: Download,
      label: "settings.backup.heading",
      summary: "settings.backup.summary",
      requires: ["saveFiles"],
    },
    {
      id: "groups",
      part: "vault",
      icon: Tags,
      label: "settings.groups.heading",
      summary: "settings.groups.summary",
    },
    {
      id: "trash",
      part: "vault",
      icon: Trash2,
      label: "settings.trash.heading",
      summary: "settings.trash.summary",
    },
    {
      id: "import",
      part: "vault",
      icon: Import,
      label: "settings.import.heading",
      summary: "settings.import.summary",
    },
  ];

export function offeredVaultTabs(capabilities: Capabilities) {
  return vaultTabs.filter(
    (tab) => !tab.requires || tab.requires.some((key) => capabilities[key]),
  );
}

export function settingsParent(section: SettingsSection | null) {
  return vaultTabs.some((tab) => tab.id === section) ? "vaults" : section;
}

export function offeredSettingsDestinations(capabilities: Capabilities) {
  return [
    ...offeredSections(capabilities),
    ...offeredVaultTabs(capabilities).filter((tab) => tab.id !== "vaults"),
  ];
}

/** offeredSections is the sections a host with these capabilities can use, in their order. */
export function offeredSections(
  capabilities: Capabilities,
): SettingsSectionEntry[] {
  return settingsSections.filter(
    (section) =>
      !section.requires ||
      section.requires.some((capability) => capabilities[capability]),
  );
}

export interface SettingsPartEntry {
  part: SettingsPart;
  sections: SettingsSectionEntry[];
}

/** settingsParts groups sections by part, in the order each part first appears. */
export function settingsParts(
  sections: readonly SettingsSectionEntry[],
): SettingsPartEntry[] {
  const parts = new Map<SettingsPart, SettingsSectionEntry[]>();
  for (const section of sections) {
    const part = parts.get(section.part);
    if (part) part.push(section);
    else parts.set(section.part, [section]);
  }
  return [...parts].map(([part, members]) => ({ part, sections: members }));
}
