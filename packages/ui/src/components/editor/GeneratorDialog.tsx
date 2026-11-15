import { toast } from "sonner";
import { useTranslator } from "../../i18n/translator.tsx";
import type { VaultApi } from "../../vault-api.ts";
import { PasswordForm, useWordGenerator } from "../generator/PasswordForm.tsx";
import {
  ResponsiveDialog,
  ResponsiveDialogContent,
  ResponsiveDialogFooter,
  ResponsiveDialogHeader,
  ResponsiveDialogTitle,
} from "../ResponsiveDialog.tsx";
import { Button } from "../ui/button.tsx";

/** What the generator asks of the host. */
export type GeneratorHost = Pick<
  VaultApi,
  "seedWordlist" | "recordGeneratedPassword"
>;

/** GeneratorDialog shows a new password and its options; `onUse` takes the one shown, and the history keeps it. */
export function GeneratorDialog({
  open,
  host,
  onUse,
  onClose,
}: {
  open: boolean;
  host: GeneratorHost;
  onUse: (password: string) => void;
  onClose: () => void;
}) {
  const { t, failure } = useTranslator();
  const generator = useWordGenerator(host, open);

  return (
    <ResponsiveDialog
      open={open}
      onOpenChange={(next) => {
        if (!next) onClose();
      }}
    >
      <ResponsiveDialogContent className="sm:max-w-[420px]">
        <ResponsiveDialogHeader>
          <ResponsiveDialogTitle>{t("generator.title")}</ResponsiveDialogTitle>
        </ResponsiveDialogHeader>
        {generator && (
          <PasswordForm
            generator={generator}
            footer={(password, mode) => (
              <ResponsiveDialogFooter>
                <Button
                  type="button"
                  variant="quiet"
                  size="pill"
                  onClick={onClose}
                >
                  {t("generator.cancel")}
                </Button>
                <Button
                  type="button"
                  variant="raised"
                  size="pill"
                  onClick={() => {
                    onUse(password);
                    host
                      .recordGeneratedPassword(password, mode)
                      .catch((cause: unknown) =>
                        toast.error(failure(cause, "generator.error.record")),
                      );
                  }}
                >
                  {t("generator.use")}
                </Button>
              </ResponsiveDialogFooter>
            )}
          />
        )}
      </ResponsiveDialogContent>
    </ResponsiveDialog>
  );
}
