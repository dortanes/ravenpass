import { cardLine } from "@ravenpass/ui/cards/card.ts";
import { CardTile } from "@ravenpass/ui/components/workspace/CardFace.tsx";
import {
  ItemRowContent,
  itemRowClass,
} from "@ravenpass/ui/components/workspace/ItemRowContent.tsx";
import { useSiteIcon } from "@ravenpass/ui/components/workspace/SiteIcons.tsx";
import { validityTone } from "@ravenpass/ui/components/workspace/tones.ts";
import { useTranslator } from "@ravenpass/ui/i18n/translator.tsx";
import { Validity } from "@ravenpass/ui/identities/dates.ts";
import { SelectionGroup } from "@ravenpass/ui/motion/SelectionIndicator.tsx";
import { cn } from "cn";
import { CreditCard } from "lucide-react";
import { useEffect, useState, useSyncExternalStore } from "react";
import type { CardOption } from "../link/client.ts";
import { ask, type CardListing, isFrameMessage } from "../messages.ts";
import { sendIgnoringClosedPort } from "../messaging/send.ts";
import { FillProgress } from "./fill-progress.ts";
import { useHighlight } from "./highlight.ts";
import {
  FailureNote,
  LockedRow,
  MenuFooter,
  NotOpenRow,
  RowChevron,
  StateRow,
  VerificationRow,
} from "./Rows.tsx";
import { watchVaultState } from "./vault-state.ts";

/** What last failed for a reason other than Ravenpass locked or closed. */
type CardFailure = "fill" | "unlock" | "declined";

/** A card field's menu; choosing a card waits for the owner to confirm it on the device. A locked listing asks again
 * once Ravenpass leaves the locked state. */
export function CardMenu({
  token,
  initial,
}: {
  token: string;
  initial: CardListing;
}) {
  const { t } = useTranslator();
  const [listing, setListing] = useState(initial);
  const [failed, setFailed] = useState<CardFailure | null>(null);
  const [busy, setBusy] = useState(false);
  const [progress] = useState(() => new FillProgress());
  const confirming = useSyncExternalStore(progress.subscribe, progress.view);
  const highlight = useHighlight();

  useEffect(() => {
    const onMessage = (
      message: unknown,
      sender: chrome.runtime.MessageSender,
    ): undefined => {
      if (
        sender.id === chrome.runtime.id &&
        isFrameMessage(message, token) &&
        message.kind === "fill-progress"
      ) {
        progress.report(message.progress);
      }
    };
    chrome.runtime.onMessage.addListener(onMessage);
    return () => chrome.runtime.onMessage.removeListener(onMessage);
  }, [progress, token]);

  const locked = listing.state === "locked";
  useEffect(() => {
    if (!locked) return;
    return watchVaultState((state) => {
      if (state === "locked") return;
      void sendIgnoringClosedPort(ask({ kind: "card-review", token })).then(
        (answer) => {
          if (answer?.listing) setListing(answer.listing);
        },
      );
    });
  }, [locked, token]);

  async function fill(id: string) {
    if (busy) return;
    setBusy(true);
    setFailed(null);
    const answer = await progress
      .track(ask({ kind: "menu-fill-card", token, id }))
      .catch(() => ({ ok: false, reason: "failed" }) as const);
    setBusy(false);
    if (answer.ok) return;
    if (answer.reason === "locked" || answer.reason === "not-open") {
      setListing({ state: answer.reason });
    } else {
      setFailed(answer.reason === "declined" ? "declined" : "fill");
    }
  }

  async function unlock() {
    if (busy) return;
    setFailed(null);
    const answer = await ask({ kind: "menu-unlock", token }).catch(
      () => ({ ok: false, reason: "failed" }) as const,
    );
    if (answer.ok) return;
    if (answer.reason === "not-open") setListing({ state: "not-open" });
    else setFailed("unlock");
  }

  const empty = listing.state === "list" && listing.cards.length === 0;
  return (
    <>
      {listing.state === "list" && confirming && (
        <VerificationRow progress={confirming} subject="fill" />
      )}
      {empty && (
        <StateRow
          icon={CreditCard}
          title={t("extension.menu.cards.empty.title")}
          detail={t("extension.menu.cards.empty.detail")}
        />
      )}
      {listing.state === "list" && !empty && (
        <nav
          className="flex max-h-[284px] flex-col gap-1 overflow-y-auto"
          aria-label={t("extension.menu.cards.label")}
          hidden={confirming !== null}
          {...highlight.list}
        >
          <SelectionGroup id="card-menu">
            {listing.cards.map((card) => (
              <CardRow
                key={card.id}
                card={card}
                highlight={highlight}
                onChoose={() => void fill(card.id)}
              />
            ))}
          </SelectionGroup>
        </nav>
      )}
      {listing.state === "locked" && (
        <LockedRow
          detail={t("extension.menu.cards.locked.detail")}
          onUnlock={() => void unlock()}
        />
      )}
      {listing.state === "not-open" && (
        <NotOpenRow detail={t("extension.menu.cards.not-open.detail")} />
      )}
      {failed && <FailureNote>{t(failureNotes[failed])}</FailureNote>}
      <MenuFooter
        actions={listing.state === "list" && !empty ? ["fill"] : []}
      />
    </>
  );
}

const failureNotes = {
  fill: "extension.menu.cards.error",
  unlock: "extension.menu.unlock.error",
  declined: "extension.fill.declined",
} as const;

function CardRow({
  card,
  highlight,
  onChoose,
}: {
  card: CardOption;
  highlight: ReturnType<typeof useHighlight>;
  onChoose: () => void;
}) {
  const { t } = useTranslator();
  const logo = useSiteIcon(card.site, card.site !== "");
  const active = highlight.highlighted === card.id;
  return (
    <button
      type="button"
      className={cn(itemRowClass(false, "menu"), "relative w-full")}
      {...highlight.row(card.id)}
      onClick={onChoose}
    >
      <ItemRowContent
        active={active}
        avatar={
          <CardTile
            network={card.network}
            color={card.color}
            logo={logo}
            size="row"
          />
        }
        title={card.label || t("card.untitled")}
        detail={{
          text: cardLine(card.bankName, card.network, card.lastFour),
          tone: validityTone(Validity.of(card.expiresOn), new Date()),
        }}
        trailing={<RowChevron active={active} />}
        look="menu"
      />
    </button>
  );
}
