import { Button } from "@ravenpass/ui/components/ui/button.tsx";
import {
  ItemRowContent,
  itemRowClass,
} from "@ravenpass/ui/components/workspace/ItemRowContent.tsx";
import { SiteAvatar } from "@ravenpass/ui/components/workspace/SiteAvatar.tsx";
import { useTranslator } from "@ravenpass/ui/i18n/translator.tsx";
import { SelectionGroup } from "@ravenpass/ui/motion/SelectionIndicator.tsx";
import { targetTitle } from "@ravenpass/ui/saving/offer.ts";
import { CircleCheck } from "lucide-react";
import {
  type ReactNode,
  useEffect,
  useState,
  useSyncExternalStore,
} from "react";
import {
  ask,
  isFrameMessage,
  type PasskeyCardContent,
  type PasskeyListing,
} from "../messages.ts";
import { sendIgnoringClosedPort } from "../messaging/send.ts";
import { CardHeader } from "./Card.tsx";
import { PasskeyRow } from "./Credentials.tsx";
import { DestinationPicker } from "./DestinationPicker.tsx";
import { useHighlight } from "./highlight.ts";
import {
  newItem,
  PasskeyPrompt,
  type PasskeyPromptRequests,
  type PasskeyView,
  passkeyNote,
} from "./passkey-prompt.ts";
import {
  FailureNote,
  LockedRow,
  NotOpenRow,
  StateRow,
  VerificationRow,
} from "./Rows.tsx";
import { watchVaultState } from "./vault-state.ts";

/** Closing the card refuses the page's request; Other options hands it to Chrome. */
export function PasskeyCard({
  token,
  site,
  content,
}: {
  token: string;
  site: string;
  content: PasskeyCardContent;
}) {
  const { t } = useTranslator();
  const [prompt] = useState(
    () => new PasskeyPrompt(content, requestsOf(token)),
  );
  const view = useSyncExternalStore(prompt.subscribe, prompt.view);

  useEffect(() => {
    prompt.start();
    return () => prompt.stop();
  }, [prompt]);

  useEffect(() => {
    const onMessage = (
      message: unknown,
      sender: chrome.runtime.MessageSender,
    ): undefined => {
      if (
        sender.id === chrome.runtime.id &&
        isFrameMessage(message, token) &&
        message.kind === "passkey-progress"
      ) {
        prompt.progress(message.progress);
      }
    };
    chrome.runtime.onMessage.addListener(onMessage);
    return () => chrome.runtime.onMessage.removeListener(onMessage);
  }, [prompt, token]);

  return (
    <>
      <CardHeader
        title={t(
          content.mode === "get"
            ? "extension.passkey.sign-in.title"
            : "extension.passkey.save.title",
        )}
        onClose={() => prompt.close()}
      />
      <PasskeyBody
        prompt={prompt}
        view={view}
        site={site}
        mode={content.mode}
      />
    </>
  );
}

function requestsOf(token: string): PasskeyPromptRequests {
  return {
    sign: (choice) => ask({ kind: "passkey-sign", token, ...choice }),
    save: (target) => ask({ kind: "passkey-save", token, target }),
    elsewhere: () => ask({ kind: "passkey-elsewhere", token }),
    review: () => ask({ kind: "passkey-review", token }),
    unlock: () => ask({ kind: "menu-unlock", token }),
    watchVault: (listener) => watchVaultState(listener),
    close: () => {
      void sendIgnoringClosedPort(ask({ kind: "menu-close", token }));
    },
  };
}

function PasskeyBody({
  prompt,
  view,
  site,
  mode,
}: {
  prompt: PasskeyPrompt;
  view: PasskeyView;
  site: string;
  mode: PasskeyCardContent["mode"];
}) {
  const { t } = useTranslator();
  const { listing, waiting, note } = view;
  if (waiting?.progress) {
    return <VerificationRow progress={waiting.progress} subject="passkey" />;
  }
  const noted = note && <FailureNote>{t(passkeyNote(note, mode))}</FailureNote>;
  switch (listing.state) {
    case "sign-in":
      return (
        <>
          <PasskeyList
            prompt={prompt}
            listing={listing}
            site={site}
            busy={waiting !== null}
          />
          {noted}
          <Elsewhere prompt={prompt} busy={waiting !== null} />
        </>
      );
    case "save":
      return (
        <SaveForm
          prompt={prompt}
          view={view}
          listing={listing}
          site={site}
          noted={noted}
        />
      );
    case "excluded":
      return (
        <>
          <StateRow
            icon={CircleCheck}
            title={t("extension.passkey.excluded.title")}
            detail={t("extension.passkey.excluded.detail")}
          />
          <Button
            type="button"
            variant="raised"
            size="pill"
            className="mt-1 w-full"
            data-menu-item
            onClick={() => prompt.close()}
          >
            {t("extension.card.close")}
          </Button>
        </>
      );
    case "locked":
      return (
        <>
          <LockedRow
            detail={t("extension.passkey.locked.detail")}
            onUnlock={() => void prompt.unlock()}
          />
          {view.unlockFailed && (
            <FailureNote>{t("extension.menu.unlock.error")}</FailureNote>
          )}
          <Elsewhere prompt={prompt} busy={false} />
        </>
      );
    case "not-open":
      return (
        <>
          <NotOpenRow detail={t("extension.passkey.not-open.detail")} />
          <Elsewhere prompt={prompt} busy={false} />
        </>
      );
  }
}

function PasskeyList({
  prompt,
  listing,
  site,
  busy,
}: {
  prompt: PasskeyPrompt;
  listing: Extract<PasskeyListing, { state: "sign-in" }>;
  site: string;
  busy: boolean;
}) {
  const { t } = useTranslator();
  const highlight = useHighlight();
  return (
    <nav
      className="flex max-h-[284px] flex-col gap-1 overflow-y-auto"
      aria-label={t("extension.passkey.list.label", { site })}
      aria-busy={busy}
      {...highlight.list}
    >
      <SelectionGroup id="passkey-card">
        {listing.passkeys.map((passkey) => (
          <PasskeyRow
            key={passkey.credentialId}
            passkey={passkey}
            avatar={<SiteAvatar site={site} label={passkey.label} />}
            highlight={highlight}
            onChoose={() => void prompt.sign(passkey)}
          />
        ))}
      </SelectionGroup>
    </nav>
  );
}

function SaveForm({
  prompt,
  view,
  listing,
  site,
  noted,
}: {
  prompt: PasskeyPrompt;
  view: PasskeyView;
  listing: Extract<PasskeyListing, { state: "save" }>;
  site: string;
  noted: ReactNode;
}) {
  const { t } = useTranslator();
  const busy = view.waiting !== null;
  const target =
    listing.targets.find(({ credential }) => credential === view.target) ??
    null;
  const options = [null, ...listing.targets].map((option) => {
    const id = option?.credential ?? newItem;
    return {
      id,
      key: id,
      title: targetTitle(
        option && { label: option.label, action: "add-passkey" },
        t,
      ),
      tags: option?.tags,
      detail: option ? option.account || t("credential.login.empty") : site,
      avatar: <SiteAvatar site={site} label={option?.label ?? site} />,
    };
  });

  return (
    <form
      onSubmit={(event) => {
        event.preventDefault();
        void prompt.save();
      }}
    >
      <DestinationPicker
        group="passkey-destinations"
        options={options}
        chosen={view.target}
        disabled={busy}
        onChoose={(id) => prompt.choose(id)}
      >
        <div className={itemRowClass(false, "menu")}>
          <ItemRowContent
            active={false}
            avatar={<SiteAvatar site={site} label={target?.label ?? site} />}
            title={target ? target.label || t("credential.untitled") : site}
            tags={target?.tags}
            detail={{
              text:
                (target ? target.account : listing.account) ||
                t("credential.login.empty"),
            }}
            look="menu"
          />
        </div>
      </DestinationPicker>
      {noted}
      <div className="mt-1 flex gap-1.5">
        <Button
          type="button"
          variant="quiet"
          size="pill"
          className="flex-1"
          disabled={busy}
          onClick={() => void prompt.elsewhere()}
        >
          {t("extension.passkey.elsewhere")}
        </Button>
        <Button
          type="submit"
          variant="raised"
          size="pill"
          className="flex-1"
          disabled={busy}
        >
          {t("extension.passkey.save")}
        </Button>
      </div>
    </form>
  );
}

function Elsewhere({ prompt, busy }: { prompt: PasskeyPrompt; busy: boolean }) {
  const { t } = useTranslator();
  return (
    <Button
      type="button"
      variant="quiet"
      size="pill"
      className="mt-1 w-full"
      data-menu-item
      disabled={busy}
      onClick={() => void prompt.elsewhere()}
    >
      {t("extension.passkey.elsewhere")}
    </Button>
  );
}
