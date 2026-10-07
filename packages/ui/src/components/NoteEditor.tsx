import { cn } from "cn";
import { useState } from "react";
import { useTranslator } from "../i18n/translator.tsx";
import type { Group, Note, NoteInput, NoteLimits } from "../vault-api.ts";
import {
  bareField,
  EditorHeader,
  EditorRow,
  fieldSection,
  GroupField,
  TagField,
  type Tagging,
  toggled,
  useRemaining,
} from "./editor/EditorFields.tsx";
import { Input } from "./ui/input.tsx";
import { Switch } from "./ui/switch.tsx";
import { Textarea } from "./ui/textarea.tsx";

const formID = "note-editor";

const block = "min-w-0 shrink-0 overflow-hidden rounded-row bg-field";

export function NoteEditor({
  initial,
  initialGroups,
  tagging,
  groups,
  limits,
  busy,
  onSave,
  onCancel,
}: {
  initial?: Note;
  /** The groups the note starts in, which for a new one is the chosen default. */
  initialGroups: string[];
  tagging: Tagging;
  groups: Group[];
  limits: NoteLimits | null;
  busy: boolean;
  onSave: (input: NoteInput, groups: string[]) => void;
  onCancel: () => void;
}) {
  const { t } = useTranslator();
  const counter = useRemaining();
  const [label, setLabel] = useState(initial?.label ?? "");
  const [body, setBody] = useState(initial?.body ?? "");
  const [hidden, setHidden] = useState(initial?.hidden ?? false);
  const [membership, setMembership] = useState<string[]>(initialGroups);
  const [tags, setTags] = useState<string[]>(initial?.tags ?? []);
  const name = label.trim();
  const remaining = counter(body, limits?.body, 500);

  function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    onSave({ label: name, body, hidden, tags }, membership);
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-[11px]">
      <EditorHeader
        title={t(initial ? "note.editor.edit.title" : "note.editor.new.title")}
        name={name}
        form={formID}
        submit={t(initial ? "workspace.editor.update" : "note.editor.create")}
        canSubmit={Boolean(name)}
        busy={busy}
        onCancel={onCancel}
      />

      <form
        id={formID}
        onSubmit={submit}
        className="flex min-h-0 flex-1 flex-col gap-2"
      >
        <div className={block}>
          <EditorRow
            label={t("note.field.label")}
            htmlFor="note-name"
            counter={counter(label, limits?.label, 20)}
          >
            <Input
              id="note-name"
              className={bareField}
              value={label}
              onChange={(event) => setLabel(event.target.value)}
              placeholder={t("note.field.label.placeholder")}
              maxLength={limits?.label}
              disabled={busy}
              required
            />
          </EditorRow>
          <EditorRow label={t("note.field.hidden")} htmlFor="note-hidden">
            <Switch
              id="note-hidden"
              checked={hidden}
              onCheckedChange={setHidden}
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

        <section className={cn(fieldSection, "flex min-h-0 flex-1 flex-col")}>
          <div className="mb-1 flex shrink-0 items-baseline gap-2">
            <label
              className="text-[11px] text-muted-foreground"
              htmlFor="note-body"
            >
              {t("note.field.body")}
            </label>
            {remaining && (
              <span className="ml-auto text-[11px] text-faint tabular-nums">
                {remaining}
              </span>
            )}
          </div>
          <Textarea
            id="note-body"
            value={body}
            onChange={(event) => setBody(event.target.value)}
            maxLength={limits?.body}
            className={cn(
              bareField,
              "field-sizing-fixed min-h-0 flex-1 resize-none text-[13px] leading-[1.6]",
            )}
            disabled={busy}
          />
        </section>

        <p className="shrink-0 px-1 text-[11px] text-faint">
          {t("note.editor.requirement")}
        </p>
      </form>
    </div>
  );
}
