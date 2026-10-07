import type * as models from "./bindings/github.com/dortanes/ravenpass/packages/app/api/models.ts";
import * as service from "./bindings/github.com/dortanes/ravenpass/packages/app/api/service.ts";
import { isCardNetwork } from "./cards/networks.ts";
import { isItemKindName } from "./components/workspace/places.ts";
import { CountWatcher } from "./count-watcher.ts";
import { isSignInStyle, signInStyleOf } from "./extensions/sign-in-style.ts";
import { appearanceOf, isAppearance } from "./host/appearance.ts";
import { isDocumentType, isScanMediaType } from "./identities/identity.ts";
import {
  isImportFormat,
  isImportOrigin,
  isImportReason,
} from "./import/preview.ts";
import { isSeedChecksum, isSeedFormat } from "./seeds/phrase.ts";
import type {
  AutoBackup,
  BackupInterval,
  Card,
  CardNetwork,
  CardSummary,
  CodeSetup,
  Confirmation,
  Credential,
  CredentialSummary,
  ExportStatus,
  ExtensionLinks,
  GeneratorHistoryEntry,
  Identity,
  IdentityAddresses,
  IdentityDocument,
  IdentitySummary,
  ImportCardNetworks,
  ImportChoice,
  ImportFormat,
  ImportOptions,
  ImportPreview,
  ImportResult,
  ImportSource,
  LinkedExtension,
  Note,
  NoteSummary,
  PhraseCheck,
  ScanDraft,
  ScanMediaType,
  ScanSummary,
  Seed,
  SeedChecksum,
  SeedFormat,
  SeedSummary,
  SignInStyleSetting,
  StorageChange,
  StorageKind,
  StorageReason,
  StorageStatus,
  Trash,
  UnlockRequester,
  Vault,
  VaultApi,
  VaultState,
  VerificationReason,
} from "./vault-api.ts";

/** The host sends an empty list as null. */
function listed<Value>(values: Value[] | null | undefined): Value[] {
  return values ?? [];
}

async function getState(): Promise<VaultState> {
  const state = await service.GetState();
  if (
    state.phase !== "storage" &&
    state.phase !== "setup" &&
    state.phase !== "locked" &&
    state.phase !== "ready"
  ) {
    throw new Error(
      "Ravenpass could not read your vault state. Reopen the app.",
    );
  }
  return { phase: state.phase };
}

const storageKinds: StorageKind[] = ["local-file", "document"];
const storageReasons: StorageReason[] = [
  "",
  "unreachable",
  "selection-unusable",
];

function knownKind(value: string): value is StorageKind {
  return (storageKinds as string[]).includes(value);
}

/** A vault of a kind this version does not know reads as none. */
function knownVault(entry: models.Vault): Vault | null {
  if (!knownKind(entry.kind)) return null;
  return {
    kind: entry.kind,
    path: entry.path,
    name: entry.name,
    place: entry.place,
    current: entry.current,
  };
}

async function getStorage(): Promise<StorageStatus> {
  const status = await service.GetStorage();
  const kinds = listed(status.kinds).filter(knownKind);
  if (!kinds.length || (status.kind !== "" && !knownKind(status.kind))) {
    throw new Error(
      "Ravenpass could not read where your vault is kept. Reopen the app.",
    );
  }
  const reason = (storageReasons as string[]).includes(status.reason)
    ? (status.reason as StorageReason)
    : "";
  const vaults: Vault[] = [];
  for (const entry of listed(status.vaults)) {
    const vault = knownVault(entry);
    if (vault) vaults.push(vault);
  }
  return {
    kinds,
    chosen: listed(status.chosen).filter(knownKind),
    kind: status.kind === "" ? "" : (status.kind as StorageKind),
    path: status.path,
    name: status.name,
    place: status.place,
    vaults,
    available: status.available,
    restricted: status.restricted,
    reason,
    shared: status.shared,
    missing: status.missing,
  };
}

/** The host reports an empty previous vault, of no kind, when nothing moved. */
function storageChange(change: models.StorageChange): StorageChange {
  return {
    changed: change.changed,
    path: change.path,
    previous: knownVault(change.previous),
    previousRemoved: change.previousRemoved,
  };
}

async function selectStorageLocation(
  kind: StorageKind,
): Promise<StorageChange> {
  return storageChange(await service.SelectStorageLocation(kind));
}

async function moveStorageLocation(): Promise<StorageChange> {
  return storageChange(await service.MoveStorageLocation());
}

async function createVault(): Promise<StorageChange> {
  return storageChange(await service.CreateVault());
}

async function openVault(): Promise<StorageChange> {
  return storageChange(await service.OpenVault());
}

async function exportStatus(): Promise<ExportStatus> {
  const status = await service.ExportStatus();
  if (
    status.state !== "unknown" &&
    status.state !== "current" &&
    status.state !== "stale"
  ) {
    throw new Error(
      "Ravenpass could not check your encrypted copy. Reopen Ravenpass.",
    );
  }
  return { state: status.state };
}

const backupIntervals: BackupInterval[] = ["daily", "weekly", "monthly"];

function knownInterval(value: string): value is BackupInterval {
  return (backupIntervals as string[]).includes(value);
}

/** An unknown recorded interval fails the read; unknown offered intervals are left out. */
async function autoBackup(): Promise<AutoBackup> {
  const backup = await service.GetAutoBackup();
  if (!knownInterval(backup.interval)) {
    throw new Error(
      "Ravenpass could not read your automatic backup settings. Reopen Ravenpass.",
    );
  }
  return {
    enabled: backup.enabled,
    interval: backup.interval,
    intervals: listed(backup.intervals).filter(knownInterval),
    keep: backup.keep,
    keeps: listed(backup.keeps),
    folder: backup.folder
      ? { name: backup.folder.name, place: backup.folder.place }
      : null,
    lastBackupAt: backup.lastBackupAt,
    failed: backup.failed,
  };
}

async function listCredentials(): Promise<CredentialSummary[]> {
  const credentials: models.CredentialSummary[] | null =
    await service.ListCredentials();
  return listed(credentials).map((credential) => ({
    ...credential,
    groups: listed(credential.groups),
    tags: listed(credential.tags),
    sites: listed(credential.sites),
  }));
}

async function readCredential(id: string): Promise<Credential> {
  const credential: models.Credential = await service.ReadCredential(id);
  return {
    ...credential,
    groups: listed(credential.groups),
    tags: listed(credential.tags),
    websites: listed(credential.websites),
    passkeys: listed(credential.passkeys),
    apps: listed(credential.apps),
  };
}

async function getCodeSetup(): Promise<CodeSetup | null> {
  const setup = await service.GetCodeSetup();
  return setup.pending
    ? { token: setup.token, issuer: setup.issuer, account: setup.account }
    : null;
}

async function listIdentities(): Promise<IdentitySummary[]> {
  const identities: models.IdentitySummary[] | null =
    await service.ListIdentities();
  return listed(identities).map((identity) => ({
    ...identity,
    groups: listed(identity.groups),
    tags: listed(identity.tags),
  }));
}

async function readIdentity(id: string): Promise<Identity> {
  const identity: models.Identity = await service.ReadIdentity(id);
  const documents: IdentityDocument[] = [];
  for (const document of listed(identity.documents)) {
    if (!isDocumentType(document.type)) {
      throw new Error(
        "Ravenpass could not read a document in this identity. Update Ravenpass to open it.",
      );
    }
    documents.push({
      ...document,
      type: document.type,
      scans: listed(document.scans),
    });
  }
  const attachments: ScanSummary[] = [];
  for (const scan of listed(identity.attachments)) {
    attachments.push({ ...scan, mediaType: knownMediaType(scan.mediaType) });
  }
  return {
    ...identity,
    groups: listed(identity.groups),
    tags: listed(identity.tags),
    emails: listed(identity.emails),
    phones: listed(identity.phones),
    addresses: listed(identity.addresses),
    documents,
    attachments,
  };
}

function knownNetwork(value: string): CardNetwork {
  if (!isCardNetwork(value)) {
    throw new Error(
      "Ravenpass could not read a card's network. Update Ravenpass to open it.",
    );
  }
  return value;
}

async function listCards(): Promise<CardSummary[]> {
  const cards: models.CardSummary[] | null = await service.ListCards();
  return listed(cards).map((card) => ({
    ...card,
    network: knownNetwork(card.network),
    groups: listed(card.groups),
    tags: listed(card.tags),
  }));
}

async function readCard(id: string): Promise<Card> {
  const card: models.Card = await service.ReadCard(id);
  return {
    ...card,
    network: knownNetwork(card.network),
    groups: listed(card.groups),
    tags: listed(card.tags),
  };
}

async function listIdentityAddresses(): Promise<IdentityAddresses[]> {
  const identities: models.IdentityAddresses[] | null =
    await service.ListIdentityAddresses();
  return listed(identities).map((identity) => ({
    ...identity,
    addresses: listed(identity.addresses),
  }));
}

/** Items of a kind this version does not know are left out. */
async function listTrash(): Promise<Trash> {
  const trash = await service.ListTrash();
  return {
    retentionDays: trash.retentionDays,
    items: listed(trash.items).flatMap((item) =>
      isItemKindName(item.kind) ? [{ ...item, kind: item.kind }] : [],
    ),
  };
}

async function listNotes(): Promise<NoteSummary[]> {
  const notes: models.NoteSummary[] | null = await service.ListNotes();
  return listed(notes).map((note) => ({
    ...note,
    groups: listed(note.groups),
    tags: listed(note.tags),
  }));
}

async function readNote(id: string): Promise<Note> {
  const note: models.Note = await service.ReadNote(id);
  return { ...note, groups: listed(note.groups), tags: listed(note.tags) };
}

function knownFormat(value: string): SeedFormat {
  if (!isSeedFormat(value)) {
    throw new Error(
      "Ravenpass could not read a seed's format. Update Ravenpass to open it.",
    );
  }
  return value;
}

function knownChecksum(value: string): SeedChecksum {
  if (!isSeedChecksum(value)) {
    throw new Error(
      "Ravenpass could not check this seed phrase. Update Ravenpass to open it.",
    );
  }
  return value;
}

async function listSeeds(): Promise<SeedSummary[]> {
  const seeds: models.SeedSummary[] | null = await service.ListSeeds();
  return listed(seeds).map((seed) => ({
    ...seed,
    format: knownFormat(seed.format),
    groups: listed(seed.groups),
    tags: listed(seed.tags),
  }));
}

async function readSeed(id: string): Promise<Seed> {
  const seed: models.Seed = await service.ReadSeed(id);
  return {
    ...seed,
    format: knownFormat(seed.format),
    checksum: knownChecksum(seed.checksum),
    groups: listed(seed.groups),
    tags: listed(seed.tags),
    words: listed(seed.words),
    codes: listed(seed.codes),
    addresses: listed(seed.addresses),
  };
}

async function checkSeedPhrase(words: string[]): Promise<PhraseCheck> {
  const check: models.PhraseCheck = await service.CheckSeedPhrase(words);
  const checksum = knownChecksum(check.checksum);
  if (checksum === "") {
    throw new Error("Ravenpass could not check this seed phrase. Try again.");
  }
  return { checksum, unknownWords: listed(check.unknownWords) };
}

function knownMediaType(value: string): ScanMediaType {
  if (!isScanMediaType(value)) {
    throw new Error(
      "Ravenpass could not read a scan in this identity. Update Ravenpass to open it.",
    );
  }
  return value;
}

function scanDraft(draft: models.ScanDraft): ScanDraft {
  if (!draft.chosen) {
    return {
      chosen: false,
      token: "",
      name: "",
      mediaType: "image/jpeg",
      thumbnail: "",
    };
  }
  return { ...draft, mediaType: knownMediaType(draft.mediaType) };
}

async function chooseScan(): Promise<ScanDraft> {
  return scanDraft(await service.ChooseScan());
}

async function chooseScanPhoto(): Promise<ScanDraft> {
  return scanDraft(await service.ChooseScanPhoto());
}

function knownImportFormat(value: string): ImportFormat {
  if (!isImportFormat(value)) {
    throw new Error(
      "Ravenpass could not read this export file. Update Ravenpass.",
    );
  }
  return value;
}

/** Entries of a kind, origin or reason this version does not know are left out. */
function importPreview(preview: models.ImportPreview): ImportPreview {
  return {
    format: knownImportFormat(preview.format),
    items: preview.items,
    kinds: listed(preview.kinds).flatMap((entry) =>
      isItemKindName(entry.kind)
        ? [
            {
              kind: entry.kind,
              count: entry.count,
              duplicates: entry.duplicates,
              oneTimeCodes: entry.oneTimeCodes,
              converted: listed(entry.converted).flatMap((conversion) =>
                isImportOrigin(conversion.from)
                  ? [{ from: conversion.from, count: conversion.count }]
                  : [],
              ),
            },
          ]
        : [],
    ),
    groups: preview.groups,
    groupsWithoutDuplicates: preview.groupsWithoutDuplicates,
    cardIssuers: listed(preview.cardIssuers),
    skipped: listed(preview.skipped).flatMap((skip) =>
      isImportReason(skip.reason)
        ? [
            {
              label: skip.label,
              origin: isImportOrigin(skip.origin) ? skip.origin : null,
              reason: skip.reason,
            },
          ]
        : [],
    ),
    attachments: preview.attachments,
    passkeys: preview.passkeys,
    importedPasskeys: preview.importedPasskeys,
  };
}

async function chooseImportFile(source: ImportSource): Promise<ImportChoice> {
  const choice = await service.ChooseImportFile(source);
  return {
    chosen: choice.chosen,
    name: choice.name,
    locked: choice.locked,
    preview:
      choice.chosen && !choice.locked ? importPreview(choice.preview) : null,
  };
}

async function unlockImportFile(password: string): Promise<ImportPreview> {
  return importPreview(await service.UnlockImportFile(password));
}

async function importItems(
  options: ImportOptions,
  cardNetworks: ImportCardNetworks,
): Promise<ImportResult> {
  const result: models.ImportResult = await service.ImportItems({
    ...options,
    cardNetworks: { ...cardNetworks },
  });
  return {
    added: result.added,
    kinds: listed(result.kinds).flatMap((total) =>
      isItemKindName(total.kind)
        ? [{ kind: total.kind, count: total.count }]
        : [],
    ),
    groups: result.groups,
    format: knownImportFormat(result.format),
    located: result.located,
  };
}

async function extensionLinks(): Promise<ExtensionLinks> {
  const links = await service.ExtensionLinks();
  return { extensions: listed(links.extensions), reachable: links.reachable };
}

/** The styles this interface cannot name are left out of the offered ones. */
async function signInStyle(): Promise<SignInStyleSetting> {
  const setting = await service.GetSignInStyle();
  return {
    style: signInStyleOf(setting.style),
    offered: listed(setting.offered).filter(isSignInStyle),
  };
}

/** Cancels a waiting host call once signal aborts: Wails then cancels the call's context. */
function cancelOnAbort<Value>(
  call: Promise<Value> & { cancel(): void },
  signal: AbortSignal,
): Promise<Value> {
  if (signal.aborted) call.cancel();
  else signal.addEventListener("abort", () => call.cancel(), { once: true });
  return call;
}

/** Cancelling the host call ends the key it waits on. */
function awaitExtensionLink(signal: AbortSignal): Promise<LinkedExtension> {
  return cancelOnAbort(service.AwaitExtensionLink(), signal);
}

const verificationReasons: VerificationReason[] = [
  "share",
  "save-passkey",
  "sign-in",
  "fill",
  "change-unlock",
];

function isVerificationReason(value: string): value is VerificationReason {
  return (verificationReasons as string[]).includes(value);
}

const unlockRequesters: UnlockRequester[] = ["extension", "autofill"];

function isUnlockRequester(value: string): value is UnlockRequester {
  return (unlockRequesters as string[]).includes(value);
}

/** Cancelling ends only the wait, not the requests; an empty ID means no request waits. */
async function awaitConfirmation(
  shown: string,
  signal: AbortSignal,
): Promise<Confirmation | null> {
  const request = await cancelOnAbort<models.Confirmation>(
    service.AwaitConfirmation(shown),
    signal,
  );
  if (request.id === "") return null;
  if (request.kind === "unlock" && isUnlockRequester(request.requester)) {
    return { id: request.id, kind: "unlock", requester: request.requester };
  }
  if (request.kind === "verify" && isVerificationReason(request.reason)) {
    return {
      id: request.id,
      kind: "verify",
      reason: request.reason,
      file: request.file,
      identity: request.identity,
      site: request.site,
      account: request.account,
    };
  }
  throw new Error("Ravenpass could not read this request. Reopen Ravenpass.");
}

function awaitVaultChange(seen: number, signal: AbortSignal): Promise<number> {
  return cancelOnAbort(service.AwaitVaultChange(seen), signal);
}

function watchLanguage(onChange: () => void): () => void {
  return new CountWatcher((seen, signal) =>
    cancelOnAbort(service.AwaitLanguageChange(seen), signal),
  ).watch(onChange);
}

function generatorHistoryEntryOf(
  entry: models.GeneratedPassword,
): GeneratorHistoryEntry {
  return { ...entry, mode: entry.mode === "words" ? "words" : "characters" };
}

/** The vault API of a Wails host, desktop or mobile: both bind the same Go service. */
export const hostApi: VaultApi = {
  capabilities: service.Capabilities,
  getState,
  getStorage,
  async getLanguage() {
    const settings = await service.GetLanguage();
    return { ...settings, languages: settings.languages ?? [] };
  },
  setLanguage: service.SetLanguage,
  watchLanguage,
  async clipboardClearing() {
    const clearing = await service.GetClipboardClearing();
    return { ...clearing, offered: clearing.offered ?? [] };
  },
  setClipboardClearing: service.SetClipboardClearing,
  async autoLock() {
    const lock = await service.GetAutoLock();
    return { ...lock, offered: lock.offered ?? [] };
  },
  setAutoLock: service.SetAutoLock,
  dockIcon: service.GetDockIcon,
  setDockIcon: service.SetDockIcon,
  async interfaceSize() {
    const size = await service.GetInterfaceSize();
    return { ...size, offered: size.offered ?? [] };
  },
  setInterfaceSize: service.SetInterfaceSize,
  async appearance() {
    const setting = await service.GetAppearance();
    return {
      appearance: appearanceOf(setting.appearance),
      offered: listed(setting.offered).filter(isAppearance),
    };
  },
  setAppearance: service.SetAppearance,
  siteIcons: service.GetSiteIcons,
  setSiteIcons: service.SetSiteIcons,
  siteIcon: service.SiteIcon,
  identityList: service.GetIdentityList,
  setIdentityList: service.SetIdentityList,
  screenshotsAllowed: service.ScreenshotsAllowed,
  allowScreenshots: service.AllowScreenshots,
  systemAutofill: service.GetSystemAutofill,
  chooseSystemAutofill: service.ChooseSystemAutofill,
  async shortcuts() {
    const recorded = (await service.GetShortcuts()) ?? {};
    return Object.fromEntries(
      Object.entries(recorded).filter(
        (entry): entry is [string, string] => entry[1] !== undefined,
      ),
    );
  },
  setShortcut: service.SetShortcut,
  selectStorageLocation,
  chooseStorageMove: service.ChooseStorageMove,
  moveStorageLocation,
  retryStorage: service.RetryStorage,
  switchVault: service.SwitchVault,
  knowsVault: service.KnowsVault,
  createVault,
  cancelVaultCreation: service.CancelVaultCreation,
  openVault,
  forgetVault: service.ForgetVault,
  deleteVault: service.DeleteVault,
  beginCreation: service.BeginCreation,
  confirmCreation: service.ConfirmCreation,
  copyRecoveryKey: service.CopyRecoveryKey,
  saveRecoveryKey: service.SaveRecoveryKey,
  printRecoveryKey: service.PrintRecoveryKey,
  beginRecoveryPhraseChange: service.BeginRecoveryPhraseChange,
  confirmRecoveryPhraseChange: service.ConfirmRecoveryPhraseChange,
  cancelRecoveryPhraseChange: service.CancelRecoveryPhraseChange,
  unlock: service.Unlock,
  promptsUnlock: service.PromptsUnlock,
  unlockMethods: service.GetUnlockMethods,
  unlockWithPin: service.UnlockWithPIN,
  adoptChangedVault: service.AdoptChangedVault,
  setPin: service.SetPIN,
  removePin: service.RemovePIN,
  setBiometryUnlock: service.SetBiometryUnlock,
  beginRecovery: service.BeginRecovery,
  confirmRecovery: service.ConfirmRecovery,
  listCredentials,
  credentialLimits: service.GetCredentialLimits,
  readCredential,
  generateOneTimeCode: service.GenerateOneTimeCode,
  getCodeSetup,
  addCodeSetup: service.AddCodeSetup,
  createCredentialFromCodeSetup: service.CreateCredentialFromCodeSetup,
  dismissCodeSetup: service.DismissCodeSetup,
  clearSelection: service.ClearSelection,
  createCredential: service.CreateCredential,
  updateCredential: service.UpdateCredential,
  listIdentities,
  identityLimits: service.GetIdentityLimits,
  readIdentity,
  createIdentity: service.CreateIdentity,
  updateIdentity: service.UpdateIdentity,
  chooseIdentityPhoto: service.ChooseIdentityPhoto,
  cropIdentityPhoto: service.CropIdentityPhoto,
  discardIdentityPhoto: service.DiscardIdentityPhoto,
  chooseScan,
  chooseScanPhoto,
  discardScans: service.DiscardScans,
  copyScan: service.CopyScan,
  saveScan: service.SaveScan,
  listCards,
  cardLimits: service.GetCardLimits,
  readCard,
  createCard: service.CreateCard,
  updateCard: service.UpdateCard,
  copyCardField: service.CopyCardField,
  listIdentityAddresses,
  bankDetails: service.GetBankDetails,
  setBankDetails: service.SetBankDetails,
  breachChecks: service.GetBreachChecks,
  setBreachChecks: service.SetBreachChecks,
  async generatorHistorySetting() {
    const setting = await service.GetGeneratorHistorySetting();
    return { ...setting, offered: setting.offered ?? [] };
  },
  setGeneratorHistorySetting: service.SetGeneratorHistorySetting,
  countGeneratorHistoryPast: service.CountGeneratorHistoryPast,
  async checkBreaches() {
    const check = await service.CheckBreaches();
    return { checked: check.checked, breaches: listed(check.breaches) };
  },
  checkPassword: service.CheckPassword,
  lookupBank: service.LookupBank,
  lookupSite: service.LookupSite,
  readCodeSetup: service.ReadCodeSetup,
  tagLimits: service.GetTagLimits,
  confirmReveal: service.ConfirmReveal,
  listNotes,
  noteLimits: service.GetNoteLimits,
  readNote,
  createNote: service.CreateNote,
  updateNote: service.UpdateNote,
  copyNote: service.CopyNote,
  listSeeds,
  seedLimits: service.GetSeedLimits,
  readSeed,
  createSeed: service.CreateSeed,
  updateSeed: service.UpdateSeed,
  copySeedField: service.CopySeedField,
  spendBackupCode: service.SpendBackupCode,
  recordSeedCheck: service.RecordSeedCheck,
  checkSeedPhrase,
  async seedWordlist() {
    return listed(await service.SeedWordlist());
  },
  async listGroups() {
    return (await service.ListGroups()) ?? [];
  },
  async generatorHistory() {
    return listed(await service.GeneratorHistory()).map(
      generatorHistoryEntryOf,
    );
  },
  recordGeneratedPassword: service.RecordGeneratedPassword,
  clearGeneratorHistory: service.ClearGeneratorHistory,
  copyGeneratedPassword: service.CopyGeneratedPassword,
  createGroup: service.CreateGroup,
  renameGroup: service.RenameGroup,
  deleteGroup: service.DeleteGroup,
  setItemGroups: service.SetItemGroups,
  defaultGroup: service.DefaultGroup,
  setDefaultGroup: service.SetDefaultGroup,
  trashItem: service.TrashItem,
  restoreItem: service.RestoreItem,
  deleteItem: service.DeleteItem,
  listTrash,
  emptyTrash: service.EmptyTrash,
  setTrashRetention: service.SetTrashRetention,
  mergeCredentials: service.MergeCredentials,
  duplicateItem: service.DuplicateItem,
  setPinned: service.SetPinned,
  copyCredentialField: service.CopyCredentialField,
  copyIdentityField: service.CopyIdentityField,
  exportStatus,
  openWebsite: service.OpenWebsite,
  supportDetails: service.SupportDetails,
  thirdPartyNotices: service.ThirdPartyNotices,
  exportEncryptedCopy: service.ExportEncryptedCopy,
  autoBackup,
  setAutoBackup: service.SetAutoBackup,
  chooseBackupFolder: service.ChooseBackupFolder,
  chooseImportFile,
  unlockImportFile,
  importItems,
  cancelImport: service.CancelImport,
  trashImportFile: service.TrashImportFile,
  revealImportFile: service.RevealImportFile,
  extensionLinks,
  beginExtensionLink: service.BeginExtensionLink,
  awaitExtensionLink,
  cancelExtensionLink: service.CancelExtensionLink,
  copyExtensionLinkKey: service.CopyExtensionLinkKey,
  unlinkExtension: service.UnlinkExtension,
  renameExtension: service.RenameExtension,
  signInStyle,
  setSignInStyle: service.SetSignInStyle,
  confirmExtensionFills: service.ConfirmExtensionFills,
  setConfirmExtensionFills: service.SetConfirmExtensionFills,
  awaitConfirmation,
  confirmWithPin: service.ConfirmWithPIN,
  unlockFromConfirmation: service.UnlockFromConfirmation,
  recoverFromConfirmation: service.RecoverFromConfirmation,
  declineConfirmation: service.DeclineConfirmation,
  fitConfirmation: service.FitConfirmation,
  awaitVaultChange,
  lock: service.Lock,
};
