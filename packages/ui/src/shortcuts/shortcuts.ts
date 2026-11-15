import {
  type HotkeyChord,
  normalizeHotkey,
  parseHotkey,
  validateHotkey,
} from "@tanstack/react-hotkeys";
import type { MessageKey } from "../i18n/messages.ts";

export type ShortcutAction = "palette" | "search";

export interface ShortcutDefinition {
  action: ShortcutAction;
  label: MessageKey;
  hotkey: HotkeyChord;
}

/** In the order settings list them, with default hotkeys. */
export const shortcutDefinitions: readonly ShortcutDefinition[] = [
  {
    action: "palette",
    label: "settings.shortcuts.palette",
    hotkey: "Mod+K",
  },
  {
    action: "search",
    label: "settings.shortcuts.search",
    hotkey: "Mod+F",
  },
];

/** Taken by text editing and the macOS app menu. */
const reservedHotkeys: readonly HotkeyChord[] = [
  "Mod+A",
  "Mod+C",
  "Mod+V",
  "Mod+X",
  "Mod+Z",
  "Mod+Shift+Z",
  "Mod+H",
  "Mod+M",
  "Mod+Q",
  "Mod+W",
  "Mod+,",
];

/** `bare` has no modifier, so typing would trigger it. */
export type ShortcutRefusal =
  | { kind: "bare" }
  | { kind: "reserved" }
  | { kind: "taken"; by: ShortcutAction };

/** The user's choices over the defaults. */
export class Shortcuts {
  static readonly defaults = new Shortcuts(new Map());

  private readonly chosen: ReadonlyMap<ShortcutAction, HotkeyChord>;

  private constructor(chosen: ReadonlyMap<ShortcutAction, HotkeyChord>) {
    this.chosen = chosen;
  }

  /** Keeps only known actions with valid hotkeys. */
  static fromRecorded(recorded: Readonly<Record<string, string>>): Shortcuts {
    const chosen = new Map<ShortcutAction, HotkeyChord>();
    for (const { action } of shortcutDefinitions) {
      const hotkey = recorded[action];
      if (hotkey && validateHotkey(hotkey).valid) {
        chosen.set(action, hotkey as HotkeyChord);
      }
    }
    return new Shortcuts(chosen);
  }

  hotkey(action: ShortcutAction): HotkeyChord {
    return this.chosen.get(action) ?? definitionOf(action).hotkey;
  }

  isDefault(action: ShortcutAction): boolean {
    return !this.chosen.has(action);
  }

  /** Null when `hotkey` can run `action`. */
  refusal(hotkey: HotkeyChord, action: ShortcutAction): ShortcutRefusal | null {
    const parsed = parseHotkey(hotkey);
    if (!parsed.ctrl && !parsed.alt && !parsed.meta) return { kind: "bare" };
    const wanted = normalizeHotkey(hotkey);
    if (
      reservedHotkeys.some((reserved) => normalizeHotkey(reserved) === wanted)
    )
      return { kind: "reserved" };
    const holder = shortcutDefinitions.find(
      (definition) =>
        definition.action !== action &&
        normalizeHotkey(this.hotkey(definition.action)) === wanted,
    );
    return holder ? { kind: "taken", by: holder.action } : null;
  }

  /** `null` restores the default. */
  with(action: ShortcutAction, hotkey: HotkeyChord | null): Shortcuts {
    const chosen = new Map(this.chosen);
    if (hotkey === null) chosen.delete(action);
    else chosen.set(action, hotkey);
    return new Shortcuts(chosen);
  }
}

export function definitionOf(action: ShortcutAction): ShortcutDefinition {
  const definition = shortcutDefinitions.find((item) => item.action === action);
  if (!definition) throw new Error(`unknown shortcut action ${action}`);
  return definition;
}
