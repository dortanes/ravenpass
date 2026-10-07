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

/** RowTags are an item's tags on its row's title line; short of room, each tag cuts its own text. */
export function RowTags({ tags }: { tags: readonly string[] }) {
  if (!tags.length) return null;
  return (
    <span className="flex min-w-0 shrink items-center gap-1 overflow-hidden">
      {tags.map((tag) => (
        <span
          key={tag}
          className="max-w-[96px] min-w-7 shrink truncate rounded-full bg-foreground/8 px-1.5 text-[10.5px] leading-4 text-secondary-foreground"
        >
          {tag}
        </span>
      ))}
    </span>
  );
}
