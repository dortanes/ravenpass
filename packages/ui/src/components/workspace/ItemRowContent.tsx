import { cn } from "cn";
import { AnimatePresence } from "motion/react";
import type { ReactNode } from "react";
import {
  aboveIndicator,
  SelectionIndicator,
} from "../../motion/SelectionIndicator.tsx";
import { RowTags } from "./TagBadge.tsx";
import { type Tone, toneText } from "./tones.ts";

/** The second line of a row, and whether it carries a warning. */
export interface RowDetail {
  text: string;
  tone?: Tone;
}

/** Where a row is drawn: a workspace list, or the browser extension's menu on a web page. */
export type ItemRowLook = "list" | "menu";

const looks: Record<
  ItemRowLook,
  {
    row: string;
    selection: string;
    title: string;
    detail: string;
    /** The title's and the detail's line when they are values glanced at. */
    line: string;
    /** Whether the selection fades out when no row takes it over. */
    fades: boolean;
  }
> = {
  list: {
    row: "rounded-row border border-transparent bg-card px-[11px] py-2 focus-visible:border-ring max-sm:min-h-[46px]",
    selection:
      "-inset-px rounded-row border border-foreground/12 bg-raised group-focus-visible:border-ring",
    title: "text-[13px]",
    detail: "text-[11px]",
    line: "truncate",
    fades: false,
  },
  menu: {
    row: "min-h-[46px] rounded-[10px] px-2 py-1.5",
    selection:
      "rounded-[10px] bg-foreground/7.5 inset-ring inset-ring-foreground/4",
    title: "text-[13px] font-medium",
    detail: "mt-px text-[11.5px]",
    line: "truncate leading-4",
    fades: true,
  },
};

/** itemRowClass is the class of an item row's element; `hover` is false for the active row. */
export function itemRowClass(
  hover: boolean,
  look: ItemRowLook = "list",
): string {
  return cn(
    "group flex items-center gap-[11px] text-left outline-none",
    looks[look].row,
    hover && "hover:bg-control",
  );
}

/** ItemRowContent fills a positioned element that carries `itemRowClass` of the same look. */
export function ItemRowContent({
  active,
  avatar,
  title,
  tags = [],
  detail,
  trailing,
  wrap = false,
  look = "list",
}: {
  active: boolean;
  avatar: ReactNode;
  title: string;
  /** The item's tags, shown beside its title. */
  tags?: readonly string[];
  /** Unset for a row that names an action, which the title alone says. */
  detail?: RowDetail;
  trailing?: ReactNode;
  /** Set when the title and detail are sentences, which wrap. */
  wrap?: boolean;
  look?: ItemRowLook;
}) {
  const style = looks[look];
  const line = wrap ? "leading-[1.4]" : style.line;
  const selection = active && (
    <SelectionIndicator key="selection" className={style.selection} />
  );
  return (
    <>
      {style.fades ? <AnimatePresence>{selection}</AnimatePresence> : selection}
      <span className={`${aboveIndicator} flex shrink-0`}>{avatar}</span>
      <span className={`${aboveIndicator} min-w-0 flex-1`}>
        {tags.length > 0 ? (
          <span
            className={cn("flex min-w-0 items-center gap-1.5", style.title)}
          >
            <span className={cn("min-w-0 shrink-[2]", line)}>{title}</span>
            <RowTags tags={tags} />
          </span>
        ) : (
          <span className={cn("block", style.title, line)}>{title}</span>
        )}
        {detail && (
          <span
            className={cn(
              "block",
              style.detail,
              line,
              toneText[detail.tone ?? "default"],
            )}
          >
            {detail.text}
          </span>
        )}
      </span>
      {trailing && (
        <span className={`${aboveIndicator} flex shrink-0`}>{trailing}</span>
      )}
    </>
  );
}
