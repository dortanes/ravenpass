import { cn } from "cn";
import { KeySquare, Settings2 } from "lucide-react";
import { type ComponentType, useState } from "react";
import { useTranslator } from "../../i18n/translator.tsx";
import {
  aboveIndicator,
  SelectionGroup,
  SelectionIndicator,
} from "../../motion/SelectionIndicator.tsx";
import { Button } from "../ui/button.tsx";
import { itemPlaces, type WorkspacePlace } from "./places.ts";

export function WorkspaceRail({
  place,
  onPlace,
}: {
  place: WorkspacePlace;
  onPlace: (place: WorkspacePlace) => void;
}) {
  const { t } = useTranslator();
  const [moved, setMoved] = useState({ to: place, from: place });
  if (moved.to !== place) setMoved({ to: place, from: moved.to });
  // The indicator travels within a cluster and grows in when it crosses to the other one.
  const appear = clusters[moved.from] !== clusters[place];

  return (
    <nav
      className="flex w-11 shrink-0 flex-col items-center gap-1.5"
      aria-label={t("workspace.rail.label")}
    >
      <SelectionGroup id="rail-items">
        {itemPlaces.map((item) => (
          <RailItem
            key={item.id}
            label={t(item.label)}
            icon={item.icon}
            active={place === item.id}
            appear={appear}
            onClick={() => onPlace(item.id)}
          />
        ))}
      </SelectionGroup>
      <SelectionGroup id="rail-utilities">
        <RailItem
          label={t("workspace.rail.generator")}
          icon={KeySquare}
          active={place === "generator"}
          appear={appear}
          onClick={() => onPlace("generator")}
          className="mt-auto"
        />
        <RailItem
          label={t("workspace.rail.settings")}
          icon={Settings2}
          active={place === "settings"}
          appear={appear}
          onClick={() => onPlace("settings")}
        />
      </SelectionGroup>
    </nav>
  );
}

const clusters: Record<WorkspacePlace, "items" | "utilities"> = {
  passwords: "items",
  identities: "items",
  cards: "items",
  notes: "items",
  seeds: "items",
  generator: "utilities",
  settings: "utilities",
};

function RailItem({
  label,
  icon: Icon,
  active,
  appear,
  onClick,
  className,
}: {
  label: string;
  icon: ComponentType<{ className?: string }>;
  active: boolean;
  appear: boolean;
  onClick: () => void;
  className?: string;
}) {
  return (
    <Button
      type="button"
      variant="ghost"
      size="icon"
      aria-current={active ? "page" : undefined}
      aria-label={label}
      title={label}
      onClick={onClick}
      className={cn(
        "relative size-9 rounded-lg border",
        active
          ? "border-transparent text-(--action-foreground) hover:bg-transparent hover:text-(--action-foreground)"
          : "bg-control text-muted-foreground hover:bg-secondary hover:text-foreground",
        className,
      )}
    >
      {active && (
        <SelectionIndicator
          appear={appear}
          className="-inset-px action-fill rounded-lg"
        />
      )}
      <Icon className={`${aboveIndicator} size-[19px]`} />
    </Button>
  );
}
