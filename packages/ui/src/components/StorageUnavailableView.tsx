import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { useCapabilities } from "../host/capabilities.tsx";
import type { MessageKey } from "../i18n/messages.ts";
import { useTranslator } from "../i18n/translator.tsx";
import { queryKeys } from "../query/keys.ts";
import { vaultStorageQuery } from "../query/vault-storage.ts";
import type { VaultApi, VaultState } from "../vault-api.ts";
import { AccessShell, ActionBar } from "./AccessShell.tsx";
import { Button } from "./ui/button.tsx";
import { VaultMenu } from "./VaultMenu.tsx";

interface Attempt {
  /** Resolves true when the vault file changed. */
  run: () => Promise<boolean>;
  failure: MessageKey;
}

/** StorageUnavailableView stands in when the vault file cannot be reached. */
export function StorageUnavailableView({
  api,
  onOpened,
}: {
  api: VaultApi;
  onOpened: (phase: VaultState["phase"]) => void;
}) {
  const { t, failure, language } = useTranslator();
  const { storageLocations } = useCapabilities();
  const client = useQueryClient();
  const storageKey = queryKeys.storage(language);
  const status =
    useQuery(
      vaultStorageQuery(api, language, "storage-unavailable.errors.unreadable"),
    ).data ?? null;

  const attempt = useMutation({
    mutationFn: async ({ run }: Attempt) => {
      if (!(await run())) return;
      const state = await api.getState();
      if (state.phase !== "storage") {
        onOpened(state.phase);
        return;
      }
      await client.invalidateQueries({ queryKey: storageKey });
      toast.error(t("storage-unavailable.errors.unreachable"));
    },
    onError: (cause, { failure: message }) => {
      toast.error(failure(cause, message));
    },
  });
  const busy = attempt.isPending;

  function retry() {
    attempt.mutate({
      run: async () => {
        await api.retryStorage();
        return true;
      },
      failure: "storage-unavailable.errors.unreachable",
    });
  }

  function switchVault(path: string) {
    attempt.mutate({
      run: async () => {
        await api.switchVault(path);
        return true;
      },
      failure: "storage-unavailable.errors.switch-failed",
    });
  }

  function create() {
    attempt.mutate({
      run: async () => (await api.createVault()).changed,
      failure: "storage-unavailable.errors.create-failed",
    });
  }

  function locate() {
    attempt.mutate({
      run: async () => (await api.openVault()).changed,
      failure: "storage-unavailable.errors.open-failed",
    });
  }

  const detail = describe();

  function describe() {
    if (status?.reason === "selection-unusable") {
      return t("storage-unavailable.errors.unreadable");
    }
    return status?.path
      ? t("storage-unavailable.description-folder", { place: status.place })
      : t("storage-unavailable.description");
  }

  return (
    <AccessShell title={t("storage-unavailable.title")} description={detail}>
      <div className="flex flex-col gap-3">
        {status && status.vaults.length > 1 && (
          <div className="flex justify-center">
            <VaultMenu
              vaults={status.vaults}
              currentPath={status.path}
              busy={busy}
              onSwitch={switchVault}
              onCreate={create}
              onOpen={locate}
            />
          </div>
        )}
        {status?.path && (
          <div className="rounded-row bg-field px-[13px] py-2.5 select-text">
            <p className="truncate text-[13px]">{status.name}</p>
            <p className="truncate text-[11px] text-muted-foreground">
              {status.place}
            </p>
          </div>
        )}
        <p className="text-[11px] text-faint">
          {t("storage-unavailable.note")}
        </p>
      </div>
      <ActionBar>
        {storageLocations && (
          <Button
            type="button"
            variant="quiet"
            size="pill"
            onClick={locate}
            disabled={busy}
          >
            {t("storage-unavailable.actions.locate")}
          </Button>
        )}
        <Button
          type="button"
          variant="raised"
          size="pill"
          onClick={retry}
          disabled={busy}
        >
          {busy
            ? t("storage-unavailable.actions.retry-busy")
            : t("storage-unavailable.actions.retry")}
        </Button>
      </ActionBar>
    </AccessShell>
  );
}
