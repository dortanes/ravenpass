import { ChromeAutofill } from "../chrome-autofill.ts";
import { logRejection } from "../failures.ts";
import { StoredLanguage } from "../language.ts";
import { LinkClient, LinkError } from "../link/client.ts";
import { ConnectionKey } from "../link/key.ts";
import { WebCryptoKeys } from "../link/keys.ts";
import { linkName } from "../link/name.ts";
import { LinkSocket } from "../link/socket.ts";
import { IndexedLinkStore } from "../link/store.ts";
import {
  type Answer,
  type Answers,
  type MenuRequest,
  type OfferRequest,
  type OfferShow,
  type PasskeyPageRequest,
  type Request,
  vaultStatePort,
} from "../messages.ts";
import { sendIgnoringClosedPort } from "../messaging/send.ts";
import { StoredSignInStyle } from "../sign-in-style.ts";
import { ColorSchemePage } from "../toolbar/color-scheme-page.ts";
import { toolbarIcon } from "../toolbar/icon.ts";
import { CardFrames } from "./card-frames.ts";
import { ContextMenu } from "./context-menu.ts";
import { MenuRouter, type TabMessenger } from "./menus.ts";
import { PagePasskeys } from "./page-passkeys.ts";
import { PendingPasskeys } from "./pending-passkeys.ts";
import { PendingSignIns } from "./pending-sign-ins.ts";
import { RecentFills } from "./recent-fills.ts";
import { SaveOffers } from "./save-offers.ts";
import { MenuSessions } from "./sessions.ts";
import { SignInCards } from "./sign-in-cards.ts";
import { TabOffers } from "./tab-offers.ts";
import { VaultWatch } from "./vault-watch.ts";

const vaultStatusEveryMs = 1500;

const tabCleanupFailure = "Ravenpass could not forget a closed tab.";
const contextMenuFailure = "Ravenpass could not build its context menu.";
const contextMenuClickFailure =
  "Ravenpass could not open the menu chosen from the context menu.";
const colorSchemeFailure = "Ravenpass could not match the toolbar icon.";
const chromeAutofillFailure = "Ravenpass could not change Chrome's autofill.";

const language = new StoredLanguage();

const signIn = new StoredSignInStyle();

const client = new LinkClient({
  store: new IndexedLinkStore(),
  language,
  signIn,
  connect: LinkSocket.connect,
  keys: new WebCryptoKeys(),
});

const sessions = new MenuSessions({
  area: chrome.storage.session,
  extensionId: chrome.runtime.id,
});

const relay: TabMessenger = (target, message) =>
  sendIgnoringClosedPort(
    chrome.tabs.sendMessage(
      target.tabId,
      message,
      "documentId" in target
        ? { documentId: target.documentId }
        : { frameId: target.frameId },
    ),
  );

const cards = new SignInCards({ sessions, relay });

const passkeys = new PagePasskeys({
  client,
  sessions,
  cards,
  pending: new PendingPasskeys({ area: chrome.storage.session }),
  relay,
});

const menus = new MenuRouter({
  client,
  sessions,
  cards,
  passkeys,
  recentFills: new RecentFills({ area: chrome.storage.session }),
  pendingSignIns: new PendingSignIns({ area: chrome.storage.session }),
  cardFrames: new CardFrames({ area: chrome.storage.session }),
  signInStyle: () => signIn.signInStyle(),
  relay,
});

const offerShow: OfferShow = { kind: "offer-show" };

const saveOffers = new SaveOffers({
  client,
  sessions,
  offers: new TabOffers({ area: chrome.storage.session, sessions }),
  relay,
  show: async (tabId) => {
    await sendIgnoringClosedPort(
      chrome.tabs.sendMessage(tabId, offerShow, { frameId: 0 }),
    );
  },
});

const colorSchemePage = new ColorSchemePage();

const chromeAutofill = new ChromeAutofill();

const contextMenu = new ContextMenu({
  language,
  menus: chrome.contextMenus,
  send: (tabId, frameId, message) =>
    sendIgnoringClosedPort(
      chrome.tabs.sendMessage(tabId, message, { frameId }),
    ),
});

const vaultWatch = new VaultWatch({
  status: () => client.status(),
  everyMs: vaultStatusEveryMs,
});

async function link(text: string): Promise<Answers["link"]> {
  let key: ConnectionKey;
  try {
    key = await ConnectionKey.parse(text);
  } catch {
    return { ok: false, reason: "not-key" };
  }
  try {
    await client.link(key, linkName(navigator.userAgentData));
    return { ok: true };
  } catch (error) {
    return {
      ok: false,
      reason: error instanceof LinkError ? error.reason : "failed",
    };
  }
}

/** A tab's frames hold content scripts and menus, which may not ask about the link. */
async function servePage(
  request: Exclude<Request, MenuRequest | OfferRequest | PasskeyPageRequest>,
  sender: chrome.runtime.MessageSender,
): Promise<Answer> {
  if (sender.tab) {
    throw new Error(`A tab's frame may not make the ${request.kind} request.`);
  }
  switch (request.kind) {
    case "link":
      return link(request.key);
    case "status":
      return client.status();
    case "unlink":
      await client.unlink();
      return { ok: true };
    case "color-scheme":
      await chrome.action.setIcon({ path: toolbarIcon(request.scheme) });
      return { ok: true };
  }
}

function serve(
  request: Request,
  sender: chrome.runtime.MessageSender,
): Promise<Answer> {
  switch (request.kind) {
    case "link":
    case "status":
    case "unlink":
    case "color-scheme":
      return servePage(request, sender);
    case "capture":
    case "card-capture":
    case "form-gone":
    case "offer-open":
    case "offer-review":
    case "offer-save":
    case "offer-discard":
      return saveOffers.serve(request, sender);
    case "passkey-request":
    case "passkey-withdraw":
    case "passkey-linked":
      return passkeys.serve(request, sender);
    default:
      return menus.serve(request, sender);
  }
}

chrome.runtime.onMessage.addListener((request: Request, sender, respond) => {
  serve(request, sender).then(respond, () => respond(null));
  return true;
});

chrome.runtime.onConnect.addListener((port) => {
  if (port.name === vaultStatePort && port.sender?.id === chrome.runtime.id) {
    vaultWatch.connect(port);
  }
});

chrome.tabs.onRemoved.addListener((tabId) => {
  void logRejection(
    Promise.all([
      menus.tabClosed(tabId),
      saveOffers.tabClosed(tabId),
      passkeys.tabClosed(tabId),
    ]),
    tabCleanupFailure,
  );
});

chrome.contextMenus.onClicked.addListener((info, tab) => {
  void logRejection(contextMenu.clicked(info, tab), contextMenuClickFailure);
});

language.watchLanguage(() => {
  void logRejection(contextMenu.retitle(), contextMenuFailure);
});

function applyChromeAutofill(): void {
  void logRejection(chromeAutofill.apply(), chromeAutofillFailure);
}

chromeAutofill.watch(applyChromeAutofill);

function openColorSchemePage(): void {
  void logRejection(colorSchemePage.open(), colorSchemeFailure);
}

function prepare(): void {
  openColorSchemePage();
  applyChromeAutofill();
  void logRejection(contextMenu.create(), contextMenuFailure);
}

// Chrome wakes the service worker at startup and install only for these listeners.
chrome.runtime.onStartup.addListener(prepare);
chrome.runtime.onInstalled.addListener(prepare);
openColorSchemePage();
// Chrome drops the extension's settings while it is disabled, and enabling it fires neither event.
applyChromeAutofill();
