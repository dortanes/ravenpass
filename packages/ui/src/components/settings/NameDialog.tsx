import { useId, useState } from "react";
import {
  ResponsiveDialog,
  ResponsiveDialogContent,
  ResponsiveDialogFooter,
  ResponsiveDialogHeader,
  ResponsiveDialogTitle,
} from "../ResponsiveDialog.tsx";
import { Button } from "../ui/button.tsx";
import { Input } from "../ui/input.tsx";

/** NameDialog asks for a name, trimmed and never empty; `onSave` resolves true once it is kept. */
export function NameDialog({
  open,
  title,
  label,
  placeholder,
  maxLength,
  save,
  cancel,
  initial = "",
  busy,
  onSave,
  onClose,
  onDelete,
}: {
  open: boolean;
  title: string;
  label: string;
  placeholder?: string;
  maxLength: number;
  save: string;
  cancel: string;
  initial?: string;
  busy: boolean;
  onSave: (name: string) => Promise<boolean>;
  onClose: () => void;
  /** Shown beside the other actions with its label. */
  onDelete?: { label: string; run: () => void };
}) {
  const id = useId();
  const [name, setName] = useState(initial);

  async function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!name.trim()) return;
    if (await onSave(name.trim())) {
      setName(initial);
      onClose();
    }
  }

  function close() {
    setName(initial);
    onClose();
  }

  return (
    <ResponsiveDialog
      open={open}
      dismissible={!busy}
      onOpenChange={(next) => {
        if (!next) close();
      }}
    >
      <ResponsiveDialogContent
        className={onDelete ? "sm:max-w-[440px]" : "sm:max-w-[360px]"}
      >
        <form onSubmit={submit} className="grid min-w-0 gap-3">
          <ResponsiveDialogHeader>
            <ResponsiveDialogTitle>{title}</ResponsiveDialogTitle>
          </ResponsiveDialogHeader>
          <label className="sr-only" htmlFor={id}>
            {label}
          </label>
          <Input
            id={id}
            value={name}
            placeholder={placeholder}
            maxLength={maxLength}
            autoFocus
            disabled={busy}
            onChange={(event) => setName(event.target.value)}
          />
          <ResponsiveDialogFooter className="sm:flex-wrap">
            {onDelete && (
              <Button
                type="button"
                variant="quiet"
                size="pill"
                className="text-destructive sm:mr-auto"
                disabled={busy}
                onClick={onDelete.run}
              >
                {onDelete.label}
              </Button>
            )}
            <Button
              type="button"
              variant="quiet"
              size="pill"
              disabled={busy}
              onClick={close}
            >
              {cancel}
            </Button>
            <Button
              type="submit"
              variant="raised"
              size="pill"
              disabled={busy || !name.trim()}
            >
              {save}
            </Button>
          </ResponsiveDialogFooter>
        </form>
      </ResponsiveDialogContent>
    </ResponsiveDialog>
  );
}
