import { Eye, EyeOff, LoaderCircle } from "lucide-react";
import { type FormEvent, useId, useRef, useState } from "react";
import { useTranslator } from "../i18n/translator.tsx";
import type { UnlockMethods } from "../vault-api.ts";
import { Block, BlockRow } from "./Block.tsx";
import { checkNewPin } from "./new-pin.ts";
import { PhraseFields, type PhraseFieldsHandle } from "./PhraseFields.tsx";
import { PinField } from "./PinField.tsx";
import { PhraseEntry } from "./phrase-entry.ts";
import {
  ResponsiveDialog,
  ResponsiveDialogContent,
  ResponsiveDialogDescription,
  ResponsiveDialogFooter,
  ResponsiveDialogHeader,
  ResponsiveDialogTitle,
} from "./ResponsiveDialog.tsx";
import { recoveryKeyWords } from "./recovery-challenge.ts";
import { Button } from "./ui/button.tsx";
import { Switch } from "./ui/switch.tsx";
import { heldWays, ownerCheck, type UnlockPending } from "./unlock-change.ts";

const noWords = PhraseEntry.empty(recoveryKeyWords);

/** A change to the ways in that runs once it has the current PIN or recovery key, empty where none is asked; true once saved. */
type HeldChange = (current: string) => Promise<boolean>;

/** UnlockOptions offers the ways a vault may be opened on the device, in setup and settings alike. */
export function UnlockOptions({
  methods,
  busy,
  pending = null,
  keepOne = true,
  descriptions = true,
  confirmChanges = false,
  onSetPin,
  onRemovePin,
  onBiometry,
}: {
  methods: UnlockMethods | null;
  busy: boolean;
  /** The change the device is applying; every control waits for it. */
  pending?: UnlockPending | null;
  /** Whether the last way in must stay on, as it must for a vault that exists. */
  keepOne?: boolean;
  descriptions?: boolean;
  /**
   * Set where each change applies to the device at once, so the current PIN confirms it where the device cannot, and
   * the recovery key where neither can.
   */
  confirmChanges?: boolean;
  onSetPin: (pin: string, current: string) => Promise<boolean>;
  onRemovePin: (current: string) => Promise<boolean>;
  onBiometry: (enabled: boolean, current: string) => Promise<boolean>;
}) {
  const { t } = useTranslator();
  const [editing, setEditing] = useState(false);
  const [held, setHeld] = useState<HeldChange | null>(null);

  const busyOrUnknown = busy || pending !== null || !methods;
  const enablingBiometry = pending === "biometry-on";
  // Null where changes apply without confirming the owner, as in setup.
  const asked = confirmChanges && methods !== null ? ownerCheck(methods) : null;
  const askPin = asked === "pin";
  const askKey = asked === "recovery-key";
  // Device authentication that comes back leaves nothing to ask for.
  if (held !== null && !askPin && !askKey) setHeld(null);
  // The last way in cannot be given up, so its switch is held where nothing would replace it.
  const kept = keepOne ? heldWays(methods) : { biometry: false, pin: false };
  const biometryIsLast = kept.biometry;
  const pinIsLast = kept.pin;

  function apply(change: HeldChange) {
    if (askPin || askKey) {
      setHeld(() => change);
      return;
    }
    void change("");
  }

  // The recovery key is asked after the new PIN, in a dialog of its own.
  function savePin(pin: string, current: string): Promise<boolean> {
    if (!askKey) return onSetPin(pin, current);
    apply((key) => onSetPin(pin, key));
    return Promise.resolve(true);
  }

  return (
    <>
      <Block>
        <BlockRow
          title={t("unlock-methods.biometry.title")}
          detail={
            enablingBiometry
              ? t("unlock-methods.biometry.creating")
              : methods && !methods.biometryAvailable
                ? t("unlock-methods.biometry.unavailable")
                : !descriptions && biometryIsLast
                  ? t("unlock-methods.biometry.last")
                  : descriptions
                    ? t("unlock-methods.biometry.description")
                    : undefined
          }
          htmlFor="unlock-biometry"
        >
          {enablingBiometry && (
            <LoaderCircle
              className="size-4 animate-spin text-muted-foreground motion-reduce:animate-none"
              aria-hidden="true"
            />
          )}
          <Switch
            id="unlock-biometry"
            checked={Boolean(methods?.biometryEnabled) || enablingBiometry}
            aria-busy={enablingBiometry || undefined}
            disabled={
              busyOrUnknown ||
              (!methods?.biometryAvailable && !methods?.biometryEnabled) ||
              biometryIsLast
            }
            onCheckedChange={(enabled) =>
              apply((current) => onBiometry(enabled, current))
            }
          />
        </BlockRow>
        <BlockRow
          title={t("unlock-methods.pin.title")}
          detail={
            descriptions
              ? t("unlock-methods.pin.description", {
                  min: methods?.pinMinLength ?? 6,
                  max: methods?.pinMaxLength ?? 12,
                })
              : pinIsLast
                ? t(
                    methods?.biometryAvailable
                      ? "unlock-methods.pin.last"
                      : "unlock-methods.pin.only",
                  )
                : undefined
          }
          htmlFor="unlock-pin"
        >
          {methods?.pinSet && (
            <Button
              type="button"
              variant="quiet"
              size="pill-sm"
              disabled={busyOrUnknown}
              onClick={() => setEditing(true)}
            >
              {t("unlock-methods.pin.change")}
            </Button>
          )}
          <Switch
            id="unlock-pin"
            checked={Boolean(methods?.pinSet)}
            disabled={busyOrUnknown || pinIsLast}
            onCheckedChange={(enabled) => {
              if (enabled) {
                setEditing(true);
                return;
              }
              apply((current) => onRemovePin(current));
            }}
          />
        </BlockRow>
      </Block>

      <PinDialog
        open={editing}
        changing={Boolean(methods?.pinSet)}
        askCurrent={askPin}
        attemptsLeft={methods?.pinAttemptsLeft}
        minimum={methods?.pinMinLength ?? 6}
        maximum={methods?.pinMaxLength ?? 12}
        onSave={savePin}
        onClose={() => setEditing(false)}
      />
      <CurrentPinDialog
        open={held !== null && askPin}
        note={
          enablingBiometry ? t("unlock-methods.biometry.creating") : undefined
        }
        attemptsLeft={methods?.pinAttemptsLeft}
        minimum={methods?.pinMinLength ?? 6}
        maximum={methods?.pinMaxLength ?? 12}
        onConfirm={(current) => held?.(current) ?? Promise.resolve(false)}
        onClose={() => setHeld(null)}
      />
      <CurrentKeyDialog
        open={held !== null && askKey}
        onConfirm={(current) => held?.(current) ?? Promise.resolve(false)}
        onClose={() => setHeld(null)}
      />
    </>
  );
}

/** PinDialog takes a new PIN twice, after the current one where the device cannot confirm the change. */
function PinDialog({
  open,
  changing,
  askCurrent,
  attemptsLeft,
  minimum,
  maximum,
  onSave,
  onClose,
}: {
  open: boolean;
  changing: boolean;
  askCurrent: boolean;
  attemptsLeft?: number;
  minimum: number;
  maximum: number;
  onSave: (pin: string, current: string) => Promise<boolean>;
  onClose: () => void;
}) {
  const { t } = useTranslator();
  const id = useId();
  const [current, setCurrent] = useState("");
  const [pin, setPin] = useState("");
  const [repeat, setRepeat] = useState("");
  const [saving, setSaving] = useState(false);
  const [rejections, setRejections] = useState(0);

  const check = checkNewPin(pin, repeat, minimum);
  const mismatched = check === "mismatched";
  const ready =
    check === "matched" && (!askCurrent || current.length >= minimum);

  function close() {
    setCurrent("");
    setPin("");
    setRepeat("");
    onClose();
  }

  async function save(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!ready || saving) return;
    setSaving(true);
    try {
      if (await onSave(pin, askCurrent ? current : "")) {
        close();
      } else {
        setCurrent("");
        setRejections((count) => count + 1);
      }
    } finally {
      setSaving(false);
    }
  }

  return (
    <ResponsiveDialog
      open={open}
      dismissible={!saving}
      onOpenChange={(next) => {
        if (!next) close();
      }}
    >
      <ResponsiveDialogContent className="bg-background sm:max-w-[360px]">
        <ResponsiveDialogHeader>
          <ResponsiveDialogTitle>
            {t(
              changing ? "unlock-methods.pin.change" : "unlock-methods.pin.set",
            )}
          </ResponsiveDialogTitle>
          <ResponsiveDialogDescription>
            {t("unlock-methods.pin.weaker")}
          </ResponsiveDialogDescription>
        </ResponsiveDialogHeader>
        <form id={`${id}-form`} className="grid gap-2" onSubmit={save}>
          {askCurrent && (
            <PinField
              id={`${id}-current`}
              label={t("unlock-methods.pin.current")}
              value={current}
              onChange={setCurrent}
              maxLength={maximum}
              attemptsLeft={attemptsLeft}
              disabled={saving}
              rejections={rejections}
            />
          )}
          <PinField
            id={`${id}-new`}
            label={t("unlock-methods.pin.new")}
            autoFocus={!askCurrent}
            value={pin}
            onChange={setPin}
            maxLength={maximum}
            disabled={saving}
          />
          <PinField
            id={`${id}-repeat`}
            label={t("unlock-methods.pin.repeat")}
            autoFocus={false}
            invalid={mismatched}
            value={repeat}
            onChange={setRepeat}
            maxLength={maximum}
            disabled={saving}
          />
          {mismatched ? (
            <p
              role="alert"
              className="text-center text-[11px] text-destructive"
            >
              {t("unlock-methods.pin.mismatch")}
            </p>
          ) : (
            <p className="text-center text-[11px] text-muted-foreground">
              {t("unlock-methods.pin.description", {
                min: minimum,
                max: maximum,
              })}
            </p>
          )}
        </form>
        <ResponsiveDialogFooter>
          <Button
            type="button"
            variant="quiet"
            size="pill"
            disabled={saving}
            onClick={close}
          >
            {t("unlock-methods.pin.cancel")}
          </Button>
          <Button
            type="submit"
            form={`${id}-form`}
            variant="raised"
            size="pill"
            disabled={!ready || saving}
          >
            {t(
              saving ? "unlock-methods.pin.saving" : "unlock-methods.pin.save",
            )}
          </Button>
        </ResponsiveDialogFooter>
      </ResponsiveDialogContent>
    </ResponsiveDialog>
  );
}

/** CurrentKeyDialog takes the vault's recovery key to confirm a change where neither a PIN nor the device can. */
function CurrentKeyDialog({
  open,
  onConfirm,
  onClose,
}: {
  open: boolean;
  onConfirm: (current: string) => Promise<boolean>;
  onClose: () => void;
}) {
  const { t } = useTranslator();
  const id = useId();
  const fields = useRef<PhraseFieldsHandle>(null);
  const [entry, setEntry] = useState(noWords);
  const [reveal, setReveal] = useState(false);
  const [saving, setSaving] = useState(false);

  function close() {
    setEntry(noWords);
    setReveal(false);
    onClose();
  }

  async function confirm(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!entry.complete) {
      fields.current?.focus(entry.firstEmpty);
      return;
    }
    if (saving) return;
    setSaving(true);
    try {
      if (await onConfirm(entry.phrase)) close();
    } finally {
      setSaving(false);
    }
  }

  return (
    <ResponsiveDialog
      open={open}
      dismissible={!saving}
      onOpenChange={(next) => {
        if (!next) close();
      }}
    >
      <ResponsiveDialogContent className="bg-background sm:max-w-[480px]">
        <ResponsiveDialogHeader>
          <ResponsiveDialogTitle>
            {t("unlock-methods.confirm-key.title")}
          </ResponsiveDialogTitle>
          <ResponsiveDialogDescription>
            {t("unlock-methods.confirm-key.description")}
          </ResponsiveDialogDescription>
        </ResponsiveDialogHeader>
        <form id={`${id}-form`} className="grid gap-2" onSubmit={confirm}>
          <div className="flex justify-end">
            <Button
              className="text-muted-foreground"
              type="button"
              variant="ghost"
              size="icon-sm"
              aria-label={t(
                reveal ? "recovery.phrase.hide" : "recovery.phrase.show",
              )}
              onClick={() => setReveal((shown) => !shown)}
              disabled={saving}
            >
              {reveal ? <EyeOff /> : <Eye />}
            </Button>
          </div>
          <PhraseFields
            ref={fields}
            label={t("recovery.phrase.label")}
            entry={entry}
            onEntry={setEntry}
            concealed={!reveal}
            disabled={saving}
          />
        </form>
        <ResponsiveDialogFooter>
          <Button
            type="button"
            variant="quiet"
            size="pill"
            disabled={saving}
            onClick={close}
          >
            {t("unlock-methods.pin.cancel")}
          </Button>
          <Button
            type="submit"
            form={`${id}-form`}
            variant="raised"
            size="pill"
            disabled={saving}
          >
            {t(
              saving
                ? "unlock-methods.pin.saving"
                : "unlock-methods.confirm.action",
            )}
          </Button>
        </ResponsiveDialogFooter>
      </ResponsiveDialogContent>
    </ResponsiveDialog>
  );
}

/** CurrentPinDialog takes the vault's PIN to confirm a change where the device cannot verify the owner. */
export function CurrentPinDialog({
  open,
  description,
  note,
  attemptsLeft,
  minimum,
  maximum,
  onConfirm,
  onClose,
}: {
  open: boolean;
  /** What the PIN confirms, when it is not a change to how the vault unlocks. */
  description?: string;
  /** What the change is doing while it saves. */
  note?: string;
  attemptsLeft?: number;
  minimum: number;
  maximum: number;
  onConfirm: (current: string) => Promise<boolean>;
  onClose: () => void;
}) {
  const { t } = useTranslator();
  const id = useId();
  const [current, setCurrent] = useState("");
  const [saving, setSaving] = useState(false);
  const [rejections, setRejections] = useState(0);
  const ready = current.length >= minimum;

  function close() {
    setCurrent("");
    onClose();
  }

  async function confirm(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!ready || saving) return;
    setSaving(true);
    try {
      if (await onConfirm(current)) {
        close();
      } else {
        setCurrent("");
        setRejections((count) => count + 1);
      }
    } finally {
      setSaving(false);
    }
  }

  return (
    <ResponsiveDialog
      open={open}
      dismissible={!saving}
      onOpenChange={(next) => {
        if (!next) close();
      }}
    >
      <ResponsiveDialogContent className="bg-background sm:max-w-[360px]">
        <ResponsiveDialogHeader>
          <ResponsiveDialogTitle>
            {t("unlock-methods.confirm.title")}
          </ResponsiveDialogTitle>
          <ResponsiveDialogDescription>
            {description ?? t("unlock-methods.confirm.description")}
          </ResponsiveDialogDescription>
        </ResponsiveDialogHeader>
        <form id={`${id}-form`} className="grid gap-2" onSubmit={confirm}>
          <PinField
            id={`${id}-current`}
            value={current}
            onChange={setCurrent}
            maxLength={maximum}
            attemptsLeft={attemptsLeft}
            disabled={saving}
            rejections={rejections}
          />
          {saving && note && (
            <p
              role="status"
              className="text-center text-[11px] text-muted-foreground"
            >
              {note}
            </p>
          )}
        </form>
        <ResponsiveDialogFooter>
          <Button
            type="button"
            variant="quiet"
            size="pill"
            disabled={saving}
            onClick={close}
          >
            {t("unlock-methods.pin.cancel")}
          </Button>
          <Button
            type="submit"
            form={`${id}-form`}
            variant="raised"
            size="pill"
            disabled={!ready || saving}
          >
            {t(
              saving
                ? "unlock-methods.pin.saving"
                : "unlock-methods.confirm.action",
            )}
          </Button>
        </ResponsiveDialogFooter>
      </ResponsiveDialogContent>
    </ResponsiveDialog>
  );
}
