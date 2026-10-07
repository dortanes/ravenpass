import { useQuery } from "@tanstack/react-query";
import { Import, LoaderCircle, Plus } from "lucide-react";
import {
  type ComponentType,
  type ReactNode,
  type Ref,
  useEffect,
  useEffectEvent,
  useImperativeHandle,
  useLayoutEffect,
  useRef,
  useState,
} from "react";
import { useDefaultLayout } from "react-resizable-panels";
import { toast } from "sonner";
import { useCompactLayout } from "../../host/compact.ts";
import { formatDelay } from "../../i18n/language.ts";
import type { MessageKey } from "../../i18n/messages.ts";
import { useTranslator } from "../../i18n/translator.tsx";
import { CrossFade } from "../../motion/CrossFade.tsx";
import { queryKeys } from "../../query/keys.ts";
import type {
  ClipboardClearing,
  Group,
  StorageSyncState,
  VaultApi,
} from "../../vault-api.ts";
import { quietStorage } from "../../workspace/browser-storage.ts";
import type { ListingPosition } from "../../workspace/list-positions.ts";
import {
  everyGroup,
  type ListedItem,
  type SearchValues,
  selectEntries,
  type WorkspaceSection,
} from "../../workspace/sections.ts";
import type { SelectionQueue } from "../../workspace/selection.ts";
import type { Tagging } from "../editor/EditorFields.tsx";
import { Button } from "../ui/button.tsx";
import {
  ResizableHandle,
  ResizablePanel,
  ResizablePanelGroup,
} from "../ui/resizable.tsx";
import type { DetailControls } from "./DetailHeader.tsx";
import { EmptyPane } from "./EmptyPane.tsx";
import type { ListProps } from "./ItemList.tsx";
import { sectionLabels } from "./SectionSwitch.tsx";

type Pane = "empty" | "detail" | "new" | "edit";

/** What a place starts with when it opens: nothing, the editor for a new item, or one item. */
export type OpeningPane = "empty" | "new" | { item: string };

/** What the workspace asks of the open place from outside it. */
export interface PlaceHandle {
  /** Opens the editor for a new item. */
  create(): void;
  /** Shows one item in the detail pane. */
  open(id: string): void;
  /** Rereads the item the detail pane shows; an open editor is left alone. */
  reload(): void;
  /** Closes an open editor, keeping an open item. */
  leaveEditor(): void;
  /** Closes whatever the place shows, and the host's open item with it. */
  close(): void;
  /** Drops what the place shows without asking the host, which is about to lock. */
  forget(): void;
}

/** What the shell shares with every place. */
export interface PlaceShell {
  api: VaultApi;
  groups: Group[];
  group: string;
  onGroup: (group: string) => void;
  defaultGroup: string;
  /** Every tag the vault's items carry, which editors suggest. */
  tags: string[];
  /** Lists the items of the open place carrying a tag. */
  onTag: (tag: string) => void;
  section: WorkspaceSection;
  query: string;
  clipboard: ClipboardClearing | null;
  busy: boolean;
  setBusy: (busy: boolean) => void;
  selection: SelectionQueue;
  setSyncState: (state: StorageSyncState) => void;
  /** Records how a failed write left the vault against its file. */
  settleSync: (cause: unknown) => void;
  report: (cause: unknown, message: MessageKey) => void;
  refreshExportState: () => Promise<void>;
  /** Hears whether the place shows an item or an editor. */
  onPaneShown: (shown: boolean) => void;
  /** Where the place's list was left for the section, group and search it now shows. */
  listPosition: ListingPosition;
}

/** The copy a place shows in its own words. */
export interface PlaceMessages {
  loading: MessageKey;
  detailLoading: MessageKey;
  selectTitle: MessageKey;
  selectDetail: MessageKey;
  firstTitle: MessageKey;
  firstDetail: MessageKey;
  firstAction: MessageKey;
  emptyAll: MessageKey;
  emptyRecent: MessageKey;
  emptyPinned: MessageKey;
  emptySearch: MessageKey;
  errorClose: MessageKey;
  errorOpen: MessageKey;
  errorCopy: MessageKey;
  errorPin: MessageKey;
  errorUnpin: MessageKey;
  errorSave: MessageKey;
  errorSavedPartly: MessageKey;
  errorDelete: MessageKey;
  errorDeletedPartly: MessageKey;
  errorDuplicate: MessageKey;
}

/** How a place reads and writes the one kind of item it holds. */
export interface ItemKind<Item, Input> {
  icon: ComponentType<{ className?: string }>;
  messages: PlaceMessages;
  read(id: string): Promise<Item>;
  create(input: Input, groups: string[]): Promise<string>;
  update(id: string, input: Input, groups: string[]): Promise<void>;
}

/** Copies one value through the host and says so, with when the clipboard clears. */
export type CopyAction = (
  copy: () => Promise<void>,
  notice: MessageKey,
) => void;

/** What a change to the open item reports on failure and announces once written. */
export interface ChangeMessages {
  failure: MessageKey;
  notice?: MessageKey;
  copies?: boolean;
}

/** Writes one change to the open item and rereads it; resolves whether the change was written. */
export type ChangeAction = (
  change: () => Promise<void>,
  messages: ChangeMessages,
) => Promise<boolean>;

export interface EditorProps<Item, Input> {
  initial?: Item;
  /** The groups the item starts in, which for a new one is the chosen default. */
  initialGroups: string[];
  tagging: Tagging;
  onSave: (input: Input, groups: string[]) => void;
  onCancel: () => void;
}

/** Storage key of the list width every place shares on the device. */
const columnsLayout = "ravenpass.columns";

/** ItemPlace is a list of one kind of item beside a pane that shows, creates or edits one. */
export function ItemPlace<
  Summary extends ListedItem,
  Item extends { id: string; groups: string[] },
  Input,
>({
  shell,
  kind,
  entries,
  loading,
  refresh,
  searchValues,
  opening,
  ref,
  list,
  detail,
  editor,
  onImport,
}: {
  shell: PlaceShell;
  kind: ItemKind<Item, Input>;
  entries: Summary[];
  loading: boolean;
  refresh: () => Promise<void>;
  searchValues: SearchValues<Summary>;
  opening: OpeningPane;
  ref?: Ref<PlaceHandle>;
  /** Set when the place offers to import items from another app while it holds none. */
  onImport?: () => void;
  list: (props: ListProps<Summary>) => ReactNode;
  detail: (
    item: Item,
    controls: DetailControls,
    copy: CopyAction,
    change: ChangeAction,
  ) => ReactNode;
  editor: (props: EditorProps<Item, Input>) => ReactNode;
}) {
  const { t, failure, language } = useTranslator();
  const compact = useCompactLayout();
  const columns = useDefaultLayout({
    id: columnsLayout,
    storage: quietStorage,
  });
  const {
    api,
    groups,
    group,
    busy,
    setBusy,
    selection,
    setSyncState,
    settleSync,
    report,
    refreshExportState,
    onPaneShown,
  } = shell;
  const { messages } = kind;
  const { data: tagLimits = null } = useQuery({
    queryKey: queryKeys.limits("tags"),
    queryFn: () => api.tagLimits(),
    meta: { failure: "workspace.error.read" },
  });
  const tagging: Tagging = { known: shell.tags, limits: tagLimits };
  const [pane, setPane] = useState<Pane>(
    typeof opening === "string" ? opening : "empty",
  );
  const paneShown = pane !== "empty";
  const [selected, setSelected] = useState<Item | null>(null);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [loadingDetail, setLoadingDetail] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState(false);
  const selectionRequest = useRef(0);

  function forget() {
    selectionRequest.current += 1;
    setSelected(null);
    setSelectedId(null);
    setConfirmDelete(false);
  }

  function leaveDetail(next: Pane) {
    forget();
    setPane(next);
    selection.close().catch((cause) => {
      report(cause, messages.errorClose);
    });
  }

  useImperativeHandle(ref, () => ({
    create: () => leaveDetail("new"),
    open: (id) => void open(id),
    reload: () => {
      if (pane === "detail" && selected && !loadingDetail)
        void reload(selected.id);
    },
    leaveEditor: () => {
      if (pane === "new" || pane === "edit") leaveDetail("empty");
    },
    close: () => leaveDetail("empty"),
    forget,
  }));

  const openInitial = useEffectEvent(() => {
    if (typeof opening !== "string") void open(opening.item);
  });

  useEffect(() => {
    openInitial();
  }, []);

  // Layout effect: the workspace's head and foot must leave in the same frame as the list.
  useLayoutEffect(() => {
    onPaneShown(paneShown);
    return () => onPaneShown(false);
  }, [onPaneShown, paneShown]);

  async function open(id: string) {
    // Reopening the shown item would drop its selection ticket.
    if (pane === "detail" && selectedId === id) return;
    const request = ++selectionRequest.current;
    setSelected(null);
    setSelectedId(id);
    setPane("detail");
    setLoadingDetail(true);
    setConfirmDelete(false);
    try {
      await selection.settled();
      if (request !== selectionRequest.current) return;
      const item = await kind.read(id);
      if (request === selectionRequest.current) {
        setSelected(item);
        await refresh();
      }
    } catch (cause) {
      if (request === selectionRequest.current) {
        report(cause, messages.errorOpen);
      }
    } finally {
      if (request === selectionRequest.current) setLoadingDetail(false);
    }
  }

  async function reload(id: string) {
    const request = selectionRequest.current;
    try {
      await selection.settled();
      const item = await kind.read(id);
      if (request === selectionRequest.current) setSelected(item);
    } catch (cause) {
      if (request === selectionRequest.current) {
        report(cause, messages.errorOpen);
      }
    }
  }

  function announce(notice: MessageKey, copies: boolean) {
    toast.success(t(notice), {
      duration: 2000,
      description:
        copies && shell.clipboard?.enabled
          ? t("workspace.copy.clears", {
              delay: formatDelay(shell.clipboard.seconds, language, "short"),
            })
          : undefined,
    });
  }

  async function copy(run: () => Promise<void>, notice: MessageKey) {
    setBusy(true);
    try {
      await run();
      announce(notice, true);
      await refresh();
    } catch (cause) {
      report(cause, messages.errorCopy);
    } finally {
      setBusy(false);
    }
  }

  async function change(
    run: () => Promise<void>,
    outcome: ChangeMessages,
  ): Promise<boolean> {
    if (!selected) return false;
    const { id } = selected;
    const request = selectionRequest.current;
    setBusy(true);
    setSyncState("writing");
    let written = false;
    try {
      await run();
      written = true;
      setSyncState("synced");
      if (outcome.notice) announce(outcome.notice, outcome.copies ?? false);
      const item = await kind.read(id);
      if (request === selectionRequest.current) setSelected(item);
      await refresh();
      await refreshExportState();
      return true;
    } catch (cause) {
      if (written) {
        report(cause, messages.errorOpen);
        return true;
      }
      settleSync(cause);
      report(cause, outcome.failure);
      return false;
    } finally {
      setBusy(false);
    }
  }

  async function togglePin(id: string, pinned: boolean) {
    setBusy(true);
    setSyncState("writing");
    try {
      await api.setPinned(id, pinned);
      setSyncState("synced");
      await refresh();
      await refreshExportState();
    } catch (cause) {
      settleSync(cause);
      report(cause, pinned ? messages.errorPin : messages.errorUnpin);
    } finally {
      setBusy(false);
    }
  }

  async function save(input: Input, membership: string[]) {
    setBusy(true);
    setSyncState("writing");
    let saved = false;
    try {
      let id: string;
      if (pane === "edit" && selected) {
        await kind.update(selected.id, input, membership);
        id = selected.id;
      } else {
        id = await kind.create(input, membership);
      }
      saved = true;
      setSyncState("synced");
      await refresh();
      await refreshExportState();
      await open(id);
    } catch (cause) {
      if (saved) setSyncState("synced");
      else settleSync(cause);
      if (saved) {
        leaveDetail("empty");
        toast.error(
          t(messages.errorSavedPartly, {
            detail: failure(cause, "workspace.error.refresh"),
          }),
        );
      } else {
        report(cause, messages.errorSave);
      }
    } finally {
      setBusy(false);
    }
  }

  // The copy opens once saved, as a new item does.
  async function duplicate() {
    if (!selected) return;
    setBusy(true);
    setSyncState("writing");
    let copied: string | null = null;
    try {
      copied = await api.duplicateItem(selected.id);
      setSyncState("synced");
      await refresh();
      await refreshExportState();
      await open(copied);
    } catch (cause) {
      if (copied !== null) {
        setSyncState("synced");
        leaveDetail("empty");
        toast.error(
          t(messages.errorSavedPartly, {
            detail: failure(cause, "workspace.error.refresh"),
          }),
        );
      } else {
        settleSync(cause);
        report(cause, messages.errorDuplicate);
      }
    } finally {
      setBusy(false);
    }
  }

  async function remove() {
    if (!selected) return;
    setBusy(true);
    setSyncState("writing");
    let removed = false;
    try {
      await api.deleteItem(selected.id);
      removed = true;
      setSyncState("synced");
      await refresh();
      await refreshExportState();
      leaveDetail("empty");
    } catch (cause) {
      if (removed) setSyncState("synced");
      else settleSync(cause);
      if (removed) {
        leaveDetail("empty");
        toast.error(
          t(messages.errorDeletedPartly, {
            detail: failure(cause, "workspace.error.refresh"),
          }),
        );
      } else {
        report(cause, messages.errorDelete);
      }
    } finally {
      setBusy(false);
    }
  }

  const visible = selectEntries(
    entries,
    searchValues,
    shell.section,
    shell.query,
    group,
  );
  const groupsForNew =
    group !== everyGroup
      ? [group]
      : shell.defaultGroup
        ? [shell.defaultGroup]
        : [];
  const selectedSummary = entries.find((entry) => entry.id === selectedId);
  const pinned = selectedSummary?.pinned ?? false;
  const emptyListMessage = shell.query
    ? t(messages.emptySearch)
    : shell.section === "recent"
      ? t(messages.emptyRecent)
      : shell.section === "pinned"
        ? t(messages.emptyPinned)
        : t(messages.emptyAll);
  // Keyed per item: a detail's own state, such as a reveal, must not outlive its item.
  const paneKey =
    pane === "detail" || pane === "edit"
      ? `${pane}:${selectedId}`
      : pane === "empty"
        ? `${pane}:${loading}`
        : pane;

  const listColumn = list({
    title: t(sectionLabels[shell.section]),
    entries: visible,
    all: entries,
    groups,
    group,
    onGroup: shell.onGroup,
    selectedId,
    loading,
    emptyMessage: emptyListMessage,
    onOpen: open,
    position: shell.listPosition,
  });
  const paneView = (
    <div className="flex min-h-0 min-w-0 flex-1 flex-col rounded-pane border bg-card p-4 max-sm:rounded-none max-sm:border-0 max-sm:bg-background max-sm:pt-1">
      <CrossFade id={paneKey} className="flex min-h-0 flex-1 flex-col gap-3">
        {pane === "empty" &&
          (loading ? (
            <Loading message={t(messages.loading)} />
          ) : (
            <EmptyPane
              icon={entries.length ? kind.icon : Plus}
              title={t(
                entries.length ? messages.selectTitle : messages.firstTitle,
              )}
              detail={t(
                entries.length ? messages.selectDetail : messages.firstDetail,
              )}
            >
              {!entries.length && (
                <span className="mt-1 flex flex-wrap justify-center gap-2">
                  <Button
                    type="button"
                    variant="raised"
                    size="pill"
                    onClick={() => leaveDetail("new")}
                  >
                    <Plus data-icon="inline-start" />
                    {t(messages.firstAction)}
                  </Button>
                  {onImport && (
                    <Button
                      type="button"
                      variant="quiet"
                      size="pill"
                      onClick={onImport}
                    >
                      <Import data-icon="inline-start" />
                      {t("workspace.empty.first.import")}
                    </Button>
                  )}
                </span>
              )}
            </EmptyPane>
          ))}
        {pane === "detail" && loadingDetail && (
          <Loading message={t(messages.detailLoading)} />
        )}
        {pane === "detail" &&
          selected &&
          !loadingDetail &&
          detail(
            selected,
            {
              groups: groups.filter((item) =>
                selected.groups.includes(item.id),
              ),
              tags: selectedSummary?.tags ?? [],
              onTag: shell.onTag,
              pinned,
              busy,
              confirmDelete,
              onAskDelete: () => setConfirmDelete(true),
              onCancelDelete: () => setConfirmDelete(false),
              onDelete: remove,
              onDuplicate: () => void duplicate(),
              onEdit: () => setPane("edit"),
              onTogglePin: () => togglePin(selected.id, !pinned),
            },
            copy,
            change,
          )}
        {pane === "new" &&
          editor({
            initialGroups: groupsForNew,
            tagging,
            onSave: save,
            onCancel: () => leaveDetail("empty"),
          })}
        {pane === "edit" &&
          selected &&
          editor({
            initial: selected,
            initialGroups: selected.groups,
            tagging,
            onSave: save,
            onCancel: () => setPane("detail"),
          })}
      </CrossFade>
    </div>
  );

  if (compact) {
    return (
      <CrossFade
        id={paneShown ? "pane" : "list"}
        className="relative flex min-h-0 min-w-0 flex-1 flex-col"
      >
        {paneShown ? (
          paneView
        ) : (
          <>
            {listColumn}
            <Button
              type="button"
              variant="raised"
              size="pill"
              className="absolute right-4 bottom-4 h-10 shadow-lg"
              disabled={busy}
              onClick={() => leaveDetail("new")}
            >
              <Plus data-icon="inline-start" />
              {t("workspace.toolbar.new")}
            </Button>
          </>
        )}
      </CrossFade>
    );
  }

  return (
    <ResizablePanelGroup
      id={columnsLayout}
      className="min-w-0 flex-1"
      defaultLayout={columns.defaultLayout}
      onLayoutChanged={columns.onLayoutChanged}
    >
      <ResizablePanel
        id="list"
        defaultSize="256px"
        minSize="224px"
        maxSize="480px"
        groupResizeBehavior="preserve-pixel-size"
      >
        {listColumn}
      </ResizablePanel>
      <ResizableHandle
        aria-label={t("workspace.list.resize")}
        className="w-3 bg-transparent after:w-px after:bg-transparent hover:after:bg-border active:after:bg-ring"
      />
      <ResizablePanel id="pane" className="flex h-full">
        {paneView}
      </ResizablePanel>
    </ResizablePanelGroup>
  );
}

function Loading({ message }: { message: string }) {
  return (
    <div
      className="flex items-center gap-2 text-[13px] text-muted-foreground"
      role="status"
    >
      <LoaderCircle
        className="size-4 animate-spin motion-reduce:animate-none"
        aria-hidden="true"
      />
      {message}
    </div>
  );
}
