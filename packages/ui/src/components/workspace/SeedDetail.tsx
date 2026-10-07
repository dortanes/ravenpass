import { cn } from "cn";
import { Copy, ListChecks } from "lucide-react";
import { useState } from "react";
import type { MessageKey } from "../../i18n/messages.ts";
import { useTranslator } from "../../i18n/translator.tsx";
import { CalendarDate } from "../../identities/dates.ts";
import {
  checksumLabel,
  nextUnusedCode,
  summaryLabel,
} from "../../seeds/phrase.ts";
import type { BackupCode, Seed, SeedField } from "../../vault-api.ts";
import { ConfirmDialog } from "../ConfirmDialog.tsx";
import { Button } from "../ui/button.tsx";
import { ScrollArea } from "../ui/scroll-area.tsx";
import { BackupCheck } from "./BackupCheck.tsx";
import { type DetailControls, DetailHeader } from "./DetailHeader.tsx";
import {
  actionRow,
  FieldBlock,
  FieldRow,
  NotesBlock,
  SecretRow,
  TitledBlock,
} from "./Fields.tsx";
import { PhraseGrid } from "./PhraseGrid.tsx";
import { type RevealHost, useRevealGate } from "./RevealGate.tsx";
import { seedFace } from "./SeedList.tsx";
import { toneText } from "./tones.ts";

export function SeedDetail({
  seed,
  controls,
  onCopy,
  onUseCode,
  onRecordCheck,
  host,
  onFailure,
}: {
  seed: Seed;
  controls: DetailControls;
  host: RevealHost;
  onFailure: (cause: unknown, message: MessageKey) => void;
  /** Copies one value, and names the notice that says what was copied. */
  onCopy: (field: SeedField, notice: MessageKey) => void;
  /** Copies one unused code and marks it used, resolving whether that was written. */
  onUseCode: (index: number) => Promise<boolean>;
  /** Records today as the last check, resolving whether that was written. */
  onRecordCheck: () => Promise<boolean>;
}) {
  const { t } = useTranslator();
  const { busy } = controls;
  const [checking, setChecking] = useState(false);
  const [checkRound, setCheckRound] = useState(0);
  const [confirmCopy, setConfirmCopy] = useState(false);
  const [passphraseShown, setPassphraseShown] = useState(false);
  const [keyShown, setKeyShown] = useState(false);
  const gate = useRevealGate(host, seed.label || t("seed.untitled"), onFailure);
  /** Copies a secret of the seed once the owner confirms it is them. */
  const copySecret = (field: SeedField, notice: MessageKey) =>
    gate.after(() => onCopy(field, notice));
  const words = seed.words.length;
  const nextCode = nextUnusedCode(seed.codes);
  const addresses = seed.addresses.map((address, index) => ({
    address,
    index,
  }));
  const subtitle = {
    phrase: t("seed.subtitle.phrase", { count: words }),
    key: t("seed.summary.key"),
    codes: t("seed.subtitle.codes"),
  }[seed.format];

  async function recordCheck() {
    if (await onRecordCheck()) setChecking(false);
  }

  return (
    <ScrollArea className="min-h-0 flex-1">
      <article className="flex flex-1 flex-col gap-[11px]">
        <DetailHeader
          label={seed.label}
          {...seedFace(seed.format, words)}
          title={seed.label || t("seed.untitled")}
          subtitle={[subtitle, seed.wallet].filter(Boolean).join(" · ")}
          deleteTitle={t("seed.delete.title")}
          controls={
            seed.format === "codes"
              ? controls
              : { ...controls, onEdit: () => gate.after(controls.onEdit) }
          }
        />

        {seed.format === "phrase" && (
          <>
            <PhraseStatus seed={seed} />
            <PhraseGrid words={seed.words} gate={gate} />
            <BackupCheck
              key={checkRound}
              open={checking}
              words={seed.words}
              busy={busy}
              onMatch={() => void recordCheck()}
              onClose={() => setChecking(false)}
            />
            {(seed.passphrase || seed.path) && (
              <TitledBlock title={t("seed.block.wallet")}>
                <FieldBlock>
                  {seed.passphrase && (
                    <SecretRow
                      label={t("seed.field.passphrase")}
                      value={seed.passphrase}
                      revealed={passphraseShown}
                      revealLabel={t("seed.passphrase.reveal")}
                      concealLabel={t("seed.passphrase.conceal")}
                      copyLabel={t("seed.copy.passphrase")}
                      busy={busy}
                      onReveal={() =>
                        passphraseShown
                          ? setPassphraseShown(false)
                          : gate.after(() => setPassphraseShown(true))
                      }
                      onCopy={() =>
                        copySecret(
                          { kind: "passphrase" },
                          "seed.copied.passphrase",
                        )
                      }
                    />
                  )}
                  {seed.path && (
                    <FieldRow
                      label={t("seed.field.path")}
                      action={t("seed.copy.path")}
                      icon={Copy}
                      disabled={busy}
                      onAction={() =>
                        onCopy({ kind: "path" }, "seed.copied.path")
                      }
                    >
                      <span className="min-w-0 flex-1 truncate font-mono text-[13px]">
                        {seed.path}
                      </span>
                    </FieldRow>
                  )}
                </FieldBlock>
              </TitledBlock>
            )}
          </>
        )}

        {seed.format === "key" && (
          <TitledBlock title={t("seed.block.key")}>
            <FieldBlock>
              <SecretRow
                label={t("seed.field.key")}
                value={seed.key}
                revealed={keyShown}
                wrap
                revealLabel={t("seed.key.reveal")}
                concealLabel={t("seed.key.conceal")}
                copyLabel={t("seed.copy.key")}
                busy={busy}
                onReveal={() =>
                  keyShown
                    ? setKeyShown(false)
                    : gate.after(() => setKeyShown(true))
                }
                onCopy={() => copySecret({ kind: "key" }, "seed.copied.key")}
              />
            </FieldBlock>
          </TitledBlock>
        )}

        {seed.format === "codes" && (
          <CodeGrid
            codes={seed.codes}
            busy={busy}
            onUse={(index) => void onUseCode(index)}
          />
        )}

        {addresses.length > 0 && (
          <TitledBlock title={t("seed.block.addresses")}>
            <FieldBlock>
              {addresses.map(({ address, index }) => (
                <FieldRow
                  key={index}
                  label={
                    address.label ||
                    t("seed.address.untitled", { number: index + 1 })
                  }
                  action={t("seed.copy.address")}
                  icon={Copy}
                  disabled={busy}
                  onAction={() =>
                    onCopy({ kind: "address", index }, "seed.copied.address")
                  }
                >
                  <span className="min-w-0 flex-1 truncate font-mono text-[13px] select-text">
                    {address.value}
                  </span>
                </FieldRow>
              ))}
            </FieldBlock>
          </TitledBlock>
        )}

        {seed.notes && (
          <NotesBlock
            label={t("workspace.field.notes")}
            notes={seed.notes}
            copyLabel={t("workspace.notes.copy")}
            busy={busy}
            onCopy={() => onCopy({ kind: "notes" }, "workspace.copy.notes")}
          />
        )}

        <div className="mt-auto flex shrink-0 gap-2">
          {seed.format === "phrase" && (
            <>
              <Button
                type="button"
                variant="raised"
                size="pill"
                className="flex-1 text-[13px]"
                disabled={busy}
                onClick={() => {
                  setCheckRound((round) => round + 1);
                  setChecking(true);
                }}
              >
                <ListChecks data-icon="inline-start" />
                {t("seed.check.open")}
              </Button>
              <Button
                type="button"
                variant="quiet"
                size="pill"
                className="flex-1 text-[13px]"
                disabled={busy}
                onClick={() => setConfirmCopy(true)}
              >
                <Copy data-icon="inline-start" />
                {t("seed.copy.phrase")}
              </Button>
            </>
          )}
          {seed.format === "key" && (
            <Button
              type="button"
              variant="raised"
              size="pill"
              className="flex-1 text-[13px]"
              disabled={busy}
              onClick={() => copySecret({ kind: "key" }, "seed.copied.key")}
            >
              <Copy data-icon="inline-start" />
              {t("seed.copy.key")}
            </Button>
          )}
          {seed.format === "codes" && (
            <Button
              type="button"
              variant="raised"
              size="pill"
              className="flex-1 text-[13px]"
              disabled={busy || nextCode < 0}
              onClick={() => void onUseCode(nextCode)}
            >
              <Copy data-icon="inline-start" />
              {t("seed.copy.next-code")}
            </Button>
          )}
        </div>
      </article>

      {gate.dialog}
      <ConfirmDialog
        open={confirmCopy}
        title={t("seed.copy-phrase.title")}
        detail={t("seed.copy-phrase.detail")}
        confirm={t("seed.copy-phrase.confirm")}
        cancel={t("workspace.delete.cancel")}
        destructive
        busy={busy}
        onConfirm={() => {
          setConfirmCopy(false);
          copySecret({ kind: "phrase" }, "seed.copied.phrase");
        }}
        onCancel={() => setConfirmCopy(false)}
      />
    </ScrollArea>
  );
}

function PhraseStatus({ seed }: { seed: Seed }) {
  const { t, language } = useTranslator();
  const checked = CalendarDate.parse(seed.checkedOn);
  const checksum = seed.checksum ? checksumLabel(seed.checksum) : null;

  return (
    <p className="flex shrink-0 flex-wrap items-center gap-x-1.5 px-[3px] text-[11px]">
      {checksum && (
        <>
          <span className={toneText[checksum.warning ? "warning" : "default"]}>
            {t(checksum.key)}
          </span>
          <span className="text-faint" aria-hidden="true">
            ·
          </span>
        </>
      )}
      <span className={toneText[checked ? "default" : "warning"]}>
        {checked
          ? t("seed.checked.on", { date: checked.format(language) })
          : t("seed.checked.never")}
      </span>
    </p>
  );
}

/** CodeGrid marks an unused code used when it is chosen. */
function CodeGrid({
  codes,
  busy,
  onUse,
}: {
  codes: readonly BackupCode[];
  busy: boolean;
  onUse: (index: number) => void;
}) {
  const { t } = useTranslator();
  const left = summaryLabel({
    format: "codes",
    total: codes.length,
    used: codes.filter((code) => code.used).length,
  });
  const cells = codes.map((code, index) => ({ code, position: index + 1 }));

  return (
    <TitledBlock title={t("seed.block.codes")}>
      <div className="rounded-row bg-field p-[11px]">
        <p
          className={cn(
            "mb-2 px-0.5 text-[11px]",
            toneText[left.warning ? "warning" : "default"],
          )}
        >
          {t(left.key, left.values)}
        </p>
        <ul className="grid grid-cols-2 gap-1.5">
          {cells.map(({ code, position }) => (
            <li key={position} className="min-w-0">
              {code.used ? (
                <span className="flex h-8 min-w-0 items-center rounded-md px-2.5 font-mono text-[13px] text-faint">
                  <span className="truncate line-through">{code.value}</span>
                  <span className="sr-only">{t("seed.code.used")}</span>
                </span>
              ) : (
                <button
                  type="button"
                  aria-label={t("seed.code.use", { number: position })}
                  title={t("seed.code.use", { number: position })}
                  disabled={busy}
                  onClick={() => onUse(position - 1)}
                  className={`flex h-8 w-full min-w-0 items-center gap-2 rounded-md bg-tile px-2.5 text-left font-mono text-[13px] ${actionRow}`}
                >
                  <span className="min-w-0 flex-1 truncate">{code.value}</span>
                  <Copy className="size-3.5 shrink-0 text-muted-foreground" />
                </button>
              )}
            </li>
          ))}
        </ul>
      </div>
    </TitledBlock>
  );
}
