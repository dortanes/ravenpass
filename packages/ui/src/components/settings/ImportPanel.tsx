import {
  Check,
  ChevronLeft,
  CircleAlert,
  Download,
  FileArchive,
  FileText,
  LoaderCircle,
  TriangleAlert,
} from "lucide-react";
import { Fragment, type ReactNode, useEffect, useRef, useState } from "react";
import { toast } from "sonner";
import { failureCode } from "../../failures.ts";
import type { MessageKey } from "../../i18n/messages.ts";
import { useTranslator } from "../../i18n/translator.tsx";
import { isEncryptedFormat } from "../../import/preview.ts";
import {
  defaultImportOptions,
  type ImportKindReview,
  ImportReview,
} from "../../import/review.ts";
import { type ImportSourceGuide, importSources } from "../../import/sources.ts";
import type {
  ImportCardNetworks,
  ImportChoice,
  ImportOptions,
  ImportPreview,
  ImportResult,
  ImportSource,
  ItemKindName,
} from "../../vault-api.ts";
import {
  Block,
  BlockHint,
  BlockIconTile,
  BlockNote,
  BlockRow,
  BlockRowButton,
  BlockTile,
} from "../Block.tsx";
import {
  ResponsiveDialog,
  ResponsiveDialogContent,
  ResponsiveDialogDescription,
  ResponsiveDialogFooter,
  ResponsiveDialogHeader,
  ResponsiveDialogTitle,
} from "../ResponsiveDialog.tsx";
import { StepNumber } from "../StepNumber.tsx";
import { Alert, AlertDescription, AlertTitle } from "../ui/alert.tsx";
import { Button } from "../ui/button.tsx";
import { Field, FieldError, FieldGroup, FieldLabel } from "../ui/field.tsx";
import { Input } from "../ui/input.tsx";
import { Switch } from "../ui/switch.tsx";
import { kindPlaces, placeOf } from "../workspace/places.ts";

/** What the Import section asks of the workspace. `run` reports its own failure. */
export interface ImportActions {
  choose(source: ImportSource): Promise<ImportChoice>;
  unlock(password: string): Promise<ImportPreview>;
  run(
    options: ImportOptions,
    cardNetworks: ImportCardNetworks,
  ): Promise<ImportResult>;
  cancel(): Promise<void>;
  trash(): Promise<void>;
  reveal(): Promise<void>;
}

type DoneFlow = {
  step: "done";
  source: ImportSourceGuide;
  name: string;
  result: ImportResult;
  file: "kept" | "trashing" | "trashed";
};

/** Why the chosen file was not taken, and the message to fall back on for an unexpected cause. */
type Refusal = { cause: unknown; fallback: MessageKey };

type Flow =
  | { step: "sources" }
  | {
      step: "source";
      source: ImportSourceGuide;
      reading: boolean;
      refusal: Refusal | null;
    }
  | { step: "unlocking"; source: ImportSourceGuide; name: string }
  | {
      step: "review";
      source: ImportSourceGuide;
      name: string;
      review: ImportReview;
    }
  | {
      step: "importing";
      source: ImportSourceGuide;
      name: string;
      review: ImportReview;
    }
  | DoneFlow;

const listing: Flow = { step: "sources" };

function opened(
  source: ImportSourceGuide,
  refusal: Refusal | null = null,
): Flow {
  return { step: "source", source, reading: false, refusal };
}

function reviewing(
  source: ImportSourceGuide,
  name: string,
  preview: ImportPreview,
): Flow {
  return {
    step: "review",
    source,
    name,
    review: new ImportReview(preview, defaultImportOptions),
  };
}

function kindLabel(kind: ItemKindName): MessageKey {
  return placeOf(kindPlaces[kind]).label;
}

/** The review lists this many items it cannot import and counts the rest. */
const listedSkips = 5;

/** A file Ravenpass cannot locate is deleted by the person, wherever they saved it. */
function warningDetail({ format, located }: ImportResult): MessageKey {
  if (format === "zip") {
    return located
      ? "settings.import.warning.zip"
      : "settings.import.warning.zip.manual";
  }
  return located
    ? "settings.import.warning.detail"
    : "settings.import.warning.manual";
}

/** A failed cancel leaves the file staged until the host drops it at lock or window close. */
function keepStaged() {}

/** ImportPanel takes a file exported from another app through reviewing and importing. */
export function ImportPanel({
  actions,
  busy,
}: {
  actions: ImportActions;
  busy: boolean;
}) {
  const [flow, setFlow] = useState<Flow>(listing);

  // Leaving the section drops the staged file; the ref holds the latest actions and flow at unmount.
  const leaving = useRef({ actions, flow });
  leaving.current = { actions, flow };
  useEffect(
    () => () => {
      const { actions: current, flow: last } = leaving.current;
      if (last.step !== "sources") void current.cancel().catch(keepStaged);
    },
    [],
  );

  function backTo(source: ImportSourceGuide) {
    void actions.cancel().catch(keepStaged);
    setFlow(opened(source));
  }

  // The host keeps a staged file until another one is chosen.
  async function choose(source: ImportSourceGuide, previous: Flow) {
    setFlow({ step: "source", source, reading: true, refusal: null });
    try {
      const choice = await actions.choose(source.id);
      if (!choice.chosen) {
        setFlow(previous);
      } else if (choice.preview) {
        setFlow(reviewing(source, choice.name, choice.preview));
      } else {
        setFlow({ step: "unlocking", source, name: choice.name });
      }
    } catch (cause) {
      setFlow(
        opened(source, { cause, fallback: "settings.import.error.choose" }),
      );
    }
  }

  async function unlock(
    source: ImportSourceGuide,
    name: string,
    password: string,
  ) {
    try {
      setFlow(reviewing(source, name, await actions.unlock(password)));
      return true;
    } catch (cause) {
      if (failureCode(cause) === "import-password-wrong") return false;
      setFlow(
        opened(source, { cause, fallback: "settings.import.error.unlock" }),
      );
      return true;
    }
  }

  // The workspace reports a failed import; the review stays unless the host dropped the file.
  async function run(
    source: ImportSourceGuide,
    name: string,
    review: ImportReview,
  ) {
    setFlow({ step: "importing", source, name, review });
    try {
      const result = await actions.run(review.options, review.cardNetworks);
      setFlow({ step: "done", source, name, result, file: "kept" });
    } catch (cause) {
      setFlow(
        failureCode(cause) === "import-not-active"
          ? opened(source)
          : { step: "review", source, name, review },
      );
    }
  }

  return (
    <>
      {(flow.step === "review" ||
        flow.step === "importing" ||
        flow.step === "done") && (
        <ImportProgress complete={flow.step === "done"} />
      )}
      {flow.step === "sources" && (
        <SourceList busy={busy} onOpen={(source) => setFlow(opened(source))} />
      )}
      {(flow.step === "source" || flow.step === "unlocking") && (
        <SourceStep
          source={flow.source}
          reading={flow.step === "source" && flow.reading}
          refusal={flow.step === "source" ? flow.refusal : null}
          busy={busy}
          onBack={() => setFlow(listing)}
          onChoose={() => void choose(flow.source, flow)}
        />
      )}
      {flow.step === "unlocking" && (
        <PasswordDialog
          name={flow.name}
          onUnlock={(password) => unlock(flow.source, flow.name, password)}
          onCancel={() => backTo(flow.source)}
        />
      )}
      {flow.step === "review" && (
        <Review
          source={flow.source}
          name={flow.name}
          review={flow.review}
          busy={busy}
          onOptions={(options) =>
            setFlow({ ...flow, review: flow.review.withOptions(options) })
          }
          onChange={() => void choose(flow.source, flow)}
          onCancel={() => backTo(flow.source)}
          onRun={() => void run(flow.source, flow.name, flow.review)}
        />
      )}
      {flow.step === "importing" && (
        <Importing name={flow.name} review={flow.review} />
      )}
      {flow.step === "done" && (
        <Done
          flow={flow}
          actions={actions}
          onFile={(file) =>
            setFlow((current) =>
              current.step === "done" ? { ...current, file } : current,
            )
          }
          onAnother={() => backTo(flow.source)}
        />
      )}
    </>
  );
}

function ImportProgress({ complete }: { complete: boolean }) {
  const { t } = useTranslator();
  const stages = [
    "settings.import.stage.file",
    "settings.import.stage.review",
    "settings.import.stage.result",
  ] as const;
  return (
    <ol className="mb-3.5 flex flex-wrap items-center gap-1.5 text-[11px] text-muted-foreground">
      {stages.map((stage, index) => (
        <Fragment key={stage}>
          {index > 0 && <li aria-hidden="true">›</li>}
          <li
            aria-current={index === (complete ? 2 : 1) ? "step" : undefined}
            className={
              index === (complete ? 2 : 1)
                ? "font-medium text-foreground"
                : undefined
            }
          >
            {t(stage)}
          </li>
        </Fragment>
      ))}
    </ol>
  );
}

function SourceList({
  busy,
  onOpen,
}: {
  busy: boolean;
  onOpen: (source: ImportSourceGuide) => void;
}) {
  const { t } = useTranslator();

  return (
    <Block>
      {importSources.map((source) => (
        <BlockRowButton
          key={source.id}
          leading={<SourceTile source={source} />}
          title={source.name}
          detail={t(source.formats)}
          disabled={busy}
          onClick={() => onOpen(source)}
        />
      ))}
    </Block>
  );
}

/** SourceStep is one app: how to export from it, and the file to import. */
function SourceStep({
  source,
  reading,
  refusal,
  busy,
  onBack,
  onChoose,
}: {
  source: ImportSourceGuide;
  reading: boolean;
  refusal: Refusal | null;
  busy: boolean;
  onBack: () => void;
  onChoose: () => void;
}) {
  const { t, failure } = useTranslator();

  return (
    <>
      <Button
        type="button"
        variant="ghost"
        size="pill-sm"
        className="-ml-2.5 mb-2 text-muted-foreground"
        disabled={reading}
        onClick={onBack}
      >
        <ChevronLeft data-icon="inline-start" />
        {t("settings.import.sources")}
      </Button>
      <Block>
        <BlockRow
          leading={<SourceTile source={source} />}
          title={source.name}
          detail={t(source.formats)}
        />
      </Block>

      {refusal && (
        <Alert variant="destructive" className="mt-3.5">
          <CircleAlert />
          <AlertTitle className="text-[13px] font-normal">
            {t("settings.import.failed.title")}
          </AlertTitle>
          <AlertDescription className="text-[11px] leading-[1.5]">
            {failure(refusal.cause, refusal.fallback, { source: source.name })}
          </AlertDescription>
        </Alert>
      )}

      <SectionLabel>{t("settings.import.steps")}</SectionLabel>
      <Block>
        {source.steps.map((step, index) => (
          <BlockRow
            key={step.title}
            leading={<StepNumber value={index + 1} />}
            title={t(step.title)}
            detail={t(step.detail)}
            wrap
          />
        ))}
      </Block>

      <div className="mt-4 flex justify-end">
        <Button
          type="button"
          variant="raised"
          size="pill-sm"
          className="h-auto whitespace-normal py-[5px] font-normal"
          disabled={busy || reading}
          onClick={onChoose}
        >
          {t(
            reading
              ? "settings.import.reading"
              : refusal
                ? "settings.import.failed.choose"
                : "settings.import.choose",
          )}
        </Button>
      </div>
      <BlockNote>{t("settings.import.note")}</BlockNote>
    </>
  );
}

/** A logo that is its own tile fills the tile; a mark without a background sits on one. */
function SourceTile({ source }: { source: ImportSourceGuide }) {
  if (source.logoFit === "mark") {
    return (
      <BlockTile>
        <img src={source.logo} alt="" draggable={false} className="size-5" />
      </BlockTile>
    );
  }
  return (
    <img
      src={source.logo}
      alt=""
      draggable={false}
      className="size-[30px] shrink-0 rounded-[10px]"
    />
  );
}

function Review({
  source,
  name,
  review,
  busy,
  onOptions,
  onChange,
  onCancel,
  onRun,
}: {
  source: ImportSourceGuide;
  name: string;
  review: ImportReview;
  busy: boolean;
  onOptions: (options: ImportOptions) => void;
  onChange: () => void;
  onCancel: () => void;
  onRun: () => void;
}) {
  const { t } = useTranslator();
  const { preview, options } = review;
  const listed = preview.skipped.slice(0, listedSkips);
  const unlisted = preview.skipped.length - listed.length;

  function detailOf(kind: ImportKindReview) {
    const parts: string[] = [];
    if (kind.duplicates > 0) {
      parts.push(t("settings.import.in-file", { count: kind.count }));
    }
    if (kind.oneTimeCodes > 0) {
      parts.push(
        t("settings.import.one-time-codes", { count: kind.oneTimeCodes }),
      );
    }
    if (kind.passkeys > 0) {
      parts.push(t("settings.import.passkeys", { count: kind.passkeys }));
    }
    if (kind.duplicates > 0) {
      parts.push(
        t(
          options.skipDuplicates
            ? "settings.import.duplicates.skipped"
            : "settings.import.duplicates.kept",
          { count: kind.duplicates },
        ),
      );
    }
    for (const conversion of kind.converted) {
      parts.push(
        t(`settings.import.from.${conversion.from}`, {
          count: conversion.count,
        }),
      );
    }
    return parts.join(" · ");
  }

  return (
    <>
      <Block>
        <FileRow name={name} preview={preview}>
          <Button
            type="button"
            variant="ghost"
            size="pill-sm"
            disabled={busy}
            onClick={onChange}
          >
            {t("settings.import.change")}
          </Button>
        </FileRow>
      </Block>

      {review.kinds.length > 0 && (
        <>
          <SectionLabel>{t("settings.import.adding")}</SectionLabel>
          <Block>
            {review.kinds.map((kind) => (
              <BlockRow
                key={kind.kind}
                title={t(kindLabel(kind.kind))}
                detail={detailOf(kind)}
              >
                <Count value={kind.adding} />
              </BlockRow>
            ))}
            {review.newGroups > 0 && (
              <BlockRow
                title={t("settings.groups.heading")}
                detail={t(source.grouping.from, { source: source.name })}
              >
                <Count value={review.newGroups} />
              </BlockRow>
            )}
          </Block>
        </>
      )}

      {(review.folders > 0 || review.duplicates > 0) && (
        <Block className="mt-3.5">
          {review.folders > 0 && (
            <BlockRow
              title={t(source.grouping.option)}
              detail={t(source.grouping.detail)}
              htmlFor="import-groups"
            >
              <Switch
                id="import-groups"
                checked={options.groups}
                onCheckedChange={(groups) => onOptions({ ...options, groups })}
              />
            </BlockRow>
          )}
          {review.duplicates > 0 && (
            <BlockRow
              title={t("settings.import.option.skip")}
              detail={t("settings.import.option.skip.detail")}
              htmlFor="import-skip-duplicates"
            >
              <Switch
                id="import-skip-duplicates"
                checked={options.skipDuplicates}
                onCheckedChange={(skipDuplicates) =>
                  onOptions({ ...options, skipDuplicates })
                }
              />
            </BlockRow>
          )}
        </Block>
      )}

      {(review.leftBehind || review.droppedFolders > 0) && (
        <>
          <SectionLabel>{t("settings.import.left")}</SectionLabel>
          <Block>
            {listed.map((skip, index) => (
              <BlockRow
                // biome-ignore lint/suspicious/noArrayIndexKey: skipped items carry no id, their labels repeat or are empty, and the list never reorders.
                key={index}
                title={
                  skip.label ||
                  t(
                    skip.origin
                      ? `settings.import.origin.${skip.origin}`
                      : "settings.import.origin.unknown",
                  )
                }
                detail={t(`settings.import.reason.${skip.reason}`)}
              />
            ))}
            {unlisted > 0 && (
              <BlockRow
                title={t("settings.import.more", { count: unlisted })}
              />
            )}
            {preview.attachments > 0 && (
              <BlockRow
                title={t("settings.import.attachments", {
                  count: preview.attachments,
                })}
                detail={t("settings.import.attachments.detail")}
              />
            )}
            {preview.passkeys > 0 && (
              <BlockRow
                title={t("settings.import.passkeys", {
                  count: preview.passkeys,
                })}
                detail={t("settings.import.passkeys.detail")}
              />
            )}
            {review.droppedFolders > 0 && (
              <BlockRow
                title={t(source.grouping.dropped, {
                  count: review.droppedFolders,
                })}
                detail={t("settings.import.dropped.detail")}
              />
            )}
          </Block>
        </>
      )}

      <BlockNote>{t(source.review, { source: source.name })}</BlockNote>
      {review.empty && <BlockNote>{t("settings.import.nothing")}</BlockNote>}

      <div className="mt-4 flex flex-wrap justify-end gap-2">
        <Button
          type="button"
          variant="quiet"
          size="pill-sm"
          className="bg-tile font-normal"
          onClick={onCancel}
        >
          {t("settings.import.cancel")}
        </Button>
        <Button
          type="button"
          variant="raised"
          size="pill-sm"
          className="h-auto whitespace-normal py-[5px] font-normal"
          disabled={busy || review.empty}
          onClick={onRun}
        >
          <Download data-icon="inline-start" />
          {t("settings.import.run", { count: review.total })}
        </Button>
      </div>
    </>
  );
}

function Importing({ name, review }: { name: string; review: ImportReview }) {
  const { t } = useTranslator();

  return (
    <>
      <Block>
        <FileRow name={name} preview={review.preview} />
        <BlockRow
          title={t("settings.import.importing")}
          detail={t("settings.import.items", { count: review.total })}
        >
          <LoaderCircle
            className="size-4 animate-spin text-muted-foreground motion-reduce:animate-none"
            aria-hidden="true"
          />
        </BlockRow>
      </Block>
      <BlockNote>{t("settings.import.importing.note")}</BlockNote>
    </>
  );
}

function Done({
  flow,
  actions,
  onFile,
  onAnother,
}: {
  flow: DoneFlow;
  actions: ImportActions;
  onFile: (file: DoneFlow["file"]) => void;
  onAnother: () => void;
}) {
  const { t, failure } = useTranslator();
  const { name, result, file } = flow;

  async function trash() {
    onFile("trashing");
    try {
      await actions.trash();
      onFile("trashed");
    } catch (cause) {
      toast.error(failure(cause, "settings.import.error.trash"));
      onFile("kept");
    }
  }

  function reveal() {
    actions.reveal().catch((cause: unknown) => {
      toast.error(failure(cause, "settings.import.error.reveal"));
    });
  }

  return (
    <>
      <Block>
        <BlockRow
          leading={
            <span
              className="flex size-[30px] shrink-0 items-center justify-center rounded-full bg-primary text-primary-foreground"
              aria-hidden="true"
            >
              <Check className="size-4" />
            </span>
          }
          title={t("settings.import.done.title", { count: result.added })}
          detail={t("settings.import.done.detail", { name })}
        />
        {result.kinds.map((total) => (
          <BlockRow key={total.kind} title={t(kindLabel(total.kind))}>
            <Count value={total.count} />
          </BlockRow>
        ))}
        {result.groups > 0 && (
          <BlockRow title={t("settings.import.done.groups")}>
            <Count value={result.groups} />
          </BlockRow>
        )}
      </Block>

      {!isEncryptedFormat(result.format) &&
        (file === "trashed" ? (
          <BlockHint>{t("settings.import.trashed")}</BlockHint>
        ) : (
          <Alert className="mt-3.5 text-warning">
            <TriangleAlert />
            <AlertTitle className="text-[13px] font-normal">
              {t("settings.import.warning.title")}
            </AlertTitle>
            <AlertDescription className="text-[11px] leading-[1.5]">
              <p>{t(warningDetail(result))}</p>
              {result.located && (
                <div className="mt-1.5 flex gap-1.5">
                  <Button
                    type="button"
                    variant="quiet"
                    size="pill-sm"
                    disabled={file === "trashing"}
                    onClick={() => void trash()}
                  >
                    {t("settings.import.trash")}
                  </Button>
                  <Button
                    type="button"
                    variant="ghost"
                    size="pill-sm"
                    disabled={file === "trashing"}
                    onClick={reveal}
                  >
                    {t("settings.import.reveal")}
                  </Button>
                </div>
              )}
            </AlertDescription>
          </Alert>
        ))}

      <div className="mt-4 flex justify-end">
        <Button
          type="button"
          variant="raised"
          size="pill-sm"
          className="h-auto whitespace-normal py-[5px] font-normal"
          disabled={file === "trashing"}
          onClick={onAnother}
        >
          {t("settings.import.another")}
        </Button>
      </div>
    </>
  );
}

/** A wrong password keeps PasswordDialog open with the field focused. */
function PasswordDialog({
  name,
  onUnlock,
  onCancel,
}: {
  name: string;
  /** Resolves false when the password does not open the file. */
  onUnlock: (password: string) => Promise<boolean>;
  onCancel: () => void;
}) {
  const { t } = useTranslator();
  const [password, setPassword] = useState("");
  const [error, setError] = useState<MessageKey | null>(null);
  const [opening, setOpening] = useState(false);
  const field = useRef<HTMLInputElement | null>(null);

  function refuse(message: MessageKey) {
    setError(message);
    field.current?.focus();
    field.current?.select();
  }

  async function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!password) {
      refuse("settings.import.password.empty");
      return;
    }
    setOpening(true);
    try {
      if (!(await onUnlock(password))) refuse("failure.import-password-wrong");
    } finally {
      setOpening(false);
    }
  }

  return (
    <ResponsiveDialog
      open
      dismissible={!opening}
      onOpenChange={(next) => {
        if (!next) onCancel();
      }}
    >
      <ResponsiveDialogContent className="sm:max-w-[420px]">
        <form onSubmit={submit}>
          <ResponsiveDialogHeader>
            <ResponsiveDialogTitle>
              {t("settings.import.password.title")}
            </ResponsiveDialogTitle>
            <ResponsiveDialogDescription>
              {t("settings.import.password.detail", { name })}
            </ResponsiveDialogDescription>
          </ResponsiveDialogHeader>
          <FieldGroup className="py-5">
            <Field data-invalid={error ? true : undefined}>
              <FieldLabel htmlFor="import-password">
                {t("settings.import.password.label")}
              </FieldLabel>
              <Input
                ref={field}
                id="import-password"
                type="password"
                autoComplete="off"
                autoFocus
                value={password}
                readOnly={opening}
                aria-invalid={error ? true : undefined}
                onChange={(event) => {
                  setPassword(event.target.value);
                  setError(null);
                }}
              />
              {error && <FieldError>{t(error)}</FieldError>}
            </Field>
          </FieldGroup>
          <ResponsiveDialogFooter>
            <Button
              type="button"
              variant="quiet"
              size="pill"
              disabled={opening}
              onClick={onCancel}
            >
              {t("settings.import.password.cancel")}
            </Button>
            <Button
              type="submit"
              variant="raised"
              size="pill"
              disabled={opening}
            >
              {t(
                opening
                  ? "settings.import.password.opening"
                  : "settings.import.password.continue",
              )}
            </Button>
          </ResponsiveDialogFooter>
        </form>
      </ResponsiveDialogContent>
    </ResponsiveDialog>
  );
}

function FileRow({
  name,
  preview,
  children,
}: {
  name: string;
  preview: ImportPreview;
  children?: ReactNode;
}) {
  const { t } = useTranslator();

  return (
    <BlockRow
      leading={
        <BlockIconTile
          icon={
            preview.format === "zip" || preview.format === "encrypted-zip"
              ? FileArchive
              : FileText
          }
        />
      }
      title={name}
      detail={t("settings.import.file.detail", {
        format: t(`settings.import.format.${preview.format}`),
        count: preview.items,
      })}
    >
      {children}
    </BlockRow>
  );
}

function Count({ value }: { value: number }) {
  return <span className="text-[13px] tabular-nums">{value}</span>;
}

function SectionLabel({ children }: { children: ReactNode }) {
  return (
    <h3 className="mt-4 mb-1.5 px-[3px] text-[11px] text-muted-foreground">
      {children}
    </h3>
  );
}
