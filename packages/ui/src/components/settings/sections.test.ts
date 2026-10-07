import assert from "node:assert/strict";
import test from "node:test";
import type { Capabilities } from "../../vault-api.ts";
import {
  offeredSections,
  offeredSettingsDestinations,
  offeredVaultTabs,
  settingsParent,
  settingsParts,
  settingsSections,
} from "./sections.ts";

const everything: Capabilities = {
  extensions: true,
  shortcuts: true,
  dockIcon: true,
  storageLocations: true,
  saveFiles: true,
  unlockOnShow: true,
  interfaceSize: true,
  appearance: true,
  photoPicker: true,
  identityList: true,
  copyScans: true,
  lockWhenHidden: true,
  screenshots: true,
  systemAutofill: true,
  autoBackups: true,
  print: true,
  qrCodes: true,
};

const nothing: Capabilities = {
  extensions: false,
  shortcuts: false,
  dockIcon: false,
  storageLocations: false,
  saveFiles: false,
  unlockOnShow: false,
  interfaceSize: false,
  appearance: false,
  photoPicker: false,
  identityList: false,
  copyScans: false,
  lockWhenHidden: false,
  screenshots: false,
  systemAutofill: false,
  autoBackups: false,
  print: false,
  qrCodes: false,
};

/** What the Mac and the Android phone offer. */
const mac: Capabilities = {
  ...nothing,
  extensions: true,
  shortcuts: true,
  dockIcon: true,
  storageLocations: true,
  saveFiles: true,
  appearance: true,
  identityList: true,
  copyScans: true,
  print: true,
  qrCodes: true,
};
const android: Capabilities = {
  ...nothing,
  storageLocations: true,
  unlockOnShow: true,
  interfaceSize: true,
  appearance: true,
  photoPicker: true,
  copyScans: true,
  lockWhenHidden: true,
  screenshots: true,
  systemAutofill: true,
  print: true,
};

const ids = (sections: readonly { id: string }[]) =>
  sections.map((section) => section.id);

test("a host that offers everything keeps every section in order", () => {
  assert.deepEqual(offeredSections(everything), [...settingsSections]);
});

test("a host that offers nothing keeps only the sections every host can use", () => {
  assert.deepEqual(ids(offeredSections(nothing)), [
    "general",
    "security",
    "privacy",
    "vaults",
    "about",
  ]);
});

test("each capability brings back only the section that needs it", () => {
  const needs = {
    shortcuts: "shortcuts",
    systemAutofill: "autofill",
    identityList: "autofill",
    extensions: "autofill",
  } as const;
  const base = ids(offeredSections(nothing));
  for (const [capability, section] of Object.entries(needs)) {
    const offered = ids(offeredSections({ ...nothing, [capability]: true }));
    assert.deepEqual(
      offered.filter((id) => !base.includes(id)),
      [section],
      `${capability} brings back the wrong sections`,
    );
  }
  for (const capability of [
    "dockIcon",
    "storageLocations",
    "saveFiles",
    "unlockOnShow",
    "interfaceSize",
    "photoPicker",
    "copyScans",
    "lockWhenHidden",
    "screenshots",
    "print",
  ] as const) {
    assert.deepEqual(
      ids(offeredSections({ ...nothing, [capability]: true })),
      base,
      `${capability} changes the sections`,
    );
  }
});

test("each host sees its sections in two parts: the device, then the vault", () => {
  const partsOf = (capabilities: Capabilities) =>
    settingsParts(offeredSections(capabilities)).map(({ part, sections }) => [
      part,
      ids(sections),
    ]);
  assert.deepEqual(partsOf(mac), [
    ["device", ["general", "security", "privacy", "autofill", "shortcuts"]],
    ["vault", ["vaults", "about"]],
  ]);
  assert.deepEqual(partsOf(android), [
    ["device", ["general", "security", "privacy", "autofill"]],
    ["vault", ["vaults", "about"]],
  ]);
  assert.deepEqual(partsOf(nothing), [
    ["device", ["general", "security", "privacy"]],
    ["vault", ["vaults", "about"]],
  ]);
  assert.deepEqual(settingsParts([]), []);
});

test("vault tabs preserve import, groups and the trash on every host and gate backups", () => {
  assert.deepEqual(ids(offeredVaultTabs(mac)), [
    "vaults",
    "backups",
    "groups",
    "trash",
    "import",
  ]);
  for (const host of [android, nothing]) {
    assert.deepEqual(ids(offeredVaultTabs(host)), [
      "vaults",
      "groups",
      "trash",
      "import",
    ]);
  }
});

test("direct vault destinations select the vault navigation entry", () => {
  for (const target of [
    "vaults",
    "backups",
    "groups",
    "trash",
    "import",
  ] as const) {
    assert.equal(settingsParent(target), "vaults");
  }
  for (const target of [
    null,
    "general",
    "security",
    "privacy",
    "autofill",
    "shortcuts",
    "about",
  ] as const) {
    assert.equal(settingsParent(target), target);
  }
});

test("command palette retains direct import and group destinations without duplicating vault", () => {
  for (const host of [mac, android, nothing]) {
    const destinations = ids(offeredSettingsDestinations(host));
    assert(destinations.includes("groups"));
    assert(destinations.includes("import"));
    assert.equal(destinations.includes("backups"), host.saveFiles);
    assert.equal(destinations.length, new Set(destinations).size);
  }
});
