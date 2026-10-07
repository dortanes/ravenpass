import { Copy, IdCard } from "lucide-react";
import { useState } from "react";
import type { MessageKey } from "../../i18n/messages.ts";
import { useTranslator } from "../../i18n/translator.tsx";
import { CalendarDate } from "../../identities/dates.ts";
import { scansOf } from "../../identities/identity.ts";
import type { Identity, IdentityField, ScanSummary } from "../../vault-api.ts";
import { ConfirmDialog } from "../ConfirmDialog.tsx";
import { ScrollArea } from "../ui/scroll-area.tsx";
import { AddressCard } from "./AddressCard.tsx";
import { type DetailControls, DetailHeader } from "./DetailHeader.tsx";
import { DocumentCard } from "./DocumentCard.tsx";
import { FieldBlock, FieldRow, NotesBlock, TitledBlock } from "./Fields.tsx";
import type { ScanActions } from "./ScanStrip.tsx";

export function IdentityDetail({
  identity,
  controls,
  scans,
  onCopy,
}: {
  identity: Identity;
  controls: DetailControls;
  scans: ScanActions;
  /** Copies one value, and names the notice that says what was copied. */
  onCopy: (field: IdentityField, notice: MessageKey) => void;
}) {
  const { t, language } = useTranslator();
  const { busy } = controls;
  // The scan whose unencrypted copy waits for the user to confirm it.
  const [saving, setSaving] = useState<ScanSummary | null>(null);
  const today = new Date();
  const birthday = CalendarDate.parse(identity.birthday)?.format(language);
  const contact = identity.emails.length > 0 || identity.phones.length > 0;

  return (
    <ScrollArea className="min-h-0 flex-1">
      <article className="flex flex-1 flex-col gap-[11px]">
        <DetailHeader
          label={identity.label}
          photo={identity.photo}
          title={identity.label || t("identity.untitled")}
          icon={IdCard}
          subtitle={birthday}
          controls={controls}
        />

        {(identity.fullName || birthday) && (
          <TitledBlock title={t("identity.block.personal")}>
            <FieldBlock>
              {identity.fullName && (
                <FieldRow
                  label={t("identity.field.full-name")}
                  action={t("identity.copy.full-name")}
                  icon={Copy}
                  disabled={busy}
                  onAction={() =>
                    onCopy({ kind: "fullName" }, "identity.copied.full-name")
                  }
                >
                  <span className="min-w-0 flex-1 truncate text-[13px]">
                    {identity.fullName}
                  </span>
                </FieldRow>
              )}
              {birthday && (
                <FieldRow
                  label={t("identity.field.birthday")}
                  action={t("identity.copy.birthday")}
                  icon={Copy}
                  disabled={busy}
                  onAction={() =>
                    onCopy({ kind: "birthday" }, "identity.copied.birthday")
                  }
                >
                  <span className="min-w-0 flex-1 truncate text-[13px]">
                    {birthday}
                  </span>
                </FieldRow>
              )}
            </FieldBlock>
          </TitledBlock>
        )}

        {contact && (
          <TitledBlock title={t("identity.block.contact")}>
            <FieldBlock>
              {identity.emails.map((email, index) => (
                <FieldRow
                  // biome-ignore lint/suspicious/noArrayIndexKey: an email is addressed by its position, which the copy reference names.
                  key={`email-${index}`}
                  label={t("identity.field.email")}
                  action={t("identity.copy.email")}
                  icon={Copy}
                  disabled={busy}
                  onAction={() =>
                    onCopy({ kind: "email", index }, "workspace.copy.email")
                  }
                >
                  <span className="min-w-0 flex-1 truncate text-[13px]">
                    {email}
                  </span>
                </FieldRow>
              ))}
              {identity.phones.map((phone, index) => (
                <FieldRow
                  // biome-ignore lint/suspicious/noArrayIndexKey: a phone is addressed by its position, which the copy reference names.
                  key={`phone-${index}`}
                  label={t("identity.field.phone")}
                  action={t("identity.copy.phone")}
                  icon={Copy}
                  disabled={busy}
                  onAction={() =>
                    onCopy({ kind: "phone", index }, "identity.copied.phone")
                  }
                >
                  <span className="min-w-0 flex-1 truncate text-[13px]">
                    {phone}
                  </span>
                </FieldRow>
              ))}
            </FieldBlock>
          </TitledBlock>
        )}

        {identity.addresses.length > 0 && (
          <TitledBlock title={t("identity.block.addresses")}>
            {identity.addresses.map((address, index) => (
              <AddressCard
                // biome-ignore lint/suspicious/noArrayIndexKey: an address is addressed by its position, which the copy reference names.
                key={`address-${index}`}
                address={address}
                title={
                  address.label ||
                  t("identity.address.untitled", { number: index + 1 })
                }
                busy={busy}
                onCopy={(part, notice) =>
                  onCopy(
                    part
                      ? { kind: "address", index, part }
                      : { kind: "address", index },
                    notice,
                  )
                }
              />
            ))}
          </TitledBlock>
        )}

        {identity.documents.length > 0 && (
          <TitledBlock title={t("identity.block.documents")}>
            {identity.documents.map((document, index) => (
              <DocumentCard
                // biome-ignore lint/suspicious/noArrayIndexKey: a document is addressed by its position, which the copy reference names.
                key={`document-${index}`}
                document={document}
                scans={scansOf(document, identity.attachments)}
                today={today}
                busy={busy}
                onCopy={() =>
                  onCopy({ kind: "document", index }, "identity.copied.number")
                }
                scanMenu={{
                  onCopy: (scan) => scans.copy(scan.id),
                  onSave: setSaving,
                }}
              />
            ))}
          </TitledBlock>
        )}

        {identity.notes && (
          <NotesBlock
            label={t("workspace.field.notes")}
            notes={identity.notes}
            copyLabel={t("workspace.notes.copy")}
            busy={busy}
            onCopy={() => onCopy({ kind: "notes" }, "workspace.copy.notes")}
          />
        )}
      </article>
      <ConfirmDialog
        open={saving !== null}
        title={t("identity.scan.save.title")}
        detail={t("identity.scan.save.detail")}
        confirm={t("identity.scan.save.confirm")}
        cancel={t("workspace.delete.cancel")}
        busy={busy}
        onConfirm={() => {
          if (saving) void scans.save(saving.id);
          setSaving(null);
        }}
        onCancel={() => setSaving(null)}
      />
    </ScrollArea>
  );
}
