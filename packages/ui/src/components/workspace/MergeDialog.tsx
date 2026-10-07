import { cn } from "cn";
import { useState } from "react";
import { accountOf } from "../../credentials/credential.ts";
import {
  CredentialMerge,
  type MergeBounds,
  type MergeChoice,
  type MergeChoices,
  type MergeField,
  mergeCandidates,
} from "../../credentials/merge.ts";
import type { MessageKey } from "../../i18n/messages.ts";
import { useTranslator } from "../../i18n/translator.tsx";
import type {
  Credential,
  CredentialInput,
  CredentialSummary,
  VaultApi,
} from "../../vault-api.ts";
import {
  ResponsiveDialog,
  ResponsiveDialogContent,
  ResponsiveDialogDescription,
  ResponsiveDialogFooter,
  ResponsiveDialogHeader,
  ResponsiveDialogTitle,
} from "../ResponsiveDialog.tsx";
import { SearchField } from "../SearchField.tsx";
import { Button } from "../ui/button.tsx";
import { ChoiceRow } from "./ChoiceRow.tsx";
import { RevealButton } from "./Fields.tsx";

/** What the merge asks of the host. */
export type MergeHost = Pick<VaultApi, "readCredential">;

const fieldLabels: Record<MergeField | "notes", MessageKey> = {
  label: "credential.field.label",
  login: "credential.field.login",
  email: "credential.field.email",
  password: "credential.field.password",
  totp: "credential.field.totp",
  notes: "workspace.field.notes",
};

/** Fields shown concealed until revealed, as the detail shows them. */
const secretFields: ReadonlySet<MergeField | "notes"> = new Set([
  "password",
  "totp",
]);

/** MergeDialog asks for another password, then for each value that differs, and merges it into `kept`. */
export function MergeDialog({
  kept,
  summary,
  credentials,
  bounds,
  host,
  busy,
  onMerge,
  onClose,
  onFailure,
}: {
  kept: Credential;
  /** The kept password as its list shows it, which the candidates are ranked against. */
  summary: CredentialSummary;
  credentials: CredentialSummary[];
  bounds: MergeBounds;
  host: MergeHost;
  busy: boolean;
  /** Resolves whether the merge was saved. */
  onMerge: (
    from: string,
    input: CredentialInput,
    groups: string[],
  ) => Promise<boolean>;
  onClose: () => void;
  onFailure: (cause: unknown, message: MessageKey) => void;
}) {
  const { t } = useTranslator();
  const [query, setQuery] = useState("");
  const [reading, setReading] = useState<string | null>(null);
  const [merge, setMerge] = useState<CredentialMerge | null>(null);
  const keptName = kept.label || t("credential.untitled");

  async function choose(id: string) {
    setReading(id);
    try {
      setMerge(
        new CredentialMerge(kept, await host.readCredential(id), bounds),
      );
    } catch (cause) {
      onFailure(cause, "merge.error.read");
    } finally {
      setReading(null);
    }
  }

  return (
    <ResponsiveDialog
      open
      dismissible={!busy}
      onOpenChange={(open) => {
        if (!open) onClose();
      }}
    >
      <ResponsiveDialogContent className="bg-background sm:max-w-[440px]">
        {merge ? (
          <MergeChoicesStep
            merge={merge}
            keptName={keptName}
            busy={busy}
            onBack={() => setMerge(null)}
            onMerge={async (choices) => {
              const done = await onMerge(
                merge.other.id,
                merge.input(choices),
                merge.groups(),
              );
              if (done) onClose();
            }}
          />
        ) : (
          <>
            <ResponsiveDialogHeader>
              <ResponsiveDialogTitle>{t("merge.title")}</ResponsiveDialogTitle>
              <ResponsiveDialogDescription>
                {t("merge.choose.detail", { name: keptName })}
              </ResponsiveDialogDescription>
            </ResponsiveDialogHeader>
            <SearchField
              id="merge-search"
              value={query}
              onChange={setQuery}
              label={t("workspace.toolbar.search")}
              placeholder={t("workspace.toolbar.search.placeholder")}
            />
            <div className="grid max-h-[min(320px,45dvh)] gap-1 overflow-y-auto">
              {mergeCandidates(summary, credentials, query).map((other) => (
                <ChoiceRow
                  key={other.id}
                  site={other.site}
                  title={other.label || t("credential.untitled")}
                  tags={other.tags}
                  detail={accountOf(other) || t("credential.login.empty")}
                  busy={busy || reading !== null}
                  working={reading === other.id}
                  onChoose={() => void choose(other.id)}
                />
              ))}
            </div>
            <ResponsiveDialogFooter>
              <Button
                type="button"
                variant="quiet"
                size="pill"
                disabled={busy}
                onClick={onClose}
              >
                {t("merge.cancel")}
              </Button>
            </ResponsiveDialogFooter>
          </>
        )}
      </ResponsiveDialogContent>
    </ResponsiveDialog>
  );
}

function MergeChoicesStep({
  merge,
  keptName,
  busy,
  onBack,
  onMerge,
}: {
  merge: CredentialMerge;
  keptName: string;
  busy: boolean;
  onBack: () => void;
  onMerge: (choices: MergeChoices) => void;
}) {
  const { t } = useTranslator();
  const [choices, setChoices] = useState<MergeChoices>({});
  const otherName = merge.other.label || t("credential.untitled");
  const moved = merge.other.passkeys.filter(
    (passkey) => !merge.kept.passkeys.some((held) => held.id === passkey.id),
  ).length;
  const notes = [
    moved > 0 && t("merge.passkeys", { count: moved }),
    merge.dropped.websites > 0 &&
      t("merge.dropped.websites", { count: merge.dropped.websites }),
    merge.dropped.tags > 0 &&
      t("merge.dropped.tags", { count: merge.dropped.tags }),
  ].filter((note): note is string => Boolean(note));

  return (
    <>
      <ResponsiveDialogHeader>
        <ResponsiveDialogTitle>{t("merge.title")}</ResponsiveDialogTitle>
        <ResponsiveDialogDescription>
          {t(
            merge.conflicts.length
              ? "merge.resolve.detail"
              : "merge.resolve.none",
            { kept: keptName, other: otherName },
          )}
        </ResponsiveDialogDescription>
      </ResponsiveDialogHeader>
      <div className="grid max-h-[min(420px,55dvh)] gap-3 overflow-y-auto">
        {merge.conflicts.map((field) => (
          <Conflict
            key={field}
            field={field}
            kept={merge.kept[field]}
            other={merge.other[field]}
            keptName={keptName}
            otherName={otherName}
            both={field === "notes" && merge.notesFitTogether}
            choice={choices[field] ?? "kept"}
            onChoose={(choice) =>
              setChoices((current) => ({ ...current, [field]: choice }))
            }
          />
        ))}
        {notes.map((note) => (
          <p key={note} className="px-[3px] text-xs text-muted-foreground">
            {note}
          </p>
        ))}
      </div>
      <ResponsiveDialogFooter>
        <Button
          type="button"
          variant="quiet"
          size="pill"
          disabled={busy}
          onClick={onBack}
        >
          {t("merge.back")}
        </Button>
        <Button
          type="button"
          variant="raised"
          size="pill"
          disabled={busy}
          onClick={() => onMerge(choices)}
        >
          {t(busy ? "merge.busy" : "merge.confirm")}
        </Button>
      </ResponsiveDialogFooter>
    </>
  );
}

function Conflict({
  field,
  kept,
  other,
  keptName,
  otherName,
  both,
  choice,
  onChoose,
}: {
  field: MergeField | "notes";
  kept: string;
  other: string;
  keptName: string;
  otherName: string;
  /** Offers keeping both values. */
  both: boolean;
  choice: MergeChoice;
  onChoose: (choice: MergeChoice) => void;
}) {
  const { t } = useTranslator();
  const secret = secretFields.has(field);
  const [revealed, setRevealed] = useState(false);
  const shown = (value: string) =>
    secret && !revealed ? "•".repeat(Math.min(value.length, 12)) : value;

  return (
    <fieldset className="grid gap-1.5">
      <legend className="mb-1.5 flex w-full items-center justify-between px-[3px] text-[11px] text-muted-foreground">
        {t(fieldLabels[field])}
        {secret && (
          <RevealButton
            shown={revealed}
            revealLabel={t("merge.reveal")}
            concealLabel={t("merge.conceal")}
            onToggle={() => setRevealed((current) => !current)}
          />
        )}
      </legend>
      <Option
        selected={choice === "kept"}
        source={keptName}
        value={shown(kept)}
        secret={secret}
        onChoose={() => onChoose("kept")}
      />
      <Option
        selected={choice === "other"}
        source={otherName}
        value={shown(other)}
        secret={secret}
        onChoose={() => onChoose("other")}
      />
      {both && (
        <Option
          selected={choice === "both"}
          source={t("merge.both")}
          value={t("merge.both.detail")}
          secret={false}
          onChoose={() => onChoose("both")}
        />
      )}
    </fieldset>
  );
}

function Option({
  selected,
  source,
  value,
  secret,
  onChoose,
}: {
  selected: boolean;
  /** Which password the value comes from. */
  source: string;
  value: string;
  secret: boolean;
  onChoose: () => void;
}) {
  return (
    <button
      type="button"
      aria-pressed={selected}
      className={cn(
        "grid gap-0.5 rounded-row border px-[11px] py-2 text-left outline-none transition-colors focus-visible:border-ring",
        selected
          ? "border-foreground/24 bg-raised"
          : "border-transparent bg-card hover:bg-control",
      )}
      onClick={onChoose}
    >
      <span className="text-[11px] text-muted-foreground">{source}</span>
      <span
        className={cn(
          "line-clamp-3 text-[13px] break-words whitespace-pre-wrap",
          secret && "font-mono",
        )}
      >
        {value}
      </span>
    </button>
  );
}
