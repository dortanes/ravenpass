import { cn } from "cn";
import { Copy, Pencil, Star, Trash2 } from "lucide-react";
import type { ComponentType, ReactNode } from "react";
import { useTranslator } from "../../i18n/translator.tsx";
import type { Group } from "../../vault-api.ts";
import type { SiteIconState } from "../../workspace/site-icons.ts";
import { ConfirmDialog } from "../ConfirmDialog.tsx";
import { Button } from "../ui/button.tsx";
import { Avatar } from "./Avatar.tsx";
import { GroupBadge } from "./GroupBadge.tsx";
import { TagBadge } from "./TagBadge.tsx";

/** What an open item's pane can do with the item as a whole, whatever its kind. */
export interface DetailControls {
  /** The groups this item belongs to, named. */
  groups: Group[];
  tags: string[];
  /** Lists the items carrying a tag. */
  onTag: (tag: string) => void;
  pinned: boolean;
  busy: boolean;
  confirmDelete: boolean;
  onAskDelete: () => void;
  onCancelDelete: () => void;
  onDelete: () => void;
  onDuplicate: () => void;
  onEdit: () => void;
  onTogglePin: () => void;
}

const paneActionClass =
  "size-[26px] rounded-md bg-raised hover:bg-tile max-sm:size-9";

/** DetailHeader names an open item and holds its actions; deleting asks first. */
export function DetailHeader({
  label,
  mark,
  photo,
  logo,
  avatar,
  title,
  icon,
  subtitle,
  deleteTitle,
  controls: {
    groups,
    tags,
    onTag,
    pinned,
    busy,
    confirmDelete,
    onAskDelete,
    onCancelDelete,
    onDelete,
    onDuplicate,
    onEdit,
    onTogglePin,
  },
}: {
  /** The item's own name, which the avatar takes its letter from. */
  label: string;
  /** As in `Avatar`. */
  mark?: string;
  /** The item's photo in base64, for a kind that has one. */
  photo?: string;
  /** What is known of the item's logo, for a kind that has one. */
  logo?: SiteIconState;
  /** The item's face, for a kind that draws its own in place of the avatar. */
  avatar?: ReactNode;
  /** The name shown, which stands in for a missing label. */
  title: string;
  icon: ComponentType<{ className?: string }>;
  subtitle?: ReactNode;
  deleteTitle: string;
  controls: DetailControls;
}) {
  const { t } = useTranslator();

  return (
    <>
      <header className="flex shrink-0 items-center gap-[11px]">
        {avatar ?? (
          <Avatar
            label={label}
            mark={mark}
            photo={photo}
            logo={logo}
            icon={icon}
            size="header"
            shape="circle"
            emphasis="fill"
          />
        )}
        <span className="min-w-0 flex-1">
          <h2 className="truncate text-[17px]">{title}</h2>
          <p className="flex min-w-0 items-center gap-1.5 text-[11px] text-muted-foreground">
            {subtitle && <span className="truncate">{subtitle}</span>}
            {groups.map((group) => (
              <GroupBadge key={group.id} name={group.name} />
            ))}
            {tags.map((tag) => (
              <TagBadge
                key={tag}
                tag={tag}
                label={t("workspace.tags.show", { tag })}
                onChoose={onTag}
              />
            ))}
          </p>
        </span>
        <span className="flex shrink-0 items-center gap-1">
          <PaneAction
            label={t("workspace.detail.edit")}
            icon={Pencil}
            disabled={busy}
            onClick={onEdit}
          />
          <PaneAction
            label={t(
              pinned ? "workspace.detail.unpin" : "workspace.detail.pin",
            )}
            icon={pinned ? PinnedStar : Star}
            disabled={busy}
            onClick={onTogglePin}
          />
          <PaneAction
            label={t("workspace.detail.duplicate")}
            icon={Copy}
            disabled={busy}
            onClick={onDuplicate}
          />
          <PaneAction
            label={t("workspace.detail.delete")}
            icon={Trash2}
            disabled={busy}
            destructive
            onClick={onAskDelete}
          />
        </span>
      </header>

      <ConfirmDialog
        open={confirmDelete}
        title={deleteTitle}
        detail={t("workspace.delete.detail")}
        confirm={t(busy ? "workspace.delete.busy" : "workspace.delete.confirm")}
        cancel={t("workspace.delete.cancel")}
        destructive
        busy={busy}
        onConfirm={onDelete}
        onCancel={onCancelDelete}
      />
    </>
  );
}

function PinnedStar({ className }: { className?: string }) {
  return <Star className={cn(className, "fill-current")} />;
}

export function PaneAction({
  label,
  icon: Icon,
  disabled,
  destructive = false,
  onClick,
}: {
  label: string;
  icon: ComponentType<{ className?: string }>;
  disabled: boolean;
  /** Set for an action that removes something, which takes the destructive hue. */
  destructive?: boolean;
  onClick: () => void;
}) {
  return (
    <Button
      type="button"
      variant="ghost"
      size="icon"
      aria-label={label}
      title={label}
      disabled={disabled}
      onClick={onClick}
      className={cn(
        paneActionClass,
        destructive && "text-destructive hover:text-destructive",
      )}
    >
      <Icon className="size-[15px]" />
    </Button>
  );
}
