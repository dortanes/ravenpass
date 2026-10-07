import { useQuery } from "@tanstack/react-query";
import { NotebookText } from "lucide-react";
import type { Ref } from "react";
import { queryKeys } from "../../query/keys.ts";
import type { Note, NoteInput, NoteSummary } from "../../vault-api.ts";
import { noteSearchValues } from "../../workspace/sections.ts";
import { NoteEditor } from "../NoteEditor.tsx";
import {
  type ItemKind,
  ItemPlace,
  type OpeningPane,
  type PlaceHandle,
  type PlaceMessages,
  type PlaceShell,
} from "./ItemPlace.tsx";
import { NoteDetail } from "./NoteDetail.tsx";
import { NoteList } from "./NoteList.tsx";

const messages: PlaceMessages = {
  loading: "note.loading",
  detailLoading: "note.detail.loading",
  selectTitle: "note.empty.select.title",
  selectDetail: "note.empty.select.detail",
  firstTitle: "note.empty.first.title",
  firstDetail: "note.empty.first.detail",
  firstAction: "note.empty.first.action",
  emptyAll: "note.list.empty.all",
  emptyRecent: "note.list.empty.recent",
  emptyPinned: "note.list.empty.pinned",
  emptySearch: "note.list.empty.search",
  errorClose: "note.error.close",
  errorOpen: "note.error.open",
  errorCopy: "note.error.copy",
  errorPin: "note.error.pin",
  errorUnpin: "note.error.unpin",
  errorSave: "note.error.save",
  errorSavedPartly: "note.error.saved-partly",
  errorDelete: "note.error.delete",
  errorDuplicate: "note.error.duplicate",
  errorDeletedPartly: "note.error.deleted-partly",
};

/** NotesPlace is the workspace's place for notes. */
export function NotesPlace({
  shell,
  entries,
  loading,
  refresh,
  opening,
  ref,
}: {
  shell: PlaceShell;
  entries: NoteSummary[];
  loading: boolean;
  refresh: () => Promise<void>;
  opening: OpeningPane;
  ref?: Ref<PlaceHandle>;
}) {
  const { api, groups, busy, report } = shell;
  const { data: limits = null } = useQuery({
    queryKey: queryKeys.limits("notes"),
    queryFn: () => api.noteLimits(),
    meta: { failure: "workspace.error.read" },
  });

  const kind: ItemKind<Note, NoteInput> = {
    icon: NotebookText,
    messages,
    read: (id) => api.readNote(id),
    create: (input, membership) => api.createNote(input, membership),
    update: (id, input, membership) => api.updateNote(id, input, membership),
  };

  return (
    <ItemPlace
      ref={ref}
      shell={shell}
      kind={kind}
      entries={entries}
      loading={loading}
      refresh={refresh}
      searchValues={noteSearchValues}
      opening={opening}
      list={(props) => <NoteList {...props} />}
      detail={(note, controls, copy) => (
        <NoteDetail
          note={note}
          controls={controls}
          host={api}
          onFailure={report}
          onCopy={() => copy(() => api.copyNote(note.id), "note.copied")}
        />
      )}
      editor={(props) => (
        <NoteEditor {...props} groups={groups} limits={limits} busy={busy} />
      )}
    />
  );
}
