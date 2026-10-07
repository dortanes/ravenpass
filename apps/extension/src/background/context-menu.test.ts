import assert from "node:assert/strict";
import test from "node:test";
import { ContextMenu } from "./context-menu.ts";

function contextMenu(language = "en") {
  const created: chrome.contextMenus.CreateProperties[] = [];
  const updated: [string, string | undefined][] = [];
  const sent: [number, number, unknown][] = [];
  const menu = new ContextMenu({
    language: {
      getLanguage: async () => ({
        languages: ["en", "ru"],
        language,
        chosen: true,
      }),
    },
    menus: {
      removeAll: async () => {
        created.length = 0;
      },
      create: (properties) => {
        created.push(properties);
        return properties.id ?? "";
      },
      update: async (id, properties) => {
        updated.push([String(id), properties.title]);
      },
    },
    send: async (tabId, frameId, message) => {
      sent.push([tabId, frameId, message]);
    },
  });
  return { menu, created, updated, sent };
}

test("the passwords, codes and cards items show on editable fields only, beside the upload item", async () => {
  const { menu, created } = contextMenu();
  await menu.create();

  const items = created.filter(({ parentId }) => parentId === "ravenpass");
  assert.deepEqual(
    items.map(({ id, title }) => [id, title]),
    [
      ["show-passwords", "Show passwords"],
      ["show-codes", "Show one-time codes"],
      ["show-cards", "Show cards"],
      ["upload-identity-file", "Upload identity file…"],
    ],
  );
  assert.deepEqual(items[0]?.contexts, ["editable"]);
  assert.deepEqual(items[1]?.contexts, ["editable"]);
  assert.deepEqual(items[2]?.contexts, ["editable"]);
  assert.ok(items[3]?.contexts?.includes("page"));
});

test("a chosen item asks the clicked frame to open its menu", async () => {
  const { menu, sent } = contextMenu();
  await menu.clicked({ menuItemId: "show-passwords", frameId: 3 }, { id: 7 });
  await menu.clicked({ menuItemId: "show-codes" }, { id: 7 });
  await menu.clicked({ menuItemId: "show-cards", frameId: 2 }, { id: 7 });
  await menu.clicked(
    { menuItemId: "upload-identity-file", frameId: 1 },
    { id: 7 },
  );

  assert.deepEqual(sent, [
    [7, 3, { kind: "field-menu-open", field: "login" }],
    [7, 0, { kind: "field-menu-open", field: "code" }],
    [7, 2, { kind: "field-menu-open", field: "card" }],
    [7, 1, { kind: "file-menu-open" }],
  ]);
});

test("a click without a tab or on another extension's item sends nothing", async () => {
  const { menu, sent } = contextMenu();
  await menu.clicked({ menuItemId: "show-passwords" }, undefined);
  await menu.clicked({ menuItemId: "other" }, { id: 7 });
  assert.deepEqual(sent, []);
});

test("the items follow the language", async () => {
  const { menu, updated } = contextMenu("ru");
  await menu.retitle();
  assert.deepEqual(updated, [
    ["show-passwords", "Показать пароли"],
    ["show-codes", "Показать одноразовые коды"],
    ["show-cards", "Показать карты"],
    ["upload-identity-file", "Загрузить файл из профиля…"],
  ]);
});
