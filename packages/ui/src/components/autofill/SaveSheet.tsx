import { cn } from "cn";
import { Check } from "lucide-react";
import { useState, useSyncExternalStore } from "react";
import type { AutofillApi, SaveOpening } from "../../autofill/autofill-api.ts";
import { SaveReview } from "../../autofill/save-review.ts";
import { useTranslator } from "../../i18n/translator.tsx";
import { SelectionGroup } from "../../motion/SelectionIndicator.tsx";
import {
  destinationsOf,
  destinationTitle,
  saveActionOf,
  targetOf,
} from "../../saving/offer.ts";
import { ResponsiveDialogFooter } from "../ResponsiveDialog.tsx";
import { DestinationAvatar } from "../saving/DestinationAvatar.tsx";
import { OfferField } from "../saving/OfferField.tsx";
import { Button } from "../ui/button.tsx";
import { ItemRowContent, itemRowClass } from "../workspace/ItemRowContent.tsx";
import { AutofillNote, AutofillSheet } from "./AutofillSheet.tsx";

/** SaveSheet asks whether a saved password goes to a new item or to a credential the vault offers. */
export function SaveSheet({
  api,
  opening,
}: {
  api: AutofillApi;
  opening: SaveOpening;
}) {
  const { t } = useTranslator();
  const [review] = useState(
    () => new SaveReview(opening.offer, (choice) => api.save(choice)),
  );
  const view = useSyncExternalStore(review.state.subscribe, review.state.get);
  const { offer } = review;
  const { choice, saving, refused } = view;
  const target = targetOf(offer, choice.target);

  return (
    <AutofillSheet
      title={t("saving.title")}
      description={offer.site || undefined}
      busy={saving}
      onClose={() => api.cancel()}
    >
      <form
        id="autofill-save"
        className="grid gap-3"
        onSubmit={(event) => {
          event.preventDefault();
          void review.save();
        }}
      >
        {offer.targets.length > 0 && (
          <fieldset className="grid gap-1" disabled={saving}>
            <legend className="px-3 pb-1 text-xs text-muted-foreground">
              {t("saving.where")}
            </legend>
            <SelectionGroup id="autofill-destinations">
              {destinationsOf(offer).map((destination) => {
                const shown = targetOf(offer, destination);
                const chosen = destination === choice.target;
                return (
                  <button
                    key={destination || "new"}
                    type="button"
                    className={cn(itemRowClass(!chosen), "relative w-full")}
                    aria-pressed={chosen}
                    onClick={() => review.edit({ target: destination })}
                  >
                    <ItemRowContent
                      active={chosen}
                      avatar={
                        <DestinationAvatar offer={offer} target={shown} />
                      }
                      title={destinationTitle(offer, destination, t)}
                      tags={shown?.tags}
                      detail={{
                        text: shown
                          ? shown.account || t("credential.login.empty")
                          : offer.site,
                      }}
                      trailing={
                        chosen ? (
                          <Check
                            className="size-4 text-foreground/75"
                            aria-hidden="true"
                          />
                        ) : null
                      }
                    />
                  </button>
                );
              })}
            </SelectionGroup>
          </fieldset>
        )}
        {!target && (
          <>
            <OfferField
              label={t("saving.name")}
              value={choice.name}
              error={refused === "name" ? t("saving.name.refused") : ""}
              disabled={saving}
              onChange={(name) => review.edit({ name })}
            />
            <OfferField
              label={t("saving.account")}
              value={choice.account}
              error={refused === "account" ? t("saving.account.refused") : ""}
              disabled={saving}
              onChange={(account) => review.edit({ account })}
            />
          </>
        )}
        {view.failed && <AutofillNote>{t("saving.error")}</AutofillNote>}
      </form>
      <ResponsiveDialogFooter>
        <Button
          type="button"
          variant="quiet"
          size="pill"
          disabled={saving}
          onClick={() => api.cancel()}
        >
          {t("saving.not-now")}
        </Button>
        <Button
          type="submit"
          form="autofill-save"
          variant="raised"
          size="pill"
          disabled={saving}
        >
          {t(saveActionOf(target))}
        </Button>
      </ResponsiveDialogFooter>
    </AutofillSheet>
  );
}
