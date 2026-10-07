import { Button } from "@ravenpass/ui/components/ui/button.tsx";
import {
  ItemRowContent,
  itemRowClass,
} from "@ravenpass/ui/components/workspace/ItemRowContent.tsx";
import { useTranslator } from "@ravenpass/ui/i18n/translator.tsx";
import { SelectionGroup } from "@ravenpass/ui/motion/SelectionIndicator.tsx";
import { cn } from "cn";
import { Check, ChevronDown } from "lucide-react";
import { type ReactNode, useId, useState } from "react";
import { useHighlight } from "./highlight.ts";

/** `id` is the value chosen; `key` names the row, which an empty `id` cannot. */
export interface DestinationOption {
  readonly id: string;
  readonly key: string;
  readonly title: string;
  readonly tags?: readonly string[];
  readonly detail: string;
  readonly avatar: ReactNode;
}

/** The "where" row of a save; its toggle lists the destinations in place of `children`. */
export function DestinationPicker({
  group,
  options,
  chosen,
  disabled,
  onChoose,
  children,
}: {
  group: string;
  options: readonly DestinationOption[];
  chosen: string;
  disabled: boolean;
  onChoose: (id: string) => void;
  children: ReactNode;
}) {
  const { t } = useTranslator();
  const [choosing, setChoosing] = useState(false);
  const list = useId();
  const title = options.find(({ id }) => id === chosen)?.title ?? "";

  return (
    <>
      <div className="flex h-8 items-center gap-2 pr-0.5 pl-2.5">
        <span className="shrink-0 text-[11px] text-muted-foreground">
          {t("saving.where")}
        </span>
        {options.length > 1 ? (
          <Button
            type="button"
            variant="quiet"
            size="pill-sm"
            className="ml-auto min-w-0"
            aria-expanded={choosing}
            aria-controls={list}
            disabled={disabled}
            onClick={() => setChoosing(!choosing)}
          >
            <span className="truncate">{title}</span>
            <ChevronDown
              className={cn(
                "transition-transform duration-150 motion-reduce:transition-none",
                choosing && "rotate-180",
              )}
            />
          </Button>
        ) : (
          <span className="ml-auto truncate pr-2 text-[12px]">{title}</span>
        )}
      </div>
      {choosing ? (
        <Destinations
          id={list}
          group={group}
          options={options}
          chosen={chosen}
          onChoose={(id) => {
            onChoose(id);
            setChoosing(false);
          }}
        />
      ) : (
        children
      )}
    </>
  );
}

function Destinations({
  id,
  group,
  options,
  chosen,
  onChoose,
}: {
  id: string;
  group: string;
  options: readonly DestinationOption[];
  chosen: string;
  onChoose: (id: string) => void;
}) {
  const { t } = useTranslator();
  const highlight = useHighlight();
  return (
    <fieldset
      id={id}
      aria-label={t("saving.where")}
      className="flex max-h-[284px] flex-col gap-1 overflow-y-auto"
      {...highlight.list}
    >
      <SelectionGroup id={group}>
        {options.map((option) => (
          <button
            key={option.key}
            type="button"
            className={cn(itemRowClass(false, "menu"), "relative w-full")}
            aria-pressed={option.id === chosen}
            {...highlight.row(option.key)}
            onClick={() => onChoose(option.id)}
          >
            <ItemRowContent
              active={highlight.highlighted === option.key}
              avatar={option.avatar}
              title={option.title}
              tags={option.tags}
              detail={{ text: option.detail }}
              trailing={
                option.id === chosen ? (
                  <Check className="size-3.5 text-foreground/75" />
                ) : null
              }
              look="menu"
            />
          </button>
        ))}
      </SelectionGroup>
    </fieldset>
  );
}
