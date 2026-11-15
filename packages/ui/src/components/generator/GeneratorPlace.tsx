import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { toast } from "sonner";
import type { GeneratorMode } from "../../credentials/password-generator.ts";
import { useTranslator } from "../../i18n/translator.tsx";
import { queryKeys } from "../../query/keys.ts";
import type {
  KeptPassword,
  PasswordHistorySetting,
  VaultApi,
} from "../../vault-api.ts";
import { Block, BlockHeading, BlockRow } from "../Block.tsx";
import { ConfirmDialog } from "../ConfirmDialog.tsx";
import { Button } from "../ui/button.tsx";
import { CopyButton, RevealButton } from "../workspace/Fields.tsx";
import { PasswordForm, useWordGenerator } from "./PasswordForm.tsx";

/** What the Generator place asks of the host. */
export type GeneratorPlaceApi = Pick<
  VaultApi,
  | "seedWordlist"
  | "keptPasswords"
  | "recordGeneratedPassword"
  | "clearPasswordHistory"
  | "copyGeneratedPassword"
>;

/** GeneratorPlace makes a password to copy, with the history of those copied or used below it. */
export function GeneratorPlace({
  api,
  setting,
}: {
  api: GeneratorPlaceApi;
  setting: PasswordHistorySetting | null;
}) {
  const { t, failure } = useTranslator();
  const client = useQueryClient();
  const generator = useWordGenerator(api, true);
  const kept = useQuery({
    queryKey: queryKeys.keptPasswords,
    queryFn: () => api.keptPasswords(),
    meta: { failure: "generator.error.history" },
  });
  const [clearing, setClearing] = useState(false);
  const refresh = () =>
    client.invalidateQueries({ queryKey: queryKeys.keptPasswords });

  const copy = useMutation({
    mutationFn: async (password: { value: string; mode?: GeneratorMode }) => {
      await api.copyGeneratedPassword(password.value);
      if (password.mode) {
        await api
          .recordGeneratedPassword(password.value, password.mode)
          .catch((cause: unknown) =>
            toast.error(failure(cause, "generator.error.record")),
          );
      }
    },
    meta: { failure: "generator.error.copy" },
    onSuccess: () => toast.success(t("generator.copied")),
    onSettled: refresh,
  });
  const clear = useMutation({
    mutationFn: () => api.clearPasswordHistory(),
    meta: { failure: "generator.error.clear" },
    onSuccess: () => setClearing(false),
    onSettled: refresh,
  });
  const passwords = kept.data ?? [];

  return (
    <div className="min-h-0 flex-1 overflow-y-auto">
      <div className="mx-auto grid w-full max-w-[460px] gap-6 px-5 py-6">
        {generator && (
          <PasswordForm
            generator={generator}
            footer={(value, mode) => (
              <Button
                type="button"
                variant="raised"
                size="pill"
                className="justify-self-end"
                disabled={copy.isPending}
                onClick={() => copy.mutate({ value, mode })}
              >
                {t("generator.copy")}
              </Button>
            )}
          />
        )}
        <section className="grid gap-2">
          <BlockHeading>{t("generator.history")}</BlockHeading>
          <Block>
            {passwords.map((password) => (
              <KeptRow
                key={`${password.at}-${password.value}`}
                password={password}
                busy={copy.isPending}
                onCopy={() => copy.mutate({ value: password.value })}
              />
            ))}
            {!passwords.length && kept.isSuccess && (
              <p className="px-[13px] py-3 text-[13px] text-muted-foreground">
                {setting?.enabled === false
                  ? t("generator.history.off")
                  : t("generator.history.empty", {
                      count: setting?.days ?? 30,
                    })}
              </p>
            )}
          </Block>
          {passwords.length > 0 && (
            <Button
              type="button"
              variant="quiet"
              size="pill-sm"
              className="justify-self-start bg-tile font-normal text-destructive hover:bg-field-hover"
              disabled={clear.isPending}
              onClick={() => setClearing(true)}
            >
              {t("generator.history.clear")}
            </Button>
          )}
        </section>
      </div>
      <ConfirmDialog
        open={clearing}
        title={t("generator.history.clear.title", { count: passwords.length })}
        detail={t("generator.history.clear.detail")}
        confirm={t("generator.history.clear.confirm")}
        cancel={t("workspace.delete.cancel")}
        destructive
        busy={clear.isPending}
        onConfirm={() => clear.mutate()}
        onCancel={() => setClearing(false)}
      />
    </div>
  );
}

function KeptRow({
  password,
  busy,
  onCopy,
}: {
  password: KeptPassword;
  busy: boolean;
  onCopy: () => void;
}) {
  const { t, language } = useTranslator();
  const [shown, setShown] = useState(false);
  return (
    <BlockRow
      title={shown ? password.value : "••••••••••••"}
      detail={new Intl.DateTimeFormat(language, {
        dateStyle: "medium",
        timeStyle: "short",
      }).format(new Date(password.at))}
    >
      <RevealButton
        shown={shown}
        revealLabel={t("generator.history.show")}
        concealLabel={t("generator.history.hide")}
        onToggle={() => setShown(!shown)}
      />
      <CopyButton label={t("generator.copy")} busy={busy} onCopy={onCopy} />
    </BlockRow>
  );
}
