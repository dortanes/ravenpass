import { cn } from "cn";
import {
  ClipboardPaste,
  FolderOpen,
  Globe,
  LoaderCircle,
  QrCode,
  ShieldAlert,
  WandSparkles,
} from "lucide-react";
import { useReducedMotion } from "motion/react";
import { useState } from "react";
import { appKey } from "../credentials/apps.ts";
import {
  type CredentialDraft,
  emptyCredential,
  readyToSave,
} from "../credentials/credential.ts";
import { SiteNaming } from "../credentials/site-naming.ts";
import { useCapabilities } from "../host/capabilities.tsx";
import type { MessageKey } from "../i18n/messages.ts";
import { useTranslator } from "../i18n/translator.tsx";
import { typeIn } from "../motion/type-in.ts";
import { useBreachCount } from "../query/breaches.ts";
import type {
  Credential,
  CredentialInput,
  CredentialLimits,
  Group,
  LinkedApp,
  VaultApi,
} from "../vault-api.ts";
import { hintSeen, markHintSeen } from "../workspace/hints.ts";
import {
  bareField,
  EditorHeader,
  EditorHint,
  EditorRow,
  GroupField,
  NotesField,
  RemoveButton,
  roomFor,
  TagField,
  type Tagging,
  toggled,
  useRemaining,
  useRows,
} from "./editor/EditorFields.tsx";
import {
  GeneratorDialog,
  type GeneratorHost,
} from "./editor/GeneratorDialog.tsx";
import { Button } from "./ui/button.tsx";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "./ui/dropdown-menu.tsx";
import { Input } from "./ui/input.tsx";
import { ScrollArea } from "./ui/scroll-area.tsx";
import {
  RevealButton,
  RowButton,
  rowButtonClass,
} from "./workspace/Fields.tsx";
import { LinkedApps } from "./workspace/LinkedApps.tsx";
import { Passkeys } from "./workspace/Passkeys.tsx";

const formID = "credential-editor";

/** What the editor asks of the host besides saving. */
export type CredentialEditorHost = Pick<
  VaultApi,
  "lookupSite" | "readCodeSetup" | "breachChecks" | "checkPassword"
> &
  GeneratorHost;

export function CredentialEditor({
  initial,
  initialGroups,
  tagging,
  groups,
  limits,
  busy,
  host,
  onSave,
  onCancel,
  onFailure,
}: {
  initial?: Credential;
  /** The groups the credential starts in, which for a new one is the chosen default. */
  initialGroups: string[];
  tagging: Tagging;
  groups: Group[];
  limits: CredentialLimits | null;
  busy: boolean;
  host: CredentialEditorHost;
  onSave: (draft: CredentialDraft, groups: string[]) => void;
  onCancel: () => void;
  onFailure: (cause: unknown, message: MessageKey) => void;
}) {
  const { t } = useTranslator();
  const { qrCodes } = useCapabilities();
  const counter = useRemaining();
  const [input, setInput] = useState<CredentialInput>(
    initial ?? emptyCredential,
  );
  const websites = useRows(initial?.websites.length ? initial.websites : [""]);
  const [membership, setMembership] = useState<string[]>(initialGroups);
  const [tags, setTags] = useState<string[]>(initial?.tags ?? []);
  const [showPassword, setShowPassword] = useState(false);
  const [generating, setGenerating] = useState(false);
  const breached = useBreachCount(host, input.password);
  const [showTotp, setShowTotp] = useState(false);
  const [removedPasskeys, setRemovedPasskeys] = useState<string[]>([]);
  const passkeys = (initial?.passkeys ?? []).filter(
    (passkey) => !removedPasskeys.includes(passkey.id),
  );
  const [apps, setApps] = useState<LinkedApp[]>(initial?.apps ?? []);
  const [siteNaming] = useState(
    () => new SiteNaming(initial?.websites[0] ?? ""),
  );
  const [lookingUp, setLookingUp] = useState(false);
  const reduceMotion = useReducedMotion() ?? false;
  // Shown once, on the first new password.
  const [hintOpen, setHintOpen] = useState(
    () => !initial && !hintSeen("credential-site"),
  );

  function update(
    field: Exclude<keyof CredentialInput, "websites" | "apps">,
    value: string,
  ) {
    setInput((current) => ({ ...current, [field]: value }));
  }

  function dismissHint() {
    if (!hintOpen) return;
    setHintOpen(false);
    markHintSeen("credential-site");
  }

  /** Cuts a typed or pasted first website to its domain and names a password still unnamed after the site. */
  async function lookUpSite(key: number, website: string) {
    const site = siteNaming.begin(website);
    if (!site) return;
    dismissHint();
    const naming = siteNaming.names(input.label);
    setLookingUp(naming);
    try {
      const found = await host.lookupSite(site, naming);
      if (found.website && found.website !== site) {
        siteNaming.cut(found.website);
        websites.update(key, (current) =>
          current.trim() === site ? found.website : current,
        );
      }
      if (!naming || !found.name) return;
      siteNaming.filled(found.name);
      await typeIn(found.name, (shown) => update("label", shown), reduceMotion);
    } catch (cause) {
      onFailure(cause, "credential.error.lookup");
    } finally {
      setLookingUp(false);
    }
  }

  async function readCodeSetup(source: "file" | "clipboard") {
    try {
      const setup = await host.readCodeSetup(source);
      if (setup) update("totp", setup);
    } catch (cause) {
      onFailure(cause, "credential.error.qr");
    }
  }

  function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    onSave(
      {
        input: readyToSave({ ...input, websites: websites.values, apps, tags }),
        removedPasskeys,
      },
      membership,
    );
  }

  const name = input.label.trim();
  const holdsPasskeys = passkeys.length > 0;

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-[11px]">
      <EditorHeader
        title={t(
          initial
            ? "credential.editor.edit.title"
            : "credential.editor.new.title",
        )}
        name={name}
        form={formID}
        submit={t(
          initial ? "workspace.editor.update" : "credential.editor.create",
        )}
        canSubmit={Boolean(
          name && (input.password || input.totp.trim() || holdsPasskeys),
        )}
        busy={busy}
        onCancel={onCancel}
      />

      <ScrollArea className="min-h-0 flex-1">
        <form
          id={formID}
          onSubmit={submit}
          className="flex flex-1 flex-col gap-2"
        >
          <div className="shrink-0 overflow-hidden rounded-row bg-field">
            <EditorRow
              label={t("credential.field.label")}
              htmlFor="credential-name"
              counter={counter(input.label, limits?.label, 20)}
            >
              <Input
                id="credential-name"
                className={bareField}
                value={input.label}
                onChange={(event) => update("label", event.target.value)}
                placeholder="GitHub"
                maxLength={limits?.label}
                disabled={busy || lookingUp}
                required
              />
            </EditorRow>
            {websites.rows.map((row, index) => {
              const field = (
                <EditorRow
                  key={row.key}
                  label={t("credential.field.website")}
                  htmlFor={`credential-website-${row.key}`}
                  counter={counter(row.value, limits?.website, 20)}
                >
                  <Input
                    id={`credential-website-${row.key}`}
                    className={bareField}
                    value={row.value}
                    onChange={(event) => {
                      websites.change(row.key, event.target.value);
                      if (index === 0) dismissHint();
                    }}
                    onBlur={
                      index === 0
                        ? (event) =>
                            void lookUpSite(row.key, event.target.value)
                        : undefined
                    }
                    placeholder={index === 0 ? "github.com" : undefined}
                    maxLength={limits?.website}
                    autoCapitalize="none"
                    spellCheck={false}
                    disabled={busy}
                  />
                  {index === 0 && lookingUp && (
                    <span
                      className="flex shrink-0 items-center gap-1.5 text-[11px] text-muted-foreground"
                      role="status"
                    >
                      <LoaderCircle
                        className="size-3.5 animate-spin motion-reduce:animate-none"
                        aria-hidden="true"
                      />
                      {t("credential.site.looking-up")}
                    </span>
                  )}
                  {websites.rows.length > 1 && (
                    <RemoveButton
                      label={t("credential.remove.website")}
                      busy={busy}
                      onRemove={() => websites.remove(row.key)}
                    />
                  )}
                </EditorRow>
              );
              return index === 0 ? (
                <EditorHint
                  key={row.key}
                  open={hintOpen}
                  text={t("credential.hint.site")}
                  onDismiss={dismissHint}
                >
                  <div className="border-b">{field}</div>
                </EditorHint>
              ) : (
                field
              );
            })}
            <EditorRow
              label={t("credential.field.login")}
              htmlFor="credential-username"
              counter={counter(input.login, limits?.login, 20)}
            >
              <Input
                id="credential-username"
                className={bareField}
                value={input.login}
                onChange={(event) => update("login", event.target.value)}
                maxLength={limits?.login}
                autoCapitalize="none"
                spellCheck={false}
                disabled={busy}
              />
            </EditorRow>
            <EditorRow
              label={t("credential.field.email")}
              htmlFor="credential-email"
              counter={counter(input.email, limits?.email, 20)}
            >
              <Input
                id="credential-email"
                type="email"
                className={bareField}
                value={input.email}
                onChange={(event) => update("email", event.target.value)}
                maxLength={limits?.email}
                autoCapitalize="none"
                spellCheck={false}
                disabled={busy}
              />
            </EditorRow>
            <EditorRow
              label={t("credential.field.password")}
              htmlFor="credential-password"
            >
              <Input
                id="credential-password"
                className={cn(bareField, "font-mono")}
                type={showPassword ? "text" : "password"}
                value={input.password}
                onChange={(event) => update("password", event.target.value)}
                maxLength={limits?.password}
                autoComplete="new-password"
                spellCheck={false}
                disabled={busy}
                required={!holdsPasskeys}
              />
              <RevealButton
                shown={showPassword}
                revealLabel={t("credential.password.reveal")}
                concealLabel={t("credential.password.conceal")}
                busy={busy}
                onToggle={() => setShowPassword((visible) => !visible)}
              />
              <RowButton
                label={t("generator.open")}
                icon={WandSparkles}
                busy={busy}
                onClick={() => setGenerating(true)}
              />
            </EditorRow>
            {breached > 0 && (
              <p
                className="flex items-center gap-2 border-b px-[13px] py-2 text-xs text-warning"
                role="status"
              >
                <ShieldAlert className="size-3.5 shrink-0" aria-hidden="true" />
                {t("breach.editor", { count: breached })}
              </p>
            )}
            <TagField
              tags={tags}
              tagging={tagging}
              busy={busy}
              onChange={setTags}
            />
            {groups.length > 0 && (
              <GroupField
                groups={groups}
                membership={membership}
                busy={busy}
                onToggle={(id) =>
                  setMembership((current) => toggled(current, id))
                }
              />
            )}
            <EditorRow
              label={t("credential.field.totp")}
              htmlFor="credential-totp"
            >
              <Input
                id="credential-totp"
                className={cn(bareField, "font-mono text-xs")}
                type={showTotp ? "text" : "password"}
                value={input.totp}
                onChange={(event) => update("totp", event.target.value)}
                placeholder={t("credential.totp.placeholder")}
                maxLength={limits?.totp}
                autoCapitalize="none"
                autoComplete="off"
                spellCheck={false}
                disabled={busy}
              />
              <RevealButton
                shown={showTotp}
                revealLabel={t("credential.totp.reveal")}
                concealLabel={t("credential.totp.conceal")}
                busy={busy}
                onToggle={() => setShowTotp((visible) => !visible)}
              />
              {qrCodes && (
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon-sm"
                      className={rowButtonClass}
                      aria-label={t("credential.totp.qr")}
                      title={t("credential.totp.qr")}
                      disabled={busy}
                    >
                      <QrCode className="size-4" />
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end">
                    <DropdownMenuItem
                      onSelect={() => void readCodeSetup("file")}
                    >
                      <FolderOpen />
                      {t("credential.totp.qr.file")}
                    </DropdownMenuItem>
                    <DropdownMenuItem
                      onSelect={() => void readCodeSetup("clipboard")}
                    >
                      <ClipboardPaste />
                      {t("credential.totp.qr.clipboard")}
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              )}
            </EditorRow>
          </div>

          <fieldset className="flex shrink-0 flex-wrap items-center gap-1.5 px-0.5">
            <legend className="sr-only">{t("credential.editor.add")}</legend>
            {roomFor(websites.rows.length, limits?.websites) && (
              <Button
                type="button"
                variant="quiet"
                size="pill-sm"
                disabled={busy}
                onClick={() => websites.add("")}
              >
                <Globe data-icon="inline-start" />
                {t("credential.field.website")}
              </Button>
            )}
          </fieldset>

          {holdsPasskeys && (
            <Passkeys
              passkeys={passkeys}
              action={(passkey) => (
                <RemoveButton
                  label={t("credential.remove.passkey")}
                  busy={busy}
                  onRemove={() =>
                    setRemovedPasskeys((current) => [...current, passkey.id])
                  }
                />
              )}
            />
          )}

          {apps.length > 0 && (
            <LinkedApps
              apps={apps}
              action={(app) => (
                <RemoveButton
                  label={t("credential.remove.app")}
                  busy={busy}
                  onRemove={() =>
                    setApps((current) =>
                      current.filter((kept) => appKey(kept) !== appKey(app)),
                    )
                  }
                />
              )}
            />
          )}

          <NotesField
            id="credential-notes"
            value={input.notes}
            limit={limits?.notes}
            busy={busy}
            onChange={(value) => update("notes", value)}
          />

          <p className="shrink-0 px-1 text-[11px] text-faint">
            {t(
              holdsPasskeys
                ? "credential.editor.requirement.passkeys"
                : "credential.editor.requirement",
            )}
          </p>
        </form>
      </ScrollArea>
      <GeneratorDialog
        open={generating}
        host={host}
        onUse={(password) => {
          update("password", password);
          setShowPassword(true);
          setGenerating(false);
        }}
        onClose={() => setGenerating(false)}
      />
    </div>
  );
}
