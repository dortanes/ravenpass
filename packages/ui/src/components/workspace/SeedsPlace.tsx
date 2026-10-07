import { useQuery } from "@tanstack/react-query";
import { Sprout } from "lucide-react";
import type { Ref } from "react";
import { queryKeys } from "../../query/keys.ts";
import type { Seed, SeedInput, SeedSummary } from "../../vault-api.ts";
import { seedSearchValues } from "../../workspace/sections.ts";
import { SeedEditor } from "../SeedEditor.tsx";
import {
  type ItemKind,
  ItemPlace,
  type OpeningPane,
  type PlaceHandle,
  type PlaceMessages,
  type PlaceShell,
} from "./ItemPlace.tsx";
import { SeedDetail } from "./SeedDetail.tsx";
import { SeedList } from "./SeedList.tsx";

const messages: PlaceMessages = {
  loading: "seed.loading",
  detailLoading: "seed.detail.loading",
  selectTitle: "seed.empty.select.title",
  selectDetail: "seed.empty.select.detail",
  firstTitle: "seed.empty.first.title",
  firstDetail: "seed.empty.first.detail",
  firstAction: "seed.empty.first.action",
  emptyAll: "seed.list.empty.all",
  emptyRecent: "seed.list.empty.recent",
  emptyPinned: "seed.list.empty.pinned",
  emptySearch: "seed.list.empty.search",
  errorClose: "seed.error.close",
  errorOpen: "seed.error.open",
  errorCopy: "seed.error.copy",
  errorPin: "seed.error.pin",
  errorUnpin: "seed.error.unpin",
  errorSave: "seed.error.save",
  errorSavedPartly: "seed.error.saved-partly",
  errorDelete: "seed.error.delete",
  errorDuplicate: "seed.error.duplicate",
  errorDeletedPartly: "seed.error.deleted-partly",
};

/** SeedsPlace is the workspace's place for seed phrases, private keys and backup codes. */
export function SeedsPlace({
  shell,
  entries,
  loading,
  refresh,
  opening,
  ref,
}: {
  shell: PlaceShell;
  entries: SeedSummary[];
  loading: boolean;
  refresh: () => Promise<void>;
  opening: OpeningPane;
  ref?: Ref<PlaceHandle>;
}) {
  const { api, groups, busy, report } = shell;
  const { data: limits = null } = useQuery({
    queryKey: queryKeys.limits("seeds"),
    queryFn: () => api.seedLimits(),
    meta: { failure: "workspace.error.read" },
  });

  const kind: ItemKind<Seed, SeedInput> = {
    icon: Sprout,
    messages,
    read: (id) => api.readSeed(id),
    create: (input, membership) => api.createSeed(input, membership),
    update: (id, input, membership) => api.updateSeed(id, input, membership),
  };

  return (
    <ItemPlace
      ref={ref}
      shell={shell}
      kind={kind}
      entries={entries}
      loading={loading}
      refresh={refresh}
      searchValues={seedSearchValues}
      opening={opening}
      list={(props) => <SeedList {...props} />}
      detail={(seed, controls, copy, change) => (
        <SeedDetail
          seed={seed}
          controls={controls}
          host={api}
          onFailure={report}
          onCopy={(field, notice) =>
            copy(() => api.copySeedField(seed.id, field), notice)
          }
          onUseCode={(index) =>
            change(() => api.spendBackupCode(seed.id, index), {
              failure: "seed.error.use-code",
              notice: "seed.copied.code",
              copies: true,
            })
          }
          onRecordCheck={() =>
            change(() => api.recordSeedCheck(seed.id), {
              failure: "seed.error.check",
              notice: "seed.check.done",
            })
          }
        />
      )}
      editor={(props) => (
        <SeedEditor
          {...props}
          groups={groups}
          limits={limits}
          busy={busy}
          host={api}
        />
      )}
    />
  );
}
