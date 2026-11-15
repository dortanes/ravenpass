import {
  CreditCard,
  IdCard,
  KeyRound,
  type LucideIcon,
  NotebookText,
  Sprout,
} from "lucide-react";
import type { MessageKey } from "../../i18n/messages.ts";
import type { ItemKindName } from "../../vault-api.ts";

/** A place holding one kind of item. */
export type ItemPlaceName =
  | "passwords"
  | "identities"
  | "cards"
  | "notes"
  | "seeds";

/** A place beside the items that fills the workspace on its own. */
export type UtilityPlace = "generator" | "settings";

export type WorkspacePlace = ItemPlaceName | UtilityPlace;

export interface ItemPlaceEntry {
  id: ItemPlaceName;
  icon: LucideIcon;
  label: MessageKey;
}

/** The places that hold items, in the order the workspace offers them. */
export const itemPlaces: readonly ItemPlaceEntry[] = [
  { id: "passwords", icon: KeyRound, label: "workspace.rail.passwords" },
  { id: "identities", icon: IdCard, label: "workspace.rail.identities" },
  { id: "cards", icon: CreditCard, label: "workspace.rail.cards" },
  { id: "notes", icon: NotebookText, label: "workspace.rail.notes" },
  { id: "seeds", icon: Sprout, label: "workspace.rail.seeds" },
];

/** The place that holds each kind of item. */
export const kindPlaces: Readonly<Record<ItemKindName, ItemPlaceName>> = {
  credential: "passwords",
  identity: "identities",
  card: "cards",
  note: "notes",
  seed: "seeds",
};

const itemKindNames = Object.keys(kindPlaces);

export function isItemKindName(value: string): value is ItemKindName {
  return itemKindNames.includes(value);
}

export function holdsItems(place: WorkspacePlace): place is ItemPlaceName {
  return itemPlaces.some((item) => item.id === place);
}

export function placeOf(id: ItemPlaceName): ItemPlaceEntry {
  const place = itemPlaces.find((item) => item.id === id);
  if (!place) throw new Error(`unknown place ${id}`);
  return place;
}
