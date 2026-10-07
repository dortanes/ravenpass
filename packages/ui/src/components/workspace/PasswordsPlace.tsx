import { useQuery } from "@tanstack/react-query";
import { KeyRound } from "lucide-react";
import type { Ref } from "react";
import type { CredentialDraft } from "../../credentials/credential.ts";
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
  refresh,
  opening,
  onImport,
  ref,
}: {
  shell: PlaceShell;
  entries: CredentialSummary[];
  loading: boolean;
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
      list={(props) => <CredentialList {...props} />}
      detail={(credential, controls, copy) => (
        <CredentialDetail
          credential={credential}
          controls={controls}
          onCopy={(field, notice) =>
            copy(() => api.copyCredentialField(credential.id, field), notice)
          }
          onGenerateCode={api.generateOneTimeCode}
          onOpenWebsite={openWebsite}
        />
      )}
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
