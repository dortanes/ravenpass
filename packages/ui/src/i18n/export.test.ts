import assert from "node:assert/strict";
import test from "node:test";
import { exportMessages } from "./export.ts";
import { catalogs } from "./messages.ts";

test("a host's names take their messages in every language", () => {
  assert.deepEqual(
    exportMessages({
      app_name: "app.name",
      search: "workspace.toolbar.search",
    }),
    {
      app_name: { en: "Ravenpass", ru: "Ravenpass", zh: "Ravenpass" },
      search: { en: "Search passwords", ru: "Поиск паролей", zh: "搜索密码" },
    },
  );
});

test("a key no catalog holds or a message that takes values fails the export", () => {
  assert.throws(() => exportMessages({ missing: "autofill.nothing" }));
  assert.throws(() => exportMessages({ site: "system.reason.save-passkey" }));
  assert.throws(() => exportMessages({ wrong: 7 }));
});

test("a plural message fails the export even for a host that fills values", () => {
  assert.throws(
    () => exportMessages({ attempts: "unlock.pin.attempts" }, { values: true }),
    /cannot fill/,
  );
});

test("a host that fills values itself takes messages with placeholders", () => {
  assert.deepEqual(
    exportMessages({ unlock: "system.reason.save-passkey" }, { values: true }),
    {
      unlock: {
        en: "save a passkey for {site}",
        ru: "сохранить ключ доступа для сайта {site}",
        zh: "为 {site} 保存通行密钥",
      },
    },
  );
});

test("apostrophes and placeholders come through as the catalogs write them", () => {
  const keys = {
    mismatch: "unlock-methods.pin.mismatch",
    password: "settings.import.password.detail",
  } as const;
  const exported = exportMessages(keys, { values: true });
  for (const [name, key] of Object.entries(keys)) {
    assert.deepEqual(exported[name], {
      en: catalogs.en[key],
      ru: catalogs.ru[key],
      zh: catalogs.zh[key],
    });
  }
  assert.equal(exported.mismatch?.en, "The PINs don't match.");
});
