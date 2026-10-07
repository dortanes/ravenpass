import { cn } from "cn";
import { Mail, MapPin, Phone } from "lucide-react";
import { useCallback, useEffect, useEffectEvent, useState } from "react";
import type { MessageKey } from "../i18n/messages.ts";
import { useTranslator } from "../i18n/translator.tsx";
import {
  documentTypes,
  emptyAddress,
  emptyDocument,
  labelled,
  readyToSave,
  scansOf,
  stagedScan,
  storedScans,
  unnamedDocuments,
} from "../identities/identity.ts";
import type {
  Address,
  Group,
  Identity,
  IdentityDocument,
  IdentityInput,
  IdentityLimits,
  ScanSummary,
  VaultApi,
} from "../vault-api.ts";
import { ConfirmDialog } from "./ConfirmDialog.tsx";
import { AddressRows } from "./editor/AddressRows.tsx";
import { DateField } from "./editor/DateField.tsx";
import {
  bareField,
  EditorHeader,
  EditorRow,
  GroupField,
  NotesField,
  RemoveButton,
  roomFor,
  TagField,
  type Tagging,
  toggled,
  useRemaining,
  useRows,
} from "./editor/EditorFields.tsx";
import { PhoneField } from "./editor/PhoneField.tsx";
import { PhotoField, type PhotoService } from "./editor/PhotoField.tsx";
import { Button } from "./ui/button.tsx";
import { Input } from "./ui/input.tsx";
import { ScrollArea } from "./ui/scroll-area.tsx";
import {
  documentTypeIcons,
  documentTypeNames,
} from "./workspace/document-types.ts";
import { RevealButton } from "./workspace/Fields.tsx";
import { type ScanSource, ScanStrip } from "./workspace/ScanStrip.tsx";

const formID = "identity-editor";

const block = "min-w-0 shrink-0 overflow-hidden rounded-row bg-field";

/** What the editor asks of the host besides saving: the photo and the scans it stages. */
export type EditorHost = PhotoService &
  Pick<VaultApi, "chooseScan" | "chooseScanPhoto" | "discardScans">;

export function IdentityEditor({
  initial,
  initialGroups,
  tagging,
  groups,
  limits,
  busy,
  host,
  onSave,
  onCancel,
  onFailure,
}: {
  initial?: Identity;
  /** The groups the identity starts in, which for a new one is the chosen default. */
  initialGroups: string[];
  tagging: Tagging;
  groups: Group[];
  limits: IdentityLimits | null;
  busy: boolean;
  host: EditorHost;
  onSave: (input: IdentityInput, groups: string[]) => void;
  onCancel: () => void;
  onFailure: (cause: unknown, message: MessageKey) => void;
}) {
  const { t } = useTranslator();
  const counter = useRemaining();
  const [label, setLabel] = useState(initial?.label ?? "");
  const [fullName, setFullName] = useState(initial?.fullName ?? "");
  const [birthday, setBirthday] = useState(initial?.birthday ?? "");
  const [notes, setNotes] = useState(initial?.notes ?? "");
  const [photo, setPhoto] = useState(initial?.photo ?? "");
  const [confirmingLostScans, setConfirmingLostScans] = useState(false);
  // Stored scans of documents removed in this session; a save deletes them with the document.
  const [lostScans, setLostScans] = useState<string[]>([]);
  // Tiles for the scans chosen in this session, which the vault does not hold yet.
  const [staged, setStaged] = useState<ScanSummary[]>([]);
  const [membership, setMembership] = useState<string[]>(initialGroups);
  const [tags, setTags] = useState<string[]>(initial?.tags ?? []);
  const emails = useRows(initial?.emails ?? []);
  const phones = useRows(initial?.phones ?? []);
  const addresses = useRows<Address>(initial?.addresses ?? []);
  const documents = useRows<IdentityDocument>(initial?.documents ?? []);
  const [unfinishedDates, setUnfinishedDates] = useState<ReadonlySet<string>>(
    () => new Set(),
  );

  const reportDate = useCallback((id: string, complete: boolean) => {
    setUnfinishedDates((current) => {
      if (complete !== current.has(id)) return current;
      const next = new Set(current);
      if (complete) next.delete(id);
      else next.add(id);
      return next;
    });
  }, []);

  const draft: IdentityInput = {
    label: label.trim(),
    fullName,
    birthday,
    emails: emails.values,
    phones: phones.values,
    addresses: addresses.values,
    documents: documents.values,
    notes,
    photo,
    tags,
  };

  // Leaving the editor any way drops its staged scans; after a save the host already used them.
  const discardScans = useEffectEvent(() => {
    host.discardScans().catch((cause) => {
      onFailure(cause, "identity.error.scan-discard");
    });
  });
  useEffect(() => () => discardScans(), []);

  const knownScans = [...(initial?.attachments ?? []), ...staged];

  async function chooseScan(source: ScanSource): Promise<string | null> {
    try {
      const chosen = await (source === "photo"
        ? host.chooseScanPhoto()
        : host.chooseScan());
      if (!chosen.chosen) return null;
      const tile = stagedScan(chosen);
      setStaged((current) => [...current, tile]);
      return tile.id;
    } catch (cause) {
      onFailure(cause, "identity.error.scan-attach");
      return null;
    }
  }

  function removeDocument(key: number, document: IdentityDocument) {
    setLostScans((current) => [...current, ...storedScans(document)]);
    documents.remove(key);
  }

  // A save that drops a document with stored scans deletes them, so it asks first.
  function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (lostScans.length > 0) {
      setConfirmingLostScans(true);
      return;
    }
    onSave(readyToSave(draft), membership);
  }

  const name = draft.label;
  const datesUnfinished = unfinishedDates.size > 0;
  const namesMissing = unnamedDocuments(draft) > 0;
  const roomForDocument = roomFor(documents.rows.length, limits?.documents);

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-[11px]">
      <EditorHeader
        title={t(
          initial ? "identity.editor.edit.title" : "identity.editor.new.title",
        )}
        name={name}
        form={formID}
        submit={t(
          initial ? "workspace.editor.update" : "identity.editor.create",
        )}
        canSubmit={Boolean(name) && !datesUnfinished && !namesMissing}
        busy={busy}
        onCancel={onCancel}
      />

      <ConfirmDialog
        open={confirmingLostScans}
        title={t("identity.editor.drop-scans.title")}
        detail={t("identity.editor.drop-scans.detail", {
          count: lostScans.length,
        })}
        confirm={t("identity.editor.drop-scans.confirm")}
        cancel={t("workspace.delete.cancel")}
        destructive
        busy={busy}
        onConfirm={() => {
          setConfirmingLostScans(false);
          onSave(readyToSave(draft), membership);
        }}
        onCancel={() => setConfirmingLostScans(false)}
      />

      <ScrollArea className="min-h-0 flex-1">
        <form
          id={formID}
          onSubmit={submit}
          className="flex flex-1 flex-col gap-2"
        >
          <PhotoField
            label={name}
            value={photo}
            photos={host}
            busy={busy}
            onChange={setPhoto}
            onFailure={onFailure}
          />
          <div className={block}>
            <EditorRow
              label={t("identity.field.label")}
              htmlFor="identity-name"
              counter={counter(label, limits?.label, 20)}
            >
              <Input
                id="identity-name"
                className={bareField}
                value={label}
                onChange={(event) => setLabel(event.target.value)}
                placeholder={t("identity.field.label.placeholder")}
                maxLength={limits?.label}
                disabled={busy}
                required
              />
            </EditorRow>
            <EditorRow
              label={t("identity.field.full-name")}
              htmlFor="identity-full-name"
              counter={counter(fullName, limits?.fullName, 20)}
            >
              <Input
                id="identity-full-name"
                className={bareField}
                value={fullName}
                onChange={(event) => setFullName(event.target.value)}
                maxLength={limits?.fullName}
                autoComplete="off"
                disabled={busy}
              />
            </EditorRow>
            <EditorRow
              label={t("identity.field.birthday")}
              htmlFor="identity-birthday"
            >
              <DateField
                id="identity-birthday"
                value={birthday}
                span="past"
                disabled={busy}
                onChange={setBirthday}
                onValidity={reportDate}
              />
            </EditorRow>
            <TagField
              tags={tags}
              tagging={tagging}
              busy={busy}
              onChange={setTags}
            />
            {groups.length > 0 && (
              <GroupField
                groups={groups}
                membership={membership}
                busy={busy}
                onToggle={(id) =>
                  setMembership((current) => toggled(current, id))
                }
              />
            )}
          </div>

          {(emails.rows.length > 0 || phones.rows.length > 0) && (
            <div className={block}>
              {emails.rows.map((row) => (
                <EditorRow
                  key={row.key}
                  label={t("identity.field.email")}
                  htmlFor={`identity-email-${row.key}`}
                  counter={counter(row.value, limits?.email, 20)}
                >
                  <Input
                    id={`identity-email-${row.key}`}
                    type="email"
                    className={bareField}
                    value={row.value}
                    onChange={(event) =>
                      emails.change(row.key, event.target.value)
                    }
                    maxLength={limits?.email}
                    autoCapitalize="none"
                    spellCheck={false}
                    disabled={busy}
                  />
                  <RemoveButton
                    label={t("identity.remove.email")}
                    busy={busy}
                    onRemove={() => emails.remove(row.key)}
                  />
                </EditorRow>
              ))}
              {phones.rows.map((row) => (
                <EditorRow
                  key={row.key}
                  label={t("identity.field.phone")}
                  htmlFor={`identity-phone-${row.key}`}
                >
                  <PhoneField
                    id={`identity-phone-${row.key}`}
                    value={row.value}
                    maxLength={limits?.phone}
                    disabled={busy}
                    onChange={(value) => phones.change(row.key, value)}
                  />
                  <RemoveButton
                    label={t("identity.remove.phone")}
                    busy={busy}
                    onRemove={() => phones.remove(row.key)}
                  />
                </EditorRow>
              ))}
            </div>
          )}

          {addresses.rows.map((row) => (
            <AddressFields
              key={row.key}
              id={`identity-address-${row.key}`}
              address={row.value}
              limits={limits}
              busy={busy}
              onChange={(address) => addresses.change(row.key, address)}
              onRemove={() => addresses.remove(row.key)}
            />
          ))}

          {documents.rows.map((row) => (
            <DocumentFields
              key={row.key}
              id={`identity-document-${row.key}`}
              document={row.value}
              limits={limits}
              busy={busy}
              scans={scansOf(row.value, knownScans)}
              onChange={(document) => documents.change(row.key, document)}
              onRemove={() => removeDocument(row.key, row.value)}
              onDateValidity={reportDate}
              onAttach={async (source) => {
                const id = await chooseScan(source);
                if (!id) return;
                documents.update(row.key, (document) => ({
                  ...document,
                  scans: [...document.scans, id],
                }));
              }}
              onRemoveScan={(id) =>
                documents.update(row.key, (document) => ({
                  ...document,
                  scans: document.scans.filter((scan) => scan !== id),
                }))
              }
            />
          ))}

          <fieldset className="flex shrink-0 flex-wrap items-center gap-1.5 px-0.5">
            <legend className="sr-only">{t("identity.editor.add")}</legend>
            {roomFor(emails.rows.length, limits?.emails) && (
              <Button
                type="button"
                variant="quiet"
                size="pill-sm"
                disabled={busy}
                onClick={() => emails.add("")}
              >
                <Mail data-icon="inline-start" />
                {t("identity.field.email")}
              </Button>
            )}
            {roomFor(phones.rows.length, limits?.phones) && (
              <Button
                type="button"
                variant="quiet"
                size="pill-sm"
                disabled={busy}
                onClick={() => phones.add("")}
              >
                <Phone data-icon="inline-start" />
                {t("identity.field.phone")}
              </Button>
            )}
            {roomFor(addresses.rows.length, limits?.addresses) && (
              <Button
                type="button"
                variant="quiet"
                size="pill-sm"
                disabled={busy}
                onClick={() => addresses.add(emptyAddress)}
              >
                <MapPin data-icon="inline-start" />
                {t("identity.field.address")}
              </Button>
            )}
            {roomForDocument &&
              documentTypes.map((type) => {
                const Icon = documentTypeIcons[type];
                return (
                  <Button
                    key={type}
                    type="button"
                    variant="quiet"
                    size="pill-sm"
                    disabled={busy}
                    onClick={() => documents.add(emptyDocument(type))}
                  >
                    <Icon data-icon="inline-start" />
                    {t(documentTypeNames[type])}
                  </Button>
                );
              })}
          </fieldset>

          <NotesField
            id="identity-notes"
            value={notes}
            limit={limits?.notes}
            busy={busy}
            onChange={setNotes}
          />

          <p className="shrink-0 px-1 text-[11px] text-faint">
            {t("identity.editor.requirement")}
          </p>
          {datesUnfinished && (
            <p className="shrink-0 px-1 text-[11px] text-destructive">
              {t("identity.editor.unfinished-date")}
            </p>
          )}
        </form>
      </ScrollArea>
    </div>
  );
}

/** The head of a part made of several rows: what the part is, and how to take it out. */
function PartHeader({
  icon: Icon,
  title,
  removeLabel,
  busy,
  onRemove,
}: {
  icon: React.ComponentType<{ className?: string }>;
  title: string;
  removeLabel: string;
  busy: boolean;
  onRemove: () => void;
}) {
  return (
    <div className="flex min-h-[35px] items-center gap-2 border-b py-1 pr-1.5 pl-[13px]">
      <Icon className="size-3.5 shrink-0 text-muted-foreground" />
      <span className="min-w-0 flex-1 truncate text-[11px] text-muted-foreground">
        {title}
      </span>
      <RemoveButton label={removeLabel} busy={busy} onRemove={onRemove} />
    </div>
  );
}

function AddressFields({
  id,
  address,
  limits,
  busy,
  onChange,
  onRemove,
}: {
  id: string;
  address: Address;
  limits: IdentityLimits | null;
  busy: boolean;
  onChange: (address: Address) => void;
  onRemove: () => void;
}) {
  const { t } = useTranslator();

  return (
    <fieldset className={block}>
      <legend className="sr-only">{t("identity.field.address")}</legend>
      <PartHeader
        icon={MapPin}
        title={t("identity.field.address")}
        removeLabel={t("identity.remove.address")}
        busy={busy}
        onRemove={onRemove}
      />
      <AddressRows
        id={id}
        address={address}
        named
        limits={limits}
        busy={busy}
        onChange={onChange}
      />
    </fieldset>
  );
}

function DocumentFields({
  id,
  document,
  limits,
  busy,
  scans,
  onChange,
  onRemove,
  onDateValidity,
  onAttach,
  onRemoveScan,
}: {
  id: string;
  document: IdentityDocument;
  limits: IdentityLimits | null;
  busy: boolean;
  /** The tiles of the scans the document holds, stored or chosen in this session. */
  scans: ScanSummary[];
  onChange: (document: IdentityDocument) => void;
  onRemove: () => void;
  onDateValidity: (id: string, complete: boolean) => void;
  onAttach: (source: ScanSource) => void;
  onRemoveScan: (id: string) => void;
}) {
  const { t } = useTranslator();
  const [showNumber, setShowNumber] = useState(false);
  const title = t(documentTypeNames[document.type]);

  function update(
    field: Exclude<keyof IdentityDocument, "type" | "scans">,
    value: string,
  ) {
    onChange({ ...document, [field]: value });
  }

  return (
    <fieldset className={block}>
      <legend className="sr-only">{title}</legend>
      <PartHeader
        icon={documentTypeIcons[document.type]}
        title={title}
        removeLabel={t("identity.remove.document")}
        busy={busy}
        onRemove={onRemove}
      />
      {labelled(document.type) && (
        <EditorRow
          label={t("identity.field.document-name")}
          htmlFor={`${id}-label`}
        >
          <Input
            id={`${id}-label`}
            className={bareField}
            value={document.label}
            onChange={(event) => update("label", event.target.value)}
            placeholder={t("identity.field.document-name.placeholder")}
            maxLength={limits?.documentLabel}
            autoComplete="off"
            disabled={busy}
            required
          />
        </EditorRow>
      )}
      <EditorRow label={t("identity.field.number")} htmlFor={`${id}-number`}>
        <Input
          id={`${id}-number`}
          className={cn(bareField, "font-mono")}
          type={showNumber ? "text" : "password"}
          value={document.number}
          onChange={(event) => update("number", event.target.value)}
          maxLength={limits?.documentNumber}
          autoComplete="off"
          spellCheck={false}
          disabled={busy}
        />
        <RevealButton
          shown={showNumber}
          revealLabel={t("identity.number.reveal")}
          concealLabel={t("identity.number.conceal")}
          busy={busy}
          onToggle={() => setShowNumber((shown) => !shown)}
        />
      </EditorRow>
      <EditorRow label={t("identity.field.issuer")} htmlFor={`${id}-issuer`}>
        <Input
          id={`${id}-issuer`}
          className={bareField}
          value={document.issuer}
          onChange={(event) => update("issuer", event.target.value)}
          maxLength={limits?.issuer}
          autoComplete="off"
          disabled={busy}
        />
      </EditorRow>
      <EditorRow
        label={t("identity.field.issued-on")}
        htmlFor={`${id}-issued-on`}
      >
        <DateField
          id={`${id}-issued-on`}
          value={document.issuedOn}
          span="past"
          disabled={busy}
          onChange={(value) => update("issuedOn", value)}
          onValidity={onDateValidity}
        />
      </EditorRow>
      <EditorRow
        label={t("identity.field.expires-on")}
        htmlFor={`${id}-expires-on`}
      >
        <DateField
          id={`${id}-expires-on`}
          value={document.expiresOn}
          span="any"
          disabled={busy}
          onChange={(value) => update("expiresOn", value)}
          onValidity={onDateValidity}
        />
      </EditorRow>
      <ScanStrip
        scans={scans}
        busy={busy}
        className="px-[13px] py-2.5"
        onRemove={(scan) => onRemoveScan(scan.id)}
        onAttach={onAttach}
      />
    </fieldset>
  );
}
