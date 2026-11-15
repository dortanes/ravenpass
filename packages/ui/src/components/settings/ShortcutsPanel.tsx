import {
  formatForDisplay,
  type HotkeyChord,
  useHotkeyRecorder,
} from "@tanstack/react-hotkeys";
import { cn } from "cn";
import { RotateCcw } from "lucide-react";
import { useEffect, useState } from "react";
import { useTranslator } from "../../i18n/translator.tsx";
import {
  definitionOf,
  type ShortcutAction,
  type ShortcutRefusal,
  type Shortcuts,
  shortcutDefinitions,
} from "../../shortcuts/shortcuts.ts";
import { Block, BlockNote, BlockRow, blockControl } from "../Block.tsx";
import { Button } from "../ui/button.tsx";

/** ShortcutsPanel lists every action a shortcut runs and records new keys for one of them. */
export function ShortcutsPanel({
  shortcuts,
  busy,
  onChange,
  onRecording,
}: {
  shortcuts: Shortcuts;
  busy: boolean;
  /** Records an action's hotkey; `null` restores its default. */
  onChange: (action: ShortcutAction, hotkey: HotkeyChord | null) => void;
  /** Shortcuts stay silent while new keys are being recorded. Must keep its identity. */
  onRecording: (recording: boolean) => void;
}) {
  const { t } = useTranslator();
  const [editing, setEditing] = useState<ShortcutAction | null>(null);
  const [refused, setRefused] = useState<{
    action: ShortcutAction;
    hotkey: HotkeyChord;
    refusal: ShortcutRefusal;
  } | null>(null);
  const recorder = useHotkeyRecorder({
    ignoreInputs: false,
    onRecord: (hotkey) => {
      if (!editing) return;
      const refusal = shortcuts.refusal(hotkey, editing);
      if (refusal) {
        setRefused({ action: editing, hotkey, refusal });
      } else if (hotkey !== shortcuts.hotkey(editing)) {
        onChange(editing, hotkey);
      }
      setEditing(null);
    },
    onClear: () => {
      if (editing && !shortcuts.isDefault(editing)) onChange(editing, null);
      setEditing(null);
    },
    onCancel: () => setEditing(null),
  });
  const { isRecording, startRecording, cancelRecording } = recorder;

  useEffect(() => {
    onRecording(isRecording);
  }, [isRecording, onRecording]);

  // The recorder stops when the panel goes; the shortcuts must not stay silenced with it.
  useEffect(() => () => onRecording(false), [onRecording]);

  function record(action: ShortcutAction) {
    setRefused(null);
    if (isRecording && editing === action) {
      cancelRecording();
      setEditing(null);
      return;
    }
    setEditing(action);
    startRecording();
  }

  function explain(refusal: ShortcutRefusal, hotkey: HotkeyChord): string {
    const keys = formatForDisplay(hotkey);
    switch (refusal.kind) {
      case "bare":
        return t("settings.shortcuts.bare");
      case "reserved":
        return t("settings.shortcuts.reserved", { keys });
      case "taken":
        return t("settings.shortcuts.taken", {
          keys,
          action: t(definitionOf(refusal.by).label),
        });
    }
  }

  return (
    <>
      <Block>
        {shortcutDefinitions.map(({ action, label }) => {
          const recording = isRecording && editing === action;
          const title = t(label);
          return (
            <BlockRow
              key={action}
              title={title}
              detail={
                refused?.action === action ? (
                  <span className="text-destructive">
                    {explain(refused.refusal, refused.hotkey)}
                  </span>
                ) : undefined
              }
            >
              {!shortcuts.isDefault(action) && (
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  className="size-7 text-muted-foreground"
                  aria-label={t("settings.shortcuts.reset", { action: title })}
                  title={t("settings.shortcuts.reset", { action: title })}
                  disabled={busy}
                  onClick={() => {
                    setRefused(null);
                    onChange(action, null);
                  }}
                >
                  <RotateCcw className="size-3.5" />
                </Button>
              )}
              <Button
                type="button"
                variant="quiet"
                aria-label={t("settings.shortcuts.change", { action: title })}
                aria-pressed={recording}
                disabled={busy}
                onClick={() => record(action)}
                className={cn(
                  blockControl,
                  "min-w-[92px] justify-center font-mono",
                  recording && "text-muted-foreground",
                )}
              >
                {recording
                  ? t("settings.shortcuts.recording")
                  : formatForDisplay(shortcuts.hotkey(action))}
              </Button>
            </BlockRow>
          );
        })}
      </Block>
      <BlockNote>{t("settings.shortcuts.note")}</BlockNote>
    </>
  );
}
