import { cn } from "cn";
import { Folder } from "lucide-react";

/** GroupBadge names one group an item belongs to; given `onChoose`, it narrows the list to that group. */
export function GroupBadge({
  name,
  label,
  onChoose,
}: {
  name: string;
  /** What choosing the group does, for assistive technology and the pointer's tooltip. */
  label?: string;
  onChoose?: () => void;
}) {
  const badge =
    "inline-flex shrink-0 items-center gap-1 rounded-full bg-foreground/8 px-2 py-px text-[11px] text-secondary-foreground";
  if (!onChoose) {
    return (
      <span className={badge}>
        <Folder className="size-2.5 shrink-0 opacity-60" />
        <span className="truncate">{name}</span>
      </span>
    );
  }
  return (
    <button
      type="button"
      title={label}
      aria-label={label}
      className={cn(
        badge,
        "cursor-pointer outline-none hover:bg-control hover:text-foreground focus-visible:ring-[3px] focus-visible:ring-ring/40",
      )}
      onClick={onChoose}
    >
      <Folder className="size-2.5 shrink-0 opacity-60" />
      <span className="truncate">{name}</span>
    </button>
  );
}
