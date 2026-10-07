import { useQuery } from "@tanstack/react-query";
import { cn } from "cn";
import { MapPin } from "lucide-react";
import {
  useDeferredValue,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
} from "react";
import { useCompactLayout } from "../host/compact.ts";
import type { MessageKey } from "../i18n/messages.ts";
import { useTranslator } from "../i18n/translator.tsx";
import { queryKeys } from "../query/keys.ts";
import {
  checksumLabel,
  codeLines,
  completeWord,
  completions,
  keptCodes,
  seedFormats,
  splitWords,
  wordAt,
} from "../seeds/phrase.ts";
import type {
  Group,
  PhraseCheck,
  Seed,
  SeedAddress,
  SeedFormat,
  SeedInput,
  SeedLimits,
  VaultApi,
} from "../vault-api.ts";
import { BlockNote } from "./Block.tsx";
import {
  bareField,
  EditorHeader,
  EditorRow,
  editorRow,
  fieldSection,
  GroupField,
  labelColumn,
  NotesField,
  RemoveButton,
  roomFor,
  TagField,
  type Tagging,
  toggled,
  useRemaining,
  useRows,
} from "./editor/EditorFields.tsx";
import { Button } from "./ui/button.tsx";
import { Input } from "./ui/input.tsx";
import { ScrollArea } from "./ui/scroll-area.tsx";
import { Textarea } from "./ui/textarea.tsx";
import { RevealButton } from "./workspace/Fields.tsx";
import { Segmented } from "./workspace/Segmented.tsx";
import { toneText } from "./workspace/tones.ts";

const formID = "seed-editor";

const block = "min-w-0 shrink-0 overflow-hidden rounded-row bg-field";

const longField = cn(
  bareField,
  "min-h-20 resize-none font-mono text-[13px] leading-[1.6]",
);

const formatChoiceRow = cn(
  editorRow,
  "max-sm:flex-col max-sm:items-stretch max-sm:gap-2 max-sm:py-2.5",
);

const noWords: readonly string[] = [];

const formatNames: Record<SeedFormat, MessageKey> = {
  phrase: "seed.format.phrase",
  key: "seed.format.key",
  codes: "seed.format.codes",
};

const requirements: Record<SeedFormat, MessageKey> = {
  phrase: "seed.editor.requirement.phrase",
  key: "seed.editor.requirement.key",
  codes: "seed.editor.requirement.codes",
};

const emptySeedAddress: SeedAddress = { label: "", value: "" };

/** What the editor asks of the host besides saving. */
export type SeedEditorHost = Pick<VaultApi, "checkSeedPhrase" | "seedWordlist">;

function characters(value: string): number {
  return [...value].length;
}

export function SeedEditor({
  initial,
  initialGroups,
  tagging,
  groups,
  limits,
  busy,
  host,
  onSave,
  onCancel,
}: {
  initial?: Seed;
  /** The groups the seed starts in, which for a new one is the chosen default. */
  initialGroups: string[];
  tagging: Tagging;
  groups: Group[];
  limits: SeedLimits | null;
  busy: boolean;
  host: SeedEditorHost;
  onSave: (input: SeedInput, groups: string[]) => void;
  onCancel: () => void;
}) {
  const { t } = useTranslator();
  const compact = useCompactLayout();
  const counter = useRemaining();
  const [label, setLabel] = useState(initial?.label ?? "");
  const [format, setFormat] = useState<SeedFormat>(initial?.format ?? "phrase");
  const [phrase, setPhrase] = useState(() => initial?.words.join(" ") ?? "");
  // Null keeps an opened seed from offering completions before the caret moves.
  const [caret, setCaret] = useState<number | null>(null);
  const [passphrase, setPassphrase] = useState(initial?.passphrase ?? "");
  const [passphraseShown, setPassphraseShown] = useState(false);
  const [path, setPath] = useState(initial?.path ?? "");
  const [key, setKey] = useState(initial?.key ?? "");
  const [keyShown, setKeyShown] = useState(false);
  const [codes, setCodes] = useState(
    () => initial?.codes.map((code) => code.value).join("\n") ?? "",
  );
  // Keyed by value: a code keeps its mark wherever it moves in the field.
  const [used, setUsed] = useState<ReadonlySet<string>>(
    () =>
      new Set(
        initial?.codes.filter((code) => code.used).map((code) => code.value),
      ),
  );
  const [wallet, setWallet] = useState(initial?.wallet ?? "");
  const addresses = useRows<SeedAddress>(initial?.addresses ?? []);
  const [notes, setNotes] = useState(initial?.notes ?? "");
  const [membership, setMembership] = useState<string[]>(initialGroups);
  const [tags, setTags] = useState<string[]>(initial?.tags ?? []);
  const [check, setCheck] = useState<{
    phrase: string;
    result: PhraseCheck;
  } | null>(null);
  const phraseField = useRef<HTMLTextAreaElement | null>(null);
  const caretToPlace = useRef<number | null>(null);

  const words = splitWords(phrase);
  const checkedPhrase = words.join(" ");
  const lines = codeLines(codes);

  const wordlist =
    useQuery({
      queryKey: queryKeys.seedWordlist,
      queryFn: () => host.seedWordlist(),
      meta: { failure: "seed.error.wordlist" },
    }).data ?? noWords;

  // Security: the phrase goes to the host directly and never into the query cache or a query key.
  const deferredPhrase = useDeferredValue(
    format === "phrase" ? checkedPhrase : "",
  );
  useEffect(() => {
    if (!deferredPhrase) return;
    const superseded = new AbortController();
    host.checkSeedPhrase(deferredPhrase.split(" ")).then(
      (result) => {
        if (!superseded.signal.aborted) {
          setCheck({ phrase: deferredPhrase, result });
        }
      },
      // The findings are advisory: a phrase the host cannot check shows none and still saves.
      () => {
        if (!superseded.signal.aborted) setCheck(null);
      },
    );
    return () => superseded.abort();
  }, [host, deferredPhrase]);

  const phraseCheck =
    check && check.phrase === checkedPhrase ? check.result : null;
  const suggestions =
    caret === null ? [] : completions(wordAt(phrase, caret).word, wordlist);

  useLayoutEffect(() => {
    const at = caretToPlace.current;
    if (at === null) return;
    caretToPlace.current = null;
    phraseField.current?.focus();
    phraseField.current?.setSelectionRange(at, at);
  });

  function complete(word: string) {
    if (caret === null) return;
    const next = completeWord(phrase, caret, word);
    setPhrase(next.text);
    setCaret(next.caret);
    caretToPlace.current = next.caret;
  }

  const tooManyWords = limits !== null && words.length > limits.words;
  const wordTooLong =
    limits !== null && words.some((word) => characters(word) > limits.word);
  const tooManyCodes = limits !== null && lines.length > limits.codes;
  const codeTooLong =
    limits !== null && lines.some((line) => characters(line) > limits.code);
  const keptAddresses = addresses.values.filter(
    (address) => address.label.trim() || address.value.trim(),
  );
  const addressMissing = keptAddresses.some((address) => !address.value.trim());
  const name = label.trim();

  const keyCounter = counter(key, limits?.key, 100);
  const filled = {
    phrase: words.length > 0 && !tooManyWords && !wordTooLong,
    key: key.trim() !== "",
    codes: lines.length > 0 && !tooManyCodes && !codeTooLong,
  }[format];
  const unfinished =
    addressMissing ||
    (format === "phrase" && (tooManyWords || wordTooLong)) ||
    (format === "codes" && (tooManyCodes || codeTooLong));

  function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    onSave(
      {
        label: name,
        format,
        words: format === "phrase" ? words : [],
        passphrase: format === "phrase" ? passphrase : "",
        path: format === "phrase" ? path : "",
        key: format === "key" ? key : "",
        codes: format === "codes" ? keptCodes(lines, used) : [],
        wallet,
        addresses: keptAddresses,
        notes,
        tags,
      },
      membership,
    );
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-[11px]">
      <EditorHeader
        title={t(initial ? "seed.editor.edit.title" : "seed.editor.new.title")}
        name={name}
        form={formID}
        submit={t(initial ? "workspace.editor.update" : "seed.editor.create")}
        canSubmit={Boolean(name) && filled && !unfinished}
        busy={busy}
        onCancel={onCancel}
      />

      <ScrollArea className="min-h-0 flex-1">
        <form
          id={formID}
          onSubmit={submit}
          className="flex flex-1 flex-col gap-2"
        >
          <div className={block}>
            <EditorRow
              label={t("seed.field.label")}
              htmlFor="seed-name"
              counter={counter(label, limits?.label, 20)}
            >
              <Input
                id="seed-name"
                className={bareField}
                value={label}
                onChange={(event) => setLabel(event.target.value)}
                placeholder={t("seed.field.label.placeholder")}
                maxLength={limits?.label}
                disabled={busy}
                required
              />
            </EditorRow>
            <div className={initial ? editorRow : formatChoiceRow}>
              <span className={labelColumn}>{t("seed.format.label")}</span>
              {initial ? (
                <span className="min-w-0 flex-1 truncate text-[13px]">
                  {t(formatNames[format])}
                </span>
              ) : (
                <Segmented
                  id="seed-format"
                  legend={t("seed.format.label")}
                  options={seedFormats.map((value) => ({
                    value,
                    label: t(formatNames[value]),
                  }))}
                  value={format}
                  disabled={busy}
                  stretch={compact}
                  onChange={setFormat}
                />
              )}
            </div>
            <EditorRow
              label={t("seed.field.wallet")}
              htmlFor="seed-wallet"
              counter={counter(wallet, limits?.wallet, 20)}
            >
              <Input
                id="seed-wallet"
                className={bareField}
                value={wallet}
                onChange={(event) => setWallet(event.target.value)}
                placeholder={t("seed.field.wallet.placeholder")}
                maxLength={limits?.wallet}
                autoComplete="off"
                disabled={busy}
              />
            </EditorRow>
            <TagField
              tags={tags}
              tagging={tagging}
              busy={busy}
              onChange={setTags}
            />
            {groups.length > 0 && (
              <GroupField
                groups={groups}
                membership={membership}
                busy={busy}
                onToggle={(id) =>
                  setMembership((current) => toggled(current, id))
                }
              />
            )}
          </div>

          {format === "phrase" && (
            <>
              <section className={fieldSection}>
                <div className="mb-1 flex items-baseline gap-2">
                  <label
                    className="text-[11px] text-muted-foreground"
                    htmlFor="seed-phrase"
                  >
                    {t("seed.field.phrase")}
                  </label>
                  {words.length > 0 && (
                    <span className="ml-auto text-[11px] text-faint tabular-nums">
                      {t("seed.summary.words", { count: words.length })}
                    </span>
                  )}
                </div>
                <Textarea
                  id="seed-phrase"
                  ref={phraseField}
                  value={phrase}
                  onChange={(event) => {
                    setPhrase(event.target.value);
                    setCaret(event.target.selectionStart);
                  }}
                  onSelect={(event) =>
                    setCaret(event.currentTarget.selectionStart)
                  }
                  placeholder={t("seed.editor.phrase.placeholder")}
                  rows={3}
                  autoComplete="off"
                  autoCapitalize="none"
                  autoCorrect="off"
                  spellCheck={false}
                  aria-invalid={tooManyWords || wordTooLong ? true : undefined}
                  className={longField}
                  disabled={busy}
                />
                {suggestions.length > 0 && (
                  <fieldset className="mt-2 flex flex-wrap gap-1.5">
                    <legend className="sr-only">
                      {t("seed.editor.phrase.suggestions")}
                    </legend>
                    {suggestions.map((word) => (
                      <Button
                        key={word}
                        type="button"
                        variant="quiet"
                        size="pill-sm"
                        className="font-mono"
                        disabled={busy}
                        onMouseDown={(event) => event.preventDefault()}
                        onClick={() => complete(word)}
                      >
                        {word}
                      </Button>
                    ))}
                  </fieldset>
                )}
                <PhraseFindings
                  check={phraseCheck}
                  tooManyWords={tooManyWords}
                  wordTooLong={wordTooLong}
                  limits={limits}
                />
              </section>

              <div>
                <div className={block}>
                  <EditorRow
                    label={t("seed.field.passphrase")}
                    htmlFor="seed-passphrase"
                  >
                    <Input
                      id="seed-passphrase"
                      type={passphraseShown ? "text" : "password"}
                      className={cn(bareField, "font-mono")}
                      value={passphrase}
                      onChange={(event) => setPassphrase(event.target.value)}
                      maxLength={limits?.passphrase}
                      autoComplete="off"
                      spellCheck={false}
                      disabled={busy}
                    />
                    <RevealButton
                      shown={passphraseShown}
                      revealLabel={t("seed.passphrase.reveal")}
                      concealLabel={t("seed.passphrase.conceal")}
                      busy={busy}
                      onToggle={() => setPassphraseShown((shown) => !shown)}
                    />
                  </EditorRow>
                  <EditorRow
                    label={t("seed.field.path")}
                    htmlFor="seed-path"
                    counter={counter(path, limits?.path, 20)}
                  >
                    <Input
                      id="seed-path"
                      className={cn(bareField, "font-mono")}
                      value={path}
                      onChange={(event) => setPath(event.target.value)}
                      maxLength={limits?.path}
                      autoComplete="off"
                      spellCheck={false}
                      disabled={busy}
                    />
                  </EditorRow>
                </div>
                <BlockNote>{t("seed.field.path.note")}</BlockNote>
              </div>
            </>
          )}

          {format === "key" && (
            <section className={fieldSection}>
              <div className="mb-1 flex items-center gap-2">
                <label
                  className="text-[11px] text-muted-foreground"
                  htmlFor="seed-key"
                >
                  {t("seed.field.key")}
                </label>
                <span className="ml-auto flex items-center gap-2">
                  {keyCounter && (
                    <span className="text-[11px] text-faint tabular-nums">
                      {keyCounter}
                    </span>
                  )}
                  <RevealButton
                    shown={keyShown}
                    revealLabel={t("seed.key.reveal")}
                    concealLabel={t("seed.key.conceal")}
                    busy={busy}
                    onToggle={() => setKeyShown((shown) => !shown)}
                  />
                </span>
              </div>
              <Textarea
                id="seed-key"
                value={key}
                onChange={(event) => setKey(event.target.value)}
                maxLength={limits?.key}
                rows={3}
                autoComplete="off"
                autoCapitalize="none"
                autoCorrect="off"
                spellCheck={false}
                className={cn(
                  longField,
                  "break-all",
                  !keyShown && "[-webkit-text-security:disc]",
                )}
                disabled={busy}
              />
            </section>
          )}

          {format === "codes" && (
            <section className={fieldSection}>
              <div className="mb-1 flex items-baseline gap-2">
                <label
                  className="text-[11px] text-muted-foreground"
                  htmlFor="seed-codes"
                >
                  {t("seed.field.codes")}
                </label>
                {lines.length > 0 && (
                  <span className="ml-auto text-[11px] text-faint tabular-nums">
                    {t("seed.editor.codes.count", { count: lines.length })}
                  </span>
                )}
              </div>
              <Textarea
                id="seed-codes"
                value={codes}
                onChange={(event) => setCodes(event.target.value)}
                placeholder={t("seed.editor.codes.placeholder")}
                rows={4}
                autoComplete="off"
                autoCapitalize="none"
                autoCorrect="off"
                spellCheck={false}
                aria-invalid={tooManyCodes || codeTooLong ? true : undefined}
                className={longField}
                disabled={busy}
              />
              {tooManyCodes && limits && (
                <p className="mt-1.5 text-[11px] text-destructive">
                  {t("seed.editor.codes.too-many", { limit: limits.codes })}
                </p>
              )}
              {codeTooLong && limits && (
                <p className="mt-1.5 text-[11px] text-destructive">
                  {t("seed.editor.codes.too-long", { limit: limits.code })}
                </p>
              )}
              {initial && lines.length > 0 && (
                <UsedCodes
                  codes={[...new Set(lines)]}
                  used={used}
                  busy={busy}
                  onToggle={(value) =>
                    setUsed((current) => {
                      const next = new Set(current);
                      if (next.has(value)) next.delete(value);
                      else next.add(value);
                      return next;
                    })
                  }
                />
              )}
            </section>
          )}

          {addresses.rows.map((row) => (
            <fieldset key={row.key} className={block}>
              <legend className="sr-only">{t("seed.field.address")}</legend>
              <EditorRow
                label={t("seed.field.address-label")}
                htmlFor={`seed-address-${row.key}-label`}
              >
                <Input
                  id={`seed-address-${row.key}-label`}
                  className={bareField}
                  value={row.value.label}
                  onChange={(event) =>
                    addresses.change(row.key, {
                      ...row.value,
                      label: event.target.value,
                    })
                  }
                  maxLength={limits?.addressLabel}
                  autoComplete="off"
                  disabled={busy}
                />
                <RemoveButton
                  label={t("seed.editor.address.remove")}
                  busy={busy}
                  onRemove={() => addresses.remove(row.key)}
                />
              </EditorRow>
              <EditorRow
                label={t("seed.field.address")}
                htmlFor={`seed-address-${row.key}-value`}
              >
                <Input
                  id={`seed-address-${row.key}-value`}
                  className={cn(bareField, "font-mono")}
                  value={row.value.value}
                  onChange={(event) =>
                    addresses.change(row.key, {
                      ...row.value,
                      value: event.target.value,
                    })
                  }
                  maxLength={limits?.address}
                  autoComplete="off"
                  autoCapitalize="none"
                  spellCheck={false}
                  disabled={busy}
                />
              </EditorRow>
            </fieldset>
          ))}

          {roomFor(addresses.rows.length, limits?.addresses) && (
            <div className="flex shrink-0 flex-wrap items-center gap-1.5 px-0.5">
              <Button
                type="button"
                variant="quiet"
                size="pill-sm"
                disabled={busy}
                onClick={() => addresses.add(emptySeedAddress)}
              >
                <MapPin data-icon="inline-start" />
                {t("seed.editor.address.add")}
              </Button>
            </div>
          )}

          <NotesField
            id="seed-notes"
            value={notes}
            limit={limits?.notes}
            busy={busy}
            onChange={setNotes}
          />

          <p className="shrink-0 px-1 text-[11px] text-faint">
            {t(requirements[format])}
          </p>
          {addressMissing && (
            <p className="shrink-0 px-1 text-[11px] text-destructive">
              {t("seed.editor.address.missing")}
            </p>
          )}
          {unfinished && !addressMissing && (
            <p className="shrink-0 px-1 text-[11px] text-destructive">
              {t("seed.editor.unfinished")}
            </p>
          )}
        </form>
      </ScrollArea>
    </div>
  );
}

/** Words outside the BIP-39 list still save; the findings only warn. */
function PhraseFindings({
  check,
  tooManyWords,
  wordTooLong,
  limits,
}: {
  check: PhraseCheck | null;
  tooManyWords: boolean;
  wordTooLong: boolean;
  limits: SeedLimits | null;
}) {
  const { t } = useTranslator();
  const checksum = check ? checksumLabel(check.checksum) : null;

  return (
    <div
      className="mt-1.5 flex flex-col gap-0.5 text-[11px]"
      aria-live="polite"
    >
      {check && check.unknownWords.length > 0 && (
        <p className={toneText.warning}>
          {t("seed.editor.phrase.unknown", {
            positions: check.unknownWords
              .map((position) => position + 1)
              .join(", "),
          })}
        </p>
      )}
      {checksum && (
        <p className={toneText[checksum.warning ? "warning" : "default"]}>
          {t(checksum.key)}
        </p>
      )}
      {tooManyWords && limits && (
        <p className="text-destructive">
          {t("seed.editor.phrase.too-many", { limit: limits.words })}
        </p>
      )}
      {wordTooLong && limits && (
        <p className="text-destructive">
          {t("seed.editor.phrase.too-long", { limit: limits.word })}
        </p>
      )}
    </div>
  );
}

/** The codes of an existing seed, each a toggle for whether it is already used. */
function UsedCodes({
  codes,
  used,
  busy,
  onToggle,
}: {
  codes: readonly string[];
  used: ReadonlySet<string>;
  busy: boolean;
  onToggle: (value: string) => void;
}) {
  const { t } = useTranslator();

  return (
    <fieldset className="mt-2.5 flex flex-col gap-1.5">
      <legend className="mb-1.5 text-[11px] text-muted-foreground">
        {t("seed.editor.codes.used")}
      </legend>
      <p className="text-[11px] text-faint">
        {t("seed.editor.codes.used.detail")}
      </p>
      <div className="grid grid-cols-2 gap-1.5">
        {codes.map((code) => {
          const marked = used.has(code);
          return (
            <Button
              key={code}
              type="button"
              variant="quiet"
              size="pill-sm"
              aria-pressed={marked}
              disabled={busy}
              onClick={() => onToggle(code)}
              className={cn(
                "justify-start font-mono",
                marked && "text-faint line-through",
              )}
            >
              <span className="truncate">{code}</span>
            </Button>
          );
        })}
      </div>
    </fieldset>
  );
}
