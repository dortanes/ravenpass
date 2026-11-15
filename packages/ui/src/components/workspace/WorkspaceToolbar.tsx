import { cn } from "cn";
import { KeySquare, LockKeyhole, Plus, Settings2 } from "lucide-react";
import type { ComponentType, RefObject } from "react";
import { useCompactLayout } from "../../host/compact.ts";
import { useTranslator } from "../../i18n/translator.tsx";
import type { Vault } from "../../vault-api.ts";
import type { WorkspaceSection } from "../../workspace/sections.ts";
import { SearchField } from "../SearchField.tsx";
import { Button } from "../ui/button.tsx";
import { VaultMenu } from "../VaultMenu.tsx";
import { SectionSwitch } from "./SectionSwitch.tsx";

/** WorkspaceToolbar is the workspace's head; a compact screen stacks it. */
export function WorkspaceToolbar({
  vaults,
  currentVaultPath,
  onSwitchVault,
  onCreateVault,
  onOpenVault,
  title,
  count,
  section,
  onSection,
  query,
  onQuery,
  searchRef,
  searchLabel,
  searchPlaceholder,
  searchShortcut,
  onNew,
  onLock,
  onGenerator,
  onSettings,
  busy,
}: {
  vaults: Vault[];
  currentVaultPath: string;
  onSwitchVault: (path: string) => void;
  onCreateVault: () => void;
  onOpenVault: () => void;
  /** The open place's name, which a compact screen shows above the search. */
  title: string;
  /** How many items the open place holds. */
  count: number;
  section: WorkspaceSection;
  onSection: (section: WorkspaceSection) => void;
  query: string;
  onQuery: (value: string) => void;
  searchRef: RefObject<HTMLInputElement | null>;
  /** The search is named for the open place, and says what it matches. */
  searchLabel: string;
  searchPlaceholder: string;
  /** The platform's notation for the keys that focus the search, on a host that takes shortcuts. */
  searchShortcut?: string;
  /** Creates an item of the kind the open place holds. */
  onNew: () => void;
  onLock: () => void;
  /** Opens the Generator place, which a compact screen reaches from the toolbar. */
  onGenerator: () => void;
  /** Opens settings, which a compact screen reaches from the toolbar. */
  onSettings: () => void;
  busy: boolean;
}) {
  const { t } = useTranslator();
  const compact = useCompactLayout();

  const vaultMenu = (
    <VaultMenu
      vaults={vaults}
      currentPath={currentVaultPath}
      busy={busy}
      onSwitch={onSwitchVault}
      onCreate={onCreateVault}
      onOpen={onOpenVault}
    />
  );
  const search = (
    <SearchField
      id="item-search"
      inputRef={searchRef}
      value={query}
      onChange={onQuery}
      label={searchLabel}
      placeholder={searchPlaceholder}
      shortcut={searchShortcut}
      className={compact ? undefined : "min-w-0 flex-1"}
    />
  );
  const lock = (
    <VaultAction
      label={t("workspace.toolbar.lock")}
      icon={LockKeyhole}
      compact={compact}
      disabled={busy}
      onClick={onLock}
    />
  );

  if (compact) {
    return (
      <header className="flex shrink-0 flex-col gap-3 bg-background px-4 pt-[env(safe-area-inset-top)] pb-2">
        <div className="flex h-14 items-center gap-2">
          {vaultMenu}
          <span className="ml-auto flex items-center gap-2">
            {lock}
            <VaultAction
              label={t("workspace.rail.generator")}
              icon={KeySquare}
              compact
              onClick={onGenerator}
            />
            <VaultAction
              label={t("workspace.rail.settings")}
              icon={Settings2}
              compact
              onClick={onSettings}
            />
          </span>
        </div>
        <h1 className="flex min-w-0 items-baseline gap-2">
          <span className="truncate text-[22px] font-medium">{title}</span>
          <span className="text-[11px] text-muted-foreground">{count}</span>
        </h1>
        {search}
        <SectionSwitch section={section} onSection={onSection} stretch />
      </header>
    );
  }

  return (
    <header className="drag-region flex h-13 shrink-0 items-center gap-2.5 bg-background pr-3.5 pl-[88px]">
      {vaultMenu}
      <SectionSwitch section={section} onSection={onSection} />
      {search}
      <Button
        type="button"
        variant="raised"
        size="pill-sm"
        className="h-[29px] px-3 has-[>svg]:px-3"
        onClick={onNew}
        disabled={busy}
      >
        <Plus data-icon="inline-start" />
        {t("workspace.toolbar.new")}
      </Button>
      {lock}
    </header>
  );
}

function VaultAction({
  label,
  icon: Icon,
  compact,
  disabled,
  onClick,
}: {
  label: string;
  icon: ComponentType<{ className?: string }>;
  compact: boolean;
  disabled?: boolean;
  onClick: () => void;
}) {
  return (
    <Button
      type="button"
      variant="ghost"
      size="icon"
      aria-label={label}
      title={label}
      className={cn(
        "border bg-control hover:bg-secondary",
        compact ? "size-10 rounded-xl" : "size-[29px] rounded-lg",
      )}
      onClick={onClick}
      disabled={disabled}
    >
      <Icon className={compact ? "size-[18px]" : "size-4"} />
    </Button>
  );
}
