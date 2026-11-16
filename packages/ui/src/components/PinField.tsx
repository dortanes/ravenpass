import { cn } from "cn";
import { useEffect, useRef } from "react";
import { useTranslator } from "../i18n/translator.tsx";
import { AccessField } from "./AccessField.tsx";
import { PinAttemptsWarning } from "./PinAttemptsWarning.tsx";

/** PinField takes a masked PIN and drops anything but letters and digits as typed. */
export function PinField({
  id,
  value,
  onChange,
  maxLength,
  attemptsLeft,
  disabled,
  label,
  autoFocus = true,
  invalid = false,
  rejections = 0,
  className,
}: {
  id: string;
  value: string;
  onChange: (pin: string) => void;
  maxLength?: number;
  /** Left out while the attempts are not known. */
  attemptsLeft?: number;
  disabled: boolean;
  /** The label and placeholder of a PIN being chosen; left out for the vault's PIN. */
  label?: string;
  /** Unset for a field that follows another in its form. */
  autoFocus?: boolean;
  invalid?: boolean;
  /** Counts the PINs turned down; each one gives the field the focus for the next try once it is enabled. */
  rejections?: number;
  /** Sizes the field beyond the access field's own height. */
  className?: string;
}) {
  const { t } = useTranslator();
  const field = useRef<HTMLInputElement>(null);
  const answered = useRef(rejections);
  useEffect(() => {
    if (disabled || answered.current === rejections) return;
    answered.current = rejections;
    field.current?.focus();
  }, [disabled, rejections]);

  return (
    <>
      <label className="sr-only" htmlFor={id}>
        {label ?? t("unlock.pin.label")}
      </label>
      <AccessField
        ref={field}
        id={id}
        className={cn("font-mono text-[19px] tracking-[0.4em]", className)}
        type="password"
        autoComplete="off"
        autoCorrect="off"
        autoCapitalize="none"
        spellCheck={false}
        autoFocus={autoFocus}
        value={value}
        placeholder={label ?? t("unlock.pin.placeholder")}
        maxLength={maxLength}
        disabled={disabled}
        aria-invalid={invalid || undefined}
        onChange={(event) =>
          onChange(event.target.value.replace(/[^0-9a-zA-Z]/g, ""))
        }
      />
      {attemptsLeft !== undefined && (
        <PinAttemptsWarning attemptsLeft={attemptsLeft} />
      )}
    </>
  );
}
