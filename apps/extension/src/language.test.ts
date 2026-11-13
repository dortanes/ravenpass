import assert from "node:assert/strict";
import test from "node:test";
import { StoredLanguage } from "./language.ts";
import { MemoryLocalArea } from "./test-doubles.test-support.ts";

function storedLanguage() {
  const area = new MemoryLocalArea();
  const language = new StoredLanguage({ area, browserLanguage: () => "ru-RU" });
  return { area, language };
}

test("the browser's language is in use until a language is kept", async () => {
  const { language } = storedLanguage();
  assert.deepEqual(await language.getLanguage(), {
    languages: ["en", "ru", "zh"],
    language: "ru-RU",
    chosen: false,
  });
});

test("the popup's choice is in use over the browser's language", async () => {
  const { language } = storedLanguage();
  await language.setLanguage("en");
  assert.deepEqual(await language.getLanguage(), {
    languages: ["en", "ru", "zh"],
    language: "en",
    chosen: true,
  });
});

test("the desktop app's language is in use over the popup's choice until it is forgotten", async () => {
  const { language } = storedLanguage();
  await language.setLanguage("en");
  await language.keepDesktopLanguage("ru");
  assert.equal((await language.getLanguage()).language, "ru");
  await language.setLanguage("en");
  assert.equal((await language.getLanguage()).language, "ru");
  await language.forgetDesktopLanguage();
  assert.deepEqual(await language.getLanguage(), {
    languages: ["en", "ru", "zh"],
    language: "en",
    chosen: true,
  });
});

test("watchers hear each change of either language until they stop", async () => {
  const { area, language } = storedLanguage();
  let heard = 0;
  const stop = language.watchLanguage(() => {
    heard += 1;
  });
  await language.keepDesktopLanguage("ru");
  await language.setLanguage("en");
  await language.forgetDesktopLanguage();
  await area.set({ unrelated: true });
  assert.equal(heard, 3);
  stop();
  await language.keepDesktopLanguage("en");
  assert.equal(heard, 3);
  assert.equal(area.listeners.size, 0);
});
