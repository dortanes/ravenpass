import type { Hotkey } from "@tanstack/react-hotkeys";
import { cn } from "cn";
import type { Appearance } from "../host/appearance.ts";
import { useCapabilities } from "../host/capabilities.tsx";
import { useCompactLayout } from "../host/compact.ts";
import { useTranslator } from "../i18n/translator.tsx";
import { CrossFade } from "../motion/CrossFade.tsx";
import type { ShortcutAction, Shortcuts } from "../shortcuts/shortcuts.ts";
import type {
  AboutApi,
  AppearanceSetting,
  AutofillSettings,
  AutoLock,
  BackupSettings,
  BankDetails,
  BreachChecks,
  ClipboardClearing,
  DockIcon,
  ExportState,
  ExtensionLinking,
  Group,
  InterfaceSize,
  ScreenshotSettings,
  SiteIcons,
  StorageKind,
  StorageStatus,
  UnlockMethods,
  Vault,
} from "../vault-api.ts";
import type { ListedItem } from "../workspace/sections.ts";
import { WrapBlockText } from "./Block.tsx";
import { AboutPanel } from "./settings/AboutPanel.tsx";
import { AutofillPanel } from "./settings/AutofillPanel.tsx";
import { BackupsPanel } from "./settings/BackupsPanel.tsx";
import { GeneralPanel } from "./settings/GeneralPanel.tsx";
import { GroupsPanel } from "./settings/GroupsPanel.tsx";
import { type ImportActions, ImportPanel } from "./settings/ImportPanel.tsx";
import { PrivacyPanel } from "./settings/PrivacyPanel.tsx";
import type { RecoveryKeyChangeApi } from "./settings/RecoveryKeyChange.tsx";
import { SecurityPanel } from "./settings/SecurityPanel.tsx";
import { SettingsNav } from "./settings/SettingsNav.tsx";
import { ShortcutsPanel } from "./settings/ShortcutsPanel.tsx";
import {
  offeredSections,
  offeredVaultTabs,
  type SettingsSection,
  settingsParent,
} from "./settings/sections.ts";
import {
  type TrashActions,
  type TrashApi,
  TrashPanel,
} from "./settings/TrashPanel.tsx";
import { VaultsPanel } from "./settings/VaultsPanel.tsx";
import { ScrollArea } from "./ui/scroll-area.tsx";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "./ui/tabs.tsx";
import { ownerCheck, type UnlockPending } from "./unlock-change.ts";
import { BackRow } from "./workspace/BackRow.tsx";

/** SettingsView is one pane with its own navigation; a compact screen shows the list or one subsection. */
export function SettingsView({
  section,
  onSection,
  leaveLabel,
  onLeave,
  busy,
  autoLock,
  onAutoLock,
  interfaceSize,
  onInterfaceSize,
  appearance,
  onAppearance,
  dockIcon,
  onDockIcon,
  clipboard,
  onClipboard,
  siteIcons,
  onSiteIcons,
  bankDetails,
  onBankDetails,
  breachChecks,
  onBreachChecks,
  screenshots,
  storage,
  movingStorage,
  onMoveStorage,
  groups,
  items,
  defaultGroup,
  onCreateGroup,
  onRenameGroup,
  onDeleteGroup,
  onDefaultGroup,
  onForgetVault,
  onDeleteVault,
  shortcuts,
  onShortcut,
  onRecordingShortcut,
  unlockMethods,
  unlockPending,
  onSetPin,
  onRemovePin,
  onBiometry,
  recoveryKey,
  autofill,
  extensionLinking,
  backups,
  exportState,
  onExport,
  onBackedUp,
  importActions,
  trash,
  trashActions,
  about,
  onOpenWebsite,
}: {
  /** The subsection chosen; none while a compact screen lists them. */
  section: SettingsSection | null;
  onSection: (section: SettingsSection | null) => void;
  /** Names, in a compact screen's back row, where leaving settings returns to. */
  leaveLabel: string;
  onLeave: () => void;
  busy: boolean;
  autoLock: AutoLock | null;
  onAutoLock: (enabled: boolean, seconds: number) => void;
  interfaceSize: InterfaceSize | null;
  onInterfaceSize: (percent: number) => void;
  appearance: AppearanceSetting | null;
  onAppearance: (appearance: Appearance) => void;
  dockIcon: DockIcon | null;
  onDockIcon: (hideWithWindow: boolean) => void;
  clipboard: ClipboardClearing | null;
  onClipboard: (enabled: boolean, seconds: number) => void;
  siteIcons: SiteIcons | null;
  onSiteIcons: (enabled: boolean) => void;
  bankDetails: BankDetails | null;
  onBankDetails: (enabled: boolean) => void;
  breachChecks: BreachChecks | null;
  onBreachChecks: (enabled: boolean) => void;
  screenshots: ScreenshotSettings;
  storage: StorageStatus | null;
  movingStorage: boolean;
  onMoveStorage: (kind: StorageKind) => void;
  groups: Group[];
  /** Every item of the open vault, of every kind. */
  items: readonly ListedItem[];
  defaultGroup: string;
  onCreateGroup: (name: string) => Promise<boolean>;
  onRenameGroup: (id: string, name: string) => Promise<boolean>;
  onDeleteGroup: (id: string) => void;
  onDefaultGroup: (id: string) => void;
  onForgetVault: (vault: Vault) => void;
  onDeleteVault: (vault: Vault) => void;
  shortcuts: Shortcuts;
  onShortcut: (action: ShortcutAction, hotkey: Hotkey | null) => void;
  onRecordingShortcut: (recording: boolean) => void;
  unlockMethods: UnlockMethods | null;
  unlockPending: UnlockPending | null;
  onSetPin: (pin: string, current: string) => Promise<boolean>;
  onRemovePin: (current: string) => Promise<boolean>;
  onBiometry: (enabled: boolean, current: string) => Promise<boolean>;
  recoveryKey: RecoveryKeyChangeApi;
  autofill: AutofillSettings;
  extensionLinking: ExtensionLinking;
  backups: BackupSettings;
  exportState: ExportState;
  onExport: () => void;
  onBackedUp: () => void;
  importActions: ImportActions;
  trash: TrashApi;
  trashActions: TrashActions;
  about: AboutApi;
  /** Opens a web address in the browser. */
  onOpenWebsite: (address: string) => void;
}) {
  const { t } = useTranslator();
  const compact = useCompactLayout();
  const capabilities = useCapabilities();
  const sections = offeredSections(capabilities);
  const vaultTabs = offeredVaultTabs(capabilities);
  const vaultTab = vaultTabs.find((tab) => tab.id === section)?.id ?? "vaults";
  // A wide screen always shows a subsection; a compact one lists them until one is chosen.
  const current =
    sections.find((item) => item.id === settingsParent(section)) ??
    (compact ? undefined : sections[0]);

  const panel = current && (
    <WrapBlockText>
      <h2
        className={cn(
          "mb-[21px] font-medium leading-[1.35]",
          compact ? "text-[22px]" : "text-[17px]",
        )}
      >
        {t(current.label)}
      </h2>
      <div>
        {current.id === "general" && (
          <GeneralPanel
            interfaceSize={interfaceSize}
            onInterfaceSize={onInterfaceSize}
            appearance={appearance}
            onAppearance={onAppearance}
            dockIcon={dockIcon}
            onDockIcon={onDockIcon}
            busy={busy}
          />
        )}
        {current.id === "security" && (
          <SecurityPanel
            methods={unlockMethods}
            busy={busy}
            pending={unlockPending}
            onSetPin={onSetPin}
            onRemovePin={onRemovePin}
            onBiometry={onBiometry}
            autoLock={autoLock}
            onAutoLock={onAutoLock}
            recoveryKey={recoveryKey}
          />
        )}
        {current.id === "privacy" && (
          <PrivacyPanel
            busy={busy}
            siteIcons={siteIcons}
            onSiteIcons={onSiteIcons}
            bankDetails={bankDetails}
            onBankDetails={onBankDetails}
            breachChecks={breachChecks}
            onBreachChecks={onBreachChecks}
            clipboard={clipboard}
            onClipboard={onClipboard}
            screenshots={screenshots}
          />
        )}
        {current.id === "autofill" && (
          <AutofillPanel
            settings={autofill}
            linking={extensionLinking}
            verifiable={
              unlockMethods !== null &&
              ownerCheck(unlockMethods) !== "recovery-key"
            }
            busy={busy}
          />
        )}
        {current.id === "shortcuts" && (
          <ShortcutsPanel
            shortcuts={shortcuts}
            busy={busy}
            onChange={onShortcut}
            onRecording={onRecordingShortcut}
          />
        )}
        {current.id === "vaults" && (
          <Tabs
            value={vaultTab}
            onValueChange={(value) => {
              const tab = vaultTabs.find((item) => item.id === value);
              if (tab) onSection(tab.id);
            }}
            className="-mt-[7px] gap-5"
          >
            <TabsList
              aria-label={t("settings.vaults.heading")}
              className="h-auto w-full flex-wrap justify-start gap-1 rounded-none border-b bg-transparent p-0 pb-[9px] group-data-[orientation=horizontal]/tabs:h-auto"
            >
              {vaultTabs.map((tab) => {
                const Icon = tab.icon;
                return (
                  <TabsTrigger
                    key={tab.id}
                    value={tab.id}
                    className="h-auto flex-none rounded-lg whitespace-normal border-0 px-[11px] py-[7px] text-xs font-normal data-[state=active]:bg-tile data-[state=active]:shadow-none"
                  >
                    <Icon aria-hidden="true" />
                    {t(tab.label)}
                  </TabsTrigger>
                );
              })}
            </TabsList>
            <TabsContent value="vaults">
              <VaultsPanel
                storage={storage}
                busy={busy}
                moving={movingStorage}
                onMove={onMoveStorage}
                onForget={onForgetVault}
                onDelete={onDeleteVault}
              />
            </TabsContent>
            {capabilities.saveFiles && (
              <TabsContent value="backups">
                <BackupsPanel
                  settings={backups}
                  exportState={exportState}
                  busy={busy}
                  onExport={onExport}
                  onBackedUp={onBackedUp}
                />
              </TabsContent>
            )}
            <TabsContent value="groups">
              <GroupsPanel
                groups={groups}
                items={items}
                defaultGroup={defaultGroup}
                busy={busy}
                onCreate={onCreateGroup}
                onRename={onRenameGroup}
                onDelete={onDeleteGroup}
                onDefaultGroup={onDefaultGroup}
              />
            </TabsContent>
            <TabsContent value="trash">
              <TrashPanel trash={trash} busy={busy} {...trashActions} />
            </TabsContent>
            <TabsContent value="import">
              <ImportPanel actions={importActions} busy={busy} />
            </TabsContent>
          </Tabs>
        )}
        {current.id === "about" && (
          <AboutPanel about={about} onOpenWebsite={onOpenWebsite} />
        )}
      </div>
    </WrapBlockText>
  );

  if (compact) {
    return (
      <CrossFade
        id={current?.id ?? "sections"}
        className="flex min-h-0 flex-1 flex-col"
      >
        <BackRow
          label={current ? t("settings.nav.heading") : leaveLabel}
          onBack={current ? () => onSection(null) : onLeave}
        />
        <ScrollArea className="min-h-0 flex-1">
          {current ? (
            <div className="px-4 pt-1 pb-6">{panel}</div>
          ) : (
            <WrapBlockText>
              <SettingsNav sections={sections} onSection={onSection} />
            </WrapBlockText>
          )}
        </ScrollArea>
      </CrossFade>
    );
  }

  return (
    <div className="flex min-h-0 flex-1">
      <SettingsNav
        sections={sections}
        section={current?.id}
        onSection={onSection}
      />
      <ScrollArea className="min-h-0 flex-1">
        <CrossFade id={current?.id ?? ""} className="max-w-[524px] p-5">
          {panel}
        </CrossFade>
      </ScrollArea>
    </div>
  );
}
