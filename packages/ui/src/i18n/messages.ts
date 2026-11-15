import { en } from "./en/index.ts";
import type { Language } from "./language.ts";
import { ru } from "./ru/index.ts";
import { zh } from "./zh/index.ts";

/** English is the source catalog: it defines the keys every language must carry. */
export type Messages = typeof en;

export type MessageKey = keyof Messages;

export const catalogs: Record<Language, Messages> = { en, ru, zh };
