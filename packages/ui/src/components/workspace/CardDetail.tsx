import { Copy, CreditCard, ExternalLink } from "lucide-react";
import { useState } from "react";
import {
  concealedDigits,
  expiryDisplay,
  expiryEnd,
  expiryStart,
  maskedGroups,
  numberGroups,
} from "../../cards/card.ts";
import { networkName } from "../../cards/networks.ts";
import type { MessageKey } from "../../i18n/messages.ts";
import { useTranslator } from "../../i18n/translator.tsx";
import { Validity } from "../../identities/dates.ts";
import { addressName } from "../../identities/identity.ts";
import type { Card, CardField } from "../../vault-api.ts";
import { ScrollArea } from "../ui/scroll-area.tsx";
import { AddressCard } from "./AddressCard.tsx";
import { CardFace, CardTile } from "./CardFace.tsx";
import { type DetailControls, DetailHeader } from "./DetailHeader.tsx";
import {
  CopyButton,
  FieldBlock,
  FieldRow,
  NotesBlock,
  SecretRow,
  TitledBlock,
} from "./Fields.tsx";
import { useSiteIcon } from "./SiteIcons.tsx";
import { ValidityHeading, type ValidityMessages } from "./ValidityHeading.tsx";

const cardValidity: ValidityMessages = {
  expires: "card.validity.expires",
  expired: "card.validity.expired",
  none: "card.validity.none",
};

/** CardDetail owns every reveal, the turned face included, so each ends with this pane. */
export function CardDetail({
  card,
  controls,
  onCopy,
  onOpenBankSite,
}: {
  card: Card;
  controls: DetailControls;
  /** Copies one value, and names the notice that says what was copied. */
  onCopy: (field: CardField, notice: MessageKey) => void;
  onOpenBankSite: () => void;
}) {
  const { t } = useTranslator();
  const { busy } = controls;
  const logo = useSiteIcon(card.site);
  const [turned, setTurned] = useState(false);
  const [numberShown, setNumberShown] = useState(false);
  const [pinShown, setPinShown] = useState(false);
  const today = new Date();
  const expiry = expiryDisplay(card.expiry);
  const billing = card.linked?.address ?? card.billing;
  const billingTitle = card.linked
    ? [
        card.linked.identityLabel || t("identity.untitled"),
        addressName(card.linked.address),
      ]
        .filter(Boolean)
        .join(" · ")
    : t("card.billing.own");

  return (
    <ScrollArea className="min-h-0 flex-1">
      <article className="flex flex-1 flex-col gap-[11px]">
        <DetailHeader
          label={card.label}
          avatar={
            <CardTile
              network={card.network}
              color={card.color}
              logo={logo}
              size="header"
            />
          }
          title={card.label || t("card.untitled")}
          icon={CreditCard}
          subtitle={[networkName(card.network), card.bankName]
            .filter(Boolean)
            .join(" · ")}
          controls={controls}
        />

        <CardFace
          card={card}
          logo={logo}
          turned={turned}
          onTurn={() => setTurned((shown) => !shown)}
        />

        <TitledBlock title={t("card.block.card")}>
          <FieldBlock>
            {expiry && (
              <ValidityHeading
                validity={Validity.of(
                  expiryEnd(card.expiry),
                  expiryStart(card.expiry),
                )}
                today={today}
                title={t("card.validity.title", { expiry })}
                messages={cardValidity}
                action={{
                  label: t("card.copy.expiry"),
                  icon: Copy,
                  disabled: busy,
                  run: () => onCopy("expiry", "card.copied.expiry"),
                }}
              />
            )}
            <SecretRow
              label={t("card.field.number")}
              value={numberGroups(card.number, card.network).join(" ")}
              concealed={maskedGroups(card.number, card.network).join(" ")}
              revealed={numberShown}
              revealLabel={t("card.number.reveal")}
              concealLabel={t("card.number.conceal")}
              copyLabel={t("card.copy.number")}
              busy={busy}
              onReveal={() => setNumberShown((shown) => !shown)}
              onCopy={() => onCopy("number", "card.copied.number")}
            />
            {card.holder && (
              <FieldRow
                label={t("card.field.holder")}
                action={t("card.copy.holder")}
                icon={Copy}
                disabled={busy}
                onAction={() => onCopy("holder", "card.copied.holder")}
              >
                <span className="min-w-0 flex-1 truncate text-[13px]">
                  {card.holder}
                </span>
              </FieldRow>
            )}
            {card.securityCode && (
              <SecretRow
                label={t("card.field.security-code")}
                value={card.securityCode}
                concealed={concealedDigits(card.securityCode)}
                revealed={turned}
                revealLabel={t("card.face.back")}
                concealLabel={t("card.face.front")}
                copyLabel={t("card.copy.security-code")}
                busy={busy}
                onReveal={() => setTurned((shown) => !shown)}
                onCopy={() =>
                  onCopy("securityCode", "card.copied.security-code")
                }
              />
            )}
            {card.pin && (
              <SecretRow
                label={t("card.field.pin")}
                value={card.pin}
                concealed={concealedDigits(card.pin)}
                revealed={pinShown}
                revealLabel={t("card.pin.reveal")}
                concealLabel={t("card.pin.conceal")}
                copyLabel={t("card.copy.pin")}
                busy={busy}
                onReveal={() => setPinShown((shown) => !shown)}
                onCopy={() => onCopy("pin", "card.copied.pin")}
              />
            )}
          </FieldBlock>
        </TitledBlock>

        {(card.bankName || card.bankSite) && (
          <TitledBlock title={t("card.block.bank")}>
            <FieldBlock>
              {card.bankName && (
                <FieldRow
                  label={t("card.field.bank-name")}
                  action={t("card.copy.bank-name")}
                  icon={Copy}
                  disabled={busy}
                  onAction={() => onCopy("bankName", "card.copied.bank-name")}
                >
                  <span className="min-w-0 flex-1 truncate text-[13px]">
                    {card.bankName}
                  </span>
                </FieldRow>
              )}
              {card.bankSite && (
                <FieldRow
                  label={t("card.field.bank-site")}
                  action={t("card.bank.open")}
                  icon={ExternalLink}
                  disabled={busy}
                  onAction={onOpenBankSite}
                  accessory={
                    <CopyButton
                      label={t("card.copy.bank-site")}
                      busy={busy}
                      onCopy={() => onCopy("bankSite", "card.copied.bank-site")}
                    />
                  }
                >
                  <span className="min-w-0 flex-1 truncate text-[13px]">
                    {card.bankSite}
                  </span>
                </FieldRow>
              )}
            </FieldBlock>
          </TitledBlock>
        )}

        {billing && (
          <TitledBlock title={t("card.block.billing")}>
            <AddressCard
              address={billing}
              title={billingTitle}
              busy={busy}
              onCopy={(part, notice) =>
                onCopy(part ? `billing:${part}` : "billing", notice)
              }
            />
          </TitledBlock>
        )}

        {card.notes && (
          <NotesBlock
            label={t("workspace.field.notes")}
            notes={card.notes}
            copyLabel={t("workspace.notes.copy")}
            busy={busy}
            onCopy={() => onCopy("notes", "workspace.copy.notes")}
          />
        )}
      </article>
    </ScrollArea>
  );
}
