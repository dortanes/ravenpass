import {
  IdCard,
  KeyRound,
  LockKeyhole,
  NotebookText,
  Plus,
  Tags,
} from "lucide-react";
import { type ComponentType, type ReactNode, useState } from "react";
import { cardLine } from "../../cards/card.ts";
import { accountOf } from "../../credentials/credential.ts";
import { useCapabilities } from "../../host/capabilities.tsx";
import type { MessageKey } from "../../i18n/messages.ts";
import { useTranslator } from "../../i18n/translator.tsx";
import type {
  CardSummary,
  CredentialSummary,
  Group,
  IdentitySummary,
  NoteSummary,
  SeedSummary,
} from "../../vault-api.ts";
import {
  cardSearchValues,
  credentialSearchValues,
  identitySearchValues,
  matchesQuery,
  noteSearchValues,
  paletteEntries,
  seedSearchValues,
} from "../../workspace/sections.ts";
import {
  offeredSettingsDestinations,
  type SettingsSection,
} from "../settings/sections.ts";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  CommandShortcut,
} from "../ui/command.tsx";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogTitle,
} from "../ui/dialog.tsx";
import { Avatar } from "./Avatar.tsx";
import { CardTile } from "./CardFace.tsx";
import { GroupBadge } from "./GroupBadge.tsx";
import { type ItemPlaceName, itemPlaces } from "./places.ts";
import { seedFace, useSeedLine } from "./SeedList.tsx";
import { useSiteIcons } from "./SiteIcons.tsx";
import { TagBadge } from "./TagBadge.tsx";

/** What the person chose in the command palette. */
export type PaletteChoice =
  | { kind: "item"; place: ItemPlaceName; id: string }
  | { kind: "group"; id: string }
  | { kind: "settings"; section: SettingsSection }
  | { kind: "new"; place: ItemPlaceName }
  | { kind: "lock" };

const newItemLabels: Record<ItemPlaceName, MessageKey> = {
  passwords: "credential.editor.new.title",
  identities: "identity.editor.new.title",
  cards: "card.editor.new.title",
  notes: "note.editor.new.title",
  seeds: "seed.editor.new.title",
};

/** CommandPalette finds items, groups, settings and actions in the open vault by name. */
export function CommandPalette({
  open,
  onOpenChange,
  credentials,
  identities,
  cards,
  notes,
  seeds,
  groups,
  busy,
  onChoose,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  credentials: CredentialSummary[];
  identities: IdentitySummary[];
  cards: CardSummary[];
  notes: NoteSummary[];
  seeds: SeedSummary[];
  groups: Group[];
  /** Actions that write or lock wait while another change runs. */
  busy: boolean;
  onChoose: (choice: PaletteChoice) => void;
}) {
  const { t } = useTranslator();

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="gap-0 overflow-hidden p-0"
        showCloseButton={false}
      >
        <DialogTitle className="sr-only">
          {t("workspace.palette.title")}
        </DialogTitle>
        <DialogDescription className="sr-only">
          {t("workspace.palette.description")}
        </DialogDescription>
        <PaletteSearch
          credentials={credentials}
          identities={identities}
          cards={cards}
          notes={notes}
          seeds={seeds}
          groups={groups}
          busy={busy}
          onChoose={onChoose}
        />
      </DialogContent>
    </Dialog>
  );
}

/** PaletteSearch is mounted only while the palette is open. */
function PaletteSearch({
  credentials,
  identities,
  cards,
  notes,
  seeds,
  groups,
  busy,
  onChoose: choose,
}: {
  credentials: CredentialSummary[];
  identities: IdentitySummary[];
  cards: CardSummary[];
  notes: NoteSummary[];
  seeds: SeedSummary[];
  groups: Group[];
  busy: boolean;
  onChoose: (choice: PaletteChoice) => void;
}) {
  const { t } = useTranslator();
  const seedLine = useSeedLine();
  const capabilities = useCapabilities();
  const [query, setQuery] = useState("");

  const foundCredentials = paletteEntries(
    credentials,
    credentialSearchValues,
    query,
  );
  const foundIdentities = paletteEntries(
    identities,
    identitySearchValues,
    query,
  );
  const foundCards = paletteEntries(cards, cardSearchValues, query);
  const foundNotes = paletteEntries(notes, noteSearchValues, query);
  const foundSeeds = paletteEntries(seeds, seedSearchValues, query);
  const siteIcon = useSiteIcons([
    ...foundCredentials.map((credential) => credential.site),
    ...foundCards.map((card) => card.site),
  ]);
  // Named in the order the vault keeps its groups, as the tabs show them.
  const groupsOf = (item: { groups: string[] }) =>
    groups.filter((group) => item.groups.includes(group.id));
  const foundGroups = groups.filter((group) =>
    matchesQuery([group.name], query),
  );
  const foundSections = offeredSettingsDestinations(capabilities).filter(
    (section) => matchesQuery([t(section.label), t(section.summary)], query),
  );
  const actions = [
    ...itemPlaces.map(({ id }) => ({
      value: `new:${id}`,
      icon: Plus,
      label: t(newItemLabels[id]),
      choice: { kind: "new", place: id } as const,
    })),
    {
      value: "lock",
      icon: LockKeyhole,
      label: t("workspace.toolbar.lock"),
      choice: { kind: "lock" } as const,
    },
  ].filter((action) => matchesQuery([action.label], query));

  return (
    <Command shouldFilter={false} loop>
      <CommandInput
        value={query}
        onValueChange={setQuery}
        placeholder={t("workspace.palette.placeholder")}
        className="h-11"
      />
      <CommandList className="max-h-[360px]">
        <CommandEmpty>{t("workspace.palette.empty")}</CommandEmpty>
        {foundCredentials.length > 0 && (
          <CommandGroup heading={t("workspace.rail.passwords")}>
            {foundCredentials.map((credential) => (
              <ItemRow
                key={credential.id}
                value={`credential:${credential.id}`}
                face={
                  <Avatar
                    label={credential.label}
                    logo={siteIcon(credential.site)}
                    icon={KeyRound}
                    size="compact"
                    shape="tile"
                    emphasis="none"
                  />
                }
                label={credential.label || t("credential.untitled")}
                groups={groupsOf(credential)}
                tags={credential.tags}
                detail={accountOf(credential)}
                onSelect={() =>
                  choose({
                    kind: "item",
                    place: "passwords",
                    id: credential.id,
                  })
                }
              />
            ))}
          </CommandGroup>
        )}
        {foundIdentities.length > 0 && (
          <CommandGroup heading={t("workspace.rail.identities")}>
            {foundIdentities.map((identity) => (
              <ItemRow
                key={identity.id}
                value={`identity:${identity.id}`}
                face={
                  <Avatar
                    label={identity.label}
                    photo={identity.thumbnail}
                    icon={IdCard}
                    size="compact"
                    shape="circle"
                    emphasis="none"
                  />
                }
                label={identity.label || t("identity.untitled")}
                groups={groupsOf(identity)}
                tags={identity.tags}
                detail={identity.email}
                onSelect={() =>
                  choose({
                    kind: "item",
                    place: "identities",
                    id: identity.id,
                  })
                }
              />
            ))}
          </CommandGroup>
        )}
        {foundCards.length > 0 && (
          <CommandGroup heading={t("workspace.rail.cards")}>
            {foundCards.map((card) => (
              <ItemRow
                key={card.id}
                value={`card:${card.id}`}
                face={
                  <CardTile
                    network={card.network}
                    color={card.color}
                    logo={siteIcon(card.site)}
                    size="compact"
                  />
                }
                label={card.label || t("card.untitled")}
                groups={groupsOf(card)}
                tags={card.tags}
                detail={cardLine(card.bankName, card.network, card.lastFour)}
                onSelect={() =>
                  choose({ kind: "item", place: "cards", id: card.id })
                }
              />
            ))}
          </CommandGroup>
        )}
        {foundNotes.length > 0 && (
          <CommandGroup heading={t("workspace.rail.notes")}>
            {foundNotes.map((note) => (
              <ItemRow
                key={note.id}
                value={`note:${note.id}`}
                face={
                  <Avatar
                    label={note.label}
                    icon={NotebookText}
                    size="compact"
                    shape="tile"
                    emphasis="none"
                  />
                }
                label={note.label || t("note.untitled")}
                groups={groupsOf(note)}
                tags={note.tags}
                detail={note.hidden ? t("note.preview.hidden") : note.preview}
                onSelect={() =>
                  choose({ kind: "item", place: "notes", id: note.id })
                }
              />
            ))}
          </CommandGroup>
        )}
        {foundSeeds.length > 0 && (
          <CommandGroup heading={t("workspace.rail.seeds")}>
            {foundSeeds.map((seed) => (
              <ItemRow
                key={seed.id}
                value={`seed:${seed.id}`}
                face={
                  <Avatar
                    label={seed.label}
                    {...seedFace(seed.format, seed.total)}
                    size="compact"
                    shape="tile"
                    emphasis="none"
                  />
                }
                label={seed.label || t("seed.untitled")}
                groups={groupsOf(seed)}
                tags={seed.tags}
                detail={seedLine(seed).text}
                onSelect={() =>
                  choose({ kind: "item", place: "seeds", id: seed.id })
                }
              />
            ))}
          </CommandGroup>
        )}
        {foundGroups.length > 0 && (
          <CommandGroup heading={t("settings.groups.heading")}>
            {foundGroups.map((group) => (
              <ItemRow
                key={group.id}
                value={`group:${group.id}`}
                face={<Glyph icon={Tags} />}
                label={group.name}
                onSelect={() => choose({ kind: "group", id: group.id })}
              />
            ))}
          </CommandGroup>
        )}
        {foundSections.length > 0 && (
          <CommandGroup heading={t("workspace.rail.settings")}>
            {foundSections.map((section) => (
              <ItemRow
                key={section.id}
                value={`settings:${section.id}`}
                face={<Glyph icon={section.icon} />}
                label={t(section.label)}
                detail={t(section.summary)}
                onSelect={() =>
                  choose({ kind: "settings", section: section.id })
                }
              />
            ))}
          </CommandGroup>
        )}
        {actions.length > 0 && (
          <CommandGroup heading={t("workspace.palette.actions")}>
            {actions.map((action) => (
              <ItemRow
                key={action.value}
                value={action.value}
                face={<Glyph icon={action.icon} />}
                label={action.label}
                disabled={busy}
                onSelect={() => choose(action.choice)}
              />
            ))}
          </CommandGroup>
        )}
      </CommandList>
    </Command>
  );
}

/** A kind icon takes an avatar's room, so the labels of every group line up. */
function Glyph({
  icon: Icon,
}: {
  icon: ComponentType<{ className?: string }>;
}) {
  return (
    <span className="flex size-6 shrink-0 items-center justify-center">
      <Icon />
    </span>
  );
}

function ItemRow({
  value,
  face,
  label,
  groups = [],
  tags = [],
  detail,
  disabled,
  onSelect,
}: {
  value: string;
  face: ReactNode;
  label: string;
  /** The groups the item belongs to, named. */
  groups?: readonly Group[];
  tags?: readonly string[];
  detail?: string;
  disabled?: boolean;
  onSelect: () => void;
}) {
  return (
    <CommandItem value={value} disabled={disabled} onSelect={onSelect}>
      {face}
      <span className="shrink truncate">{label}</span>
      {groups.length > 0 && (
        <span className="flex min-w-0 shrink-[2] gap-1 overflow-hidden">
          {groups.map((group) => (
            <GroupBadge key={group.id} name={group.name} />
          ))}
        </span>
      )}
      {tags.length > 0 && (
        <span className="flex min-w-0 shrink-[2] gap-1 overflow-hidden">
          {tags.map((tag) => (
            <TagBadge key={tag} tag={tag} />
          ))}
        </span>
      )}
      {detail && (
        <CommandShortcut className="max-w-[45%] truncate tracking-normal">
          {detail}
        </CommandShortcut>
      )}
    </CommandItem>
  );
}
