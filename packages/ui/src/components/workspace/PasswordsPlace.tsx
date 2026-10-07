import { useQuery } from "@tanstack/react-query";
import { KeyRound } from "lucide-react";
import { type Ref, useState } from "react";
import type { CredentialDraft } from "../../credentials/credential.ts";
import { type BreachStatus, breachedFirst } from "../../query/breaches.ts";
import { queryKeys } from "../../query/keys.ts";
import type { Credential, CredentialSummary } from "../../vault-api.ts";
import { credentialSearchValues } from "../../workspace/sections.ts";
import { CredentialEditor } from "../CredentialEditor.tsx";
import { CredentialDetail } from "./CredentialDetail.tsx";
import { CredentialList } from "./CredentialList.tsx";
import {
  type ItemKind,
  ItemPlace,
  type OpeningPane,
  type PlaceHandle,
  type PlaceMessages,
  type PlaceShell,
} from "./ItemPlace.tsx";
import { MergeDialog } from "./MergeDialog.tsx";

const messages: PlaceMessages = {
  loading: "workspace.loading",
  detailLoading: "workspace.detail.loading",
  selectTitle: "workspace.empty.select.title",
  selectDetail: "workspace.empty.select.detail",
  firstTitle: "workspace.empty.first.title",
  firstDetail: "workspace.empty.first.detail",
  firstAction: "workspace.empty.first.action",
  emptyAll: "workspace.list.empty.all",
  emptyRecent: "workspace.list.empty.recent",
  emptyPinned: "workspace.list.empty.pinned",
  emptySearch: "workspace.list.empty.search",
  errorClose: "workspace.error.close",
  errorOpen: "workspace.error.open",
  errorCopy: "workspace.error.copy",
  errorPin: "workspace.error.pin",
  errorUnpin: "workspace.error.unpin",
  errorSave: "workspace.error.save",
  errorSavedPartly: "workspace.error.saved-partly",
  errorDelete: "workspace.error.delete",
  errorDuplicate: "workspace.error.duplicate",
  errorDeletedPartly: "workspace.error.deleted-partly",
};

/** PasswordsPlace is the workspace's place for credentials. */
export function PasswordsPlace({
  shell,
  entries,
  loading,
  breaches,
  refresh,
  opening,
  onImport,
  ref,
}: {
  shell: PlaceShell;
  entries: CredentialSummary[];
  loading: boolean;
  /** Where the breach check stands; null while checks are off. */
  breaches: BreachStatus | null;
  refresh: () => Promise<void>;
  opening: OpeningPane;
  /** Opens the import of another app's items, offered while the vault holds no password. */
  onImport: () => void;
  ref?: Ref<PlaceHandle>;
}) {
  const { api, groups, busy, report } = shell;
  const { data: limits = null } = useQuery({
    queryKey: queryKeys.limits("credentials"),
    queryFn: () => api.credentialLimits(),
    meta: { failure: "workspace.error.read" },
  });
  const { data: tagLimits = null } = useQuery({
    queryKey: queryKeys.limits("tags"),
    queryFn: () => api.tagLimits(),
    meta: { failure: "workspace.error.read" },
  });
  // A failed check knows nothing to put first or mark.
  const breached =
    breaches?.report?.failure === null ? breaches.report.counts : null;
  // The password whose merge dialog is open.
  const [merging, setMerging] = useState<string | null>(null);

  const kind: ItemKind<Credential, CredentialDraft> = {
    icon: KeyRound,
    messages,
    read: (id) => api.readCredential(id),
    create: ({ input }, membership) => api.createCredential(input, membership),
    update: (id, { input, removedPasskeys }, membership) =>
      api.updateCredential(id, input, membership, removedPasskeys),
  };

  async function openWebsite(address: string) {
    try {
      await api.openWebsite(address);
    } catch (cause) {
      report(cause, "workspace.error.website");
    }
  }

  return (
    <ItemPlace
      ref={ref}
      shell={shell}
      kind={kind}
      entries={entries}
      loading={loading}
      refresh={refresh}
      onImport={onImport}
      searchValues={credentialSearchValues}
      opening={opening}
      rank={(visible) =>
        breached ? breachedFirst(visible, breached) : visible
      }
      list={(props) => <CredentialList {...props} breaches={breaches} />}
      detail={(credential, controls, copy, change) => {
        const summary = entries.find((entry) => entry.id === credential.id);
        return (
          <>
            <CredentialDetail
              credential={credential}
              controls={controls}
              breach={
                breaches?.report
                  ? {
                      count: breaches.report.counts.get(credential.id) ?? 0,
                      failed: breaches.report.failure !== null,
                    }
                  : null
              }
              onCopy={(field, notice) =>
                copy(
                  () => api.copyCredentialField(credential.id, field),
                  notice,
                )
              }
              onGenerateCode={api.generateOneTimeCode}
              onOpenWebsite={openWebsite}
              onMerge={
                limits && tagLimits && summary && entries.length > 1
                  ? () => setMerging(credential.id)
                  : undefined
              }
            />
            {merging === credential.id && limits && tagLimits && summary && (
              <MergeDialog
                kept={credential}
                summary={summary}
                credentials={entries}
                bounds={{
                  websites: limits.websites,
                  notes: limits.notes,
                  tags: tagLimits.tags,
                }}
                host={api}
                busy={busy}
                onMerge={(from, input, membership) =>
                  change(
                    () =>
                      api.mergeCredentials(
                        credential.id,
                        from,
                        input,
                        membership,
                      ),
                    { failure: "merge.error", notice: "merge.done" },
                  )
                }
                onClose={() => setMerging(null)}
                onFailure={report}
              />
            )}
          </>
        );
      }}
      editor={(props) => (
        <CredentialEditor
          {...props}
          groups={groups}
          limits={limits}
          busy={busy}
          host={api}
          onFailure={report}
        />
      )}
    />
  );
}
