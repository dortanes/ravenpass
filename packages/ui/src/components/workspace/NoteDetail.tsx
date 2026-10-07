import { Eye, EyeOff, NotebookText } from "lucide-react";
import { useId, useState } from "react";
import type { MessageKey } from "../../i18n/messages.ts";
import { useTranslator } from "../../i18n/translator.tsx";
import { splitWords } from "../../seeds/phrase.ts";
import type { Note } from "../../vault-api.ts";
import { Button } from "../ui/button.tsx";
import { ScrollArea } from "../ui/scroll-area.tsx";
import { type DetailControls, DetailHeader } from "./DetailHeader.tsx";
import { CopyButton, RevealButton } from "./Fields.tsx";
import { type RevealHost, useRevealGate } from "./RevealGate.tsx";

/** NoteDetail owns a hidden note's reveal, so it ends with this pane; showing or copying a hidden note asks the owner first. */
export function NoteDetail({
  note,
  controls,
  onCopy,
  host,
  onFailure,
}: {
  note: Note;
  controls: DetailControls;
  onCopy: () => void;
  host: RevealHost;
  onFailure: (cause: unknown, message: MessageKey) => void;
}) {
  const { t } = useTranslator();
  const { busy } = controls;
  const heading = useId();
  const [shown, setShown] = useState(false);
  const covered = note.hidden && !shown;
  const title = note.label || t("note.untitled");
  const gate = useRevealGate(host, title, onFailure);

  return (
    <article className="flex min-h-0 flex-1 flex-col gap-[11px]">
      <DetailHeader
        label={note.label}
        title={title}
        icon={NotebookText}
        subtitle={t("note.words", { count: splitWords(note.body).length })}
        controls={
          note.hidden
            ? { ...controls, onEdit: () => gate.after(controls.onEdit) }
            : controls
        }
      />

      <section
        className="flex min-h-0 flex-1 flex-col rounded-row bg-field"
        aria-labelledby={heading}
      >
        <div className="flex min-h-[35px] shrink-0 items-center gap-2 pt-1 pr-1.5 pl-[13px]">
          <h3
            id={heading}
            className="min-w-0 flex-1 text-[11px] font-normal text-muted-foreground"
          >
            {t("note.field.body")}
          </h3>
          {note.hidden && !covered && (
            <RevealButton
              shown
              revealLabel={t("note.hidden.show")}
              concealLabel={t("note.hidden.hide")}
              onToggle={() => setShown(false)}
            />
          )}
          {note.body && (
            <CopyButton
              label={t("note.copy")}
              busy={busy}
              onCopy={() => (note.hidden ? gate.after(onCopy) : onCopy())}
            />
          )}
        </div>
        {covered ? (
          <div className="flex flex-1 flex-col items-center justify-center gap-3 px-6 pb-6 text-center">
            <EyeOff
              className="size-5 text-muted-foreground"
              aria-hidden="true"
            />
            <p className="text-[13px] text-muted-foreground">
              {t("note.hidden.detail")}
            </p>
            <Button
              type="button"
              variant="quiet"
              size="pill-sm"
              onClick={() => gate.after(() => setShown(true))}
            >
              <Eye data-icon="inline-start" />
              {t("note.hidden.show")}
            </Button>
          </div>
        ) : (
          <ScrollArea className="min-h-0 flex-1">
            {note.body ? (
              <p className="px-[13px] pb-[13px] text-[13px] leading-[1.6] break-words whitespace-pre-wrap select-text">
                {note.body}
              </p>
            ) : (
              <p className="px-[13px] pb-[13px] text-[13px] text-muted-foreground">
                {t("note.body.empty")}
              </p>
            )}
          </ScrollArea>
        )}
      </section>
      {gate.dialog}
    </article>
  );
}
