import { useTranslator } from "../../i18n/translator.tsx";
import type { Group } from "../../vault-api.ts";
import { everyGroup } from "../../workspace/sections.ts";
import {
  ResponsiveDialog,
  ResponsiveDialogAction,
  ResponsiveDialogCancel,
  ResponsiveDialogContent,
  ResponsiveDialogDescription,
  ResponsiveDialogFooter,
  ResponsiveDialogHeader,
  ResponsiveDialogTitle,
} from "../ResponsiveDialog.tsx";
import { NameDialog } from "../settings/NameDialog.tsx";

/** Shared create, rename, and delete confirmation dialogs for workspace groups. */
export function GroupDialogs({
  group,
  onGroup,
  creating,
  renaming,
  deleting,
  onCloseCreate,
  onCloseRename,
  onCloseDelete,
  onStartDelete,
  onCreate,
  onRename,
  onDelete,
  busy = false,
}: {
  group: string;
  onGroup: (group: string) => void;
  creating: boolean;
  renaming: Group | null;
  deleting: Group | null;
  onCloseCreate: () => void;
  onCloseRename: () => void;
  onCloseDelete: () => void;
  onStartDelete: (target: Group) => void;
  onCreate?: (name: string) => Promise<string | null>;
  onRename?: (id: string, name: string) => Promise<boolean>;
  onDelete?: (id: string) => void;
  busy?: boolean;
}) {
  const { t } = useTranslator();
  const groupName = {
    label: t("settings.groups.name"),
    placeholder: t("settings.groups.name.placeholder"),
    maxLength: 64,
    save: t("settings.groups.save"),
    cancel: t("settings.groups.cancel"),
  };

  return (
    <>
      <NameDialog
        open={creating}
        title={t("workspace.groups.new")}
        {...groupName}
        busy={busy}
        onSave={async (name) => {
          if (!onCreate) return false;
          const newId = await onCreate(name);
          if (newId) {
            onGroup(newId);
            return true;
          }
          return false;
        }}
        onClose={onCloseCreate}
      />

      <NameDialog
        key={renaming?.id ?? "rename"}
        open={renaming !== null}
        title={t("workspace.groups.rename")}
        {...groupName}
        initial={renaming?.name ?? ""}
        busy={busy}
        onSave={(name) =>
          renaming && onRename
            ? onRename(renaming.id, name)
            : Promise.resolve(false)
        }
        onClose={onCloseRename}
        onDelete={
          onDelete
            ? {
                label: t("settings.groups.delete"),
                run: () => {
                  const target = renaming;
                  onCloseRename();
                  if (target) onStartDelete(target);
                },
              }
            : undefined
        }
      />

      <ResponsiveDialog
        role="alertdialog"
        open={deleting !== null}
        onOpenChange={(open) => {
          if (!open) onCloseDelete();
        }}
      >
        <ResponsiveDialogContent>
          <ResponsiveDialogHeader>
            <ResponsiveDialogTitle>
              {t("settings.groups.delete.title", {
                name: deleting?.name ?? "",
              })}
            </ResponsiveDialogTitle>
            <ResponsiveDialogDescription>
              {t("settings.groups.delete.detail")}
            </ResponsiveDialogDescription>
          </ResponsiveDialogHeader>
          <ResponsiveDialogFooter>
            <ResponsiveDialogCancel>
              {t("settings.groups.cancel")}
            </ResponsiveDialogCancel>
            <ResponsiveDialogAction
              variant="destructive"
              onClick={() => {
                if (deleting && onDelete) {
                  onDelete(deleting.id);
                  if (group === deleting.id) {
                    onGroup(everyGroup);
                  }
                }
                onCloseDelete();
              }}
            >
              {t("settings.groups.delete.confirm")}
            </ResponsiveDialogAction>
          </ResponsiveDialogFooter>
        </ResponsiveDialogContent>
      </ResponsiveDialog>
    </>
  );
}
