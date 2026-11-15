import { match } from "@formatjs/intl-localematcher";

/** English is the source catalog and the fallback. */
export const languages = ["en", "ru", "zh"] as const;

export type Language = (typeof languages)[number];

const fallback: Language = "en";

export function isLanguage(value: string): value is Language {
  return (languages as readonly string[]).includes(value);
}

/** Each name is written in the language it names. */
export const languageNames: Record<Language, string> = {
  en: "English",
  ru: "Русский",
  zh: "简体中文",
};

function wellFormed(tag: string): boolean {
  try {
    Intl.getCanonicalLocales(tag);
    return true;
  } catch {
    return false;
  }
}

/** matchLanguage picks the offered language closest to a BCP 47 tag, English for any other. */
export function matchLanguage(tag: string): Language {
  const matched = match(wellFormed(tag) ? [tag] : [], languages, fallback);
  return isLanguage(matched) ? matched : fallback;
}

/** formatDelay names whole seconds in the largest of hours, minutes or seconds that divides them. */
export function formatDelay(
  seconds: number,
  language: Language,
  display: "long" | "short",
): string {
  const [unit, size] =
    seconds >= 3600 && seconds % 3600 === 0
      ? (["hour", 3600] as const)
      : seconds >= 60 && seconds % 60 === 0
        ? (["minute", 60] as const)
        : (["second", 1] as const);
  return new Intl.NumberFormat(language, {
    style: "unit",
    unit,
    unitDisplay: display,
  }).format(seconds / size);
}

export function formatPercent(percent: number, language: Language): string {
  return new Intl.NumberFormat(language, { style: "percent" }).format(
    percent / 100,
  );
}
