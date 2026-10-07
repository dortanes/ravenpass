import type { SignInStyle } from "./extensions/sign-in-style.ts";
import type { Appearance } from "./host/appearance.ts";

export interface VaultState {
  phase: "storage" | "setup" | "locked" | "ready";
}

/** A file on the device, or a file a document provider such as a cloud drive holds. */
export type StorageKind = "local-file" | "document";

export type StorageReason = "" | "unreachable" | "selection-unusable";

/** `path` is an opaque identifier never shown; `place` is worded by the host in the language in use. */
export interface Vault {
  kind: StorageKind;
  path: string;
  name: string;
  place: string;
  current: boolean;
}

/** `chosen` lists the kinds whose file the owner picks through the host's picker. */
export interface StorageStatus {
  kinds: StorageKind[];
  chosen: StorageKind[];
  kind: StorageKind | "";
  path: string;
  name: string;
  place: string;
  vaults: Vault[];
  available: boolean;
  restricted: boolean;
  reason: StorageReason;
  /** Other devices may open the file. */
  shared: boolean;
  /** The file this device opened the vault from is no longer there. */
  missing: boolean;
}

/** `deleted` is false when the owner declined the device's check, which leaves the vault as it was. */
export interface VaultDeletion {
  deleted: boolean;
  path: string;
  deviceRecordsRemoved: boolean;
  sharedIdentity: boolean;
}

export interface UnlockMethods {
  biometryAvailable: boolean;
  biometryEnabled: boolean;
  pinSet: boolean;
  pinAttemptsLeft: number;
  pinMinLength: number;
  pinMaxLength: number;
}

/** `pin` is empty when no PIN is chosen. */
export interface UnlockChoice {
  biometry: boolean;
  pin: string;
}

export interface LanguageSettings {
  languages: string[];
  language: string;
  chosen: boolean;
}

/** `seconds` and `offered` are delays in seconds. */
export interface ClipboardClearing {
  enabled: boolean;
  seconds: number;
  offered: number[];
}

/** `seconds` and `offered` are delays in seconds without input. */
export interface AutoLock {
  enabled: boolean;
  seconds: number;
  offered: number[];
}

/** Sizes in percent. */
export interface InterfaceSize {
  percent: number;
  offered: number[];
}

export interface AppearanceSetting {
  appearance: Appearance;
  offered: Appearance[];
}

export interface DockIcon {
  hideWithWindow: boolean;
}

export interface SignInStyleSetting {
  style: SignInStyle;
  offered: SignInStyle[];
}

export interface SiteIcons {
  enabled: boolean;
}

/** Whether the system keeps the vault's accounts to suggest them in its own AutoFill. */
export interface IdentityListSetting {
  enabled: boolean;
}

export interface SystemAutofillStatus {
  autofill: boolean;
  passkeys: boolean;
  /** False where the device takes no passkey providers, as before Android 14. */
  passkeyProviders: boolean;
}

/** Technical facts for a problem report; `osVersion` is empty where the system does not report it. */
export interface SupportDetails {
  platform: string;
  osVersion: string;
}

/** `image` is a base64 PNG, empty for none; `tint` is `#rrggbb`, empty for a grey icon. */
export interface SiteIcon {
  image: string;
  tint: string;
}

/** A canceled picker reports no change; `previous` is null after an action that moves no vault. */
export interface StorageChange {
  changed: boolean;
  path: string;
  previous: Vault | null;
  previousRemoved: boolean;
}

export interface RecoveryPreview {
  mayLoseNewerCredentials: boolean;
  /** True when the file is sealed under a recovery key this device saw replaced. */
  keyReplaced: boolean;
  /** True when the device holds no way in that still opens the vault. */
  needsWayIn: boolean;
}

export interface CredentialInput {
  label: string;
  /** The first is shown in the header. */
  websites: string[];
  login: string;
  email: string;
  password: string;
  notes: string;
  /** Empty when there is no second factor. */
  totp: string;
  /** Only autofill links apps, so a save can only remove them. */
  apps: LinkedApp[];
  /** Short words that tell the item apart from others like it, such as two accounts on one site. */
  tags: string[];
}

/**
 * `signer` is the lower-case hexadecimal SHA-256 digest of one signing certificate; `name` is what the app installed on
 * this device shows, empty where the device has no such app.
 */
export interface LinkedApp {
  package: string;
  signer: string;
  name: string;
}

export interface CredentialSummary {
  id: string;
  label: string;
  login: string;
  pinned: boolean;
  lastUsedAt: number;
  groups: string[];
  /** The normalized website host its icon is kept under; empty without a website. */
  site: string;
  /** The normalized host of every website in order, `site` first. */
  sites: string[];
  email: string;
  oneTimeCode: boolean;
  passkeys: number;
  tags: string[];
}

export interface Group {
  id: string;
  name: string;
}

/** Characters per value; `websites` is the most websites a credential may carry. */
export interface CredentialLimits {
  label: number;
  website: number;
  websites: number;
  login: number;
  email: number;
  password: number;
  notes: number;
  totp: number;
}

/** `id` is the credential ID in unpadded base64url; `createdAt` is in Unix milliseconds. */
export interface PasskeyView {
  id: string;
  site: string;
  account: string;
  displayName: string;
  createdAt: number;
}

export interface Credential extends CredentialInput {
  id: string;
  groups: string[];
  site: string;
  passkeys: PasskeyView[];
}

/** `website:<index>` copies the website at that position as saved. */
export type CredentialField =
  | "login"
  | "email"
  | "password"
  | "totp"
  | "notes"
  | `website:${number}`;

export type DocumentType =
  | "passport"
  | "drivers-license"
  | "id-card"
  | "tax-number"
  | "other";

/** An identity address keeps its `id` across edits; a new or card-owned address has an empty one. */
export interface Address {
  id: string;
  label: string;
  street: string;
  city: string;
  region: string;
  postalCode: string;
  country: string;
}

/** Dates are empty or `YYYY-MM-DD`. */
export interface IdentityDocument {
  type: DocumentType;
  label: string;
  number: string;
  issuer: string;
  issuedOn: string;
  expiresOn: string;
  /** On save a stored id keeps its scan, `new:<token>` adds a staged one, and an omitted scan is deleted. */
  scans: string[];
}

export type ScanMediaType = "image/jpeg" | "application/pdf";

/** `thumbnail` is a base64 JPEG, empty for a PDF. */
export interface ScanSummary {
  id: string;
  name: string;
  mediaType: ScanMediaType;
  thumbnail: string;
}

/** `chosen` is false when the dialog closed without a choice. */
export interface ScanDraft {
  chosen: boolean;
  token: string;
  name: string;
  mediaType: ScanMediaType;
  thumbnail: string;
}

/** `birthday` is empty or `YYYY-MM-DD`. */
export interface IdentityInput {
  label: string;
  fullName: string;
  birthday: string;
  emails: string[];
  phones: string[];
  addresses: Address[];
  documents: IdentityDocument[];
  notes: string;
  /** Base64; on save empty removes the photo and a PNG from `cropIdentityPhoto` replaces it. */
  photo: string;
  /** Short words that tell the item apart from others like it, such as two accounts on one site. */
  tags: string[];
}

/** `expiresOn` is the earliest document expiry, or empty. */
export interface IdentitySummary {
  id: string;
  label: string;
  email: string;
  pinned: boolean;
  lastUsedAt: number;
  groups: string[];
  expiresOn: string;
  /** A base64 JPEG, empty without a photo. */
  thumbnail: string;
  tags: string[];
}

/** `chosen` is false when the dialog closed without a choice. */
export interface PhotoDraft {
  chosen: boolean;
  /** An upright base64 JPEG; `cropIdentityPhoto` takes coordinates in its pixels. */
  preview: string;
  width: number;
  height: number;
}

export interface Identity extends IdentityInput {
  id: string;
  groups: string[];
  /** Every scan the identity's documents hold. */
  attachments: ScanSummary[];
}

/** Characters per value; `emails`, `phones`, `addresses` and `documents` are the most entries. */
export interface IdentityLimits {
  label: number;
  fullName: number;
  email: number;
  phone: number;
  addressLabel: number;
  street: number;
  city: number;
  region: number;
  postalCode: number;
  country: number;
  documentLabel: number;
  documentNumber: number;
  issuer: number;
  notes: number;
  emails: number;
  phones: number;
  addresses: number;
  documents: number;
}

export type AddressPart =
  | "street"
  | "city"
  | "region"
  | "postalCode"
  | "country";

/** `index` is zero-based; a whole address copies as lines and a document copies its number. */
/** `birthday` copies the stored `YYYY-MM-DD` date. */
export type IdentityField =
  | { readonly kind: "fullName" | "birthday" | "notes" }
  | { readonly kind: "email" | "phone" | "document"; readonly index: number }
  | {
      readonly kind: "address";
      readonly index: number;
      readonly part?: AddressPart;
    };

/** A network as `credit-card-type` names it, or empty for none. */
export type CardNetwork =
  | ""
  | "visa"
  | "mastercard"
  | "american-express"
  | "discover"
  | "diners-club"
  | "jcb"
  | "unionpay"
  | "maestro"
  | "mir"
  | "elo"
  | "hiper"
  | "hipercard"
  | "troy"
  | "verve"
  | "naranja";

/** Number, security code and PIN hold digits only; `expiry` is empty or `YYYY-MM`; `color` empty or `#rrggbb`. */
export interface CardInput {
  label: string;
  holder: string;
  number: string;
  expiry: string;
  securityCode: string;
  pin: string;
  network: CardNetwork;
  bankName: string;
  /** As the user typed it. */
  bankSite: string;
  color: string;
  /** Never set together with `billingLink`. */
  billing: Address | null;
  billingLink: AddressLink | null;
  notes: string;
  /** Short words that tell the item apart from others like it, such as two accounts on one site. */
  tags: string[];
}

export interface AddressLink {
  identityId: string;
  addressId: string;
}

export interface IdentityAddresses {
  identityId: string;
  label: string;
  addresses: Address[];
}

/** The linked identity address as it reads now. */
export interface LinkedAddress {
  identityLabel: string;
  address: Address;
}

/** `expiresOn` is the last day of the expiry month, `YYYY-MM-DD`, or empty. */
export interface CardSummary {
  id: string;
  label: string;
  bankName: string;
  lastFour: string;
  network: CardNetwork;
  color: string;
  /** The normalized bank host its icon is kept under. */
  site: string;
  pinned: boolean;
  lastUsedAt: number;
  groups: string[];
  expiresOn: string;
  tags: string[];
}

export interface Card extends CardInput {
  id: string;
  groups: string[];
  site: string;
  /** Null without a link or once the linked address is gone. */
  linked: LinkedAddress | null;
}

/** Characters per text; each `Min`/`Max` pair bounds a digit count. */
export interface CardLimits {
  label: number;
  holder: number;
  numberMin: number;
  numberMax: number;
  securityCodeMin: number;
  securityCodeMax: number;
  pinMin: number;
  pinMax: number;
  bankName: number;
  bankSite: number;
  street: number;
  city: number;
  region: number;
  postalCode: number;
  country: number;
  notes: number;
}

/** `expiry` copies as `MM/YY`; `billing` copies the own or linked address as lines. */
export type CardField =
  | "number"
  | "holder"
  | "expiry"
  | "securityCode"
  | "pin"
  | "bankName"
  | "bankSite"
  | "notes"
  | "billing"
  | `billing:${AddressPart}`;

/** Whether Ravenpass fills a card's bank name and colour from the bank's site. */
export interface BankDetails {
  enabled: boolean;
}

/** Each value is empty when the site did not say. */
export interface BankLookup {
  name: string;
  color: string;
}

/** What a credential's website suggests; both are empty for an address that names no web site. */
export interface SiteLookup {
  name: string;
  /** The address cut to its domain, as imports keep it; an http address keeps its scheme and port. */
  website: string;
}

/** `tag` is characters per tag; `tags` is the most tags an item carries. */
export interface TagLimits {
  tag: number;
  tags: number;
}

export interface NoteInput {
  label: string;
  body: string;
  hidden: boolean;
  /** Short words that tell the item apart from others like it, such as two accounts on one site. */
  tags: string[];
}

/** `preview` is the first line holding text, empty for a hidden note. */
export interface NoteSummary {
  id: string;
  label: string;
  preview: string;
  hidden: boolean;
  pinned: boolean;
  lastUsedAt: number;
  groups: string[];
  tags: string[];
}

export interface Note extends NoteInput {
  id: string;
  groups: string[];
}

/** In characters. */
export interface NoteLimits {
  label: number;
  body: number;
}

/** A recovery phrase, a private key, or a set of backup codes. */
export type SeedFormat = "phrase" | "key" | "codes";

export interface SeedAddress {
  label: string;
  value: string;
}

export interface BackupCode {
  value: string;
  used: boolean;
}

/** Only the fields of `format` hold values. */
export interface SeedInput {
  label: string;
  format: SeedFormat;
  words: string[];
  passphrase: string;
  path: string;
  key: string;
  codes: BackupCode[];
  wallet: string;
  addresses: SeedAddress[];
  notes: string;
  /** Short words that tell the item apart from others like it, such as two accounts on one site. */
  tags: string[];
}

/** `total` is a phrase's word count, a code set's code count, or 0 for a key. */
export interface SeedSummary {
  id: string;
  label: string;
  format: SeedFormat;
  wallet: string;
  total: number;
  used: number;
  pinned: boolean;
  lastUsedAt: number;
  groups: string[];
  tags: string[];
}

/** A BIP-39 checksum result: `unknown` for a phrase outside BIP-39, empty for another format. */
export type SeedChecksum = "valid" | "invalid" | "unknown" | "";

/** `checkedOn` is empty or the `YYYY-MM-DD` day the user last confirmed their copy. */
export interface Seed extends SeedInput {
  id: string;
  groups: string[];
  checkedOn: string;
  checksum: SeedChecksum;
}

/** `unknownWords` holds the zero-based positions of words outside the BIP-39 list. */
export interface PhraseCheck {
  checksum: Exclude<SeedChecksum, "">;
  unknownWords: number[];
}

/** Characters per value; `words`, `codes` and `addresses` are the most entries. */
export interface SeedLimits {
  label: number;
  words: number;
  word: number;
  passphrase: number;
  path: number;
  key: number;
  codes: number;
  code: number;
  wallet: number;
  addresses: number;
  addressLabel: number;
  address: number;
  notes: number;
}

/** An address `index` is zero-based. */
export type SeedField =
  | { readonly kind: "phrase" | "passphrase" | "path" | "key" | "notes" }
  | { readonly kind: "address"; readonly index: number };

/** `period` is in seconds; `expiresAt` is in Unix milliseconds. */
export interface OneTimeCode {
  code: string;
  digits: number;
  period: number;
  expiresAt: number;
}

/** A pending one-time code setup; its secret stays with the host, `issuer` and `account` may be empty. */
export interface CodeSetup {
  token: string;
  issuer: string;
  account: string;
}

/** How the open vault compares with the file Ravenpass keeps current. */
export type StorageSyncState = "synced" | "writing" | "failed";

export type ExportState = "unknown" | "current" | "stale";

export interface ExportStatus {
  state: ExportState;
}

export type BackupInterval = "daily" | "weekly" | "monthly";

/** `folder` is null until one is chosen; `lastBackupAt` is in Unix milliseconds, 0 for none. */
export interface AutoBackup {
  enabled: boolean;
  interval: BackupInterval;
  intervals: BackupInterval[];
  keep: number;
  keeps: number[];
  folder: { name: string; place: string } | null;
  lastBackupAt: number;
  /** The last attempt could not save a backup. */
  failed: boolean;
}

export type ImportSource = "bitwarden" | "aliasvault";

export type ImportFormat =
  | "json"
  | "encrypted-json"
  | "zip"
  | "encrypted-zip"
  | "csv";

export type ImportKind = "credential" | "card" | "identity" | "note" | "seed";

/** The kind an item had in the source. */
export type ImportOrigin =
  | "login"
  | "alias"
  | "card"
  | "identity"
  | "passport"
  | "drivers-license"
  | "note"
  | "ssh-key"
  | "bank-account";

export type ImportReason = "too-long" | "unnamed" | "unsupported";

export interface ImportConversion {
  from: ImportOrigin;
  count: number;
}

/** Counts include duplicates; `duplicates` counts the items the vault already holds. */
export interface ImportKindPreview {
  kind: ImportKind;
  count: number;
  duplicates: number;
  oneTimeCodes: number;
  converted: ImportConversion[];
}

/** `existing` names match a vault group; `dropped` names cannot become groups. */
export interface ImportGroups {
  new: number;
  existing: number;
  dropped: number;
}

/** `label` is empty for an unnamed item; `origin` is null for a kind the reader does not know. */
export interface ImportSkip {
  label: string;
  origin: ImportOrigin | null;
  reason: ImportReason;
}

export interface ImportPreview {
  format: ImportFormat;
  items: number;
  /** Kinds with no item are left out. */
  kinds: ImportKindPreview[];
  /** With duplicates kept. */
  groups: ImportGroups;
  groupsWithoutDuplicates: ImportGroups;
  /** Issuer identification numbers of the card numbers that name no network. */
  cardIssuers: string[];
  skipped: ImportSkip[];
  attachments: number;
  /** Passkeys the import leaves behind. */
  passkeys: number;
  /** Duplicates included. */
  importedPasskeys: number;
}

/** `chosen` is false when the panel closed; `preview` is null until a locked file is unlocked. */
export interface ImportChoice {
  chosen: boolean;
  name: string;
  locked: boolean;
  preview: ImportPreview | null;
}

export interface ImportOptions {
  /** Folders become groups. */
  groups: boolean;
  skipDuplicates: boolean;
}

/** Keyed by issuer; the cards of an issuer left out stay without a network. */
export type ImportCardNetworks = Readonly<
  Record<string, Exclude<CardNetwork, "">>
>;

export interface ImportKindTotal {
  kind: ImportKind;
  count: number;
}

export interface ImportResult {
  added: number;
  kinds: ImportKindTotal[];
  groups: number;
  format: ImportFormat;
  /** False for a file picked as a temporary copy, which Ravenpass cannot trash or reveal. */
  located: boolean;
}

/** `linkedAt` is in Unix milliseconds. */
export interface LinkedExtension {
  id: string;
  name: string;
  linkedAt: number;
}

/** `reachable` is false while Ravenpass cannot listen on the port the extensions know. */
export interface ExtensionLinks {
  extensions: LinkedExtension[];
  reachable: boolean;
}

/** `expiresAt` is in Unix milliseconds. */
export interface ExtensionLinkOffer {
  key: string;
  expiresAt: number;
}

export type VerificationReason =
  | "share"
  | "save-passkey"
  | "sign-in"
  | "fill"
  | "change-unlock";

/** Fields the verification `reason` does not name are empty. */
export type Confirmation =
  | {
      id: string;
      kind: "verify";
      reason: VerificationReason;
      file: string;
      identity: string;
      site: string;
      account: string;
    }
  | { id: string; kind: "unlock"; requester: UnlockRequester };

/** What asks to unlock the vault: a linked browser extension or the system AutoFill. */
export type UnlockRequester = "extension" | "autofill";

/** The features the host offers beyond the vault; the interface shows only those. */
export interface Capabilities {
  /** Browser extensions can link to this host. */
  extensions: boolean;
  shortcuts: boolean;
  /** The host has a Dock icon it can hide with its window. */
  dockIcon: boolean;
  /** The owner can choose and move the vault file's location. */
  storageLocations: boolean;
  /** The host can save a file where the owner chooses. */
  saveFiles: boolean;
  /** The locked screen asks for the device's own unlock as the app comes into view. */
  unlockOnShow: boolean;
  /** The owner sets the interface size in settings. */
  interfaceSize: boolean;
  /** The host's windows follow the appearance chosen in settings. */
  appearance: boolean;
  /** A photo picker apart from the file picker, which PDF scans still use. */
  photoPicker: boolean;
  /** The owner can let the system keep the vault's accounts for its own AutoFill. */
  identityList: boolean;
  /** The host puts a scan on the clipboard. */
  copyScans: boolean;
  /** The automatic lock counts from the moment the app is hidden and can lock at once. */
  lockWhenHidden: boolean;
  /** The host keeps its windows out of screenshots unless the owner allows the main one. */
  screenshots: boolean;
  /** Settings open the system screen that makes Ravenpass the autofill service and passkey provider. */
  systemAutofill: boolean;
  /** The host saves backups of the open vault into a folder the owner chooses. */
  autoBackups: boolean;
  /** The host prints through the system print dialog. */
  print: boolean;
  /** The editor reads a one-time code setup from a QR code in a picture file or on the clipboard. */
  qrCodes: boolean;
}

export interface VaultApi {
  capabilities(): Promise<Capabilities>;
  getState(): Promise<VaultState>;
  getStorage(): Promise<StorageStatus>;
  getLanguage(): Promise<LanguageSettings>;
  setLanguage(language: string): Promise<void>;
  /** Calls `onChange` after a language is chosen in any window; the returned function stops it. */
  watchLanguage(onChange: () => void): () => void;
  clipboardClearing(): Promise<ClipboardClearing>;
  setClipboardClearing(enabled: boolean, seconds: number): Promise<void>;
  autoLock(): Promise<AutoLock>;
  setAutoLock(enabled: boolean, seconds: number): Promise<void>;
  dockIcon(): Promise<DockIcon>;
  setDockIcon(hideWithWindow: boolean): Promise<void>;
  interfaceSize(): Promise<InterfaceSize>;
  /** `percent` is one of `InterfaceSize.offered`; the interface applies it. */
  setInterfaceSize(percent: number): Promise<void>;
  appearance(): Promise<AppearanceSetting>;
  /** The host's windows and the interface apply it. */
  setAppearance(appearance: Appearance): Promise<void>;
  siteIcons(): Promise<SiteIcons>;
  setSiteIcons(enabled: boolean): Promise<void>;
  /** The image is empty when the site has none or icons are off. */
  siteIcon(site: string): Promise<SiteIcon>;
  identityList(): Promise<IdentityListSetting>;
  /** Enabling hands the open vault's accounts to the system; disabling removes them from it. */
  setIdentityList(enabled: boolean): Promise<void>;
  screenshotsAllowed(): Promise<boolean>;
  /** Applies to the main window at once. */
  allowScreenshots(allowed: boolean): Promise<void>;
  systemAutofill(): Promise<SystemAutofillStatus>;
  /** Shows the system autofill screen and resolves once the owner leaves it. */
  chooseSystemAutofill(): Promise<void>;
  /** Hotkeys by action; an action missing here keeps its default. */
  shortcuts(): Promise<Record<string, string>>;
  /** An empty hotkey restores the default. */
  setShortcut(action: string, hotkey: string): Promise<void>;
  /** Uses the host's picker for a kind in `StorageStatus.chosen`, else a free file where the host keeps vaults. */
  selectStorageLocation(kind: StorageKind): Promise<StorageChange>;
  /** Chooses where the open vault goes as `selectStorageLocation` does, for `moveStorageLocation`; false where the owner chose nothing. */
  chooseStorageMove(kind: StorageKind): Promise<boolean>;
  /** Moves the open vault to the location `chooseStorageMove` chose last. */
  moveStorageLocation(): Promise<StorageChange>;
  retryStorage(): Promise<void>;
  switchVault(path: string): Promise<void>;
  /** Whether any location the device knows holds a vault. */
  knowsVault(): Promise<boolean>;
  /** Starts setting up a vault at a free file beside the current one, without asking. */
  createVault(): Promise<StorageChange>;
  /** Returns to the vault open before `createVault`. */
  cancelVaultCreation(): Promise<void>;
  openVault(): Promise<StorageChange>;
  forgetVault(path: string): Promise<void>;
  /** Deletes the vault file once the owner passes the device's check. */
  deleteVault(path: string): Promise<VaultDeletion>;
  beginCreation(): Promise<string>;
  /** `choice` must hold at least one way in. */
  confirmCreation(phrase: string, choice: UnlockChoice): Promise<void>;
  /** Copies the recovery key being set up, of a new vault or a new key, while `phrase` still matches it. */
  copyRecoveryKey(phrase: string): Promise<void>;
  /** Saves the recovery key being set up to a file, as `copyRecoveryKey` does. */
  saveRecoveryKey(phrase: string): Promise<void>;
  /** Shows the system print dialog for the recovery key being set up, as `copyRecoveryKey` does; resolves once it closes. */
  printRecoveryKey(phrase: string): Promise<void>;
  /**
   * Returns a new recovery key for the open vault; nothing changes until it is confirmed. `pin` is the vault's PIN
   * where `UnlockMethods.pinSet`, otherwise empty. `current` is the vault's current recovery key where the device has
   * neither a PIN nor device authentication, otherwise empty.
   */
  beginRecoveryPhraseChange(pin: string, current: string): Promise<string>;
  /** Locks the vault with the new key once `phrase` matches it; other devices then open it with the new key. */
  confirmRecoveryPhraseChange(phrase: string): Promise<void>;
  cancelRecoveryPhraseChange(): Promise<void>;
  unlock(): Promise<void>;
  /** False after the owner locked the vault with the app in view, until the app leaves view. */
  promptsUnlock(): Promise<boolean>;
  unlockMethods(): Promise<UnlockMethods>;
  unlockWithPin(pin: string): Promise<void>;
  /** After `vault-diverged`: the version another device saved replaces the device's latest changes. */
  adoptChangedVault(): Promise<void>;
  /**
   * Confirms the owner first: by device authentication where the vault opens with it, failing with `owner-unverified`
   * on a decline; else by `current`, the vault's PIN, counted as the locked screen counts it; else by `current` as the
   * vault's recovery key.
   */
  setPin(pin: string, current: string): Promise<void>;
  /** Confirms the owner first, as `setPin` does. */
  removePin(current: string): Promise<void>;
  /** Confirms the owner first, as `setPin` does, unless it is already as asked. */
  setBiometryUnlock(enabled: boolean, current: string): Promise<void>;
  beginRecovery(phrase: string): Promise<RecoveryPreview>;
  /**
   * `accepted` confirms every warning the preview gave. An empty `choice` keeps the device's ways in; a filled one
   * replaces them.
   */
  confirmRecovery(accepted: boolean, choice: UnlockChoice): Promise<void>;
  listCredentials(): Promise<CredentialSummary[]>;
  credentialLimits(): Promise<CredentialLimits>;
  readCredential(id: string): Promise<Credential>;
  generateOneTimeCode(setup: string): Promise<OneTimeCode>;
  /** Null when no setup waits; a waiting setup survives a lock until it expires. */
  getCodeSetup(): Promise<CodeSetup | null>;
  /** Replaces the credential's one-time code; fails with `code-setup-expired` once the setup no longer waits. */
  addCodeSetup(token: string, credentialId: string): Promise<void>;
  /** Names the credential by the setup's issuer, else its account; resolves with the new id. */
  createCredentialFromCodeSetup(token: string): Promise<string>;
  /** Keeps a newer setup than the one `token` names. */
  dismissCodeSetup(token: string): Promise<void>;
  clearSelection(): Promise<void>;
  createCredential(input: CredentialInput, groups: string[]): Promise<string>;
  /** Removes the passkeys `removedPasskeys` names by `PasskeyView.id` in the same write. */
  updateCredential(
    id: string,
    input: CredentialInput,
    groups: string[],
    removedPasskeys: string[],
  ): Promise<void>;
  listIdentities(): Promise<IdentitySummary[]>;
  identityLimits(): Promise<IdentityLimits>;
  readIdentity(id: string): Promise<Identity>;
  createIdentity(input: IdentityInput, groups: string[]): Promise<string>;
  updateIdentity(
    id: string,
    input: IdentityInput,
    groups: string[],
  ): Promise<void>;
  /** Uses the host's photo picker, or its file dialog where it has none. */
  chooseIdentityPhoto(): Promise<PhotoDraft>;
  /** Crops a square, in preview pixels, from the staged picture; resolves with a base64 PNG. */
  cropIdentityPhoto(x: number, y: number, size: number): Promise<string>;
  discardIdentityPhoto(): Promise<void>;
  /** Stages a picture or PDF for the editor's next save. */
  chooseScan(): Promise<ScanDraft>;
  /** A host without `photoPicker` asks with its file dialog. */
  chooseScanPhoto(): Promise<ScanDraft>;
  discardScans(): Promise<void>;
  copyScan(id: string): Promise<void>;
  /** Writes an unencrypted copy where the user picks; `saved` is false on cancel. */
  saveScan(id: string): Promise<{ saved: boolean }>;
  listCards(): Promise<CardSummary[]>;
  cardLimits(): Promise<CardLimits>;
  readCard(id: string): Promise<Card>;
  createCard(input: CardInput, groups: string[]): Promise<string>;
  updateCard(id: string, input: CardInput, groups: string[]): Promise<void>;
  copyCardField(id: string, field: CardField): Promise<void>;
  /** Only identities that have an address. */
  listIdentityAddresses(): Promise<IdentityAddresses[]>;
  bankDetails(): Promise<BankDetails>;
  setBankDetails(enabled: boolean): Promise<void>;
  /** Both values are empty while `BankDetails.enabled` is off. */
  lookupBank(site: string): Promise<BankLookup>;
  /** The site's declared name where website icons load, else its readable domain. */
  lookupSite(website: string): Promise<SiteLookup>;
  /**
   * The setup a QR code holds in a picture the owner chooses or on the clipboard; empty when the owner cancels the
   * file choice. Fails with `qr-code-missing`, `qr-code-not-setup` or `qr-code-ambiguous`.
   */
  readCodeSetup(source: "file" | "clipboard"): Promise<string>;
  tagLimits(): Promise<TagLimits>;
  /**
   * Confirms the owner before a hidden note or a seed's secret shows: by device authentication where the vault opens
   * with it, else by `pin`, failing with `pin-wrong` or `owner-unverified`. Passes where the vault offers neither.
   */
  confirmReveal(item: string, pin: string): Promise<void>;
  listNotes(): Promise<NoteSummary[]>;
  noteLimits(): Promise<NoteLimits>;
  readNote(id: string): Promise<Note>;
  createNote(input: NoteInput, groups: string[]): Promise<string>;
  updateNote(id: string, input: NoteInput, groups: string[]): Promise<void>;
  copyNote(id: string): Promise<void>;
  listSeeds(): Promise<SeedSummary[]>;
  seedLimits(): Promise<SeedLimits>;
  readSeed(id: string): Promise<Seed>;
  createSeed(input: SeedInput, groups: string[]): Promise<string>;
  updateSeed(id: string, input: SeedInput, groups: string[]): Promise<void>;
  copySeedField(id: string, field: SeedField): Promise<void>;
  /** Copies the unused code at zero-based `index`, then records it as used. */
  spendBackupCode(id: string, index: number): Promise<void>;
  /** Records today as the day the user last confirmed the phrase's copy. */
  recordSeedCheck(id: string): Promise<void>;
  /** Needs no open vault. */
  checkSeedPhrase(words: string[]): Promise<PhraseCheck>;
  /** The English BIP-39 word list; needs no open vault. */
  seedWordlist(): Promise<string[]>;
  listGroups(): Promise<Group[]>;
  createGroup(name: string): Promise<string>;
  renameGroup(id: string, name: string): Promise<void>;
  deleteGroup(id: string): Promise<void>;
  /** Acts on an item of any kind. */
  setItemGroups(id: string, groups: string[]): Promise<void>;
  /** Empty when new items join no group. */
  defaultGroup(): Promise<string>;
  setDefaultGroup(id: string): Promise<void>;
  /** Acts on an item of any kind. */
  deleteItem(id: string): Promise<void>;
  /**
   * Saves a copy of an item of any kind, named after it, and resolves with the copy's id. A password's passkeys stay
   * with the original; an identity's scans are copied.
   */
  duplicateItem(id: string): Promise<string>;
  /** Acts on an item of any kind. */
  setPinned(id: string, pinned: boolean): Promise<void>;
  copyCredentialField(id: string, field: CredentialField): Promise<void>;
  copyIdentityField(id: string, field: IdentityField): Promise<void>;
  exportStatus(): Promise<ExportStatus>;
  openWebsite(address: string): Promise<void>;
  supportDetails(): Promise<SupportDetails>;
  /** Empty for a build that ships no notices. */
  thirdPartyNotices(): Promise<string>;
  exportEncryptedCopy(): Promise<void>;
  autoBackup(): Promise<AutoBackup>;
  /** Resolves after any backup it starts; enabling fails with `backup-folder-missing` until a folder is chosen. */
  setAutoBackup(
    enabled: boolean,
    interval: BackupInterval,
    keep: number,
  ): Promise<void>;
  /** Resolves false on cancel, else true after any backup it starts. */
  chooseBackupFolder(): Promise<boolean>;
  /** Stages the chosen export file, read only. */
  chooseImportFile(source: ImportSource): Promise<ImportChoice>;
  /** Opens the staged file with the password set when exporting it. */
  unlockImportFile(password: string): Promise<ImportPreview>;
  /** Adds the staged items in one save, planned again against the vault as it is now. */
  importItems(
    options: ImportOptions,
    cardNetworks: ImportCardNetworks,
  ): Promise<ImportResult>;
  cancelImport(): Promise<void>;
  /** Moves the imported file to the Trash. */
  trashImportFile(): Promise<void>;
  /** Shows the imported file in the system file manager. */
  revealImportFile(): Promise<void>;
  extensionLinks(): Promise<ExtensionLinks>;
  /** Replaces the waiting connection key. */
  beginExtensionLink(): Promise<ExtensionLinkOffer>;
  /** Fails with `link-expired` or `link-canceled`; aborting the signal cancels the waiting key. */
  awaitExtensionLink(signal: AbortSignal): Promise<LinkedExtension>;
  cancelExtensionLink(): Promise<void>;
  /** The clipboard clears as it does after other copies. */
  copyExtensionLinkKey(): Promise<void>;
  unlinkExtension(id: string): Promise<void>;
  /** Trims `name`; fails with `extension-name-invalid` for an empty or too long one. */
  renameExtension(id: string, name: string): Promise<void>;
  signInStyle(): Promise<SignInStyleSetting>;
  /** Applies from the next request a linked extension makes. */
  setSignInStyle(style: SignInStyle): Promise<void>;
  /** Whether each password or code a linked extension fills waits for the owner to confirm it. */
  confirmExtensionFills(): Promise<boolean>;
  /** Applies from the next request a linked extension makes. */
  setConfirmExtensionFills(confirm: boolean): Promise<void>;
  /** Resolves with the oldest waiting request other than `shown`, or null once none waits and `shown` is set. */
  awaitConfirmation(
    shown: string,
    signal: AbortSignal,
  ): Promise<Confirmation | null>;
  /** Fails with `pin-wrong` (the request keeps waiting), `pin-removed`, or `confirmation-ended`. */
  confirmWithPin(id: string, pin: string): Promise<void>;
  /** Opens the vault with the device's authentication. */
  unlockFromConfirmation(id: string): Promise<void>;
  /** Ends an unlock request and brings the main window forward for recovery. */
  recoverFromConfirmation(id: string): Promise<void>;
  /** Fails with `confirmation-ended` once the request no longer waits. */
  declineConfirmation(id: string): Promise<void>;
  /** `height` is in CSS pixels. */
  fitConfirmation(height: number): Promise<void>;
  /** Resolves with the count of changes that bypassed the interface since start, once it exceeds `seen`. */
  awaitVaultChange(seen: number, signal: AbortSignal): Promise<number>;
  lock(): Promise<void>;
}

/** Sources whose language can change elsewhere offer `watchLanguage`. */
export type LanguageSource = Pick<VaultApi, "getLanguage"> &
  Partial<Pick<VaultApi, "setLanguage" | "watchLanguage">>;

export type ConfirmationHost = Pick<
  VaultApi,
  | "awaitConfirmation"
  | "confirmWithPin"
  | "unlockFromConfirmation"
  | "recoverFromConfirmation"
  | "declineConfirmation"
  | "fitConfirmation"
  | "unlockMethods"
>;

export type ExtensionLinking = Pick<
  VaultApi,
  | "extensionLinks"
  | "beginExtensionLink"
  | "awaitExtensionLink"
  | "cancelExtensionLink"
  | "copyExtensionLinkKey"
  | "unlinkExtension"
  | "renameExtension"
  | "signInStyle"
  | "setSignInStyle"
  | "confirmExtensionFills"
  | "setConfirmExtensionFills"
>;

export type AutofillSettings = Pick<
  VaultApi,
  "identityList" | "setIdentityList" | "systemAutofill" | "chooseSystemAutofill"
>;

export type ScreenshotSettings = Pick<
  VaultApi,
  "screenshotsAllowed" | "allowScreenshots"
>;

export type AboutApi = Pick<VaultApi, "supportDetails" | "thirdPartyNotices">;

export type BackupSettings = Pick<
  VaultApi,
  "autoBackup" | "setAutoBackup" | "chooseBackupFolder"
>;
