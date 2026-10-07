import { cn } from "cn";
import { type KeyboardEvent, useEffect, useState } from "react";
import { useTranslator } from "../../i18n/translator.tsx";
import { phraseColumns } from "../../seeds/phrase.ts";
import { RevealButton, TitledBlock } from "./Fields.tsx";
import { ProgressRing } from "./ProgressRing.tsx";
import type { RevealGate } from "./RevealGate.tsx";

const revealSeconds = 30;

const wordMask = "••••••";

function useTimedReveal(seconds: number) {
  const [until, setUntil] = useState<number | null>(null);
  const [now, setNow] = useState(() => Date.now());

  useEffect(() => {
    if (until === null) return;
    const ticker = setInterval(() => setNow(Date.now()), 250);
    const end = setTimeout(() => setUntil(null), until - Date.now());
    return () => {
      clearInterval(ticker);
      clearTimeout(end);
    };
  }, [until]);

  const left =
    until === null ? 0 : Math.max(0, Math.ceil((until - now) / 1000));

  return {
    shown: until !== null,
    left,
    share: left / seconds,
    toggle() {
      if (until !== null) {
        setUntil(null);
        return;
      }
      const start = Date.now();
      setNow(start);
      setUntil(start + seconds * 1000);
    },
  };
}

function holdKey(event: KeyboardEvent<HTMLButtonElement>): boolean {
  return event.key === " " || event.key === "Enter";
}

/** PhraseGrid masks every word; holding one shows it, the reveal button shows all for a while, each once the gate passes. */
export function PhraseGrid({
  words,
  gate,
}: {
  words: readonly string[];
  gate: RevealGate;
}) {
  const { t } = useTranslator();
  const reveal = useTimedReveal(revealSeconds);
  const [held, setHeld] = useState<number | null>(null);
  const cells = words.map((word, index) => ({ word, position: index + 1 }));

  return (
    <TitledBlock title={t("seed.block.phrase")}>
      <div className="rounded-row bg-field p-[11px]">
        <div className="mb-2 flex h-8 items-center justify-end gap-2">
          {reveal.shown && (
            <ProgressRing share={reveal.share}>
              <span
                role="timer"
                aria-label={t("seed.phrase.left", { seconds: reveal.left })}
                className="text-[11px] text-muted-foreground tabular-nums"
              >
                {reveal.left}
              </span>
            </ProgressRing>
          )}
          <RevealButton
            shown={reveal.shown}
            revealLabel={t("seed.phrase.reveal")}
            concealLabel={t("seed.phrase.conceal")}
            onToggle={() =>
              reveal.shown ? reveal.toggle() : gate.after(reveal.toggle)
            }
          />
        </div>
        <ol
          className={cn(
            "grid gap-1.5",
            phraseColumns(words.length) === 4 ? "grid-cols-4" : "grid-cols-3",
          )}
        >
          {cells.map(({ word, position }) => {
            const shown = reveal.shown || held === position;
            return (
              <li key={position} className="min-w-0">
                <button
                  type="button"
                  aria-label={
                    shown
                      ? undefined
                      : t("seed.phrase.hold", { number: position })
                  }
                  title={
                    reveal.shown
                      ? undefined
                      : t("seed.phrase.hold", { number: position })
                  }
                  // A word held before the owner confirmed asks first; the next hold shows it.
                  onPointerDown={() =>
                    gate.confirmed ? setHeld(position) : gate.after(() => {})
                  }
                  onPointerUp={() => setHeld(null)}
                  onPointerLeave={() => setHeld(null)}
                  onPointerCancel={() => setHeld(null)}
                  onKeyDown={(event) => {
                    if (!holdKey(event)) return;
                    event.preventDefault();
                    if (gate.confirmed) setHeld(position);
                    else if (!event.repeat) gate.after(() => {});
                  }}
                  onKeyUp={(event) => {
                    if (holdKey(event)) setHeld(null);
                  }}
                  onBlur={() => setHeld(null)}
                  onContextMenu={(event) => event.preventDefault()}
                  className="flex h-8 w-full min-w-0 touch-none items-center gap-1.5 rounded-md bg-tile px-2 text-left outline-none select-none focus-visible:ring-[3px] focus-visible:ring-ring/50"
                >
                  <span className="w-5 shrink-0 text-right text-[11px] text-faint tabular-nums">
                    {position}
                  </span>
                  <span
                    className={cn(
                      "min-w-0 flex-1 truncate font-mono text-[13px]",
                      !shown && "tracking-[0.16em] text-muted-foreground",
                    )}
                  >
                    {shown ? word : wordMask}
                  </span>
                </button>
              </li>
            );
          })}
        </ol>
      </div>
    </TitledBlock>
  );
}
