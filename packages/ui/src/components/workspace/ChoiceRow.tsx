import { cn } from "cn";
import { LoaderCircle } from "lucide-react";
import { ItemRowContent, itemRowClass } from "./ItemRowContent.tsx";
import { SiteAvatar } from "./SiteAvatar.tsx";

/** ChoiceRow is a passkey or credential to choose, with a spinner while `working`. */
export function ChoiceRow({
  site,
  title,
  tags,
  detail,
  busy,
  working,
  onChoose,
}: {
  site: string;
  title: string;
  tags?: readonly string[];
  detail: string;
  busy: boolean;
  working: boolean;
  onChoose: () => void;
}) {
  return (
    <button
      type="button"
      className={cn(itemRowClass(true), "relative w-full")}
      disabled={busy}
      aria-busy={working}
      onClick={onChoose}
    >
      <ItemRowContent
        active={false}
        avatar={<SiteAvatar site={site} label={title} />}
        title={title}
        tags={tags}
        detail={{ text: detail }}
        trailing={
          working ? (
            <LoaderCircle
              className="size-4 animate-spin text-muted-foreground"
              aria-hidden="true"
            />
          ) : null
        }
      />
    </button>
  );
}
