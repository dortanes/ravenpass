import { cn } from "cn";
import { Folder } from "lucide-react";

const badge =
  "max-w-[120px] shrink-0 truncate rounded-full border border-foreground/12 px-2 py-px text-[11px] text-secondary-foreground";

/** TagBadge names one tag of an item; given `onChoose`, it lists the items carrying the tag. */
export function TagBadge({
  tag,
  label,
  onChoose,
}: {
  tag: string;
  /** What choosing the tag does, for assistive technology and the pointer's tooltip. */
  label?: string;
  onChoose?: (tag: string) => void;
}) {
  if (!onChoose) return <span className={badge}>{tag}</span>;
  return (
    <button
      type="button"
      className={cn(
        badge,
        "outline-none hover:bg-control hover:text-foreground focus-visible:ring-[3px] focus-visible:ring-ring/40",
      )}
      title={label}
      aria-label={label}
      onClick={() => onChoose(tag)}
    >
      {tag}
    </button>
  );
}

const rowBadge =
  "max-w-[78px] min-w-6 shrink truncate rounded-full px-1.5 text-[10.5px] leading-4 text-secondary-foreground";

/**
 * RowBadges show at most one group and overflow count, then tags, on its row's title line; short of room, badges
 * yield to the item title before cutting their own text.
 */
export function RowBadges({
  groups,
  tags,
}: {
  groups: readonly string[];
  tags: readonly string[];
}) {
  if (!groups.length && !tags.length) return null;
  const visibleGroups = groups.slice(0, 1);
  const extraGroups = groups.length - visibleGroups.length;
  const visibleTags = tags.slice(0, extraGroups > 0 ? 1 : 2);
  const extraTags = tags.length - visibleTags.length;
  return (
    <span className="flex min-w-0 shrink-[2] items-center gap-1 overflow-hidden">
      {visibleGroups.map((name) => (
        <span
          key={`group:${name}`}
          title={groups.join(", ")}
          className={cn(
            rowBadge,
            "inline-flex items-center gap-1 bg-foreground/8 px-1.5",
          )}
        >
          <Folder className="size-2.5 shrink-0 opacity-60" />
          <span className="truncate">{name}</span>
        </span>
      ))}
      {extraGroups > 0 && (
        <span
          title={groups.slice(1).join(", ")}
          className="shrink-0 rounded-full bg-foreground/8 px-1.5 text-[10.5px] leading-4 text-muted-foreground"
        >
          +{extraGroups}
        </span>
      )}
      {visibleTags.map((tag) => (
        <span
          key={`tag:${tag}`}
          className={cn(rowBadge, "border border-foreground/12")}
        >
          {tag}
        </span>
      ))}
      {extraTags > 0 && (
        <span
          title={tags.slice(visibleTags.length).join(", ")}
          className="shrink-0 rounded-full border border-foreground/12 px-1.5 text-[10.5px] leading-4 text-muted-foreground"
        >
          +{extraTags}
        </span>
      )}
    </span>
  );
}
