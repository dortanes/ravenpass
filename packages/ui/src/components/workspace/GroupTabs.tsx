import { cn } from "cn";
import {
  Check,
  ChevronDown,
  Folder,
  Inbox,
  Layers,
  Pencil,
  Plus,
  Star,
  Trash2,
  X,
} from "lucide-react";
import { useState } from "react";
import { useTranslator } from "../../i18n/translator.tsx";
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
import { GroupDialogs } from "./GroupDialogs.tsx";

/** GroupTabs picks one group (or ungrouped/all) on compact layouts where the sidebar is absent. */
export function GroupTabs({
  groups,
  group,
  onGroup,
  items,
  defaultGroup = "",
  onCreate,
  onRename,
  onDelete,
  onDefaultGroup,
  busy = false,
}: {
  groups: Group[];
  group: string;
  onGroup: (group: string) => void;
  /** The items the picker counts: those of the open place. */
  items: readonly ListedItem[];
  defaultGroup?: string;
  onCreate?: (name: string) => Promise<string | null>;
  onRename?: (id: string, name: string) => Promise<boolean>;
  onDelete?: (id: string) => void;
  onDefaultGroup?: (id: string) => void;
  busy?: boolean;
}) {
  const { t } = useTranslator();
  const [creating, setCreating] = useState(false);
  const [renaming, setRenaming] = useState<Group | null>(null);
  const [deleting, setDeleting] = useState<Group | null>(null);

  if (!groups.length && !onCreate) return null;

  const activeGroup = groups.find((item) => item.id === group) ?? null;
  const activeIsDefault = activeGroup?.id === defaultGroup;
  const ungroupedCount = countUngrouped(items);
  const activeLabel =
    group === ungroupedGroup
      ? t("workspace.groups.ungrouped")
      : activeGroup
        ? activeGroup.name
        : t("workspace.groups.all");

  const populated = groups
    .map((item) => ({ item, count: countInGroup(items, item.id) }))
    .filter(({ item, count }) => count > 0 || group === item.id);
  const empty = groups
    .map((item) => ({ item, count: countInGroup(items, item.id) }))
    .filter(({ item, count }) => count === 0 && group !== item.id);

  return (
    <>
      <div className="flex w-full min-w-0 shrink-0 items-center gap-1.5 pb-0.5">
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <button
              type="button"
              aria-label={t("workspace.groups.label")}
              className="flex h-8 min-w-0 flex-1 cursor-pointer items-center gap-1.5 rounded-full bg-raised px-3 text-xs text-foreground inset-ring inset-ring-foreground/10 outline-none transition-colors hover:bg-control focus-visible:ring-2 focus-visible:ring-ring/40"
            >
              {group === ungroupedGroup ? (
                <Inbox className="size-3.5 shrink-0 opacity-75" />
              ) : activeGroup ? (
                <Folder className="size-3.5 shrink-0 opacity-75" />
              ) : (
                <Layers className="size-3.5 shrink-0 opacity-75" />
              )}
              <span className="min-w-0 flex-1 truncate text-left">
                {activeLabel}
              </span>
              <ChevronDown className="size-3.5 shrink-0 text-muted-foreground" />
            </button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="start" className="w-60">
            <DropdownMenuItem onClick={() => onGroup(everyGroup)}>
              <Layers className="size-4" />
              <span className="min-w-0 flex-1 truncate">
                {t("workspace.groups.all")}
              </span>
              <span className="text-xs text-muted-foreground">
                {items.length}
              </span>
              {group === everyGroup && <Check className="size-4" />}
            </DropdownMenuItem>
            {((ungroupedCount > 0 && groups.length > 0) ||
              group === ungroupedGroup) && (
              <DropdownMenuItem onClick={() => onGroup(ungroupedGroup)}>
                <Inbox className="size-4" />
                <span className="min-w-0 flex-1 truncate">
                  {t("workspace.groups.ungrouped")}
                </span>
                <span className="text-xs text-muted-foreground">
                  {ungroupedCount}
                </span>
                {group === ungroupedGroup && <Check className="size-4" />}
              </DropdownMenuItem>
            )}
            {(populated.length > 0 || empty.length > 0) && (
              <DropdownMenuSeparator />
            )}
            {populated.map(({ item, count }) => (
              <DropdownMenuItem key={item.id} onClick={() => onGroup(item.id)}>
                <Folder className="size-4" />
                <span className="min-w-0 flex-1 truncate">{item.name}</span>
                {item.id === defaultGroup && (
                  <Star className="size-3 fill-current text-muted-foreground" />
                )}
                <span className="text-xs text-muted-foreground">{count}</span>
                {group === item.id && <Check className="size-4" />}
              </DropdownMenuItem>
            ))}
            {empty.map(({ item }) => (
              <DropdownMenuItem
                key={item.id}
                onClick={() => onGroup(item.id)}
                className="text-muted-foreground"
              >
                <Folder className="size-4 opacity-60" />
                <span className="min-w-0 flex-1 truncate">{item.name}</span>
                {item.id === defaultGroup && (
                  <Star className="size-3 fill-current text-faint" />
                )}
                <span className="text-xs text-faint">0</span>
              </DropdownMenuItem>
            ))}
            {activeGroup && (onRename || onDelete || onDefaultGroup) && (
              <>
                <DropdownMenuSeparator />
                {onDefaultGroup && (
                  <DropdownMenuItem
                    onClick={() =>
                      onDefaultGroup(activeIsDefault ? "" : activeGroup.id)
                    }
                    disabled={busy}
                  >
                    <Star
                      className={cn(
                        "size-4",
                        activeIsDefault && "fill-current",
                      )}
                    />
                    <span>
                      {t(
                        activeIsDefault
                          ? "workspace.groups.unsetDefault"
                          : "workspace.groups.setDefault",
                      )}
                    </span>
                  </DropdownMenuItem>
                )}
                {onRename && (
                  <DropdownMenuItem
                    onClick={() => setRenaming(activeGroup)}
                    disabled={busy}
                  >
                    <Pencil className="size-4" />
                    <span>{t("workspace.groups.rename")}</span>
                  </DropdownMenuItem>
                )}
                {onDelete && (
                  <DropdownMenuItem
                    variant="destructive"
                    onClick={() => setDeleting(activeGroup)}
                    disabled={busy}
                  >
                    <Trash2 className="size-4" />
                    <span>{t("workspace.groups.delete")}</span>
                  </DropdownMenuItem>
                )}
              </>
            )}
          </DropdownMenuContent>
        </DropdownMenu>

        {group !== everyGroup && (
          <button
            type="button"
            title={t("workspace.groups.clear")}
            aria-label={t("workspace.groups.clear")}
            onClick={() => onGroup(everyGroup)}
            className="flex size-8 shrink-0 cursor-pointer items-center justify-center rounded-full bg-control text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground"
          >
            <X className="size-3.5" />
          </button>
        )}

        {onCreate && (
          <button
            type="button"
            title={t("workspace.groups.new")}
            aria-label={t("workspace.groups.new")}
            disabled={busy}
            onClick={() => setCreating(true)}
            className="flex size-8 shrink-0 cursor-pointer items-center justify-center rounded-full bg-control text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground disabled:opacity-50"
          >
            <Plus className="size-3.5" />
          </button>
        )}
      </div>

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
