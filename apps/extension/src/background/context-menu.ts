import { matchLanguage } from "@ravenpass/ui/i18n/language.ts";
import { catalogs, type MessageKey } from "@ravenpass/ui/i18n/messages.ts";
import type { StoredLanguage } from "../language.ts";
import type { FieldMenuOpen, FileMenuOpen } from "../messages.ts";

const parentId = "ravenpass";

/** `all` would also cover the toolbar button's and tab strip's menus, which have no frame. */
const pageContexts: chrome.contextMenus.CreateProperties["contexts"] = [
  "page",
  "frame",
  "selection",
  "link",
  "editable",
  "image",
  "video",
  "audio",
];

interface Item {
  readonly id: string;
  readonly title: MessageKey;
  readonly contexts: chrome.contextMenus.CreateProperties["contexts"];
  /** Sent to the clicked frame, which knows what was right-clicked. */
  readonly message: FieldMenuOpen | FileMenuOpen;
}

const items: readonly Item[] = [
  {
    id: "show-passwords",
    title: "extension.context.passwords",
    contexts: ["editable"],
    message: { kind: "field-menu-open", field: "login" },
  },
  {
    id: "show-codes",
    title: "extension.context.codes",
    contexts: ["editable"],
    message: { kind: "field-menu-open", field: "code" },
  },
  {
    id: "show-cards",
    title: "extension.context.cards",
    contexts: ["editable"],
    message: { kind: "field-menu-open", field: "card" },
  },
  {
    id: "upload-identity-file",
    title: "extension.upload.menu",
    contexts: pageContexts,
    message: { kind: "file-menu-open" },
  },
];

export interface ContextMenuDependencies {
  readonly language: Pick<StoredLanguage, "getLanguage">;
  readonly menus: Pick<
    typeof chrome.contextMenus,
    "removeAll" | "create" | "update"
  >;
  readonly send: (
    tabId: number,
    frameId: number,
    message: FieldMenuOpen | FileMenuOpen,
  ) => Promise<unknown>;
}

/** Chrome builds the menu before the right-click and reports no target; the clicked frame acts on what it saw. */
export class ContextMenu {
  private readonly language: ContextMenuDependencies["language"];
  private readonly menus: ContextMenuDependencies["menus"];
  private readonly send: ContextMenuDependencies["send"];

  constructor({ language, menus, send }: ContextMenuDependencies) {
    this.language = language;
    this.menus = menus;
    this.send = send;
  }

  async create(): Promise<void> {
    await this.menus.removeAll();
    const catalog = await this.catalog();
    this.menus.create({
      id: parentId,
      title: "Ravenpass",
      contexts: pageContexts,
    });
    for (const { id, title, contexts } of items) {
      this.menus.create({ id, parentId, title: catalog[title], contexts });
    }
  }

  async retitle(): Promise<void> {
    const catalog = await this.catalog();
    await Promise.all(
      items.map(({ id, title }) =>
        this.menus.update(id, { title: catalog[title] }),
      ),
    );
  }

  async clicked(
    info: Pick<chrome.contextMenus.OnClickData, "menuItemId" | "frameId">,
    tab: Pick<chrome.tabs.Tab, "id"> | undefined,
  ): Promise<void> {
    const item = items.find(({ id }) => id === info.menuItemId);
    if (!item || tab?.id === undefined) return;
    await this.send(tab.id, info.frameId ?? 0, item.message);
  }

  private async catalog() {
    const { language } = await this.language.getLanguage();
    return catalogs[matchLanguage(language)];
  }
}
