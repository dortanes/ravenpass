import { cn } from "cn";
import { Copy, Eye, EyeOff } from "lucide-react";
import { type ComponentType, type ReactNode, useId } from "react";
import { useTranslator } from "../../i18n/translator.tsx";
import { Button } from "../ui/button.tsx";

/** What a concealed secret shows in place of its value. */
export const secretMask = "••••••••••";

export function FieldBlock({ children }: { children: ReactNode }) {
  return <div className="overflow-hidden rounded-row bg-field">{children}</div>;
}

export function TitledBlock({
  title,
  children,
}: {
  title: string;
  children: ReactNode;
}) {
  const heading = useId();

  return (
    <section className="shrink-0" aria-labelledby={heading}>
      <h3
        id={heading}
        className="mb-1.5 px-[3px] text-[11px] font-normal text-muted-foreground"
      >
        {title}
      </h3>
      <div className="flex flex-col gap-2">{children}</div>
    </section>
  );
}

export const fieldRow =
  "flex min-h-[41px] w-full items-center gap-2.5 px-[13px] py-1.5 text-left";

export const dividedRow = "border-b last:border-b-0";

export const actionRow =
  "outline-none hover:bg-field-hover focus-visible:bg-field-hover disabled:pointer-events-none disabled:opacity-50";

/**
 * FieldRow is one labelled line; with `onAction` the whole line is the button, named by its label and value and
 * described by its `action` title, and `accessory` sits outside it.
 */
export function FieldRow({
  label,
  action,
  icon: Icon,
  disabled = false,
  onAction,
  accessory,
  children,
}: {
  label: string;
  action?: string;
  icon?: ComponentType<{ className?: string }>;
  disabled?: boolean;
  onAction?: () => void;
  accessory?: ReactNode;
  children: ReactNode;
}) {
  const content = (
    <>
      <span className="w-[72px] shrink-0 text-[11px] text-muted-foreground">
        {label}
      </span>
      <span className="flex min-w-0 flex-1 items-center gap-1">{children}</span>
      {Icon && <Icon className="size-4 shrink-0 text-muted-foreground" />}
    </>
  );

  if (!onAction) {
    return <div className={`${fieldRow} ${dividedRow}`}>{content}</div>;
  }

  const button = (
    <button
      type="button"
      title={action}
      disabled={disabled}
      onClick={onAction}
      className={cn(
        fieldRow,
        actionRow,
        accessory ? "min-w-0 flex-1" : dividedRow,
      )}
    >
      {content}
    </button>
  );

  if (!accessory) return button;

  return (
    <div className={`flex items-center gap-1 pr-[7px] ${dividedRow}`}>
      {button}
      {accessory}
    </div>
  );
}

/** The look of a small action button at the end of a row. */
export const rowButtonClass =
  "size-7 shrink-0 rounded-md text-muted-foreground hover:text-foreground";

/** RowButton is an icon button beside a row's value, such as a row's `accessory`. */
function RowButton({
  label,
  icon: Icon,
  busy,
  onClick,
}: {
  label: string;
  icon: ComponentType<{ className?: string }>;
  busy: boolean;
  onClick: () => void;
}) {
  return (
    <Button
      type="button"
      variant="ghost"
      size="icon-sm"
      className={rowButtonClass}
      onClick={onClick}
      aria-label={label}
      title={label}
      disabled={busy}
    >
      <Icon className="size-4" />
    </Button>
  );
}

export function RevealButton({
  shown,
  revealLabel,
  concealLabel,
  busy = false,
  onToggle,
}: {
  shown: boolean;
  revealLabel: string;
  concealLabel: string;
  busy?: boolean;
  onToggle: () => void;
}) {
  return (
    <RowButton
      label={shown ? concealLabel : revealLabel}
      icon={shown ? EyeOff : Eye}
      busy={busy}
      onClick={onToggle}
    />
  );
}

/** CopyButton copies a value whose row does something else when selected. */
export function CopyButton({
  label,
  busy,
  onCopy,
}: {
  label: string;
  busy: boolean;
  onCopy: () => void;
}) {
  return <RowButton label={label} icon={Copy} busy={busy} onClick={onCopy} />;
}

/** SecretRow copies on click; the caller owns `revealed`, so a reveal ends with the caller. */
export function SecretRow({
  label,
  value,
  concealed = secretMask,
  revealed,
  wrap = false,
  revealLabel,
  concealLabel,
  copyLabel,
  busy,
  onReveal,
  onCopy,
}: {
  label: string;
  value: string;
  /** What shows while the value is concealed. */
  concealed?: string;
  revealed: boolean;
  /** Whether a revealed value wraps; otherwise it is truncated. */
  wrap?: boolean;
  revealLabel: string;
  concealLabel: string;
  copyLabel: string;
  busy: boolean;
  onReveal: () => void;
  onCopy: () => void;
}) {
  const { t } = useTranslator();
  // A screen reader hears "hidden" and whatever the mask leaves shown, never a run of dots.
  const spoken = [
    t("workspace.field.hidden"),
    concealed.replaceAll("•", "").trim(),
  ]
    .filter(Boolean)
    .join(" ");

  return (
    <FieldRow
      label={label}
      action={copyLabel}
      icon={Copy}
      disabled={busy}
      onAction={onCopy}
      accessory={
        <RevealButton
          shown={revealed}
          revealLabel={revealLabel}
          concealLabel={concealLabel}
          onToggle={onReveal}
        />
      }
    >
      <span
        aria-hidden={!revealed || undefined}
        className={cn(
          "min-w-0 flex-1 font-mono text-[13px]",
          revealed && wrap ? "break-all" : "truncate",
          revealed ? "select-text" : "tracking-[0.16em]",
        )}
      >
        {revealed ? value : concealed}
      </span>
      {!revealed && <span className="sr-only">{spoken}</span>}
    </FieldRow>
  );
}

export function NotesBlock({
  label,
  notes,
  copyLabel,
  busy,
  onCopy,
}: {
  label: string;
  notes: string;
  copyLabel: string;
  busy: boolean;
  onCopy: () => void;
}) {
  const heading = useId();

  return (
    <section
      className="shrink-0 rounded-row bg-field"
      aria-labelledby={heading}
    >
      <div className="flex min-h-[35px] items-center gap-2 pt-1 pr-[7px] pl-[13px]">
        <h3
          id={heading}
          className="min-w-0 flex-1 text-[11px] font-normal text-muted-foreground"
        >
          {label}
        </h3>
        <CopyButton label={copyLabel} busy={busy} onCopy={onCopy} />
      </div>
      <p className="px-[13px] pb-[11px] text-xs leading-[1.5] break-words whitespace-pre-wrap select-text">
        {notes}
      </p>
    </section>
  );
}
