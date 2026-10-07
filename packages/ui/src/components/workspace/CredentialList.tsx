import { KeyRound, LoaderCircle, Timer, UserRoundKey } from "lucide-react";
import { accountOf } from "../../credentials/credential.ts";
import { useTranslator } from "../../i18n/translator.tsx";
import type { BreachStatus } from "../../query/breaches.ts";
import type { CredentialSummary } from "../../vault-api.ts";
import { ItemList, type ListProps } from "./ItemList.tsx";

const markClass = "size-3.5 text-faint";

export function CredentialList({
  breaches,
  ...props
}: ListProps<CredentialSummary> & {
  /** Where the breach check stands; null while breach checks are off. */
  breaches: BreachStatus | null;
}) {
  const { t } = useTranslator();
  const counts = breaches?.report?.counts;

  return (
    <ItemList
      {...props}
      label={t("workspace.list.label")}
      untitled={t("credential.untitled")}
      shape="tile"
      icon={KeyRound}
      detail={(entry) => ({
        text: accountOf(entry) || t("credential.login.empty"),
      })}
      site={(entry) => entry.site}
      alert={(entry) => (counts?.has(entry.id) ? t("breach.mark") : undefined)}
      notice={breaches && <BreachNotice status={breaches} />}
      marks={(entry) =>
        (entry.oneTimeCode || entry.passkeys > 0) && (
          <>
            {entry.oneTimeCode && (
              <Timer
                className={markClass}
                aria-label={t("credential.mark.totp")}
              />
            )}
            {entry.passkeys > 0 && (
              <UserRoundKey
                className={markClass}
                aria-label={t("credential.mark.passkey")}
              />
            )}
          </>
        )
      }
    />
  );
}

/** BreachNotice says whether the vault's passwords were checked and what was found. */
function BreachNotice({ status }: { status: BreachStatus }) {
  const { t, failure } = useTranslator();
  const { report, checking } = status;
  if (!report) {
    return (
      <p
        className="flex items-center gap-1.5 text-[11px] text-muted-foreground"
        role="status"
      >
        <LoaderCircle
          className="size-3 animate-spin motion-reduce:animate-none"
          aria-hidden="true"
        />
        {t("breach.list.checking")}
      </p>
    );
  }
  if (report.failure !== null) {
    return (
      <p className="text-[11px] text-muted-foreground" role="status">
        {failure(report.failure, "breach.list.failed")}
      </p>
    );
  }
  const found = report.counts.size;
  return (
    <p
      className={
        found
          ? "text-[11px] font-medium text-destructive"
          : "text-[11px] text-muted-foreground"
      }
      role="status"
      aria-busy={checking}
    >
      {found
        ? t("breach.list.found", { count: found, checked: report.checked })
        : t("breach.list.clean", { count: report.checked })}
    </p>
  );
}
