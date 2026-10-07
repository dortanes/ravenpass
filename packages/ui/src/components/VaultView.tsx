import {
  formatForDisplay,
  type Hotkey,
  useHotkey,
} from "@tanstack/react-hotkeys";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { cn } from "cn";
import { useEffect, useEffectEvent, useMemo, useRef, useState } from "react";
import { toast } from "sonner";
import { CountWatcher } from "../count-watcher.ts";
import { refusedBeforeWriting } from "../failures.ts";
import type { Appearance } from "../host/appearance.ts";
import {
  appearanceQuery,
  interfaceSizeQuery,
  useCapabilities,
} from "../host/capabilities.tsx";
import { useCompactLayout } from "../host/compact.ts";
import type { MessageKey } from "../i18n/messages.ts";
import { useTranslator } from "../i18n/translator.tsx";
import { CrossFade } from "../motion/CrossFade.tsx";
import { useHostSetting } from "../query/host-setting.ts";
import { queryKeys, vaultScope } from "../query/keys.ts";
import { unlockMethodsQuery } from "../query/unlock-methods.ts";
import { type VaultGroups, vaultReads } from "../query/vault-reads.ts";
import { useVaultStorage } from "../query/vault-storage.ts";
import { type ShortcutAction, Shortcuts } from "../shortcuts/shortcuts.ts";
import { vaultLocation, vaultName } from "../storage/location.ts";
import type {
  CodeSetup,
  ExportState,
  ImportCardNetworks,
  ImportOptions,
  ImportResult,
  SiteIcons,
  StorageKind,
  StorageSyncState,
  Vault,
  VaultApi,
  VaultState,
} from "../vault-api.ts";
import { ListPositions } from "../workspace/list-positions.ts";
import {
  everyGroup,
  knownTags,
  tagSearch,
  type WorkspaceSection,
} from "../workspace/sections.ts";
import { SelectionQueue } from "../workspace/selection.ts";
import { SiteIconStore } from "../workspace/site-icons.ts";
import { SettingsView } from "./SettingsView.tsx";
import type { ImportActions } from "./settings/ImportPanel.tsx";
import type { SettingsSection } from "./settings/sections.ts";
import type { UnlockPending } from "./unlock-change.ts";
import { BackRow } from "./workspace/BackRow.tsx";
import { CardsPlace } from "./workspace/CardsPlace.tsx";
import { CodeSetupDialog } from "./workspace/CodeSetupDialog.tsx";
import {
  CommandPalette,
  type PaletteChoice,
} from "./workspace/CommandPalette.tsx";
import { IdentitiesPlace } from "./workspace/IdentitiesPlace.tsx";
import type {
  OpeningPane,
  PlaceHandle,
  PlaceShell,
} from "./workspace/ItemPlace.tsx";
import { NotesPlace } from "./workspace/NotesPlace.tsx";
import { PasswordsPlace } from "./workspace/PasswordsPlace.tsx";
import {
  type ItemPlaceName,
  placeOf,
  type WorkspacePlace,
} from "./workspace/places.ts";
import { SeedsPlace } from "./workspace/SeedsPlace.tsx";
import { SiteIconsProvider } from "./workspace/SiteIcons.tsx";
import { WorkspaceRail } from "./workspace/WorkspaceRail.tsx";
import { WorkspaceStatusBar } from "./workspace/WorkspaceStatusBar.tsx";
import { WorkspaceTabs } from "./workspace/WorkspaceTabs.tsx";
import { WorkspaceToolbar } from "./workspace/WorkspaceToolbar.tsx";

const regionLabels: Record<WorkspacePlace, MessageKey> = {
  passwords: "workspace.region.credentials",
  identities: "workspace.region.identities",
  cards: "workspace.region.cards",
  notes: "workspace.region.notes",
  seeds: "workspace.region.seeds",
  settings: "workspace.region.settings",
};

const searchLabels: Record<
  ItemPlaceName,
  { label: MessageKey; placeholder: MessageKey }
> = {
  passwords: {
    label: "workspace.toolbar.search",
    placeholder: "workspace.toolbar.search.placeholder",
  },
  identities: {
    label: "identity.search.label",
    placeholder: "identity.search.placeholder",
  },
  cards: {
    label: "card.search.label",
    placeholder: "card.search.placeholder",
  },
  notes: {
    label: "note.search.label",
    placeholder: "note.search.placeholder",
  },
  seeds: {
    label: "seed.search.label",
    placeholder: "seed.search.placeholder",
  },
};

const noItems: never[] = [];

/** VaultView is the open vault: the state every place shares, around whichever place is open. */
export function VaultView({
  api,
  onPhase,
}: {
  api: VaultApi;
  /** `chosen` is set where the owner chose another vault to open, which its locked screen then asks to unlock. */
  onPhase: (phase: VaultState["phase"], chosen?: boolean) => void;
}) {
  const { t, failure } = useTranslator();
  const client = useQueryClient();
  const compact = useCompactLayout();
  const offers = useCapabilities();
  const reads = useMemo(() => vaultReads(api), [api]);
  const credentialsRead = useQuery(reads.credentials.options());
  const identitiesRead = useQuery(reads.identities.options());
  const cardsRead = useQuery(reads.cards.options());
  const notesRead = useQuery(reads.notes.options());
  const seedsRead = useQuery(reads.seeds.options());
  const credentials = credentialsRead.data ?? noItems;
  const identities = identitiesRead.data ?? noItems;
  const cards = cardsRead.data ?? noItems;
  const notes = notesRead.data ?? noItems;
  const seeds = seedsRead.data ?? noItems;
  const vaultTags = useMemo(
    () =>
      knownTags([...credentials, ...identities, ...cards, ...notes, ...seeds]),
    [credentials, identities, cards, notes, seeds],
  );
  const groupsRead = useQuery(reads.groups.options());
  const groups = groupsRead.data?.groups ?? noItems;
  const defaultGroup = groupsRead.data?.defaultGroup ?? "";
  const codeSetup = useQuery(reads.codeSetup.options()).data ?? null;
  // An unreadable export record shows as unknown.
  const exportRead = useQuery({
    queryKey: queryKeys.exportStatus,
    queryFn: () => api.exportStatus(),
  });
  const exportState: ExportState = exportRead.isError
    ? "unknown"
    : (exportRead.data?.state ?? "unknown");
  const storageRead = useVaultStorage(
    api,
    "storage-unavailable.errors.unreadable",
  );
  const storage = storageRead.data ?? null;
  const unlockMethods = useQuery(unlockMethodsQuery(api)).data ?? null;
  const interfaceSize =
    useQuery(interfaceSizeQuery(api, offers.interfaceSize)).data ?? null;
  const appearance =
    useQuery(appearanceQuery(api, offers.appearance)).data ?? null;
  const clipboardSetting = useHostSetting({
    key: queryKeys.clipboardClearing,
    read: () => api.clipboardClearing(),
    write: (enabled: boolean, seconds: number) =>
      api.setClipboardClearing(enabled, seconds),
    readFailure: "app.error.setting-read",
    writeFailure: "workspace.error.clipboard",
  });
  const autoLockSetting = useHostSetting({
    key: queryKeys.autoLock,
    read: () => api.autoLock(),
    write: (enabled: boolean, seconds: number) =>
      api.setAutoLock(enabled, seconds),
    readFailure: "app.error.setting-read",
    writeFailure: "workspace.error.auto-lock",
  });
  const dockIconSetting = useHostSetting({
    key: queryKeys.dockIcon,
    read: () => api.dockIcon(),
    write: (hideWithWindow: boolean) => api.setDockIcon(hideWithWindow),
    readFailure: "app.error.setting-read",
    writeFailure: "workspace.error.dock-icon",
    enabled: offers.dockIcon,
  });
  const siteIconsSetting = useHostSetting({
    key: queryKeys.siteIcons,
    read: () => api.siteIcons(),
    write: (enabled: boolean) => api.setSiteIcons(enabled),
    readFailure: "app.error.setting-read",
    writeFailure: "workspace.error.site-icons",
  });
  const bankDetailsSetting = useHostSetting({
    key: queryKeys.bankDetails,
    read: () => api.bankDetails(),
    write: (enabled: boolean) => api.setBankDetails(enabled),
    readFailure: "app.error.setting-read",
    writeFailure: "workspace.error.bank-details",
  });
  // A refused change leaves the previous keys working.
  const shortcutSetting = useHostSetting({
    key: queryKeys.shortcuts,
    read: () => api.shortcuts(),
    write: (action: ShortcutAction, hotkey: Hotkey | null) =>
      api.setShortcut(action, hotkey ?? ""),
    readFailure: "app.error.setting-read",
    writeFailure: "workspace.error.shortcut",
    enabled: offers.shortcuts,
  });
  const interfaceSizeChange = useMutation({
    mutationFn: (percent: number) => api.setInterfaceSize(percent),
    meta: { failure: "workspace.error.interface-size" },
    onSettled: () =>
      client.invalidateQueries({ queryKey: queryKeys.interfaceSize }),
  });
  const appearanceChange = useMutation({
    mutationFn: (next: Appearance) => api.setAppearance(next),
    meta: { failure: "workspace.error.appearance" },
    onSettled: () =>
      client.invalidateQueries({ queryKey: queryKeys.appearance }),
  });
  const unlockChange = useMutation({
    mutationFn: (change: {
      run: () => Promise<void>;
      pending: UnlockPending;
    }) => change.run(),
    meta: { failure: "unlock-methods.errors.save-failed" },
    onSettled: () =>
      client.invalidateQueries({ queryKey: queryKeys.unlockMethods }),
  });
  const clipboard = clipboardSetting.value;
  const siteIcons = siteIconsSetting.value;
  const bankDetails = bankDetailsSetting.value;
  const recordedShortcuts = shortcutSetting.value;
  const shortcuts = useMemo(
    () =>
      recordedShortcuts
        ? Shortcuts.fromRecorded(recordedShortcuts)
        : Shortcuts.defaults,
    [recordedShortcuts],
  );
  const [itemPlace, setItemPlace] = useState<ItemPlaceName>("passwords");
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [settingsSection, setSettingsSection] =
    useState<SettingsSection | null>(null);
  const [paneShown, setPaneShown] = useState(false);
  const [paletteOpen, setPaletteOpen] = useState(false);
  const [recordingShortcut, setRecordingShortcut] = useState(false);
  const [opening, setOpening] = useState<OpeningPane>("empty");
  const [chosenGroup, setGroup] = useState(everyGroup);
  const [section, setSection] = useState<WorkspaceSection>("all");
  const [query, setQuery] = useState("");
  const [working, setWorking] = useState(false);
  const [syncState, setSyncState] = useState<StorageSyncState>("synced");
  const [storageMove, setStorageMove] = useState<"choosing" | "moving" | null>(
    null,
  );
  const [addingCode, setAddingCode] = useState<string | null>(null);
  const selection = useMemo(
    () => new SelectionQueue(() => api.clearSelection()),
    [api],
  );
  const siteIconStore = useMemo(
    () => new SiteIconStore((site) => api.siteIcon(site)),
    [api],
  );
  const [listPositions] = useState(() => new ListPositions());
  const openPlace = useRef<PlaceHandle | null>(null);
  const searchField = useRef<HTMLInputElement | null>(null);
  const place: WorkspacePlace = settingsOpen ? "settings" : itemPlace;
  const group =
    chosenGroup === everyGroup || groups.some((item) => item.id === chosenGroup)
      ? chosenGroup
      : everyGroup;
  const busy =
    working ||
    clipboardSetting.changing ||
    autoLockSetting.changing ||
    dockIconSetting.changing ||
    siteIconsSetting.changing ||
    bankDetailsSetting.changing ||
    shortcutSetting.changing ||
    interfaceSizeChange.isPending ||
    appearanceChange.isPending ||
    storageMove === "choosing";
  const unlockPending = unlockChange.isPending
    ? (unlockChange.variables?.pending ?? "other")
    : null;

  function report(cause: unknown, message: MessageKey) {
    toast.error(failure(cause, message));
  }

  // A refused input never reached the file.
  function settleSync(cause: unknown) {
    setSyncState(refusedBeforeWriting(cause) ? "synced" : "failed");
  }

  useEffect(() => () => siteIconStore.clear(), [siteIconStore]);

  // Extension and autofill saves, other devices' versions and code setups bypass the workspace.
  const showVaultChange = useEffectEvent(() => {
    Promise.all([refreshGroups(), refreshItems()]).catch((cause: unknown) =>
      report(cause, "workspace.error.outside-change"),
    );
    void refreshExportState();
    void client.invalidateQueries({ queryKey: reads.codeSetup.key });
    if (place !== "settings") openPlace.current?.reload();
  });
  useEffect(
    () =>
      new CountWatcher((seen, signal) =>
        api.awaitVaultChange(seen, signal),
      ).watch(() => showVaultChange()),
    [api],
  );

  useHotkey(
    shortcuts.hotkey("palette"),
    () => setPaletteOpen((open) => !open),
    { enabled: offers.shortcuts && !recordingShortcut },
  );
  useHotkey(
    shortcuts.hotkey("search"),
    () => {
      searchField.current?.focus();
      searchField.current?.select();
    },
    { enabled: offers.shortcuts && !recordingShortcut },
  );

  function refreshCredentials() {
    return reads.credentials.refresh(client);
  }

  function refreshIdentities() {
    return reads.identities.refresh(client);
  }

  function refreshCards() {
    return reads.cards.refresh(client);
  }

  function refreshNotes() {
    return reads.notes.refresh(client);
  }

  function refreshSeeds() {
    return reads.seeds.refresh(client);
  }

  async function refreshItems() {
    await Promise.all([
      refreshCredentials(),
      refreshIdentities(),
      refreshCards(),
      refreshNotes(),
      refreshSeeds(),
    ]);
  }

  function refreshGroups() {
    return reads.groups.refresh(client);
  }

  function refreshExportState() {
    return client.invalidateQueries({ queryKey: queryKeys.exportStatus });
  }

  /** Searches the open place for a tag; a compact screen leaves the item for the list. */
  function showTag(tag: string) {
    if (compact) openPlace.current?.close();
    setQuery(tagSearch(tag));
  }

  /** The search belongs to the place it was typed in; the section and the group carry over. */
  function choosePlace(next: WorkspacePlace, opening: OpeningPane = "empty") {
    if (next === place) return;
    openPlace.current?.close();
    setQuery("");
    setOpening(opening);
    if (next === "settings") {
      setSettingsOpen(true);
      return;
    }
    setSettingsOpen(false);
    setItemPlace(next);
  }

  function chooseSection(next: WorkspaceSection) {
    setSection(next);
    if (settingsOpen) {
      setOpening("empty");
      setSettingsOpen(false);
    } else {
      openPlace.current?.leaveEditor();
    }
  }

  function showInPlace(next: ItemPlaceName, opening: "new" | { item: string }) {
    if (next !== place) {
      choosePlace(next, opening);
    } else if (opening === "new") {
      openPlace.current?.create();
    } else {
      openPlace.current?.open(opening.item);
    }
  }

  // A new item made from settings goes to the place last open.
  function createItem() {
    showInPlace(itemPlace, "new");
  }

  /** Opens settings at one subsection, or at the list a compact screen starts from. */
  function showSettings(section: SettingsSection | null) {
    setSettingsSection(section);
    choosePlace("settings");
  }

  function choose(choice: PaletteChoice) {
    setPaletteOpen(false);
    switch (choice.kind) {
      case "item":
        showInPlace(choice.place, { item: choice.id });
        break;
      case "group":
        setGroup(choice.id);
        choosePlace(itemPlace);
        break;
      case "settings":
        showSettings(choice.section);
        break;
      case "new":
        showInPlace(choice.place, "new");
        break;
      case "lock":
        void lock();
        break;
    }
  }

  // A group change rewrites the index every list and tab reads.
  async function changeGroups(change: () => Promise<void>) {
    setWorking(true);
    setSyncState("writing");
    try {
      await change();
      setSyncState("synced");
      await refreshGroups();
      await refreshItems();
      await refreshExportState();
      return true;
    } catch (cause) {
      settleSync(cause);
      report(cause, "workspace.error.group");
      return false;
    } finally {
      setWorking(false);
    }
  }

  function createGroup(name: string) {
    return changeGroups(async () => {
      await api.createGroup(name);
    });
  }

  function renameGroup(id: string, name: string) {
    return changeGroups(() => api.renameGroup(id, name));
  }

  function deleteGroup(id: string) {
    void changeGroups(() => api.deleteGroup(id));
  }

  async function chooseDefaultGroup(id: string) {
    setWorking(true);
    try {
      await api.setDefaultGroup(id);
      client.setQueryData<VaultGroups>(
        reads.groups.key,
        (current) => current && { ...current, defaultGroup: id },
      );
    } catch (cause) {
      report(cause, "workspace.error.group");
    } finally {
      setWorking(false);
    }
  }

  // Turning icons off deletes them on the host.
  async function changeSiteIcons(enabled: boolean) {
    await siteIconsSetting.change(enabled);
    if (!client.getQueryData<SiteIcons>(queryKeys.siteIcons)?.enabled) {
      siteIconStore.clear();
    }
  }

  async function exportCopy() {
    setWorking(true);
    try {
      await api.exportEncryptedCopy();
      toast.success(t("workspace.backup.saved"));
    } catch (cause) {
      report(cause, "workspace.error.export");
    } finally {
      await refreshExportState();
      setWorking(false);
    }
  }

  // The import is one save; a failed read of the lists afterwards leaves the items added.
  async function importItems(
    options: ImportOptions,
    cardNetworks: ImportCardNetworks,
  ): Promise<ImportResult> {
    setWorking(true);
    setSyncState("writing");
    try {
      const result = await api.importItems(options, cardNetworks);
      setSyncState("synced");
      await Promise.all([refreshGroups(), refreshItems()]).catch(
        (cause: unknown) => report(cause, "settings.import.error.refresh"),
      );
      await refreshExportState();
      return result;
    } catch (cause) {
      settleSync(cause);
      report(cause, "settings.import.error.run");
      throw cause;
    } finally {
      setWorking(false);
    }
  }

  function forgetCodeSetup() {
    client.setQueryData<CodeSetup | null>(reads.codeSetup.key, null);
  }

  /** `add` resolves with the credential the code went to; a refused add leaves the setup waiting. */
  async function addCodeSetup(
    add: () => Promise<string>,
    adding: string | null,
  ) {
    setWorking(true);
    setAddingCode(adding);
    setSyncState("writing");
    try {
      const id = await add();
      setSyncState("synced");
      forgetCodeSetup();
      toast.success(t("code-setup.added"));
      await refreshCredentials().catch((cause: unknown) =>
        report(cause, "workspace.error.list"),
      );
      await refreshExportState();
      // Opening the credential the pane already shows does not read it anew.
      if (place === "passwords") openPlace.current?.reload();
      showInPlace("passwords", { item: id });
    } catch (cause) {
      settleSync(cause);
      report(cause, "code-setup.error");
      await client.invalidateQueries({ queryKey: reads.codeSetup.key });
    } finally {
      setAddingCode(null);
      setWorking(false);
    }
  }

  function dismissCodeSetup(setup: CodeSetup) {
    forgetCodeSetup();
    api
      .dismissCodeSetup(setup.token)
      .catch((cause: unknown) => report(cause, "code-setup.error.dismiss"));
  }

  async function changeUnlock(
    run: () => Promise<void>,
    pending: UnlockPending = "other",
  ) {
    try {
      await unlockChange.mutateAsync({ run, pending });
      return true;
    } catch {
      // The mutation reports the failure.
      return false;
    }
  }

  function setPin(pin: string, current: string) {
    return changeUnlock(async () => {
      await api.setPin(pin, current);
      toast.success(t("unlock-methods.pin.saved"));
    });
  }

  function removePin(current: string) {
    return changeUnlock(async () => {
      await api.removePin(current);
      toast.success(t("unlock-methods.pin.removed"));
    });
  }

  function setBiometry(enabled: boolean, current: string) {
    return changeUnlock(
      async () => {
        await api.setBiometryUnlock(enabled, current);
        toast.success(
          t(
            enabled
              ? "unlock-methods.biometry.on"
              : "unlock-methods.biometry.off",
          ),
        );
      },
      enabled ? "biometry-on" : "other",
    );
  }

  // The move shows only once the owner has chosen where it goes.
  async function moveStorage(kind: StorageKind) {
    setStorageMove("choosing");
    try {
      if (!(await api.chooseStorageMove(kind))) return;
      setStorageMove("moving");
      const change = await api.moveStorageLocation();
      if (change.changed) {
        await storageRead.refetch();
        toast.success(
          change.previous && !change.previousRemoved
            ? t("workspace.storage.moved.copy-left", {
                location: vaultLocation(change.previous),
              })
            : t("workspace.storage.moved"),
        );
      }
    } catch (cause) {
      report(cause, "workspace.error.storage-move");
    } finally {
      setStorageMove(null);
    }
  }

  // Switching or deleting the open vault can leave this screen.
  async function continueAfterVaultChange(chosen: boolean) {
    const phase = (await api.getState()).phase;
    if (phase !== "ready") {
      onPhase(phase, chosen);
      return;
    }
    siteIconStore.clear();
    await Promise.all([
      client.invalidateQueries({ queryKey: vaultScope }),
      client.invalidateQueries({ queryKey: queryKeys.unlockMethods }),
      storageRead.refetch(),
    ]);
  }

  /** `chosen` is set for an action that opens the vault the owner chose. */
  async function runVaultAction(
    action: () => Promise<boolean>,
    failureMessage: MessageKey,
    chosen = false,
  ) {
    setWorking(true);
    try {
      if (await action()) {
        await continueAfterVaultChange(chosen);
      }
    } catch (cause) {
      report(cause, failureMessage);
    } finally {
      setWorking(false);
    }
  }

  function switchVault(path: string) {
    return runVaultAction(
      async () => {
        await api.switchVault(path);
        return true;
      },
      "workspace.error.vault-switch",
      true,
    );
  }

  function createVault() {
    return runVaultAction(
      async () => (await api.createVault()).changed,
      "workspace.error.vault-create",
    );
  }

  function openVault() {
    return runVaultAction(
      async () => (await api.openVault()).changed,
      "workspace.error.vault-open",
      true,
    );
  }

  function forgetVault(vault: Vault) {
    const removed = vaultName(vault.name);
    return runVaultAction(async () => {
      await api.forgetVault(vault.path);
      toast.success(t("workspace.vault.forgotten", { name: removed }));
      return true;
    }, "workspace.error.vault-forget");
  }

  function deleteVault(vault: Vault) {
    const deleted = vaultName(vault.name);
    return runVaultAction(async () => {
      const result = await api.deleteVault(vault.path);
      if (!result.deleted) return false;
      toast.success(
        result.deviceRecordsRemoved
          ? t("workspace.vault.deleted", { name: deleted })
          : t("workspace.vault.deleted.keys-kept", { name: deleted }),
      );
      return true;
    }, "workspace.error.vault-delete");
  }

  async function lock() {
    openPlace.current?.forget();
    setWorking(true);
    try {
      await api.lock();
      onPhase("locked");
    } catch (cause) {
      report(cause, "workspace.error.lock");
      setWorking(false);
    }
  }

  const shell: PlaceShell = {
    api,
    groups,
    group,
    onGroup: setGroup,
    defaultGroup,
    tags: vaultTags,
    onTag: showTag,
    section,
    query,
    clipboard,
    busy,
    setBusy: setWorking,
    selection,
    setSyncState,
    settleSync,
    report,
    refreshExportState,
    onPaneShown: setPaneShown,
    listPosition: listPositions.listing(itemPlace, section, group, query),
  };
  const importActions: ImportActions = {
    choose: (source) => api.chooseImportFile(source),
    unlock: (password) => api.unlockImportFile(password),
    run: importItems,
    cancel: () => api.cancelImport(),
    trash: () => api.trashImportFile(),
    reveal: () => api.revealImportFile(),
  };
  const search = searchLabels[itemPlace];
  const counts: Record<ItemPlaceName, number> = {
    passwords: credentials.length,
    identities: identities.length,
    cards: cards.length,
    notes: notes.length,
    seeds: seeds.length,
  };
  const count = settingsOpen
    ? Object.values(counts).reduce((total, items) => total + items, 0)
    : counts[itemPlace];
  const placeLabel = t(placeOf(itemPlace).label);
  const listShown = !settingsOpen && !paneShown;
  // A compact screen's head and foot cross-fade with the content between them.
  const compactScreen = settingsOpen ? "settings" : paneShown ? "pane" : "list";
  const iconStore = siteIcons?.enabled ? siteIconStore : null;
  const toolbar = (
    <WorkspaceToolbar
      vaults={storage?.vaults ?? []}
      currentVaultPath={storage?.path ?? ""}
      onSwitchVault={switchVault}
      onCreateVault={createVault}
      onOpenVault={openVault}
      title={placeLabel}
      count={counts[itemPlace]}
      section={section}
      onSection={chooseSection}
      query={query}
      onQuery={setQuery}
      searchRef={searchField}
      searchLabel={t(search.label)}
      searchPlaceholder={t(search.placeholder)}
      searchShortcut={
        offers.shortcuts
          ? formatForDisplay(shortcuts.hotkey("search"))
          : undefined
      }
      onNew={createItem}
      onLock={lock}
      onSettings={() => showSettings(null)}
      busy={busy}
    />
  );

  return (
    <main className="flex h-dvh min-h-0 min-w-0 flex-col overflow-hidden bg-background text-foreground">
      {compact ? (
        <CrossFade id={compactScreen} className="shrink-0">
          {compactScreen === "list" && toolbar}
          {compactScreen === "pane" && (
            <BackRow
              label={placeLabel}
              onBack={() => openPlace.current?.close()}
            />
          )}
        </CrossFade>
      ) : (
        toolbar
      )}
      <div
        className={cn("flex min-h-0 flex-1 gap-3", !compact && "px-3.5 pb-1")}
      >
        {!compact && <WorkspaceRail place={place} onPlace={choosePlace} />}
        <section
          className="flex min-w-0 flex-1"
          aria-label={t(regionLabels[place])}
        >
          <CrossFade id={place} className="flex min-w-0 flex-1 gap-3">
            {place === "settings" && (
              <div className="flex min-h-0 flex-1 flex-col overflow-hidden rounded-pane border bg-card max-sm:rounded-none max-sm:border-0 max-sm:bg-background">
                <SettingsView
                  section={settingsSection}
                  onSection={setSettingsSection}
                  leaveLabel={placeLabel}
                  onLeave={() => choosePlace(itemPlace)}
                  busy={busy}
                  autoLock={autoLockSetting.value}
                  onAutoLock={autoLockSetting.change}
                  interfaceSize={interfaceSize}
                  onInterfaceSize={(percent) =>
                    interfaceSizeChange.mutate(percent)
                  }
                  appearance={appearance}
                  onAppearance={(next) => appearanceChange.mutate(next)}
                  dockIcon={dockIconSetting.value}
                  onDockIcon={dockIconSetting.change}
                  clipboard={clipboard}
                  onClipboard={clipboardSetting.change}
                  siteIcons={siteIcons}
                  onSiteIcons={changeSiteIcons}
                  bankDetails={bankDetails}
                  onBankDetails={bankDetailsSetting.change}
                  screenshots={api}
                  storage={storage}
                  movingStorage={storageMove === "moving"}
                  onMoveStorage={moveStorage}
                  groups={groups}
                  items={[
                    ...credentials,
                    ...identities,
                    ...cards,
                    ...notes,
                    ...seeds,
                  ]}
                  defaultGroup={defaultGroup}
                  onCreateGroup={createGroup}
                  onRenameGroup={renameGroup}
                  onDeleteGroup={deleteGroup}
                  onDefaultGroup={chooseDefaultGroup}
                  onForgetVault={forgetVault}
                  onDeleteVault={deleteVault}
                  shortcuts={shortcuts}
                  onShortcut={shortcutSetting.change}
                  onRecordingShortcut={setRecordingShortcut}
                  unlockMethods={unlockMethods}
                  unlockPending={unlockPending}
                  onSetPin={setPin}
                  onRemovePin={removePin}
                  onBiometry={setBiometry}
                  recoveryKey={api}
                  autofill={api}
                  extensionLinking={api}
                  backups={api}
                  exportState={exportState}
                  onExport={exportCopy}
                  onBackedUp={() => {
                    void refreshExportState();
                  }}
                  importActions={importActions}
                  about={api}
                  onOpenWebsite={(address) => {
                    void api
                      .openWebsite(address)
                      .catch((cause: unknown) =>
                        report(cause, "workspace.error.website"),
                      );
                  }}
                />
              </div>
            )}
            {place === "passwords" && (
              <SiteIconsProvider store={iconStore}>
                <PasswordsPlace
                  ref={openPlace}
                  shell={shell}
                  entries={credentials}
                  loading={credentialsRead.isPending}
                  refresh={refreshCredentials}
                  opening={opening}
                  onImport={() => showSettings("import")}
                />
              </SiteIconsProvider>
            )}
            {place === "identities" && (
              <IdentitiesPlace
                ref={openPlace}
                shell={shell}
                entries={identities}
                loading={identitiesRead.isPending}
                refresh={refreshIdentities}
                opening={opening}
              />
            )}
            {place === "cards" && (
              <SiteIconsProvider store={iconStore}>
                <CardsPlace
                  ref={openPlace}
                  shell={shell}
                  entries={cards}
                  loading={cardsRead.isPending}
                  refresh={refreshCards}
                  opening={opening}
                  bankLookup={Boolean(bankDetails?.enabled)}
                />
              </SiteIconsProvider>
            )}
            {place === "notes" && (
              <NotesPlace
                ref={openPlace}
                shell={shell}
                entries={notes}
                loading={notesRead.isPending}
                refresh={refreshNotes}
                opening={opening}
              />
            )}
            {place === "seeds" && (
              <SeedsPlace
                ref={openPlace}
                shell={shell}
                entries={seeds}
                loading={seedsRead.isPending}
                refresh={refreshSeeds}
                opening={opening}
              />
            )}
          </CrossFade>
        </section>
      </div>
      {compact && (
        <CrossFade id={listShown ? "tabs" : "clear"} className="shrink-0">
          {listShown ? (
            <WorkspaceTabs place={itemPlace} onPlace={choosePlace} />
          ) : (
            // The tabs keep clear of the system's gesture area; a screen without them does it here.
            <div className="pb-[env(safe-area-inset-bottom)]" />
          )}
        </CrossFade>
      )}
      {!compact && <WorkspaceStatusBar syncState={syncState} count={count} />}
      {codeSetup && !credentialsRead.isPending && (
        <SiteIconsProvider store={iconStore}>
          <CodeSetupDialog
            setup={codeSetup}
            credentials={credentials}
            busy={busy}
            adding={addingCode}
            onAdd={(id) =>
              void addCodeSetup(async () => {
                await api.addCodeSetup(codeSetup.token, id);
                return id;
              }, id)
            }
            onCreate={() =>
              void addCodeSetup(
                () => api.createCredentialFromCodeSetup(codeSetup.token),
                null,
              )
            }
            onDismiss={() => dismissCodeSetup(codeSetup)}
          />
        </SiteIconsProvider>
      )}
      {offers.shortcuts && (
        <SiteIconsProvider store={iconStore}>
          <CommandPalette
            open={paletteOpen}
            onOpenChange={setPaletteOpen}
            credentials={credentials}
            identities={identities}
            cards={cards}
            notes={notes}
            seeds={seeds}
            groups={groups}
            busy={busy}
            onChoose={choose}
          />
        </SiteIconsProvider>
      )}
    </main>
  );
}
