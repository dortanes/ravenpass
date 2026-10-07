import { useVirtualizer } from "@tanstack/react-virtual";
import { cn } from "cn";
import { LoaderCircle, Star } from "lucide-react";
import { AnimatePresence, motion } from "motion/react";
import {
  type ComponentType,
  type KeyboardEvent,
  type ReactNode,
  type RefObject,
  useCallback,
  useEffect,
  useEffectEvent,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { useRovingFocus } from "../../hooks/roving-focus.ts";
import { useTranslator } from "../../i18n/translator.tsx";
import { RowArrivals } from "../../motion/arrivals.ts";
import { SelectionGroup } from "../../motion/SelectionIndicator.tsx";
import { rowPresence, vanish } from "../../motion/timings.ts";
import type { Group } from "../../vault-api.ts";
import type { ListingPosition } from "../../workspace/list-positions.ts";
import type { ListedItem } from "../../workspace/sections.ts";
import type { SiteIconState } from "../../workspace/site-icons.ts";
import { ScrollArea } from "../ui/scroll-area.tsx";
import { Avatar } from "./Avatar.tsx";
import { GroupTabs } from "./GroupTabs.tsx";
import {
  ItemRowContent,
  itemRowClass,
  type RowDetail,
} from "./ItemRowContent.tsx";
import { useSiteIcon } from "./SiteIcons.tsx";

const estimatedRowHeight = 54;
const rowGap = 4;

/** What a row's face is drawn with. */
export interface RowFace {
  selected: boolean;
  /** What is known of the row's site icon, for a kind that has a site. */
  logo: SiteIconState | undefined;
}

/** What the open place passes every list, whatever kind of item it holds. */
export interface ListProps<Item extends ListedItem> {
  title: string;
  entries: Item[];
  /** Every item of the open place, which the group tabs count. */
  all: Item[];
  groups: Group[];
  group: string;
  onGroup: (group: string) => void;
  selectedId: string | null;
  loading: boolean;
  emptyMessage: string;
  onOpen: (id: string) => void;
  /** Where the listing the entries make was left, and where to record it once it is left. */
  position: ListingPosition;
}

/** How the rows of one kind look. */
interface RowLook<Item> {
  shape: "tile" | "circle";
  icon: ComponentType<{ className?: string }>;
  /** The row's photo in base64, for a kind that has one. */
  picture?: (entry: Item) => string;
  /** The host the row's icon is kept under, for a kind that has a site. */
  site?: (entry: Item) => string;
  /** The row's face, for a kind that draws its own in place of the avatar. */
  avatar?: (entry: Item, face: RowFace) => ReactNode;
  /** Small glyphs at the row's end, before the pin, naming what the item holds. */
  marks?: (entry: Item) => ReactNode;
  /** A warning badge leading the row's title line, unset for a row with nothing to warn of. */
  alert?: (entry: Item) => string | undefined;
}

/** What a list names its rows with, whatever kind of item they hold. */
interface RowWords<Item> {
  /** What the list is called for assistive technology. */
  label: string;
  untitled: string;
  detail: (entry: Item) => RowDetail;
}

/** ItemList is a place's list column: group tabs, the section's title and count, and the rows. */
export function ItemList<Item extends ListedItem>({
  title,
  entries,
  all,
  groups,
  group,
  onGroup,
  selectedId,
  loading,
  emptyMessage,
  onOpen,
  position,
  label,
  untitled,
  detail,
  notice,
  ...look
}: ListProps<Item> &
  RowLook<Item> &
  RowWords<Item> & {
    /** A line under the section's title saying how the whole list stands, such as a check of its items. */
    notice?: ReactNode;
  }) {
  const { t } = useTranslator();
  const [arrivals] = useState(() => new RowArrivals());

  return (
    // isolate: row and tab z-indexes must stay below overlays such as the compact New button.
    <div className="isolate flex h-full min-w-0 flex-col gap-1 max-sm:px-2">
      <GroupTabs groups={groups} group={group} onGroup={onGroup} items={all} />
      <div className="flex shrink-0 items-baseline gap-[7px] px-3 pb-1 max-sm:hidden">
        <span className="text-xs text-muted-foreground">{title}</span>
        {!loading && (
          <span className="text-[11px] text-faint">{entries.length}</span>
        )}
      </div>
      {notice && <div className="shrink-0 px-3 pb-1">{notice}</div>}
      {loading && (
        <div
          className="flex items-center gap-2 px-3 py-2 text-[13px] text-muted-foreground"
          role="status"
        >
          <LoaderCircle
            className="size-4 animate-spin motion-reduce:animate-none"
            aria-hidden="true"
          />
          {t("workspace.list.loading")}
        </div>
      )}
      {!loading && !entries.length && (
        <p className="px-3 py-2 text-[13px] text-muted-foreground">
          {emptyMessage}
        </p>
      )}
      <ListRows
        key={position.key}
        entries={entries}
        groupsOf={(entry) =>
          groups
            .filter(
              (held) => held.id !== group && entry.groups.includes(held.id),
            )
            .map((held) => held.name)
        }
        selectedId={selectedId}
        onOpen={onOpen}
        position={position}
        arrivals={arrivals}
        words={{ label, untitled, detail }}
        look={look}
      />
    </div>
  );
}

/** ListRows is one listing's virtualized rows; it restores and records the listing's scroll position. */
function ListRows<Item extends ListedItem>({
  entries,
  groupsOf,
  selectedId,
  onOpen,
  position,
  arrivals,
  words,
  look,
}: {
  entries: Item[];
  /** The names of the groups a row shows, leaving out the group the list is narrowed to. */
  groupsOf: (entry: Item) => string[];
  selectedId: string | null;
  onOpen: (id: string) => void;
  position: ListingPosition;
  arrivals: RowArrivals;
  words: RowWords<Item>;
  look: RowLook<Item>;
}) {
  const viewport = useRef<HTMLDivElement>(null);
  const list = useRef<HTMLElement>(null);
  // The row the arrow keys moved to, until the scroll they started has built it.
  const pendingFocus = useRef<string | null>(null);
  const [left] = useState(() => position.recall());
  const listed = useMemo(
    () => new Set(entries.map((entry) => entry.id)),
    [entries],
  );
  // The virtualizer recomputes its layout whenever this changes identity.
  const rowKey = useCallback(
    (index: number) => entries[index]?.id ?? index,
    [entries],
  );

  const rows = useVirtualizer({
    count: entries.length,
    getScrollElement: () => viewport.current,
    estimateSize: () => estimatedRowHeight,
    gap: rowGap,
    overscan: 0,
    getItemKey: rowKey,
    initialMeasurementsCache: left.measured,
  });
  const settled = !rows.isScrolling;

  const leave = useEffectEvent(() => {
    position.remember({
      offset: rows.scrollOffset ?? 0,
      measured: rows.takeSnapshot(),
    });
  });

  // Must follow useVirtualizer, whose effects attach the viewport first.
  useLayoutEffect(() => {
    if (left.offset > 0) rows.scrollToOffset(left.offset);
    return () => leave();
  }, [rows, left]);

  function focusRow(id: string): boolean {
    const row = list.current?.querySelector<HTMLButtonElement>(
      `[data-row="${CSS.escape(id)}"]`,
    );
    row?.focus({ preventScroll: true });
    return Boolean(row);
  }

  useEffect(() => {
    if (pendingFocus.current && focusRow(pendingFocus.current)) {
      pendingFocus.current = null;
    }
  });

  const moveFocus = useRovingFocus({
    count: entries.length,
    loop: true,
    focus: (index) => {
      const entry = entries[index];
      if (!entry) return;
      rows.scrollToIndex(index);
      pendingFocus.current = focusRow(entry.id) ? null : entry.id;
    },
  });

  return (
    <ScrollArea className="min-h-0 flex-1" viewportRef={viewport}>
      <nav
        ref={list}
        className="relative shrink-0 max-sm:mb-16"
        style={{ height: rows.getTotalSize() }}
        aria-label={words.label}
      >
        <SelectionGroup id="item-list">
          <AnimatePresence custom={listed}>
            {rows.getVirtualItems().map((row) => {
              const entry = entries[row.index];
              if (!entry) return null;
              const active = entry.id === selectedId;
              return (
                <ArrivingRow
                  key={row.key}
                  id={entry.id}
                  index={row.index}
                  top={row.start}
                  root={viewport}
                  arrivals={arrivals}
                  measure={rows.measureElement}
                  active={active}
                  onOpen={onOpen}
                  onKeyDown={(event) => moveFocus(event, row.index)}
                >
                  <ItemRowContent
                    active={active}
                    avatar={
                      <RowAvatar
                        entry={entry}
                        selected={active}
                        settled={settled}
                        look={look}
                      />
                    }
                    title={entry.label || words.untitled}
                    alert={look.alert?.(entry)}
                    groups={groupsOf(entry)}
                    tags={entry.tags}
                    detail={words.detail(entry)}
                    trailing={
                      <RowEnd
                        marks={look.marks?.(entry)}
                        pinned={entry.pinned}
                      />
                    }
                  />
                </ArrivingRow>
              );
            })}
          </AnimatePresence>
        </SelectionGroup>
      </nav>
    </ScrollArea>
  );
}

function RowEnd({ marks, pinned }: { marks: ReactNode; pinned: boolean }) {
  const { t } = useTranslator();
  if (!marks && !pinned) return null;
  return (
    <span className="flex items-center gap-1.5">
      {marks}
      {pinned && (
        <Star
          className="size-3.5 fill-current text-foreground/70"
          aria-label={t("workspace.list.pinned")}
        />
      )}
    </span>
  );
}

/** RowAvatar requests the row's site icon only while the list is not scrolling. */
function RowAvatar<Item extends ListedItem>({
  entry,
  selected,
  settled,
  look,
}: {
  entry: Item;
  selected: boolean;
  settled: boolean;
  look: RowLook<Item>;
}) {
  const logo = useSiteIcon(look.site?.(entry) ?? "", settled);
  if (look.avatar) return look.avatar(entry, { selected, logo });
  return (
    <Avatar
      label={entry.label}
      photo={look.picture?.(entry)}
      logo={logo}
      icon={look.icon}
      size="row"
      shape={look.shape}
      emphasis={selected ? "selected" : "none"}
    />
  );
}

/** ArrivingRow plays its entrance only the first time the open list shows it. */
function ArrivingRow({
  id,
  index,
  top,
  root,
  arrivals,
  measure,
  active,
  onOpen,
  onKeyDown,
  children,
}: {
  id: string;
  index: number;
  top: number;
  /** The element the list scrolls in, which decides when the row is in view. */
  root: RefObject<HTMLDivElement | null>;
  arrivals: RowArrivals;
  measure: (element: HTMLButtonElement | null) => void;
  active: boolean;
  onOpen: (id: string) => void;
  onKeyDown: (event: KeyboardEvent<HTMLButtonElement>) => void;
  children: ReactNode;
}) {
  const [seen] = useState(() => arrivals.seenBefore(id));
  // The row's place among the rows that came into view with it, once it has.
  const [order, setOrder] = useState<number | null>(null);
  const presence = rowPresence(order ?? 0);

  return (
    <motion.button
      ref={measure}
      type="button"
      data-row={id}
      data-index={index}
      aria-current={active ? "true" : undefined}
      onClick={() => onOpen(id)}
      onKeyDown={onKeyDown}
      initial={seen ? false : presence.initial}
      animate={seen || order !== null ? presence.animate : undefined}
      exit="leave"
      variants={{
        leave: (listed: ReadonlySet<string>) =>
          listed.has(id) ? vanish : presence.exit,
      }}
      viewport={seen ? undefined : { root, once: true }}
      onViewportEnter={seen ? undefined : () => setOrder(arrivals.arrive(id))}
      style={{ top }}
      className={cn(itemRowClass(!active), "absolute inset-x-0")}
    >
      {children}
    </motion.button>
  );
}
