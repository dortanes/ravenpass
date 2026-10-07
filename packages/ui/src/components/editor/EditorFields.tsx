import { cn } from "cn";
import { Check, ChevronsUpDown, Plus, Sparkles, X } from "lucide-react";
import { Popover as PopoverPrimitive } from "radix-ui";
import {
  type KeyboardEvent,
  type ReactNode,
  useId,
  useRef,
  useState,
} from "react";
import { useTranslator } from "../../i18n/translator.tsx";
import type { Group, TagLimits } from "../../vault-api.ts";
import { suggestedTags, withTag } from "../../workspace/tags.ts";
import { Button } from "../ui/button.tsx";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "../ui/command.tsx";
import {
  Popover,
  PopoverAnchor,
  PopoverContent,
  PopoverTrigger,
} from "../ui/popover.tsx";
import { Textarea } from "../ui/textarea.tsx";

/** A field without a border; its row shows the focus. */
export const bareField =
  "h-auto border-0 bg-transparent px-0 py-0 text-[13px] shadow-none focus-visible:border-0 focus-visible:ring-0";

/** The first and last rows round their own corners, or the block's clip cuts the focus ring. */
export const editorRow =
  "flex min-h-[41px] items-center gap-2.5 border-b px-[13px] py-1.5 first:rounded-t-row last:rounded-b-row last:border-b-0 focus-within:bg-field-hover focus-within:inset-ring focus-within:inset-ring-foreground/14";

export const labelColumn =
  "w-[88px] shrink-0 text-[11px] text-muted-foreground";

/** A block holding one long field under its label, such as notes. */
export const fieldSection =
  "shrink-0 rounded-row bg-field px-[13px] py-[11px] focus-within:bg-field-hover focus-within:inset-ring focus-within:inset-ring-foreground/14";

/** useRemaining returns the characters left once a field nears its limit, and nothing before. */
export function useRemaining() {
  const { t } = useTranslator();
  return (value: string, limit: number | undefined, margin: number) => {
    if (!limit || value.length <= limit - margin) return undefined;
    return t("workspace.editor.remaining", { used: value.length, limit });
  };
}

export function EditorRow({
  label,
  htmlFor,
  counter,
  children,
}: {
  label: string;
  htmlFor: string;
  counter?: string;
  children: ReactNode;
}) {
  return (
    <div className={editorRow}>
      <label className={labelColumn} htmlFor={htmlFor}>
        {label}
      </label>
      <div className="flex min-w-0 flex-1 items-center gap-1">{children}</div>
      {counter && (
        <span className="shrink-0 text-[11px] text-faint tabular-nums">
          {counter}
        </span>
      )}
    </div>
  );
}

/** GroupField picks groups from a searchable list and shows the chosen ones in its row. */
export function GroupField({
  groups,
  membership,
  busy,
  onToggle,
}: {
  groups: Group[];
  membership: string[];
  busy: boolean;
  onToggle: (id: string) => void;
}) {
  const { t } = useTranslator();
  const [open, setOpen] = useState(false);
  const chosen = groups.filter((group) => membership.includes(group.id));

  return (
    <div className={editorRow}>
      <span className={labelColumn}>{t("workspace.editor.groups")}</span>
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <button
            type="button"
            disabled={busy}
            aria-expanded={open}
            className="flex min-w-0 flex-1 items-center gap-1.5 rounded-md py-1 text-left outline-none disabled:opacity-50"
          >
            <span className="flex min-w-0 flex-1 flex-wrap items-center gap-1.5">
              {chosen.map((group) => (
                <span
                  key={group.id}
                  className="max-w-[140px] truncate rounded-full bg-tile px-2 py-px text-[11px]"
                >
                  {group.name}
                </span>
              ))}
              {!chosen.length && (
                <span className="text-[13px] text-muted-foreground">
                  {t("workspace.editor.groups.none")}
                </span>
              )}
            </span>
            <ChevronsUpDown className="size-3.5 shrink-0 text-muted-foreground" />
          </button>
        </PopoverTrigger>
        <PopoverContent align="start" className="w-[240px]">
          <Command>
            <CommandInput placeholder={t("workspace.editor.groups.search")} />
            <CommandList>
              <CommandEmpty>{t("workspace.editor.groups.empty")}</CommandEmpty>
              <CommandGroup>
                {groups.map((group) => (
                  <CommandItem
                    key={group.id}
                    value={group.name}
                    onSelect={() => onToggle(group.id)}
                  >
                    <span className="min-w-0 flex-1 truncate">
                      {group.name}
                    </span>
                    {membership.includes(group.id) && (
                      <Check className="size-4 text-foreground" />
                    )}
                  </CommandItem>
                ))}
              </CommandGroup>
            </CommandList>
          </Command>
        </PopoverContent>
      </Popover>
    </div>
  );
}

/** What an editor's tag field offers: the tags the vault's items carry and the core's bounds. */
export interface Tagging {
  known: string[];
  limits: TagLimits | null;
}

/** TagField holds an item's tags: Enter or a comma adds what is typed, and Backspace in an empty field takes the last. */
export function TagField({
  tags,
  tagging,
  busy,
  onChange,
}: {
  tags: string[];
  tagging: Tagging;
  busy: boolean;
  onChange: (tags: string[]) => void;
}) {
  const { t } = useTranslator();
  const id = useId();
  const [typed, setTyped] = useState("");
  const [focused, setFocused] = useState(false);
  const { limits } = tagging;
  const full = limits !== null && tags.length >= limits.tags;
  const suggestions =
    focused && !full ? suggestedTags(tagging.known, tags, typed) : [];

  function add(value: string) {
    const next = withTag(tags, value, limits);
    if (next.length !== tags.length) onChange(next);
    setTyped("");
  }

  function keyDown(event: KeyboardEvent<HTMLInputElement>) {
    if (event.key === "Enter" || event.key === ",") {
      // Enter would save the whole item.
      event.preventDefault();
      add(typed);
    } else if (event.key === "Backspace" && !typed && tags.length) {
      onChange(tags.slice(0, -1));
    }
  }

  return (
    <div className={editorRow}>
      <label className={labelColumn} htmlFor={full ? undefined : id}>
        {t("workspace.editor.tags")}
      </label>
      <div className="flex min-w-0 flex-1 flex-wrap items-center gap-1.5">
        {tags.map((tag) => (
          <span
            key={tag}
            className="flex max-w-[160px] items-center gap-0.5 rounded-full bg-tile py-px pr-0.5 pl-2 text-[11px]"
          >
            <span className="truncate">{tag}</span>
            <button
              type="button"
              className="flex size-4 shrink-0 items-center justify-center rounded-full text-muted-foreground outline-none hover:bg-control hover:text-foreground focus-visible:ring-2 focus-visible:ring-ring/40 disabled:opacity-50"
              aria-label={t("workspace.editor.tags.remove", { tag })}
              title={t("workspace.editor.tags.remove", { tag })}
              disabled={busy}
              onClick={() => onChange(tags.filter((held) => held !== tag))}
            >
              <X className="size-3" />
            </button>
          </span>
        ))}
        {!full && (
          <input
            id={id}
            className="h-6 min-w-16 flex-1 border-0 bg-transparent p-0 text-[13px] outline-none placeholder:text-faint disabled:opacity-50"
            value={typed}
            onChange={(event) => setTyped(event.target.value)}
            onKeyDown={keyDown}
            onFocus={() => setFocused(true)}
            // What is typed is kept when the editor saves without Enter.
            onBlur={() => {
              setFocused(false);
              add(typed);
            }}
            placeholder={
              tags.length ? undefined : t("workspace.editor.tags.placeholder")
            }
            maxLength={limits?.tag}
            autoCapitalize="none"
            autoComplete="off"
            spellCheck={false}
            disabled={busy}
          />
        )}
        {suggestions.map((tag) => (
          <button
            key={tag}
            type="button"
            className="flex max-w-[160px] items-center gap-1 rounded-full border border-dashed border-foreground/16 px-2 py-px text-[11px] text-muted-foreground outline-none hover:border-foreground/30 hover:text-foreground"
            aria-label={t("workspace.editor.tags.add", { tag })}
            // Pressing would blur the field, adding the half-typed tag first.
            onMouseDown={(event) => event.preventDefault()}
            onClick={() => add(tag)}
            disabled={busy}
          >
            <Plus className="size-3 shrink-0" aria-hidden="true" />
            <span className="truncate">{tag}</span>
          </button>
        ))}
      </div>
    </div>
  );
}

interface Row<Value> {
  key: number;
  value: Value;
}

/** useRows keys each row itself, or a removal would hand one row's field to another. */
export function useRows<Value>(initial: readonly Value[]) {
  const next = useRef(initial.length);
  const [rows, setRows] = useState<Row<Value>[]>(() =>
    initial.map((value, key) => ({ key, value })),
  );

  return {
    rows,
    values: rows.map((row) => row.value),
    add(value: Value) {
      const key = next.current++;
      setRows((current) => [...current, { key, value }]);
    },
    remove(key: number) {
      setRows((current) => current.filter((row) => row.key !== key));
    },
    change(key: number, value: Value) {
      setRows((current) =>
        current.map((row) => (row.key === key ? { key, value } : row)),
      );
    },
    /** update reads the row's current value, for a change that lands after an await. */
    update(key: number, next: (value: Value) => Value) {
      setRows((current) =>
        current.map((row) =>
          row.key === key ? { key, value: next(row.value) } : row,
        ),
      );
    },
  };
}

/** Whether another row fits under a count limit. Unknown limits leave the choice to the vault. */
export function roomFor(count: number, limit: number | undefined): boolean {
  return limit === undefined || count < limit;
}

export function RemoveButton({
  label,
  busy,
  onRemove,
}: {
  label: string;
  busy: boolean;
  onRemove: () => void;
}) {
  return (
    <Button
      type="button"
      variant="ghost"
      size="icon-sm"
      className="size-7 shrink-0 rounded-md text-muted-foreground hover:text-foreground"
      onClick={onRemove}
      aria-label={label}
      title={label}
      disabled={busy}
    >
      <X className="size-4" />
    </Button>
  );
}

/** toggled adds a group to a membership or takes it out. */
export function toggled(membership: string[], id: string): string[] {
  return membership.includes(id)
    ? membership.filter((member) => member !== id)
    : [...membership, id];
}

export function NotesField({
  id,
  value,
  limit,
  busy,
  onChange,
}: {
  id: string;
  value: string;
  limit: number | undefined;
  busy: boolean;
  onChange: (value: string) => void;
}) {
  const { t } = useTranslator();
  const remaining = useRemaining()(value, limit, 500);

  return (
    <section className={fieldSection}>
      <div className="mb-1 flex items-baseline gap-2">
        <label className="text-[11px] text-muted-foreground" htmlFor={id}>
          {t("workspace.field.notes")}
        </label>
        {remaining && (
          <span className="ml-auto text-[11px] text-faint tabular-nums">
            {remaining}
          </span>
        )}
      </div>
      <Textarea
        id={id}
        value={value}
        onChange={(event) => onChange(event.target.value)}
        maxLength={limit}
        rows={3}
        className={cn(bareField, "min-h-20 resize-none text-xs leading-[1.5]")}
        disabled={busy}
      />
    </section>
  );
}

/** The header every editor carries: what is being edited, and the two ways out of it. */
export function EditorHeader({
  title,
  name,
  form,
  submit,
  canSubmit,
  busy,
  onCancel,
}: {
  title: string;
  /** The name typed so far, shown under the title once there is one. */
  name: string;
  form: string;
  submit: string;
  canSubmit: boolean;
  busy: boolean;
  onCancel: () => void;
}) {
  const { t } = useTranslator();

  return (
    <header className="flex shrink-0 items-center gap-3">
      <div className="min-w-0 flex-1">
        <h2 className="truncate text-[17px]">{title}</h2>
        {name && (
          <p className="truncate text-[11px] text-muted-foreground">{name}</p>
        )}
      </div>
      <div className="flex shrink-0 items-center gap-1.5">
        <Button
          type="button"
          variant="quiet"
          size="pill-sm"
          className="h-[29px] px-3.5"
          onClick={onCancel}
          disabled={busy}
        >
          {t("workspace.editor.cancel")}
        </Button>
        <Button
          type="submit"
          form={form}
          variant="raised"
          size="pill-sm"
          className="h-[29px] px-3.5"
          disabled={busy || !canSubmit}
        >
          {busy ? t("workspace.editor.busy") : submit}
        </Button>
      </div>
    </header>
  );
}

/**
 * EditorHint points at one field with a tip shown once, beside it where there is room. `children` is the single
 * element it points at; a row wrapped in it is no longer its block's last child, so the wrapper draws its border.
 */
export function EditorHint({
  open,
  text,
  onDismiss,
  children,
}: {
  open: boolean;
  text: string;
  onDismiss: () => void;
  children: ReactNode;
}) {
  const { t } = useTranslator();
  return (
    <Popover open={open}>
      <PopoverAnchor asChild>{children}</PopoverAnchor>
      <PopoverContent
        side="left"
        align="center"
        sideOffset={12}
        className="action-fill w-64 border-0 px-3 py-2.5 text-[12px] leading-[1.45] text-(--action-foreground) shadow-[0_12px_32px_-8px_rgb(0_0_0/0.3)] dark:shadow-[0_12px_32px_-8px_rgb(0_0_0/0.7)]"
        onOpenAutoFocus={(event) => event.preventDefault()}
        onEscapeKeyDown={onDismiss}
      >
        <PopoverPrimitive.Arrow
          width={14}
          height={7}
          className="fill-(--action-midtone)"
        />
        <p className="flex gap-2">
          <Sparkles className="mt-0.5 size-4 shrink-0" aria-hidden="true" />
          {text}
        </p>
        <div className="mt-2 flex justify-end">
          <button
            type="button"
            className="h-6 rounded-full bg-(--action-foreground) px-3 text-[11px] text-(--action-midtone) outline-none hover:bg-(--action-foreground)/85 focus-visible:ring-[3px] focus-visible:ring-(--action-foreground)/30"
            onClick={onDismiss}
          >
            {t("workspace.editor.hint.dismiss")}
          </button>
        </div>
      </PopoverContent>
    </Popover>
  );
}
