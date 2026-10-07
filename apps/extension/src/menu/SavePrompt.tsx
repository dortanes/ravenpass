import { DestinationAvatar } from "@ravenpass/ui/components/saving/DestinationAvatar.tsx";
import { OfferField } from "@ravenpass/ui/components/saving/OfferField.tsx";
import { Button } from "@ravenpass/ui/components/ui/button.tsx";
import {
  ItemRowContent,
  itemRowClass,
} from "@ravenpass/ui/components/workspace/ItemRowContent.tsx";
import { useTranslator } from "@ravenpass/ui/i18n/translator.tsx";
import {
  destinationsOf,
  destinationTitle,
  saveActionOf,
  savedTitles,
  targetOf,
} from "@ravenpass/ui/saving/offer.ts";
import { CircleCheck } from "lucide-react";
import {
  type FormEvent,
  useEffect,
  useState,
  useSyncExternalStore,
} from "react";
import { ask, type SaveOffer } from "../messages.ts";
import { sendIgnoringClosedPort } from "../messaging/send.ts";
import { CardHeader } from "./Card.tsx";
import { DestinationPicker } from "./DestinationPicker.tsx";
import {
  type AskingView,
  OfferPrompt,
  type OfferRequests,
  type ShownView,
} from "./offer-prompt.ts";
import { FailureNote, LockedRow, NotOpenRow, StateRow } from "./Rows.tsx";
import { watchVaultState } from "./vault-state.ts";

/** Closing the card and Escape discard the capture, as Not now does. */
export function SavePrompt({
  token,
  offer,
}: {
  token: string;
  offer: SaveOffer;
}) {
  const { t } = useTranslator();
  const [prompt] = useState(() => new OfferPrompt(offer, requestsOf(token)));
  const view = useSyncExternalStore(prompt.subscribe, prompt.view);

  useEffect(() => {
    prompt.start();
    return () => prompt.stop();
  }, [prompt]);

  useEffect(() => {
    const onEscape = (event: KeyboardEvent) => {
      if (event.key !== "Escape") return;
      event.preventDefault();
      event.stopImmediatePropagation();
      void prompt.dismiss();
    };
    // Capture phase: the menu's own Escape handler closes the card without discarding the capture.
    addEventListener("keydown", onEscape, true);
    return () => removeEventListener("keydown", onEscape, true);
  }, [prompt]);

  return (
    <>
      <CardHeader
        title={t("saving.title")}
        onClose={() => void prompt.dismiss()}
      />
      <OfferBody
        prompt={prompt}
        view={view.stage === "closed" ? view.shown : view}
      />
    </>
  );
}

function requestsOf(token: string): OfferRequests {
  return {
    review: () => ask({ kind: "offer-review", token }),
    save: (choice) => ask({ kind: "offer-save", token, ...choice }),
    discard: () => ask({ kind: "offer-discard", token }),
    unlock: () => ask({ kind: "menu-unlock", token }),
    watchVault: (listener) => watchVaultState(listener),
    close: () => {
      void sendIgnoringClosedPort(ask({ kind: "menu-close", token }));
    },
  };
}

function OfferBody({ prompt, view }: { prompt: OfferPrompt; view: ShownView }) {
  const { t } = useTranslator();
  switch (view.stage) {
    case "saved":
      return (
        <div role="status">
          <StateRow
            icon={CircleCheck}
            title={t(savedTitles[view.result])}
            detail={
              targetOf(view.offer, view.choice.target)?.label ||
              view.choice.name ||
              view.offer.site
            }
          />
        </div>
      );
    case "not-open":
      return <NotOpenRow detail={t("extension.offer.not-open.detail")} />;
    case "asking":
      return view.offer.state === "locked" ? (
        <>
          <LockedRow
            detail={t("extension.offer.locked.detail")}
            onUnlock={() => void prompt.unlock()}
          />
          {view.unlockFailed && (
            <FailureNote>{t("extension.menu.unlock.error")}</FailureNote>
          )}
        </>
      ) : (
        <SaveForm prompt={prompt} view={view} />
      );
  }
}

function SaveForm({ prompt, view }: { prompt: OfferPrompt; view: AskingView }) {
  const { t } = useTranslator();
  const { offer, choice, saving } = view;
  const target = targetOf(offer, choice.target);

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    void prompt.save();
  }

  const options = destinationsOf(offer).map((destination) => {
    const option = targetOf(offer, destination);
    return {
      id: destination,
      key: destination || "new",
      title: destinationTitle(offer, destination, t),
      tags: option?.tags,
      detail: option
        ? option.account || t("credential.login.empty")
        : offer.site,
      avatar: <DestinationAvatar offer={offer} target={option} />,
    };
  });

  return (
    <form onSubmit={submit}>
      <DestinationPicker
        group="offer-destinations"
        options={options}
        chosen={choice.target}
        disabled={saving}
        onChoose={(id) => prompt.edit({ target: id })}
      >
        {target ? (
          <div className={itemRowClass(false, "menu")}>
            <ItemRowContent
              active={false}
              avatar={<DestinationAvatar offer={offer} target={target} />}
              title={target.label || t("credential.untitled")}
              tags={target.tags}
              detail={{ text: target.account || t("credential.login.empty") }}
              look="menu"
            />
          </div>
        ) : (
          <div className="flex flex-col gap-2 px-2.5 pt-0.5 pb-1.5">
            <OfferField
              label={t("saving.name")}
              value={choice.name}
              error={view.refused === "name" ? t("saving.name.refused") : ""}
              disabled={saving}
              dense
              onChange={(name) => prompt.edit({ name })}
            />
            <OfferField
              label={t("saving.account")}
              value={choice.account}
              error={
                view.refused === "account" ? t("saving.account.refused") : ""
              }
              disabled={saving}
              dense
              onChange={(account) => prompt.edit({ account })}
            />
          </div>
        )}
      </DestinationPicker>
      {view.failed && <FailureNote>{t("saving.error")}</FailureNote>}
      <div className="mt-1 flex gap-1.5">
        <Button
          type="button"
          variant="quiet"
          size="pill"
          className="flex-1"
          disabled={saving}
          onClick={() => void prompt.dismiss()}
        >
          {t("saving.not-now")}
        </Button>
        <Button
          type="submit"
          variant="raised"
          size="pill"
          className="flex-1"
          disabled={saving}
        >
          {t(saveActionOf(target))}
        </Button>
      </div>
    </form>
  );
}
