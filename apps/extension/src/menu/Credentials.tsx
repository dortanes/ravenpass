import { Avatar } from "@ravenpass/ui/components/workspace/Avatar.tsx";
import {
  ItemRowContent,
  itemRowClass,
} from "@ravenpass/ui/components/workspace/ItemRowContent.tsx";
import {
  CodeDigits,
  CodeTimer,
  usePeriodLeft,
} from "@ravenpass/ui/components/workspace/OneTimeCode.tsx";
import { SiteAvatar } from "@ravenpass/ui/components/workspace/SiteAvatar.tsx";
import { maskedCode } from "@ravenpass/ui/credentials/one-time-code.ts";
import { useTranslator } from "@ravenpass/ui/i18n/translator.tsx";
import { SelectionGroup } from "@ravenpass/ui/motion/SelectionIndicator.tsx";
import type { OneTimeCode } from "@ravenpass/ui/vault-api.ts";
import { cn } from "cn";
import { KeyRound, TimerReset, UserRoundKey } from "lucide-react";
import {
  type ReactNode,
  useCallback,
  useEffect,
  useRef,
  useState,
  useSyncExternalStore,
} from "react";
import type {
  CodeSuggestion,
  PasskeyChoice,
  PasskeyOption,
  ShareProgress,
  Suggestion,
} from "../link/client.ts";
import {
  type Answers,
  ask,
  type CredentialMenuContent,
  isFrameMessage,
  type Listed,
  listsNothing,
  type MenuFailure,
  type PasskeyFailure,
} from "../messages.ts";
import { sendIgnoringClosedPort } from "../messaging/send.ts";
import { useClickGate } from "./ClickGateProvider.tsx";
import { ConfirmFill } from "./ConfirmFill.tsx";
import { CodeMenu, type CodeMenuView } from "./code-menu.ts";
import { FillConfirmation } from "./fill-confirmation.ts";
import { FillProgress } from "./fill-progress.ts";
import { useHighlight } from "./highlight.ts";
import { type PasskeyNote, passkeyNote } from "./passkey-prompt.ts";
import {
  FailureNote,
  LockedRow,
  NotOpenRow,
  RowChevron,
  StateRow,
  VerificationRow,
} from "./Rows.tsx";
import { watchVaultState } from "./vault-state.ts";

export type Highlight = ReturnType<typeof useHighlight>;

export type MenuAction = "fill" | "unlock";

/** A fill the owner declined on the Mac, or one nothing there could confirm. */
type Unconfirmed = "declined" | "unverifiable";

export interface Credentials {
  readonly content: CredentialMenuContent;
  /** What last failed for a reason other than Ravenpass locked or closed. */
  readonly failed: MenuAction | Unconfirmed | null;
  /** Why the last passkey sign-in failed, other than Ravenpass locked or closed. */
  readonly passkeyFailed: PasskeyNote | null;
  /** A chosen suggestion awaiting the person's confirmation. */
  readonly pending: Listed<Suggestion> | null;
  /** What Ravenpass asks of the person while a fill waits for the owner. */
  readonly confirming: ShareProgress | null;
  /** Runs `proceed` at once for a strong suggestion, otherwise once the person confirms it. */
  readonly choose: (
    credential: Listed<Suggestion>,
    proceed: () => void,
  ) => void;
  /** `remember` asks to remember the page for an item saved for another site. */
  readonly confirm: (remember: boolean) => void;
  readonly cancel: () => void;
  readonly fill: (id: string) => void;
  /** Rejects when Ravenpass shows no code; a code field's menu then keeps the row masked. */
  readonly reveal: (id: string) => Promise<OneTimeCode>;
  readonly signIn: (passkey: PasskeyChoice) => void;
  readonly unlock: () => void;
}

/** A request that finds Ravenpass locked or closed switches the list to that state; any other failure is noted. A
 * locked list asks again once Ravenpass leaves the locked state. */
export function useCredentials(
  token: string,
  initial: CredentialMenuContent,
): Credentials {
  const [content, setContent] = useState(initial);
  const [failed, setFailed] = useState<MenuAction | Unconfirmed | null>(null);
  const [passkeyFailed, setPasskeyFailed] = useState<PasskeyNote | null>(null);
  const [busy, setBusy] = useState(false);
  const [confirmation] = useState(() => new FillConfirmation());
  const pending = useSyncExternalStore(
    confirmation.subscribe,
    confirmation.view,
  );
  const [progress] = useState(() => new FillProgress());
  const confirming = useSyncExternalStore(progress.subscribe, progress.view);

  useEffect(() => {
    const onMessage = (
      message: unknown,
      sender: chrome.runtime.MessageSender,
    ): undefined => {
      if (
        sender.id === chrome.runtime.id &&
        isFrameMessage(message, token) &&
        message.kind === "fill-progress"
      ) {
        progress.report(message.progress);
      }
    };
    chrome.runtime.onMessage.addListener(onMessage);
    return () => chrome.runtime.onMessage.removeListener(onMessage);
  }, [progress, token]);

  const locked = content.state === "locked";
  useEffect(() => {
    if (!locked) return;
    return watchVaultState((state) => {
      if (state === "locked") return;
      void sendIgnoringClosedPort(ask({ kind: "menu-review", token })).then(
        (answer) => {
          if (answer?.listing) setContent(answer.listing);
        },
      );
    });
  }, [locked, token]);

  const unavailable = useCallback((reason: Exclude<MenuFailure, "failed">) => {
    setContent((current) => ({ state: reason, purpose: current.purpose }));
  }, []);

  const fillFailed = (reason: Exclude<PasskeyFailure, "locked" | "not-open">) =>
    setFailed(
      reason === "declined" || reason === "unverifiable" ? reason : "fill",
    );

  async function act(
    request: Promise<
      Answers["menu-fill" | "menu-fill-code" | "menu-unlock" | "passkey-sign"]
    >,
    noted: (reason: Exclude<PasskeyFailure, "locked" | "not-open">) => void,
  ) {
    setBusy(true);
    setFailed(null);
    setPasskeyFailed(null);
    const answer = await request.catch(
      () => ({ ok: false, reason: "failed" }) as const,
    );
    setBusy(false);
    if (answer.ok) return;
    if (answer.reason === "locked" || answer.reason === "not-open") {
      unavailable(answer.reason);
    } else {
      noted(answer.reason);
    }
  }

  return {
    content,
    failed,
    passkeyFailed,
    pending,
    confirming,
    choose: (credential, proceed) => confirmation.choose(credential, proceed),
    confirm: (remember) => confirmation.confirm(remember),
    cancel: () => confirmation.cancel(),
    fill(id) {
      if (busy || content.state !== "list") return;
      const confirmed = confirmation.confirmed(id);
      const remember = confirmation.remembers(id);
      void act(
        progress.track(
          content.purpose === "code"
            ? ask({ kind: "menu-fill-code", token, id, confirmed, remember })
            : ask({ kind: "menu-fill", token, id, confirmed, remember }),
        ),
        fillFailed,
      );
    },
    async reveal(id) {
      const answer = await progress.track(
        ask({ kind: "menu-code", token, id }),
      );
      if (answer.ok) return answer.code;
      if (answer.reason === "locked" || answer.reason === "not-open") {
        unavailable(answer.reason);
      } else if (answer.reason !== "failed") {
        setFailed(answer.reason);
      }
      throw new Error(`Ravenpass could not show the code (${answer.reason}).`);
    },
    signIn(passkey) {
      if (busy || content.state !== "list") return;
      void act(ask({ kind: "passkey-sign", token, ...passkey }), (reason) =>
        setPasskeyFailed(reason === "excluded" ? "failed" : reason),
      );
    },
    unlock() {
      if (!busy) {
        void act(ask({ kind: "menu-unlock", token }), () =>
          setFailed("unlock"),
        );
      }
    },
  };
}

/** A chosen suggestion that is not strong asks for confirmation in place of the list. */
export function CredentialList({
  token,
  site,
  host,
  credentials,
}: {
  token: string;
  site: string;
  host: string;
  credentials: Credentials;
}) {
  const { t } = useTranslator();
  const {
    content,
    failed,
    passkeyFailed,
    pending,
    confirming,
    choose,
    fill,
    reveal,
    signIn,
    unlock,
  } = credentials;
  const highlight = useHighlight();
  const empty = listsNothing(content);
  return (
    <>
      {content.state === "list" && confirming && (
        <VerificationRow progress={confirming} subject="fill" />
      )}
      {content.state === "list" && empty && (
        <StateRow
          icon={content.purpose === "code" ? TimerReset : KeyRound}
          title={t(
            content.purpose === "code"
              ? "extension.menu.codes.empty.title"
              : "extension.menu.empty.title",
          )}
          detail={t(
            content.purpose === "code"
              ? "extension.menu.codes.empty.detail"
              : "extension.menu.empty.detail",
          )}
        />
      )}
      {content.state === "list" && !empty && (
        <nav
          className="flex max-h-[284px] flex-col gap-1 overflow-y-auto"
          aria-label={t(
            content.purpose === "code"
              ? "extension.menu.codes.label"
              : "extension.menu.label",
            { site },
          )}
          hidden={pending !== null || confirming !== null}
          {...highlight.list}
        >
          <SelectionGroup id="fill-menu">
            {content.purpose === "code" ? (
              <CodeRows
                token={token}
                credentials={content.credentials}
                highlight={highlight}
                onChoose={choose}
                onFill={fill}
                onReveal={reveal}
              />
            ) : (
              <>
                {content.passkeys.map((passkey) => (
                  <PasskeyRow
                    key={passkey.credentialId}
                    passkey={passkey}
                    avatar={<PasskeyAvatar />}
                    highlight={highlight}
                    onChoose={() => signIn(passkey)}
                  />
                ))}
                {content.credentials.map((credential) => (
                  <CredentialRow
                    key={credential.id}
                    credential={credential}
                    highlight={highlight}
                    onChoose={() =>
                      choose(credential, () => fill(credential.id))
                    }
                    trailing={
                      <RowChevron
                        active={highlight.highlighted === credential.id}
                      />
                    }
                  />
                ))}
              </>
            )}
          </SelectionGroup>
        </nav>
      )}
      {content.state === "list" && pending && (
        <ConfirmFill
          key={pending.id}
          credential={pending}
          host={host}
          purpose={content.purpose}
          onConfirm={credentials.confirm}
          onCancel={credentials.cancel}
        />
      )}
      {content.state === "locked" && (
        <LockedRow
          detail={t("extension.menu.locked.detail")}
          onUnlock={unlock}
        />
      )}
      {content.state === "not-open" && (
        <NotOpenRow detail={t("extension.menu.not-open.detail")} />
      )}
      {failed && <FillFailure content={content} failed={failed} />}
      {passkeyFailed && (
        <FailureNote>{t(passkeyNote(passkeyFailed, "get"))}</FailureNote>
      )}
    </>
  );
}

export function PasskeyRow({
  passkey,
  avatar,
  highlight,
  onChoose,
}: {
  passkey: PasskeyOption;
  avatar: ReactNode;
  highlight: Highlight;
  onChoose: () => void;
}) {
  const { t } = useTranslator();
  const key = `passkey:${passkey.credentialId}`;
  return (
    <button
      type="button"
      className={cn(itemRowClass(false, "menu"), "relative w-full")}
      {...highlight.row(key)}
      onClick={onChoose}
    >
      <ItemRowContent
        active={highlight.highlighted === key}
        avatar={avatar}
        title={passkey.label || t("credential.untitled")}
        detail={{
          text: t("extension.menu.passkey.detail", {
            account: passkey.account || t("credential.login.empty"),
          }),
        }}
        trailing={<RowChevron active={highlight.highlighted === key} />}
        look="menu"
      />
    </button>
  );
}

function PasskeyAvatar() {
  return (
    <Avatar
      label=""
      icon={UserRoundKey}
      size="row"
      shape="tile"
      emphasis="none"
    />
  );
}

export function FillFailure({
  content,
  failed,
}: {
  content: CredentialMenuContent;
  failed: MenuAction | Unconfirmed;
}) {
  const { t } = useTranslator();
  if (failed === "unlock") {
    return <FailureNote>{t("extension.menu.unlock.error")}</FailureNote>;
  }
  if (failed === "declined" || failed === "unverifiable") {
    return <FailureNote>{t(`extension.fill.${failed}`)}</FailureNote>;
  }
  return (
    <FailureNote>
      {t(
        content.state === "list" && content.purpose === "code"
          ? "fill-code.error"
          : "fill.error",
      )}
    </FailureNote>
  );
}

export function CredentialRow({
  credential,
  highlight,
  onChoose,
  detail,
  trailing,
}: {
  credential: Suggestion;
  highlight: Highlight;
  onChoose: () => void;
  detail?: string;
  trailing: ReactNode;
}) {
  const { t } = useTranslator();
  return (
    <button
      type="button"
      className={cn(itemRowClass(false, "menu"), "relative w-full")}
      {...highlight.row(credential.id)}
      onClick={onChoose}
    >
      <ItemRowContent
        active={highlight.highlighted === credential.id}
        avatar={<SiteAvatar site={credential.site} label={credential.label} />}
        title={credential.label || t("credential.untitled")}
        tags={credential.tags}
        detail={{
          text: detail ?? (credential.account || t("credential.login.empty")),
        }}
        trailing={trailing}
        look="menu"
      />
    </button>
  );
}

function CodeRows({
  token,
  credentials,
  highlight,
  onChoose,
  onFill,
  onReveal,
}: {
  token: string;
  credentials: readonly Listed<CodeSuggestion>[];
  highlight: Highlight;
  onChoose: Credentials["choose"];
  onFill: (id: string) => void;
  onReveal: Credentials["reveal"];
}) {
  const { menu, view } = useCodeMenu({
    token,
    credentials,
    highlighted: highlight.highlighted,
    onFill,
    onReveal,
  });
  return credentials.map((credential) => (
    <CodeRow
      key={credential.id}
      credential={credential}
      highlight={highlight}
      revealed={view.revealed?.id === credential.id ? view.revealed.code : null}
      waiting={view.waiting === credential.id}
      onChoose={() => onChoose(credential, () => menu.choose(credential.id))}
    />
  ));
}

/** Option held is reported by this frame's keys or by the field's frame through `menu-option`. */
function useCodeMenu({
  token,
  credentials,
  highlighted,
  onFill,
  onReveal,
}: {
  token: string;
  credentials: readonly CodeSuggestion[];
  highlighted: string | null;
  onFill: (id: string) => void;
  onReveal: Credentials["reveal"];
}): { menu: CodeMenu; view: CodeMenuView } {
  const gate = useClickGate();
  const fill = useRef(onFill);
  const reveal = useRef(onReveal);
  useEffect(() => {
    fill.current = onFill;
    reveal.current = onReveal;
  });
  const [menu] = useState(
    () =>
      new CodeMenu({
        rows: credentials,
        reveal: async (id) => {
          if (!gate.accepts()) {
            throw new Error(
              "Ravenpass shows a code only while its menu is shown.",
            );
          }
          return reveal.current(id);
        },
        fill: (id) => fill.current(id),
      }),
  );

  useEffect(() => () => menu.stop(), [menu]);

  useEffect(() => {
    menu.highlight(highlighted);
  }, [menu, highlighted]);

  useEffect(() => {
    const onOption = (event: KeyboardEvent) => {
      if (event.key === "Alt") menu.hold(event.type === "keydown");
    };
    const onEscape = (event: KeyboardEvent) => {
      if (event.key === "Escape" && menu.cancel()) {
        event.preventDefault();
        event.stopImmediatePropagation();
      }
    };
    const release = () => menu.hold(false);
    const onMessage = (
      message: unknown,
      sender: chrome.runtime.MessageSender,
    ): undefined => {
      if (
        sender.id === chrome.runtime.id &&
        isFrameMessage(message, token) &&
        message.kind === "menu-option"
      ) {
        menu.hold(message.held);
      }
    };
    // Capture phase: must run before the menu's own Escape handler closes the menu.
    addEventListener("keydown", onEscape, true);
    addEventListener("keydown", onOption);
    addEventListener("keyup", onOption);
    addEventListener("blur", release);
    chrome.runtime.onMessage.addListener(onMessage);
    return () => {
      removeEventListener("keydown", onEscape, true);
      removeEventListener("keydown", onOption);
      removeEventListener("keyup", onOption);
      removeEventListener("blur", release);
      chrome.runtime.onMessage.removeListener(onMessage);
    };
  }, [menu, token]);

  return { menu, view: useSyncExternalStore(menu.subscribe, menu.view) };
}

function CodeRow({
  credential,
  highlight,
  revealed,
  waiting,
  onChoose,
}: {
  credential: CodeSuggestion;
  highlight: Highlight;
  revealed: string | null;
  waiting: boolean;
  onChoose: () => void;
}) {
  const { t } = useTranslator();
  const left = usePeriodLeft(credential.period);
  return (
    <CredentialRow
      credential={credential}
      highlight={highlight}
      onChoose={onChoose}
      detail={
        waiting
          ? t("extension.menu.code.waiting", { seconds: left.seconds })
          : undefined
      }
      trailing={
        <span className="flex items-center gap-2.5">
          <CodeDigits
            code={revealed ?? maskedCode(credential.digits)}
            size="row"
          />
          <CodeTimer left={left} warn />
        </span>
      }
    />
  );
}
