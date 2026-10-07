import { useEffect, useRef, useState, useSyncExternalStore } from "react";
import type {
  AutofillApi,
  AutofillCredential,
  SearchOpening,
} from "../../autofill/autofill-api.ts";
import {
  type FillNote,
  SearchScreen,
  type SignInNote,
} from "../../autofill/search-screen.ts";
import type { MessageKey } from "../../i18n/messages.ts";
import { useTranslator } from "../../i18n/translator.tsx";
import { ConfirmDialog } from "../ConfirmDialog.tsx";
import { SearchField } from "../SearchField.tsx";
import { ChoiceRow } from "../workspace/ChoiceRow.tsx";
import { AutofillNote, AutofillSheet } from "./AutofillSheet.tsx";

/** Why a choice did not fill, for a page's site or an app's field, and a sign-in or a code. */
function noteOf(note: FillNote, forSite: boolean, code: boolean): MessageKey {
  switch (note) {
    case "no-code":
      return "autofill.no-code";
    case "not-added":
      return forSite ? "autofill.add-site.error" : "autofill.link.error";
    case "not-found":
      return "autofill.failed.not-found";
    case "unreachable":
      return "autofill.failed.unreachable";
    case "outdated":
      return "autofill.failed.outdated";
    case "failed":
      return code ? "fill-code.error" : "fill.error";
  }
}

const signInNotes: Record<SignInNote, MessageKey> = {
  "not-found": "autofill.failed.not-found",
  unverifiable: "autofill.failed.unverifiable",
  unreachable: "autofill.failed.unreachable",
  outdated: "autofill.failed.outdated",
  failed: "autofill.failed.sign-in",
};

function titleOf(opening: SearchOpening): MessageKey {
  if (opening.code) return "autofill.search.code-title";
  if (opening.passkey) return "autofill.search.passkey-only-title";
  return opening.passkeys.length
    ? "autofill.search.passkey-title"
    : "autofill.search.title";
}

/** SearchSheet lists the site's passkeys, then the matching credentials or the whole vault. */
export function SearchSheet({
  api,
  opening,
}: {
  api: AutofillApi;
  opening: SearchOpening;
}) {
  const { t } = useTranslator();
  const [screen] = useState(() => new SearchScreen(opening, api));
  const view = useSyncExternalStore(screen.state.subscribe, screen.state.get);
  const field = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (opening.scope === "matches") return;
    const frame = requestAnimationFrame(() => {
      field.current?.focus();
      api.showKeyboard();
    });
    return () => cancelAnimationFrame(frame);
  }, [api, opening.scope]);

  const { results, asking, filling, signingIn, note, signInNote } = view;
  const { busy, passkeys } = screen;
  const heading =
    results.scope === "vault"
      ? t("autofill.search.everything")
      : t(
          screen.forSite
            ? "autofill.search.for-site"
            : "autofill.search.for-app",
        );
  const empty = view.query.trim()
    ? t("workspace.list.empty.search")
    : t(
        screen.passkeyOnly
          ? "autofill.search.no-passkeys"
          : "workspace.list.empty.all",
      );
  const label = (credential: AutofillCredential) =>
    credential.label || t("credential.untitled");

  return (
    <AutofillSheet
      title={t(titleOf(opening))}
      description={opening.site}
      busy={busy}
      onClose={() => api.cancel()}
      className="max-sm:min-h-[60dvh]"
    >
      <SearchField
        id="autofill-search"
        inputRef={field}
        value={view.query}
        onChange={(query) => void screen.type(query)}
        label={t("workspace.toolbar.search")}
        placeholder={t("workspace.toolbar.search.placeholder")}
        className="max-sm:sticky max-sm:top-0 max-sm:z-10 max-sm:bg-background"
      />
      {passkeys.length > 0 && (
        <section className="grid gap-1" aria-labelledby="autofill-passkeys">
          <h2
            id="autofill-passkeys"
            className="px-3 text-xs text-muted-foreground"
          >
            {t("autofill.search.passkeys")}
          </h2>
          {passkeys.map((passkey) => (
            <ChoiceRow
              key={passkey.key}
              site={passkey.site}
              title={passkey.label || t("credential.untitled")}
              detail={passkey.account || t("credential.login.empty")}
              busy={busy}
              working={signingIn === passkey.key}
              onChoose={() => void screen.signIn(passkey)}
            />
          ))}
        </section>
      )}
      {screen.passkeyOnly && !passkeys.length && (
        <p className="px-3 py-2 text-[13px] text-muted-foreground">{empty}</p>
      )}
      {!screen.passkeyOnly && (
        <section className="grid gap-1" aria-labelledby="autofill-results">
          <h2
            id="autofill-results"
            className="px-3 text-xs text-muted-foreground"
          >
            {heading}
          </h2>
          {results.credentials.map((credential) => (
            <ChoiceRow
              key={credential.id}
              site={credential.site}
              title={label(credential)}
              tags={credential.tags}
              detail={detailOf(credential, t("credential.login.empty"))}
              busy={busy}
              working={filling === credential.id}
              onChoose={() => screen.choose(credential)}
            />
          ))}
          {!results.credentials.length && (
            <p className="px-3 py-2 text-[13px] text-muted-foreground">
              {empty}
            </p>
          )}
        </section>
      )}
      {view.searchFailed && (
        <AutofillNote>{t("autofill.search.error")}</AutofillNote>
      )}
      {note && (
        <AutofillNote>
          {t(noteOf(note, screen.forSite, opening.code))}
        </AutofillNote>
      )}
      {signInNote && <AutofillNote>{t(signInNotes[signInNote])}</AutofillNote>}
      <ConfirmDialog
        open={asking !== null}
        title={
          asking?.addition === "add-site"
            ? t("autofill.add-site.title", {
                site: opening.site,
                label: label(asking.credential),
              })
            : t("autofill.link.title")
        }
        detail={
          asking?.addition === "add-site"
            ? t("autofill.add-site.detail", { site: opening.site })
            : t("autofill.link.detail", {
                label: asking ? label(asking.credential) : "",
              })
        }
        confirm={t(
          asking?.addition === "add-site"
            ? "autofill.add-site.confirm"
            : "autofill.link.confirm",
        )}
        cancel={t("autofill.cancel")}
        busy={busy}
        onConfirm={() => screen.confirm()}
        onCancel={() => screen.decline()}
      />
    </AutofillSheet>
  );
}

/** A row's second line names the credential's site when it does not match yet. */
function detailOf(credential: AutofillCredential, noAccount: string): string {
  const account = credential.account || noAccount;
  return credential.matches || !credential.site
    ? account
    : `${account} · ${credential.site}`;
}
