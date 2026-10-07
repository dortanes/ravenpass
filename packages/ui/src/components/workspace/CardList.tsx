import { CreditCard } from "lucide-react";
import { cardLine } from "../../cards/card.ts";
import { useTranslator } from "../../i18n/translator.tsx";
import { Validity } from "../../identities/dates.ts";
import type { CardSummary } from "../../vault-api.ts";
import { CardTile } from "./CardFace.tsx";
import { ItemList, type ListProps } from "./ItemList.tsx";
import { validityTone } from "./tones.ts";

export function CardList(props: ListProps<CardSummary>) {
  const { t } = useTranslator();
  const today = new Date();

  return (
    <ItemList
      {...props}
      label={t("card.list.label")}
      untitled={t("card.untitled")}
      shape="tile"
      icon={CreditCard}
      detail={(entry) => ({
        text: cardLine(entry.bankName, entry.network, entry.lastFour),
        tone: validityTone(Validity.of(entry.expiresOn), today),
      })}
      site={(entry) => entry.site}
      avatar={(entry, { selected, logo }) => (
        <CardTile
          network={entry.network}
          color={entry.color}
          logo={logo}
          size="row"
          selected={selected}
        />
      )}
    />
  );
}
