import { cn } from "cn";
import {
  ChevronRight,
  Folder,
  Inbox,
  Layers,
  MoreHorizontal,
  PanelLeftClose,
  Pencil,
  Plus,
  Star,
  Trash2,
} from "lucide-react";
import { useState } from "react";
import { useTranslator } from "../../i18n/translator.tsx";
import {
  aboveIndicator,
  SelectionGroup,
  SelectionIndicator,
} from "../../motion/SelectionIndicator.tsx";
import type { Group } from "../../vault-api.ts";
import {
  countInGroup,
  countUngrouped,
  everyGroup,
  type ListedItem,
  ungroupedGroup,
} from "../../workspace/sections.ts";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "../ui/dropdown-menu.tsx";
import { ScrollArea } from "../ui/scroll-area.tsx";
import { GroupDialogs } from "./GroupDialogs.tsx";

/** GroupSidebar shows the vault's groups beside the item list on desktop. */
export function GroupSidebar({
  groups,
  group,
  onGroup,
  items,
  defaultGroup = "",
  onCreate,
  onRename,
  onDelete,
  onDefaultGroup,
  onCollapse,
  busy = false,
}: {
  groups: Group[];
  group: string;
  onGroup: (group: string) => void;
  /** Every item of the open place, which the sidebar counts. */
  items: readonly ListedItem[];
  defaultGroup?: string;
  onCreate?: (name: string) => Promise<string | null>;
  onRename?: (id: string, name: string) => Promise<boolean>;
  onDelete?: (id: string) => void;
  onDefaultGroup?: (id: string) => void;
  onCollapse: () => void;
  busy?: boolean;
}) {
  const { t } = useTranslator();
  const [creating, setCreating] = useState(false);
  const [renaming, setRenaming] = useState<Group | null>(null);
  const [deleting, setDeleting] = useState<Group | null>(null);
  const [showEmpty, setShowEmpty] = useState(false);

  const ungroupedCount = countUngrouped(items);
  const withCounts = groups.map((item) => ({
    item,
    count: countInGroup(items, item.id),
  }));
  const populated = withCounts.filter(
    ({ item, count }) => count > 0 || group === item.id,
  );
  const empty = withCounts.filter(
    ({ item, count }) => count === 0 && group !== item.id,
  );

  function renderGroupRow(item: Group, count: number) {
    const active = group === item.id;
    const isDefault = item.id === defaultGroup;
    const hasActions = Boolean(onRename || onDelete || onDefaultGroup);
    return (
      <div
        key={item.id}
        className="group/sidebar-row relative flex items-center"
      >
        <button
          type="button"
          onClick={() => onGroup(item.id)}
          title={
            isDefault
              ? `${item.name} · ${t("workspace.groups.default")}`
              : item.name
          }
          aria-current={active ? "true" : undefined}
          className={cn(
            "relative flex h-8 w-full cursor-pointer items-center gap-2 rounded-lg px-2 text-left text-[12.5px] outline-none transition-colors focus-visible:ring-2 focus-visible:ring-ring/40",
            active
              ? "text-foreground"
              : "text-muted-foreground hover:bg-control hover:text-foreground",
          )}
        >
          {active && (
            <SelectionIndicator className="rounded-lg bg-raised inset-ring inset-ring-foreground/10" />
          )}
          <Folder
            className={cn(
              aboveIndicator,
              "size-3.5 shrink-0",
              active ? "text-foreground" : "opacity-70",
            )}
          />
          <span className={cn(aboveIndicator, "min-w-0 flex-1 truncate")}>
            {item.name}
          </span>
          {isDefault && (
            <Star
              aria-label={t("workspace.groups.default")}
              className={cn(
                aboveIndicator,
                "size-2.5 shrink-0 fill-current text-faint transition-opacity group-hover/sidebar-row:opacity-0 group-has-[[data-state=open]]/sidebar-row:opacity-0",
              )}
            />
          )}
          <span
            className={cn(
              aboveIndicator,
              "text-[11px] text-faint",
              hasActions &&
                "transition-opacity group-hover/sidebar-row:opacity-0 group-has-[[data-state=open]]/sidebar-row:opacity-0",
            )}
          >
            {count}
          </span>
        </button>
        {hasActions && (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <button
                type="button"
                aria-label={t("workspace.groups.more")}
                title={t("workspace.groups.more")}
                disabled={busy}
                className="pointer-events-none absolute right-1 z-2 flex size-6 cursor-pointer items-center justify-center rounded-md text-muted-foreground opacity-0 transition-opacity hover:bg-control hover:text-foreground focus-visible:pointer-events-auto focus-visible:opacity-100 group-hover/sidebar-row:pointer-events-auto group-hover/sidebar-row:opacity-100 data-[state=open]:pointer-events-auto data-[state=open]:bg-control data-[state=open]:text-foreground data-[state=open]:opacity-100"
              >
                <MoreHorizontal className="size-3.5" />
              </button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-44">
              {onDefaultGroup && (
                <DropdownMenuItem
                  onClick={() => onDefaultGroup(isDefault ? "" : item.id)}
                  disabled={busy}
                >
                  <Star className={cn("size-4", isDefault && "fill-current")} />
                  <span>
                    {t(
                      isDefault
                        ? "workspace.groups.unsetDefault"
                        : "workspace.groups.setDefault",
                    )}
                  </span>
                </DropdownMenuItem>
              )}
              {onRename && (
                <DropdownMenuItem
                  onClick={() => setRenaming(item)}
                  disabled={busy}
                >
                  <Pencil className="size-4" />
                  <span>{t("workspace.groups.rename")}</span>
                </DropdownMenuItem>
              )}
              {(onRename || onDefaultGroup) && onDelete && (
                <DropdownMenuSeparator />
              )}
              {onDelete && (
                <DropdownMenuItem
                  variant="destructive"
                  onClick={() => setDeleting(item)}
                  disabled={busy}
                >
                  <Trash2 className="size-4" />
                  <span>{t("workspace.groups.delete")}</span>
                </DropdownMenuItem>
              )}
            </DropdownMenuContent>
          </DropdownMenu>
        )}
      </div>
    );
  }

  return (
    <>
      <aside
        aria-label={t("workspace.groups.label")}
        className="flex h-full w-[168px] shrink-0 flex-col rounded-pane border bg-card/50 p-2 md:w-[184px]"
      >
        <div className="flex shrink-0 items-center gap-1 px-1.5 pt-0.5 pb-2">
          <span className="min-w-0 flex-1 truncate text-xs font-medium text-muted-foreground">
            {t("workspace.groups.label")}
          </span>
          {onCreate && (
            <button
              type="button"
              title={t("workspace.groups.new")}
              aria-label={t("workspace.groups.new")}
              disabled={busy}
              onClick={() => setCreating(true)}
              className="flex size-6 shrink-0 cursor-pointer items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-control hover:text-foreground disabled:opacity-50"
            >
              <Plus className="size-3.5" />
            </button>
          )}
          <button
            type="button"
            title={t("workspace.groups.sidebar.hide")}
            aria-label={t("workspace.groups.sidebar.hide")}
            onClick={onCollapse}
            className="flex size-6 shrink-0 cursor-pointer items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-control hover:text-foreground"
          >
            <PanelLeftClose className="size-3.5" />
          </button>
        </div>

        <ScrollArea className="min-h-0 flex-1">
          <nav className="flex flex-col gap-0.5">
            <SelectionGroup id="group-sidebar">
              <SidebarFilterRow
                icon={Layers}
                label={t("workspace.groups.all")}
                count={items.length}
                active={group === everyGroup}
                onClick={() => onGroup(everyGroup)}
              />
              {((ungroupedCount > 0 && groups.length > 0) ||
                group === ungroupedGroup) && (
                <SidebarFilterRow
                  icon={Inbox}
                  label={t("workspace.groups.ungrouped")}
                  count={ungroupedCount}
                  active={group === ungroupedGroup}
                  onClick={() => onGroup(ungroupedGroup)}
                />
              )}

              {groups.length > 0 && (
                <div className="my-1.5 h-px bg-border/70" />
              )}

              {populated.map(({ item, count }) => renderGroupRow(item, count))}

              {!groups.length && (
                <div className="mt-2 flex flex-col items-start gap-2 px-2 py-2 text-xs text-muted-foreground">
                  <span>{t("workspace.groups.empty")}</span>
                  {onCreate && (
                    <button
                      type="button"
                      disabled={busy}
                      onClick={() => setCreating(true)}
                      className="inline-flex cursor-pointer items-center gap-1 rounded-md bg-control px-2 py-1 text-[11px] font-medium text-foreground transition-colors hover:bg-secondary disabled:opacity-50"
                    >
                      <Plus className="size-3" />
                      {t("workspace.groups.new")}
                    </button>
                  )}
                </div>
              )}

              {empty.length > 0 && (
                <div className="mt-1.5">
                  <button
                    type="button"
                    onClick={() => setShowEmpty((current) => !current)}
                    aria-expanded={showEmpty}
                    className="flex w-full cursor-pointer items-center gap-1.5 rounded-md px-2 py-1 text-left text-[11px] text-faint transition-colors hover:text-muted-foreground"
                  >
                    <ChevronRight
                      className={cn(
                        "size-3 shrink-0 transition-transform",
                        showEmpty && "rotate-90",
                      )}
                    />
                    <span className="truncate">
                      {t("workspace.groups.emptyCount", {
                        count: empty.length,
                      })}
                    </span>
                  </button>
                  {showEmpty && (
                    <div className="mt-0.5 flex flex-col gap-0.5">
                      {empty.map(({ item, count }) =>
                        renderGroupRow(item, count),
                      )}
                    </div>
                  )}
                </div>
              )}
            </SelectionGroup>
          </nav>
        </ScrollArea>
      </aside>

      <GroupDialogs
        group={group}
        onGroup={onGroup}
        creating={creating}
        renaming={renaming}
        deleting={deleting}
        onCloseCreate={() => setCreating(false)}
        onCloseRename={() => setRenaming(null)}
        onCloseDelete={() => setDeleting(null)}
        onStartDelete={setDeleting}
        onCreate={onCreate}
        onRename={onRename}
        onDelete={onDelete}
        busy={busy}
      />
    </>
  );
}

function SidebarFilterRow({
  icon: Icon,
  label,
  count,
  active,
  onClick,
}: {
  icon: React.ComponentType<{ className?: string }>;
  label: string;
  count: number;
  active: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-current={active ? "true" : undefined}
      className={cn(
        "relative flex h-8 w-full cursor-pointer items-center gap-2 rounded-lg px-2 text-left text-[12.5px] outline-none transition-colors focus-visible:ring-2 focus-visible:ring-ring/40",
        active
          ? "text-foreground"
          : "text-muted-foreground hover:bg-control hover:text-foreground",
      )}
    >
      {active && (
        <SelectionIndicator className="rounded-lg bg-raised inset-ring inset-ring-foreground/10" />
      )}
      <Icon
        className={cn(
          aboveIndicator,
          "size-3.5 shrink-0",
          active ? "text-foreground" : "opacity-70",
        )}
      />
      <span className={cn(aboveIndicator, "min-w-0 flex-1 truncate")}>
        {label}
      </span>
      <span className={cn(aboveIndicator, "text-[11px] text-faint")}>
        {count}
      </span>
    </button>
  );
}
