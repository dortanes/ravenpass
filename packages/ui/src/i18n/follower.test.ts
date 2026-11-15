import assert from "node:assert/strict";
import test from "node:test";
import type { LanguageSettings, LanguageSource } from "../vault-api.ts";
import { LanguageFollower } from "./follower.ts";
import type { Language } from "./language.ts";

const offered = ["en", "ru"];

function settings(language: string, chosen: boolean): LanguageSettings {
  return { languages: offered, language, chosen };
}

class WatchedSource implements LanguageSource {
  current: LanguageSettings;
  readonly recorded: string[] = [];
  readonly watchers = new Set<() => void>();

  constructor(current: LanguageSettings) {
    this.current = current;
  }

  async getLanguage(): Promise<LanguageSettings> {
    return this.current;
  }

  async setLanguage(language: string): Promise<void> {
    this.recorded.push(language);
  }

  watchLanguage(onChange: () => void): () => void {
    this.watchers.add(onChange);
    return () => this.watchers.delete(onChange);
  }

  change(language: string): void {
    this.current = settings(language, true);
    for (const watcher of this.watchers) watcher();
  }
}

function settle() {
  return new Promise((resolve) => setTimeout(resolve, 0));
}

function follow(source: LanguageSource) {
  const reported: Language[] = [];
  const stop = new LanguageFollower(source).follow((language) =>
    reported.push(language),
  );
  return { reported, stop };
}

test("a chosen language is reported once from a source that offers no watch", async () => {
  const { reported } = follow({
    getLanguage: async () => settings("ru", true),
  });
  await settle();
  assert.deepEqual(reported, ["ru"]);
});

test("until a language is chosen the one the source settled on is reported and nothing is recorded", async () => {
  const source = new WatchedSource(settings("ru", false));
  const { reported } = follow(source);
  await settle();
  assert.deepEqual(reported, ["ru"]);
  assert.deepEqual(source.recorded, []);
});

test("a tag with a region reads as the language Ravenpass offers", async () => {
  const { reported } = follow({
    getLanguage: async () => settings("ru-RU", false),
  });
  await settle();
  assert.deepEqual(reported, ["ru"]);
});

test("each change a watched source reports is followed", async () => {
  const source = new WatchedSource(settings("en", true));
  const { reported } = follow(source);
  await settle();
  source.change("ru");
  await settle();
  source.change("en");
  await settle();
  assert.deepEqual(reported, ["en", "ru", "en"]);
});

test("stopping ends the watch and drops a read still under way", async () => {
  const source = new WatchedSource(settings("ru", true));
  const { reported, stop } = follow(source);
  stop();
  await settle();
  assert.equal(source.watchers.size, 0);
  assert.deepEqual(reported, []);
});

test("a read that a later one overtakes reports nothing", async () => {
  const answers: ((settings: LanguageSettings) => void)[] = [];
  const watchers: (() => void)[] = [];
  const source: LanguageSource = {
    getLanguage: () => new Promise((resolve) => answers.push(resolve)),
    watchLanguage: (onChange) => {
      watchers.push(onChange);
      return () => {};
    },
  };
  const { reported } = follow(source);
  const [watcher] = watchers;
  assert.ok(watcher);
  watcher();
  const [first, second] = answers;
  assert.ok(first && second);
  second(settings("ru", true));
  await settle();
  first(settings("en", true));
  await settle();
  assert.deepEqual(reported, ["ru"]);
});

test("the language in use is read without recording it", async () => {
  const unchosen = new WatchedSource(settings("ru", false));
  const read = (source: LanguageSource) =>
    new LanguageFollower(source).current();
  assert.equal(await read(unchosen), "ru");
  assert.equal(await read(new WatchedSource(settings("en", true))), "en");
  const gone: LanguageSource = {
    getLanguage: () => Promise.reject(new Error("The source is gone.")),
  };
  assert.equal(await read(gone), "en");
  assert.deepEqual(unchosen.recorded, []);
});

test("a source that fails to answer reports nothing", async () => {
  const { reported } = follow({
    getLanguage: () => Promise.reject(new Error("The source is gone.")),
  });
  await settle();
  assert.deepEqual(reported, []);
});
