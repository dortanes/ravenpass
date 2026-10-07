import { cn } from "cn";

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
  "max-w-[96px] min-w-7 shrink truncate rounded-full px-1.5 text-[10.5px] leading-4 text-secondary-foreground";

/**
 * RowBadges are an item's group names, filled, then its tags, outlined, on its row's title line; short of room, each
 * badge cuts its own text.
 */
export function RowBadges({
  groups,
  tags,
}: {
  groups: readonly string[];
  tags: readonly string[];
}) {
  if (!groups.length && !tags.length) return null;
  return (
    <span className="flex min-w-0 shrink items-center gap-1 overflow-hidden">
      {groups.map((name) => (
        <span key={`group:${name}`} className={cn(rowBadge, "bg-foreground/8")}>
          {name}
        </span>
      ))}
      {tags.map((tag) => (
        <span
          key={`tag:${tag}`}
          className={cn(rowBadge, "border border-foreground/12")}
        >
          {tag}
        </span>
      ))}
    </span>
  );
}
