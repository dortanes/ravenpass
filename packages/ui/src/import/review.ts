import { detectNetwork } from "../cards/card.ts";
import type {
  ImportCardNetworks,
  ImportConversion,
  ImportGroups,
  ImportOptions,
  ImportPreview,
  ItemKindName,
} from "../vault-api.ts";

export interface ImportKindReview {
  readonly kind: ItemKindName;
  /** Duplicates included. */
  readonly count: number;
  /** Items the vault already holds. */
  readonly duplicates: number;
  readonly converted: readonly ImportConversion[];
  /** Items added under the options. */
  readonly adding: number;
  /** Items in the file holding a one-time code setup, duplicates included. */
  readonly oneTimeCodes: number;
  /** Passkeys in the file's items, duplicates included; only credentials hold them. */
  readonly passkeys: number;
}

export const defaultImportOptions: ImportOptions = {
  groups: true,
  skipDuplicates: true,
};

/** What an import adds from a previewed file under a choice of options. */
export class ImportReview {
  readonly preview: ImportPreview;
  readonly options: ImportOptions;
  readonly kinds: readonly ImportKindReview[];
  readonly total: number;
  /** Whether or not they are left out. */
  readonly duplicates: number;
  readonly newGroups: number;
  /** Detected as a typed card number is; an issuer that fits no single network is left out. */
  readonly cardNetworks: ImportCardNetworks;

  constructor(preview: ImportPreview, options: ImportOptions) {
    this.preview = preview;
    this.options = options;
    this.cardNetworks = Object.fromEntries(
      preview.cardIssuers.flatMap((issuer) => {
        const network = detectNetwork(issuer);
        return network ? [[issuer, network]] : [];
      }),
    );
    this.kinds = preview.kinds.map((kind) => ({
      kind: kind.kind,
      count: kind.count,
      duplicates: kind.duplicates,
      converted: kind.converted,
      adding: kind.count - (options.skipDuplicates ? kind.duplicates : 0),
      oneTimeCodes: kind.oneTimeCodes,
      passkeys: kind.kind === "credential" ? preview.importedPasskeys : 0,
    }));
    this.total = this.kinds.reduce((sum, kind) => sum + kind.adding, 0);
    this.duplicates = this.kinds.reduce(
      (sum, kind) => sum + kind.duplicates,
      0,
    );
    this.newGroups = options.groups ? this.filed.new : 0;
  }

  private get filed(): ImportGroups {
    return this.options.skipDuplicates
      ? this.preview.groupsWithoutDuplicates
      : this.preview.groups;
  }

  get empty(): boolean {
    return this.total === 0;
  }

  /** Distinct folder names across the file. */
  get folders(): number {
    const { groups } = this.preview;
    return groups.new + groups.existing + groups.dropped;
  }

  /** Zero while folders do not become groups. */
  get droppedFolders(): number {
    return this.options.groups ? this.filed.dropped : 0;
  }

  /** The file holds a skipped item, an attachment or a passkey the import leaves behind. */
  get leftBehind(): boolean {
    return (
      this.preview.skipped.length > 0 ||
      this.preview.attachments > 0 ||
      this.preview.passkeys > 0
    );
  }

  withOptions(options: ImportOptions): ImportReview {
    return new ImportReview(this.preview, options);
  }
}
