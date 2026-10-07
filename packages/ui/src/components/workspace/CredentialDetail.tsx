import { Copy, ExternalLink, KeyRound, ShieldAlert } from "lucide-react";
import { useState } from "react";
import type { MessageKey } from "../../i18n/messages.ts";
import { useTranslator } from "../../i18n/translator.tsx";
import type {
  OneTimeCode as Code,
  Credential,
  CredentialField,
} from "../../vault-api.ts";
import { Button } from "../ui/button.tsx";
import { ScrollArea } from "../ui/scroll-area.tsx";
import { type DetailControls, DetailHeader } from "./DetailHeader.tsx";
import {
  CopyButton,
  FieldBlock,
  FieldRow,
  NotesBlock,
  SecretRow,
} from "./Fields.tsx";
import { LinkedApps } from "./LinkedApps.tsx";
import { OneTimeCode } from "./OneTimeCode.tsx";
import { Passkeys } from "./Passkeys.tsx";
import { useSiteIcon } from "./SiteIcons.tsx";

/** What the last breach check found for this password. */
export interface PasswordBreach {
  count: number;
  /** Set when the check could not run, so nothing is known. */
  failed: boolean;
}

/** CredentialDetail owns the password reveal, so it ends with this pane. */
export function CredentialDetail({
  credential,
  controls,
  breach,
  onCopy,
  onGenerateCode,
  onOpenWebsite,
  onMerge,
}: {
  credential: Credential;
  controls: DetailControls;
  /** Null while breach checks are off. */
  breach: PasswordBreach | null;
  /** Copies one value, and names the notice that says what was copied. */
  onCopy: (field: CredentialField, notice: MessageKey) => void;
  onGenerateCode: (setup: string) => Promise<Code>;
  onOpenWebsite: (address: string) => void;
  /** Unset while there is no other password to merge with. */
  onMerge?: () => void;
}) {
  const { t } = useTranslator();
  const [revealPassword, setRevealPassword] = useState(false);
  const { busy } = controls;
  const logo = useSiteIcon(credential.site);

  return (
    <ScrollArea className="min-h-0 flex-1">
      <article className="flex flex-1 flex-col gap-[11px]">
        <DetailHeader
          label={credential.label}
          logo={logo}
          title={credential.label || t("credential.untitled")}
          icon={KeyRound}
          subtitle={credential.websites[0] || t("credential.website.empty")}
          onMerge={onMerge}
          controls={controls}
        />

        {credential.password && breach && <BreachNote breach={breach} />}

        <div className="shrink-0">
          <FieldBlock>
            {credential.login && (
              <FieldRow
                label={t("credential.field.login")}
                action={t("credential.copy.login")}
                icon={Copy}
                disabled={busy}
                onAction={() => onCopy("login", "workspace.copy.login")}
              >
                <span className="min-w-0 flex-1 truncate text-[13px]">
                  {credential.login}
                </span>
              </FieldRow>
            )}
            {credential.email && (
              <FieldRow
                label={t("credential.field.email")}
                action={t("credential.copy.email")}
                icon={Copy}
                disabled={busy}
                onAction={() => onCopy("email", "workspace.copy.email")}
              >
                <span className="min-w-0 flex-1 truncate text-[13px]">
                  {credential.email}
                </span>
              </FieldRow>
            )}
            {credential.password && (
              <SecretRow
                label={t("credential.field.password")}
                value={credential.password}
                revealed={revealPassword}
                revealLabel={t("credential.password.reveal")}
                concealLabel={t("credential.password.conceal")}
                copyLabel={t("credential.copy.password")}
                busy={busy}
                onReveal={() => setRevealPassword((shown) => !shown)}
                onCopy={() => onCopy("password", "workspace.copy.password")}
              />
            )}
            {credential.websites.map((address, index) => (
              <FieldRow
                // biome-ignore lint/suspicious/noArrayIndexKey: websites carry no id and may repeat, and the list never reorders while shown.
                key={index}
                label={t("credential.field.website")}
                action={t("credential.website.open")}
                icon={ExternalLink}
                disabled={busy}
                onAction={() => onOpenWebsite(address)}
                accessory={
                  <CopyButton
                    label={t("credential.copy.website", { address })}
                    busy={busy}
                    onCopy={() =>
                      onCopy(`website:${index}`, "workspace.copy.website")
                    }
                  />
                }
              >
                <span className="min-w-0 flex-1 truncate text-[13px]">
                  {address}
                </span>
              </FieldRow>
            ))}
          </FieldBlock>
        </div>

        {credential.totp && (
          <OneTimeCode
            setup={credential.totp}
            generate={onGenerateCode}
            onCopy={() => onCopy("totp", "workspace.copy.totp")}
            busy={busy}
          />
        )}

        {credential.passkeys.length > 0 && (
          <Passkeys passkeys={credential.passkeys} />
        )}

        {credential.apps.length > 0 && <LinkedApps apps={credential.apps} />}

        {credential.notes && (
          <NotesBlock
            label={t("workspace.field.notes")}
            notes={credential.notes}
            copyLabel={t("workspace.notes.copy")}
            busy={busy}
            onCopy={() => onCopy("notes", "workspace.copy.notes")}
          />
        )}

        {credential.password && (
          <Button
            type="button"
            variant="raised"
            size="pill"
            className="mt-auto shrink-0 text-[13px]"
            disabled={busy}
            onClick={() => onCopy("password", "workspace.copy.password")}
          >
            <Copy data-icon="inline-start" />
            {t("credential.copy.password")}
          </Button>
        )}
      </article>
    </ScrollArea>
  );
}

/** BreachNote warns of a breached password, or says the check could not run; a clean password shows nothing. */
function BreachNote({ breach }: { breach: PasswordBreach }) {
  const { t } = useTranslator();
  if (breach.count > 0) {
    return (
      <p
        className="flex shrink-0 items-start gap-2 rounded-row bg-destructive/10 px-[13px] py-2.5 text-xs leading-[1.5] text-destructive"
        role="status"
      >
        <ShieldAlert className="mt-px size-4 shrink-0" aria-hidden="true" />
        {t("breach.detail", { count: breach.count })}
      </p>
    );
  }
  if (breach.failed) {
    return (
      <p className="shrink-0 px-[3px] text-xs text-muted-foreground">
        {t("breach.unreachable")}
      </p>
    );
  }
  return null;
}
