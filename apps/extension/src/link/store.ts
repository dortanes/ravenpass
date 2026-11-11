import { type DBSchema, type IDBPDatabase, openDB } from "idb";

/** What the extension keeps of its link, and nothing from the vault. */
export interface LinkRecord {
  /** The port the desktop app listens on. */
  readonly port: number;
  /** The desktop app's static public key: v1 X25519 or v2 P-256. */
  readonly desktopPublicKey: Uint8Array<ArrayBuffer>;
  /** The extension's static key pair; its private key cannot be exported. */
  readonly staticKeys: CryptoKeyPair;
  /** When the link was made, in Unix milliseconds. */
  readonly linkedAt: number;
}

export interface LinkStore {
  load(): Promise<LinkRecord | undefined>;
  save(record: LinkRecord): Promise<void>;
  clear(): Promise<void>;
}

const recordKey = "current";

interface LinkDatabase extends DBSchema {
  link: { key: typeof recordKey; value: LinkRecord };
}

/** Structured clone keeps the stored private CryptoKey non-extractable. */
export class IndexedLinkStore implements LinkStore {
  private database: Promise<IDBPDatabase<LinkDatabase>> | null = null;

  async load(): Promise<LinkRecord | undefined> {
    return (await this.open()).get("link", recordKey);
  }

  async save(record: LinkRecord): Promise<void> {
    await (await this.open()).put("link", record, recordKey);
  }

  async clear(): Promise<void> {
    await (await this.open()).delete("link", recordKey);
  }

  // The browser can terminate the connection, as when site data is cleared; the next call reopens.
  private open(): Promise<IDBPDatabase<LinkDatabase>> {
    this.database ??= openDB<LinkDatabase>("ravenpass", 1, {
      upgrade(database) {
        database.createObjectStore("link");
      },
      terminated: () => {
        this.database = null;
      },
    }).catch((error: unknown) => {
      this.database = null;
      throw error;
    });
    return this.database;
  }
}
