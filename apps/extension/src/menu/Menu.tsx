import { SiteIconsProvider } from "@ravenpass/ui/components/workspace/SiteIcons.tsx";
import { listTarget } from "@ravenpass/ui/hooks/roving-focus.ts";
import { LanguageProvider } from "@ravenpass/ui/i18n/translator.tsx";
import { MotionProvider } from "@ravenpass/ui/motion/MotionProvider.tsx";
import { SiteIconStore } from "@ravenpass/ui/workspace/site-icons.ts";
import { useCallback, useEffect, useRef, useState } from "react";
import { frostings, preferredScheme } from "../content/backdrop.ts";
import { StoredLanguage } from "../language.ts";
import {
  ask,
  type CredentialMenuContent,
  type FileMenuContent,
  isCard,
  isFrameMessage,
  listsNothing,
  type MenuContent,
} from "../messages.ts";
import { sendIgnoringClosedPort } from "../messaging/send.ts";
import { CardMenu } from "./Cards.tsx";
import { ClickGateProvider } from "./ClickGateProvider.tsx";
import { CredentialList, useCredentials } from "./Credentials.tsx";
import { FileMenu } from "./FileMenu.tsx";
import { itemSelector } from "./highlight.ts";
import { PasskeyCard } from "./PasskeyCard.tsx";
import { MenuFooter } from "./Rows.tsx";
import { SavePrompt } from "./SavePrompt.tsx";
import { SignInCard } from "./SignInCard.tsx";

const language = new StoredLanguage();

const tint = `color-mix(in srgb, var(--color-popover) ${frostings[preferredScheme()].tint * 100}%, transparent)`;

/** A click acts only while the menu is visibly shown. */
export function Menu({ token }: { token: string }) {
  const [icons] = useState(
    () => new SiteIconStore((site) => ask({ kind: "menu-icon", token, site })),
  );
  return (
    <LanguageProvider api={language}>
      <MotionProvider>
        <SiteIconsProvider store={icons}>
          <ClickGateProvider>
            <MenuSurface token={token} />
          </ClickGateProvider>
        </SiteIconsProvider>
      </MotionProvider>
    </LanguageProvider>
  );
}

function MenuSurface({ token }: { token: string }) {
  const [view, setView] = useState<{
    site: string;
    host: string;
    content: MenuContent;
  } | null>(null);
  const toField =
    view !== null && !isCard(view.content) && !isFileMenu(view.content);
  const surface = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    let active = true;
    void sendIgnoringClosedPort(ask({ kind: "menu", token })).then((answer) => {
      if (active && answer) setView(answer);
    });
    return () => {
      active = false;
    };
  }, [token]);

  useEffect(() => {
    const onMessage = (
      message: unknown,
      sender: chrome.runtime.MessageSender,
    ): undefined => {
      if (
        sender.id === chrome.runtime.id &&
        isFrameMessage(message, token) &&
        message.kind === "menu-focus"
      ) {
        surface.current?.querySelector<HTMLElement>(itemSelector)?.focus();
      }
    };
    chrome.runtime.onMessage.addListener(onMessage);
    return () => chrome.runtime.onMessage.removeListener(onMessage);
  }, [token]);

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) =>
      moveFocus(event, surface.current, token, toField);
    addEventListener("keydown", onKeyDown);
    return () => removeEventListener("keydown", onKeyDown);
  }, [token, toField]);

  const measure = useCallback(
    (element: HTMLDivElement | null) => {
      if (!element) return;
      surface.current = element;
      const observer = new ResizeObserver(() => {
        const height = Math.ceil(element.getBoundingClientRect().height);
        void sendIgnoringClosedPort(ask({ kind: "menu-size", token, height }));
      });
      observer.observe(element);
      return () => {
        observer.disconnect();
        surface.current = null;
      };
    },
    [token],
  );

  if (!view) return null;

  return (
    <div
      ref={measure}
      className="rounded-row p-[5px] text-popover-foreground shadow-[inset_0_0_0_1px_rgb(0_0_0/0.06),inset_0_1px_0_rgb(255_255_255/0.6)] dark:shadow-[inset_0_0_0_1px_rgb(255_255_255/0.055),inset_0_1px_0_rgb(255_255_255/0.05)]"
      style={{ backgroundColor: tint }}
    >
      <MenuBody
        token={token}
        site={view.site}
        host={view.host}
        content={view.content}
      />
    </div>
  );
}

function MenuBody({
  token,
  site,
  host,
  content,
}: {
  token: string;
  site: string;
  host: string;
  content: MenuContent;
}) {
  if (content.state === "offer") {
    return <SavePrompt token={token} offer={content.offer} />;
  }
  if (content.state === "sign-in-card") {
    return (
      <SignInCard token={token} site={site} host={host} content={content} />
    );
  }
  if (content.state === "passkey") {
    return <PasskeyCard token={token} site={site} content={content} />;
  }
  if (isFileMenu(content)) {
    return <FileMenu token={token} site={site} content={content} />;
  }
  if (content.state === "cards") {
    return <CardMenu token={token} initial={content.listing} />;
  }
  return (
    <CredentialMenu token={token} site={site} host={host} initial={content} />
  );
}

function isFileMenu(content: MenuContent): content is FileMenuContent {
  return content.state === "files" || content.state === "no-destination";
}

function CredentialMenu({
  token,
  site,
  host,
  initial,
}: {
  token: string;
  site: string;
  host: string;
  initial: CredentialMenuContent;
}) {
  const credentials = useCredentials(token, initial);
  const { content } = credentials;
  return (
    <>
      <CredentialList
        token={token}
        site={site}
        host={host}
        credentials={credentials}
      />
      <MenuFooter
        actions={
          content.state !== "list" || listsNothing(content)
            ? []
            : content.purpose === "code"
              ? ["show", "fill"]
              : ["fill"]
        }
      />
    </>
  );
}

/** The frame may hold focus with no row focused; ArrowUp above the first row returns to the field. */
function moveFocus(
  event: KeyboardEvent,
  surface: HTMLElement | null,
  token: string,
  toField: boolean,
) {
  if (event.key === "Escape") {
    event.preventDefault();
    void sendIgnoringClosedPort(ask({ kind: "menu-close", token }));
    return;
  }
  if (event.key !== "ArrowDown" && event.key !== "ArrowUp") return;
  event.preventDefault();
  const items = Array.from(
    surface?.querySelectorAll<HTMLElement>(itemSelector) ?? [],
  );
  const focused = document.activeElement;
  const index = focused instanceof HTMLElement ? items.indexOf(focused) : -1;
  const target = listTarget(event.key, index, items.length, false);
  if (typeof target === "number") {
    items[target]?.focus();
  } else if (event.key === "ArrowUp" && toField) {
    void sendIgnoringClosedPort(ask({ kind: "menu-focus", token }));
  }
}
