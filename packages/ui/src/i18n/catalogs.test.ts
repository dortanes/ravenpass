import assert from "node:assert/strict";
import test from "node:test";
import {
  isLiteralElement,
  isPluralElement,
  isPoundElement,
  isSelectElement,
  isTagElement,
  type MessageFormatElement,
} from "@formatjs/icu-messageformat-parser";
import { IntlMessageFormat } from "intl-messageformat";
import { MessageFormatter, parseMessage } from "./format.ts";
import {
  formatDelay,
  formatPercent,
  isLanguage,
  type Language,
  languageNames,
  languages,
  matchLanguage,
} from "./language.ts";
import { catalogs, type MessageKey } from "./messages.ts";

const keys = Object.keys(catalogs.en) as MessageKey[];

const pluralCategories: Record<Language, readonly string[]> = {
  en: ["one", "other"],
  ru: ["few", "many", "one", "other"],
};

function argumentNames(elements: MessageFormatElement[]): string[] {
  return elements.flatMap((element) => {
    if (isTagElement(element)) return argumentNames(element.children);
    if (isPluralElement(element) || isSelectElement(element)) {
      return [
        element.value,
        ...Object.values(element.options).flatMap((option) =>
          argumentNames(option.value),
        ),
      ];
    }
    if (isPoundElement(element) || isLiteralElement(element)) return [];
    return [element.value];
  });
}

function pluralOptions(elements: MessageFormatElement[]): string[][] {
  return elements.flatMap((element) => {
    if (isTagElement(element)) return pluralOptions(element.children);
    if (!isPluralElement(element) && !isSelectElement(element)) return [];
    const nested = Object.values(element.options).flatMap((option) =>
      pluralOptions(option.value),
    );
    return isPluralElement(element)
      ? [Object.keys(element.options).sort(), ...nested]
      : nested;
  });
}

test("every language carries the same keys", () => {
  const source = [...keys].sort();
  for (const language of languages) {
    assert.deepEqual(
      Object.keys(catalogs[language]).sort(),
      source,
      `${language} does not carry the same keys as English`,
    );
  }
});

test("no message is empty and none is left untranslated by accident", () => {
  for (const language of languages) {
    for (const key of keys) {
      assert.ok(catalogs[language][key].trim(), `${language}:${key} is empty`);
    }
  }
  const keptAsWritten = new Set<MessageKey>([
    "card.field.security-code",
    "app.name",
    "settings.general.macos",
    "settings.import.note.swift",
    "settings.import.note.iban",
    "settings.trash.detail",
    "system.file.vault",
  ]);
  const identical = keys.filter(
    (key) => !keptAsWritten.has(key) && catalogs.en[key] === catalogs.ru[key],
  );
  assert.deepEqual(identical, [], "these Russian messages are still English");
});

test("every message parses as ICU MessageFormat in its language", () => {
  for (const language of languages) {
    for (const key of keys) {
      assert.doesNotThrow(
        () =>
          new IntlMessageFormat(
            parseMessage(catalogs[language][key]),
            language,
          ),
        `${language}:${key} does not parse`,
      );
    }
  }
});

test("every message takes the same arguments in every language", () => {
  const names = (language: Language, key: MessageKey) =>
    [...new Set(argumentNames(parseMessage(catalogs[language][key])))].sort();
  for (const key of keys) {
    for (const language of languages) {
      assert.deepEqual(
        names(language, key),
        names("en", key),
        `${language}:${key} does not take the same arguments as English`,
      );
    }
  }
});

test("every plural names exactly the categories its language uses", () => {
  for (const language of languages) {
    for (const key of keys) {
      for (const options of pluralOptions(
        parseMessage(catalogs[language][key]),
      )) {
        assert.deepEqual(
          options,
          pluralCategories[language],
          `${language}:${key} names the wrong plural categories`,
        );
      }
    }
  }
});

test("a count reads in the plural form its language needs", () => {
  const formatter = new MessageFormatter();
  const words = (language: Language, count: number) =>
    formatter.format(language, "note.words", { count });
  assert.equal(words("en", 1), "1 word");
  assert.equal(words("en", 2), "2 words");
  assert.equal(words("en", 5), "5 words");
  assert.equal(words("en", 21), "21 words");
  assert.equal(words("ru", 1), "1 слово");
  assert.equal(words("ru", 2), "2 слова");
  assert.equal(words("ru", 5), "5 слов");
  assert.equal(words("ru", 21), "21 слово");
});

test("a named value fills its placeholder", () => {
  assert.equal(
    new MessageFormatter().format("en", "system.reason.save-passkey", {
      site: "example.com",
    }),
    "save a passkey for example.com",
  );
});

test("a message missing one of its values fails to format", () => {
  const formatter = new MessageFormatter();
  assert.throws(() => formatter.format("en", "note.words"));
  assert.throws(() => formatter.format("ru", "system.reason.save-passkey", {}));
});

test("a delay reads in the largest unit that divides it evenly", () => {
  assert.equal(formatDelay(15, "en", "long"), "15 seconds");
  assert.equal(formatDelay(60, "en", "long"), "1 minute");
  assert.equal(formatDelay(90, "en", "long"), "90 seconds");
  assert.equal(formatDelay(300, "en", "short"), "5 min");
  assert.equal(formatDelay(1800, "en", "long"), "30 minutes");
  assert.equal(formatDelay(3600, "en", "long"), "1 hour");
  assert.equal(formatDelay(5400, "en", "long"), "90 minutes");
  assert.equal(formatDelay(120, "ru", "long"), "2 минуты");
  assert.equal(formatDelay(3600, "ru", "long"), "1 час");
  assert.equal(formatDelay(30, "ru", "short"), "30 с");
});

test("a percentage reads as the language writes it", () => {
  assert.equal(formatPercent(115, "en"), "115%");
  assert.equal(formatPercent(85, "ru"), "85 %");
});

test("a system language tag maps to an offered language", () => {
  assert.equal(matchLanguage("ru-RU"), "ru");
  assert.equal(matchLanguage("RU"), "ru");
  assert.equal(matchLanguage("en-GB"), "en");
  assert.equal(matchLanguage("de-DE"), "en");
  assert.equal(matchLanguage(""), "en");
  assert.equal(matchLanguage("x_y!"), "en");
  assert.ok(isLanguage("ru"));
  assert.ok(!isLanguage("de"));
});

test("every offered language names itself", () => {
  for (const language of languages) {
    assert.ok(languageNames[language]?.trim(), `${language} has no name`);
  }
});
