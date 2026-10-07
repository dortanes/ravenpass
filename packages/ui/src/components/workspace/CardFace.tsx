import { cn } from "cn";
import { CreditCard, Nfc } from "lucide-react";
import type { CSSProperties } from "react";
import { PaymentIcon } from "react-svg-credit-card-payment-icons";
import {
  cardColor,
  concealedDigits,
  expiryDisplay,
  logoInColour,
  maskedGroups,
  networkLogo,
} from "../../cards/card.ts";
import { networkName } from "../../cards/networks.ts";
import { useTranslator } from "../../i18n/translator.tsx";
import { photoSource } from "../../identities/photo.ts";
import type { Card, CardNetwork } from "../../vault-api.ts";
import type { SiteIconState } from "../../workspace/site-icons.ts";

const grain =
  "url(\"data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='160' height='160'%3E%3Cfilter id='n'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.9' numOctaves='2' stitchTiles='stitch'/%3E%3CfeColorMatrix values='0 0 0 0 1 0 0 0 0 1 0 0 0 0 1 0 0 0 0.06 0'/%3E%3C/filter%3E%3Crect width='100%25' height='100%25' filter='url(%23n)'/%3E%3C/svg%3E\")";

function surface(color: string, network: CardNetwork): CSSProperties {
  const light = cardColor(color, network);
  return {
    color: "#fff",
    backgroundColor: "#0e0e10",
    backgroundImage: [
      grain,
      `radial-gradient(85% 115% at 100% 0%, color-mix(in oklab, ${light} 88%, transparent), color-mix(in oklab, ${light} 30%, transparent) 38%, transparent 68%)`,
      `radial-gradient(70% 90% at 0% 100%, color-mix(in oklab, ${light} 20%, transparent), transparent 70%)`,
      "linear-gradient(160deg, #1f1f22, #0b0b0d)",
    ].join(", "),
  };
}

function logoOf(logo: SiteIconState | undefined): string | null {
  return logo?.kind === "icon" ? photoSource(logo.image) : null;
}

const tileSizes = {
  compact: "h-4 w-6 rounded-[3px] text-[6px]",
  row: "h-[26px] w-10 rounded-[5px] text-[8px]",
  header: "h-[38px] w-[60px] rounded-[8px] text-[11px]",
};

const tileLogoSizes = {
  compact: "size-2.5",
  row: "size-4",
  header: "size-6",
};

const tileNetworkSizes = {
  compact: "h-2.5 w-auto",
  row: "h-4 w-auto",
  header: "h-6 w-auto",
};

const edge =
  "shadow-[inset_0_0_0_1px_rgb(255_255_255/0.08),inset_0_1px_0_rgb(255_255_255/0.14)]";

/** CardTile is a card's avatar: the bank's logo when known, else the network's mark. */
export function CardTile({
  network,
  color,
  logo,
  size,
  selected = false,
}: {
  network: CardNetwork;
  color: string;
  logo?: SiteIconState;
  size: keyof typeof tileSizes;
  selected?: boolean;
}) {
  const source = logoOf(logo);

  return (
    <span
      className={cn(
        "relative flex shrink-0 items-center justify-center overflow-hidden font-semibold tracking-wide",
        edge,
        tileSizes[size],
        selected && "outline-2 outline-offset-1 outline-foreground",
      )}
      style={surface(color, network)}
      aria-hidden="true"
    >
      {source ? (
        <img
          src={source}
          alt=""
          draggable={false}
          className={cn(tileLogoSizes[size], "rounded-[22%] object-contain")}
        />
      ) : networkLogo(network) ? (
        <NetworkMark network={network} className={tileNetworkSizes[size]} />
      ) : (
        <CreditCard className={tileLogoSizes[size]} />
      )}
    </span>
  );
}

/** NetworkMark shows the network's logo, or its name when no logo is drawn. */
function NetworkMark({
  network,
  className,
}: {
  network: CardNetwork;
  className: string;
}) {
  const logo = networkLogo(network);
  if (logo) {
    return (
      <PaymentIcon
        type={logo}
        format="logo"
        className={cn(
          className,
          !logoInColour(network) && "brightness-0 invert",
        )}
        aria-hidden="true"
      />
    );
  }
  const name = networkName(network);
  return name ? (
    <span className="text-[13px] font-semibold tracking-wide">{name}</span>
  ) : null;
}

function ChipMark() {
  return (
    <svg viewBox="0 0 38 28" className="h-[26px] w-[35px]" aria-hidden="true">
      <defs>
        <linearGradient id="card-chip" x1="0" y1="0" x2="1" y2="1">
          <stop offset="0" stopColor="#f6e7b0" />
          <stop offset="0.55" stopColor="#d9bb72" />
          <stop offset="1" stopColor="#b99447" />
        </linearGradient>
      </defs>
      <rect width="38" height="28" rx="5" fill="url(#card-chip)" />
      <path
        d="M0 9.5h11.5v9H0M38 9.5H26.5v9H38M11.5 0v28M26.5 0v28M11.5 14h15"
        fill="none"
        stroke="#8d6d2c"
        strokeOpacity="0.45"
        strokeWidth="0.8"
      />
    </svg>
  );
}

const face = `absolute inset-0 flex flex-col overflow-hidden rounded-[18px] backface-hidden ${edge} shadow-[0_24px_48px_-20px_rgb(0_0_0/0.9),0_2px_6px_rgb(0_0_0/0.3)]`;

const caption = "text-[9px] tracking-[0.08em] uppercase text-white/50";

/** CardFace draws an open card; one with a security code turns over to show it. */
export function CardFace({
  card,
  logo,
  turned,
  onTurn,
}: {
  card: Card;
  logo?: SiteIconState;
  turned: boolean;
  onTurn: () => void;
}) {
  const { t } = useTranslator();
  const source = logoOf(logo);
  const painted = surface(card.color, card.network);
  const expiry = expiryDisplay(card.expiry);
  const bank = card.bankName || card.label;

  const sides = (
    <span
      className={cn(
        "relative block aspect-[1.586] w-full transition-transform duration-500 ease-[cubic-bezier(0.3,0.7,0.2,1)] transform-3d motion-reduce:transition-none",
        turned && "rotate-y-180",
      )}
    >
      <span
        className={`${face} px-6 pt-5 pb-[18px]`}
        style={painted}
        aria-hidden={turned}
      >
        <span className="flex min-w-0 items-center gap-2.5">
          {source && (
            <img
              src={source}
              alt=""
              draggable={false}
              className="size-6 shrink-0 rounded-[6px] object-contain"
            />
          )}
          <span className="min-w-0 flex-1 truncate text-left text-[14px] font-semibold tracking-[-0.01em]">
            {bank}
          </span>
          <Nfc className="size-5 shrink-0 text-white/80" aria-hidden="true" />
        </span>
        <span className="mt-[18px] flex">
          <ChipMark />
        </span>
        <span className="mt-auto flex gap-[0.6em] text-left text-[19px] font-medium tracking-[0.06em] tabular-nums [text-shadow:0_1px_2px_rgb(0_0_0/0.35)]">
          {maskedGroups(card.number, card.network).map((group, index) => (
            // biome-ignore lint/suspicious/noArrayIndexKey: a group is placed by its position in the number.
            <span key={index}>{group}</span>
          ))}
        </span>
        <span className="mt-3.5 flex items-end gap-7">
          {card.holder && (
            <span className="flex min-w-0 flex-col items-start gap-0.5 text-left">
              <span className={caption}>{t("card.field.holder")}</span>
              <span className="w-full truncate text-[12px] font-medium tracking-[0.02em]">
                {card.holder}
              </span>
            </span>
          )}
          {expiry && (
            <span className="flex shrink-0 flex-col items-start gap-0.5">
              <span className={caption}>{t("card.face.expiry")}</span>
              <span className="text-[12px] font-medium tabular-nums">
                {expiry}
              </span>
            </span>
          )}
          <span className="ml-auto flex shrink-0">
            <NetworkMark network={card.network} className="h-7 w-auto" />
          </span>
        </span>
      </span>
      <span
        className={`${face} rotate-y-180`}
        style={painted}
        aria-hidden={!turned}
      >
        <span className="mt-5 h-10 w-full bg-[linear-gradient(180deg,#141414,#050505)]" />
        <span className="mx-6 mt-5 flex items-center gap-3">
          <span className="h-8 flex-1 rounded-[5px] bg-[repeating-linear-gradient(-45deg,rgb(255_255_255/0.85)_0_5px,rgb(255_255_255/0.72)_5px_10px)]" />
          <span className="flex flex-col items-start gap-0.5">
            <span className={caption}>{t("card.field.security-code")}</span>
            <span className="rounded-[6px] bg-white px-2.5 py-0.5 text-[15px] font-medium tracking-[0.1em] text-neutral-900 tabular-nums">
              {turned ? card.securityCode : concealedDigits(card.securityCode)}
            </span>
          </span>
        </span>
        <span className="mx-6 mt-auto mb-[18px] flex items-end justify-between gap-3">
          <span className="min-w-0 truncate text-[12px] font-medium text-white/70">
            {bank}
          </span>
          <NetworkMark network={card.network} className="h-7 w-auto" />
        </span>
      </span>
    </span>
  );

  if (!card.securityCode) {
    return <div className="w-full max-w-[360px] shrink-0">{sides}</div>;
  }

  return (
    <button
      type="button"
      onClick={onTurn}
      aria-pressed={turned}
      aria-label={t(turned ? "card.face.front" : "card.face.back")}
      title={t(turned ? "card.face.front" : "card.face.back")}
      className="w-full max-w-[360px] shrink-0 rounded-[18px] perspective-distant outline-none transition-transform duration-300 hover:-translate-y-0.5 focus-visible:ring-[3px] focus-visible:ring-ring/50 motion-reduce:transition-none motion-reduce:hover:translate-y-0"
    >
      {sides}
    </button>
  );
}
