import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import type { MessageKey } from "../../i18n/messages.ts";
import { useTranslator } from "../../i18n/translator.tsx";
import { queryKeys } from "../../query/keys.ts";
import type { ItemKindName, TrashedItem, VaultApi } from "../../vault-api.ts";
import { daysLeft, dueAt, retentionOptions } from "../../workspace/trash.ts";
import { Block, BlockIconTile, BlockRow, blockControl } from "../Block.tsx";
import { ConfirmDialog } from "../ConfirmDialog.tsx";
import { Button } from "../ui/button.tsx";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "../ui/select.tsx";
import { kindPlaces, placeOf } from "../workspace/places.ts";

/** What the trash reads from the host; its changes go through the workspace, which refreshes the lists. */
export type TrashApi = Pick<VaultApi, "listTrash">;

/** The trash's changes; each resolves whether it was written. */
export interface TrashActions {
  onRestore: (id: string) => Promise<boolean>;
  onDelete: (id: string) => Promise<boolean>;
  onEmpty: () => Promise<boolean>;
  onRetention: (days: number) => Promise<boolean>;
}

const untitled: Record<ItemKindName, MessageKey> = {
  credential: "credential.untitled",
  identity: "identity.untitled",
  card: "card.untitled",
  note: "note.untitled",
  seed: "seed.untitled",
};

/** The confirmation open: one item to delete for good, or the whole trash. */
type Deleting = TrashedItem | "all" | null;

/** TrashPanel keeps the vault's retention period and lists the deleted items to restore or delete for good. */
export function TrashPanel({
  trash,
  busy,
  onRestore,
  onDelete,
  onEmpty,
  onRetention,
}: {
  trash: TrashApi;
  busy: boolean;
} & TrashActions) {
  const { t } = useTranslator();
  const read = useQuery({
    queryKey: queryKeys.trash,
    queryFn: () => trash.listTrash(),
    meta: { failure: "settings.trash.error.read" },
  });
  const [deleting, setDeleting] = useState<Deleting>(null);
  // A shorter period that deletes items at once, waiting for confirmation.
  const [shortening, setShortening] = useState<{
    days: number;
    count: number;
  } | null>(null);
  const now = Date.now();
  const content = read.data;

  async function confirmDeleting() {
    if (!deleting) return;
    const done = await (deleting === "all" ? onEmpty() : onDelete(deleting.id));
    if (done) setDeleting(null);
  }

  if (!content) return null;
  const { items, retentionDays } = content;

  function chooseRetention(days: number) {
    const count = dueAt(
      items.map((item) => item.deletedAt),
      days,
      Date.now(),
    );
    if (count > 0) setShortening({ days, count });
    else void onRetention(days);
  }

  async function confirmShortening() {
    if (!shortening) return;
    if (await onRetention(shortening.days)) setShortening(null);
  }

  return (
    <>
      <Block className="mb-5">
        <BlockRow
          title={t("settings.trash.retention")}
          htmlFor="trash-retention"
        >
          <Select
            value={String(retentionDays)}
            onValueChange={(next) => chooseRetention(Number(next))}
            disabled={busy}
          >
            <SelectTrigger
              id="trash-retention"
              size="sm"
              className={blockControl}
            >
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectGroup>
                {retentionOptions(retentionDays).map((days) => (
                  <SelectItem key={days} value={String(days)}>
                    {t("settings.trash.days", { count: days })}
                  </SelectItem>
                ))}
              </SelectGroup>
            </SelectContent>
          </Select>
        </BlockRow>
      </Block>

      <Block>
        {items.map((item) => {
          const days = daysLeft(item.deletedAt, retentionDays, now);
          const left = days
            ? t("settings.trash.left", { count: days })
            : t("settings.trash.left.today");
          return (
            <BlockRow
              key={item.id}
              title={item.label || t(untitled[item.kind])}
              detail={
                item.detail
                  ? t("settings.trash.detail", { detail: item.detail, left })
                  : left
              }
              leading={
                <BlockIconTile icon={placeOf(kindPlaces[item.kind]).icon} />
              }
            >
              <Button
                type="button"
                variant="quiet"
                size="pill-sm"
                className="bg-tile font-normal hover:bg-field-hover"
                disabled={busy}
                onClick={() => void onRestore(item.id)}
              >
                {t("settings.trash.restore")}
              </Button>
              <Button
                type="button"
                variant="quiet"
                size="pill-sm"
                className="bg-tile font-normal text-destructive hover:bg-field-hover"
                disabled={busy}
                onClick={() => setDeleting(item)}
              >
                {t("settings.trash.delete")}
              </Button>
            </BlockRow>
          );
        })}
        {!items.length && (
          <p className="px-[13px] py-3 text-[13px] text-muted-foreground">
            {t("settings.trash.empty")}
          </p>
        )}
      </Block>
      {items.length > 0 && (
        <div className="mt-3">
          <Button
            type="button"
            variant="quiet"
            size="pill-sm"
            className="bg-tile font-normal text-destructive hover:bg-field-hover"
            disabled={busy}
            onClick={() => setDeleting("all")}
          >
            {t("settings.trash.empty-all")}
          </Button>
        </div>
      )}

      <ConfirmDialog
        open={deleting !== null}
        title={
          deleting === "all"
            ? t("settings.trash.empty-all.title", { count: items.length })
            : t("settings.trash.delete.title", {
                name:
                  deleting?.label ||
                  (deleting ? t(untitled[deleting.kind]) : ""),
              })
        }
        detail={t("settings.trash.delete.detail")}
        confirm={t(
          deleting === "all"
            ? "settings.trash.empty-all.confirm"
            : "settings.trash.delete.confirm",
        )}
        cancel={t("workspace.delete.cancel")}
        destructive
        busy={busy}
        onConfirm={() => void confirmDeleting()}
        onCancel={() => setDeleting(null)}
      />
      <ConfirmDialog
        open={shortening !== null}
        title={t("settings.trash.shorten.title", {
          count: shortening?.count ?? 0,
        })}
        detail={t("settings.trash.delete.detail")}
        confirm={t("settings.trash.shorten.confirm")}
        cancel={t("workspace.delete.cancel")}
        destructive
        busy={busy}
        onConfirm={() => void confirmShortening()}
        onCancel={() => setShortening(null)}
      />
    </>
  );
}
