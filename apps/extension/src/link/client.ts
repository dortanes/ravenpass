import type { SaveChoice, SaveTarget } from "@ravenpass/ui/saving/offer.ts";
import type { OneTimeCode, SiteIcon } from "@ravenpass/ui/vault-api.ts";
import * as v from "valibot";
import type { DesktopLanguageStore } from "../language.ts";
import {
  type CreateOptions,
  type GetOptions,
  type PasskeyUser,
  verifies,
} from "../passkeys/requests.ts";
import {
  type CreatedPasskey,
  readCreatedPasskey,
  readSignedPasskey,
  type SignedPasskey,
} from "../passkeys/responses.ts";
import type { SignInStyleStore } from "../sign-in-style.ts";
import type { ConnectionKey } from "./key.ts";
import type { KeySource } from "./keys.ts";
import { HandshakeState, type Transport } from "./noise.ts";
import {
  captureOffer,
  type cardOption,
  cardOptions,
  cardValues,
  type codeSuggestion,
  codeSuggestions,
  desktopStatus,
  fillValues,
  type identity,
  type identityFile,
  identityFiles,
  linkConfirmation,
  oneTimeCode,
  partCount,
  type passkeyOption,
  passkeyOptions,
  type passkeyTarget,
  passkeyTargets,
  type pendingCapture,
  requestId,
  savedAs,
  sessionFrame,
  sessionGreeting,
  sharedFileHead,
  shareProgress,
  siteIcon,
  type suggestion,
  suggestions,
} from "./results.ts";
import {
  type FrameSocket,
  SocketClosed,
  type SocketFactory,
  SocketUnavailable,
} from "./socket.ts";
import type { LinkRecord, LinkStore } from "./store.ts";

type Bytes = Uint8Array<ArrayBuffer>;

const host = "127.0.0.1";
const linkPath = "/v1/link";
const sessionPath = "/v1/session";

const closeNotLinked = 4401;
const closeUnauthenticated = 4403;
const closeNoKey = 4404;

const empty: Bytes = new Uint8Array(0);
const encoder = new TextEncoder();
const decoder = new TextDecoder("utf-8", { fatal: true });

export type DesktopState = "unlocked" | "locked" | "not-open";

export type LinkStatus =
  | { readonly linked: false; readonly unlinkedInRavenpass: boolean }
  | {
      readonly linked: true;
      readonly desktop: DesktopState;
      readonly linkedAt: number;
    };

export type LinkFailure = "expired" | "not-open" | "failed";

export class LinkError extends Error {
  readonly reason: LinkFailure;

  constructor(reason: LinkFailure, options?: ErrorOptions) {
    super(`The extension could not link to Ravenpass (${reason}).`, options);
    this.name = "LinkError";
    this.reason = reason;
  }
}

export type SuggestPurpose = "sign-in" | "code";

export type Suggestion = v.InferOutput<typeof suggestion>;

export type CodeSuggestion = v.InferOutput<typeof codeSuggestion>;

export type FillValues = v.InferOutput<typeof fillValues>;

export type CardOption = v.InferOutput<typeof cardOption>;

export type CardValues = v.InferOutput<typeof cardValues>;

export type IdentityFile = v.InferOutput<typeof identityFile>;

export type IdentityFiles = v.InferOutput<typeof identity>;

export type ShareProgress = v.InferOutput<typeof shareProgress>;

export type PasskeyOption = v.InferOutput<typeof passkeyOption>;

export type PasskeyChoice = Pick<PasskeyOption, "credential" | "credentialId">;

export type PasskeyTarget = v.InferOutput<typeof passkeyTarget>;

export type PasskeyTargets = v.InferOutput<typeof passkeyTargets>;

export interface SharedFile {
  readonly name: string;
  readonly mediaType: string;
  readonly content: Bytes;
}

export interface CapturedPassword {
  readonly account: string;
  readonly password: string;
  // Set when the form asked for the current password beside a new one.
  readonly current?: string;
}

/** A card typed on a page; `expiry` is YYYY-MM and `network` a credit-card-type name, each empty for none. */
export interface CapturedCard {
  readonly holder: string;
  readonly number: string;
  readonly expiry: string;
  readonly securityCode: string;
  readonly network: string;
}

export type { SaveChoice, SaveTarget };

export type PendingCapture = v.InferOutput<typeof pendingCapture>;

export type CaptureOffer = v.InferOutput<typeof captureOffer>;

export type SavedAs = v.InferOutput<typeof savedAs>;

const desktopFailures = [
  "locked",
  "no-match",
  "not-found",
  "no-code",
  "declined",
  "unverifiable",
  "invalid-origin",
  "invalid-purpose",
  "invalid-account",
  "invalid-name",
  "invalid-rp",
  "invalid-request",
  "excluded",
  "full",
  "unknown-request",
] as const;

export type SessionFailure =
  | "unlinked"
  | "not-open"
  | (typeof desktopFailures)[number]
  | "failed";

export class SessionError extends Error {
  readonly reason: SessionFailure;

  constructor(reason: SessionFailure, options?: ErrorOptions) {
    super(`Ravenpass did not serve the request (${reason}).`, options);
    this.name = "SessionError";
    this.reason = reason;
  }
}

type SessionRequest =
  | { readonly type: "status" }
  | { readonly type: "unlink" }
  | {
      readonly type: "suggest";
      readonly origin: string;
      readonly purpose: SuggestPurpose;
    }
  | {
      readonly type: "fill";
      readonly credential: string;
      readonly origin: string;
    }
  | {
      readonly type: "code";
      readonly credential: string;
      readonly origin: string;
    }
  | {
      readonly type: "add-website";
      readonly credential: string;
      readonly origin: string;
    }
  | { readonly type: "icon"; readonly site: string }
  | { readonly type: "unlock" }
  | { readonly type: "identities"; readonly origin: string }
  | {
      readonly type: "share";
      readonly identity: string;
      readonly file: string;
      readonly origin: string;
    }
  | {
      readonly type: "capture";
      readonly origin: string;
      readonly account: string;
      readonly password: string;
      readonly current: string | undefined;
    }
  | ({ readonly type: "card-capture"; readonly origin: string } & CapturedCard)
  | { readonly type: "cards"; readonly origin: string }
  | {
      readonly type: "card-fill";
      readonly card: string;
      readonly origin: string;
    }
  | { readonly type: "review" | "discard"; readonly pending: string }
  | ({ readonly type: "save"; readonly pending: string } & SaveChoice)
  | {
      readonly type: "passkeys";
      readonly origin: string;
      readonly mode: "get";
      readonly rpId: string;
      readonly allow: readonly string[];
    }
  | {
      readonly type: "passkeys";
      readonly origin: string;
      readonly mode: "create";
      readonly rpId: string;
      readonly account: string;
      readonly exclude: readonly string[];
    }
  | {
      readonly type: "passkey-create";
      readonly origin: string;
      readonly rpId: string;
      readonly rpName: string;
      readonly user: PasskeyUser;
      readonly challenge: string;
      readonly verify: boolean;
      readonly target: string;
      readonly exclude: readonly string[];
      readonly algorithms: readonly number[];
    }
  | ({
      readonly type: "passkey-sign";
      readonly origin: string;
      readonly rpId: string;
      readonly challenge: string;
      readonly verify: boolean;
    } & PasskeyChoice);

// `body` holds the raw parts that followed a head announcing a file.
interface Served {
  readonly result: unknown;
  readonly body: Bytes | null;
}

type Reply = Served | { readonly error: string };

type ProgressListener = (progress: ShareProgress) => void;

type Greeting = v.InferOutput<typeof sessionGreeting>;

export interface LinkClientDependencies {
  readonly store: LinkStore;
  readonly language: DesktopLanguageStore;
  readonly signIn: SignInStyleStore;
  readonly connect: SocketFactory;
  readonly keys: KeySource;
}

export class LinkClient {
  private readonly store: LinkStore;
  private readonly language: DesktopLanguageStore;
  private readonly signIn: SignInStyleStore;
  private readonly connect: SocketFactory;
  private readonly keys: KeySource;

  constructor({
    store,
    language,
    signIn,
    connect,
    keys,
  }: LinkClientDependencies) {
    this.store = store;
    this.language = language;
    this.signIn = signIn;
    this.connect = connect;
    this.keys = keys;
  }

  async link(key: ConnectionKey, name: string): Promise<void> {
    const staticKeys = await this.keys.generate();
    let linked: { desktopPublicKey: Bytes; greeting: Greeting };
    try {
      linked = await this.exchange(key.port, linkPath, async (socket) => {
        const handshake = await HandshakeState.link(
          { staticKeys, ephemeralKeys: this.keys },
          key.secret,
        );
        socket.send(await handshake.writeMessage(empty));
        await handshake.readMessage(await socket.receive());
        socket.send(await handshake.writeMessage(encodeJson({ name })));
        const confirmation = v.safeParse(
          linkConfirmation,
          decodeJson(await handshake.transport().open(await socket.receive())),
        );
        if (!confirmation.success) {
          throw new Error("Ravenpass did not confirm the link.");
        }
        return {
          desktopPublicKey: handshake.responderStatic(),
          greeting: confirmation.output,
        };
      });
    } catch (error) {
      throw new LinkError(linkFailure(error), { cause: error });
    }
    await this.store.save({
      port: key.port,
      desktopPublicKey: linked.desktopPublicKey,
      staticKeys,
      linkedAt: Date.now(),
    });
    await this.keep(linked.greeting);
  }

  async status(): Promise<LinkStatus> {
    const record = await this.store.load();
    if (!record) return { linked: false, unlinkedInRavenpass: false };
    try {
      const reply = await this.request(record, { type: "status" });
      return {
        linked: true,
        desktop: desktopState(reply),
        linkedAt: record.linkedAt,
      };
    } catch (error) {
      if (error instanceof SocketClosed && error.code === closeNotLinked) {
        await this.forget();
        return { linked: false, unlinkedInRavenpass: true };
      }
      return { linked: true, desktop: "not-open", linkedAt: record.linkedAt };
    }
  }

  async unlink(): Promise<void> {
    try {
      const record = await this.store.load();
      if (record) await this.request(record, { type: "unlink" });
    } catch {
      // The desktop app may be closed or may have unlinked the extension already.
    }
    await this.forget();
  }

  suggest(origin: string, purpose: "sign-in"): Promise<Suggestion[]>;
  suggest(origin: string, purpose: "code"): Promise<CodeSuggestion[]>;
  async suggest(
    origin: string,
    purpose: SuggestPurpose,
  ): Promise<Suggestion[]> {
    const result = await this.ask({ type: "suggest", origin, purpose });
    return purpose === "code"
      ? resultOf(codeSuggestions, result)
      : resultOf(suggestions, result);
  }

  /** `onProgress` hears what Ravenpass asks of the person while the owner's choice makes fills wait for them. */
  async fill(
    id: string,
    origin: string,
    onProgress?: ProgressListener,
  ): Promise<FillValues> {
    const { result } = await this.served(
      { type: "fill", credential: id, origin },
      onProgress,
    );
    return resultOf(fillValues, result);
  }

  /** `onProgress` as for `fill`. */
  async code(
    id: string,
    origin: string,
    onProgress?: ProgressListener,
  ): Promise<OneTimeCode> {
    const { result } = await this.served(
      { type: "code", credential: id, origin },
      onProgress,
    );
    return resultOf(oneTimeCode, result);
  }

  /** Ravenpass adds only an `https` origin, and only to a credential that already matches it by site. */
  async addWebsite(id: string, origin: string): Promise<void> {
    await this.ask({ type: "add-website", credential: id, origin });
  }

  async icon(site: string): Promise<SiteIcon> {
    return resultOf(siteIcon, await this.ask({ type: "icon", site }));
  }

  async unlock(): Promise<void> {
    await this.ask({ type: "unlock" });
  }

  async identities(origin: string): Promise<IdentityFiles[]> {
    return resultOf(
      identityFiles,
      await this.ask({ type: "identities", origin }),
    );
  }

  async share(
    identity: string,
    file: string,
    origin: string,
    onProgress: ProgressListener,
  ): Promise<SharedFile> {
    return sharedFile(
      await this.served({ type: "share", identity, file, origin }, onProgress),
    );
  }

  async capture(
    origin: string,
    { account, password, current }: CapturedPassword,
  ): Promise<CaptureOffer> {
    return resultOf(
      captureOffer,
      await this.ask({ type: "capture", origin, account, password, current }),
    );
  }

  /** Ravenpass refuses a page that is not `https`, or `http` on `localhost`. */
  async captureCard(origin: string, card: CapturedCard): Promise<CaptureOffer> {
    return resultOf(
      captureOffer,
      await this.ask({ type: "card-capture", origin, ...card }),
    );
  }

  /** Ravenpass refuses a page that is not `https`, or `http` on `localhost`. */
  async cards(origin: string): Promise<CardOption[]> {
    return resultOf(cardOptions, await this.ask({ type: "cards", origin }));
  }

  /** `onProgress` hears what Ravenpass asks of the person, which it does on every card fill. */
  async fillCard(
    id: string,
    origin: string,
    onProgress: ProgressListener,
  ): Promise<CardValues> {
    const { result } = await this.served(
      { type: "card-fill", card: id, origin },
      onProgress,
    );
    return resultOf(cardValues, result);
  }

  async review(pending: string): Promise<CaptureOffer> {
    return resultOf(captureOffer, await this.ask({ type: "review", pending }));
  }

  async save(
    pending: string,
    { target, account, name }: SaveChoice,
  ): Promise<SavedAs> {
    return resultOf(
      savedAs,
      await this.ask({ type: "save", pending, target, account, name }),
    );
  }

  async discard(pending: string): Promise<void> {
    await this.ask({ type: "discard", pending });
  }

  async linked(): Promise<boolean> {
    return (await this.store.load()) !== undefined;
  }

  async passkeys(
    origin: string,
    options: GetOptions,
  ): Promise<PasskeyOption[]> {
    return resultOf(
      passkeyOptions,
      await this.ask({
        type: "passkeys",
        origin,
        mode: "get",
        rpId: options.rpId ?? "",
        allow: options.allow,
      }),
    );
  }

  async passkeyTargets(
    origin: string,
    options: CreateOptions,
  ): Promise<PasskeyTargets> {
    return resultOf(
      passkeyTargets,
      await this.ask({
        type: "passkeys",
        origin,
        mode: "create",
        rpId: options.rpId ?? "",
        account: options.user.name,
        exclude: options.exclude,
      }),
    );
  }

  async createPasskey(
    origin: string,
    options: CreateOptions,
    target: string,
    onProgress: ProgressListener,
  ): Promise<CreatedPasskey> {
    const { result } = await this.served(
      {
        type: "passkey-create",
        origin,
        rpId: options.rpId ?? "",
        rpName: options.rpName,
        user: options.user,
        challenge: options.challenge,
        verify: verifies(options),
        target,
        exclude: options.exclude,
        algorithms: options.algorithms,
      },
      onProgress,
    );
    const created = readCreatedPasskey(result);
    if (!created) throw new SessionError("failed");
    return created;
  }

  async signPasskey(
    origin: string,
    options: GetOptions,
    { credential, credentialId }: PasskeyChoice,
    onProgress: ProgressListener,
  ): Promise<SignedPasskey> {
    const { result } = await this.served(
      {
        type: "passkey-sign",
        origin,
        rpId: options.rpId ?? "",
        challenge: options.challenge,
        verify: verifies(options),
        credential,
        credentialId,
      },
      onProgress,
    );
    const signed = readSignedPasskey(result);
    if (!signed) throw new SessionError("failed");
    return signed;
  }

  private async ask(request: SessionRequest): Promise<unknown> {
    return (await this.served(request)).result;
  }

  private async served(
    request: SessionRequest,
    onProgress?: ProgressListener,
  ): Promise<Served> {
    const record = await this.store.load();
    if (!record) throw new SessionError("unlinked");
    let reply: Reply;
    try {
      reply = await this.request(record, request, onProgress);
    } catch (error) {
      if (error instanceof SocketClosed && error.code === closeNotLinked) {
        await this.forget();
        throw new SessionError("unlinked", { cause: error });
      }
      throw new SessionError("not-open", { cause: error });
    }
    if ("error" in reply) throw new SessionError(desktopFailure(reply.error));
    return reply;
  }

  private async forget(): Promise<void> {
    await this.store.clear();
    await this.language.forgetDesktopLanguage();
    await this.signIn.forgetSignInStyle();
  }

  private async keep({ language, signIn }: Greeting): Promise<void> {
    await this.language.keepDesktopLanguage(language);
    await this.signIn.keepSignInStyle(signIn);
  }

  private request(
    record: LinkRecord,
    request: SessionRequest,
    onProgress?: ProgressListener,
  ): Promise<Reply> {
    return this.exchange(record.port, sessionPath, async (socket) => {
      const transport = await this.session(record, socket);
      socket.send(
        await transport.seal(encodeJson({ id: requestId, ...request })),
      );
      const receive = async () => transport.open(await socket.receive());
      for (;;) {
        const parsed = v.safeParse(sessionFrame, decodeJson(await receive()));
        if (!parsed.success) {
          throw new Error(
            `Ravenpass did not serve the ${request.type} request.`,
          );
        }
        const frame = parsed.output;
        if ("progress" in frame) {
          if (v.is(shareProgress, frame.progress)) onProgress?.(frame.progress);
          continue;
        }
        if ("error" in frame) return { error: frame.error };
        if ("parts" in frame) {
          const body = await receiveParts(receive, frame.parts);
          return "result" in frame
            ? { result: frame.result, body }
            : { result: decodeJson(body), body: null };
        }
        return { result: frame.result, body: null };
      }
    });
  }

  private async session(
    record: LinkRecord,
    socket: FrameSocket,
  ): Promise<Transport> {
    const handshake = await HandshakeState.session(
      { staticKeys: record.staticKeys, ephemeralKeys: this.keys },
      record.desktopPublicKey,
    );
    socket.send(await handshake.writeMessage(empty));
    const greeting = v.safeParse(
      sessionGreeting,
      decodeJson(await handshake.readMessage(await socket.receive())),
    );
    if (!greeting.success) {
      throw new Error("Ravenpass did not name its language and sign-in style.");
    }
    await this.keep(greeting.output);
    return handshake.transport();
  }

  private async exchange<Result>(
    port: number,
    path: string,
    run: (socket: FrameSocket) => Promise<Result>,
  ): Promise<Result> {
    const socket = await this.connect(`ws://${host}:${port}${path}`);
    try {
      return await run(socket);
    } finally {
      socket.close();
    }
  }
}

function linkFailure(error: unknown): LinkFailure {
  if (error instanceof SocketUnavailable) return "not-open";
  // 4403 while linking is a handshake that did not authenticate, as with a replaced key.
  if (
    error instanceof SocketClosed &&
    (error.code === closeNoKey || error.code === closeUnauthenticated)
  ) {
    return "expired";
  }
  return "failed";
}

function desktopState(reply: Reply): DesktopState {
  if ("result" in reply) {
    const status = v.safeParse(desktopStatus, reply.result);
    if (status.success) return status.output;
  }
  throw new Error(
    "Ravenpass answered the status request with an unknown state.",
  );
}

function desktopFailure(code: string): SessionFailure {
  return desktopFailures.find((known) => known === code) ?? "failed";
}

function resultOf<Schema extends v.GenericSchema>(
  schema: Schema,
  result: unknown,
): v.InferOutput<Schema> {
  const parsed = v.safeParse(schema, result);
  if (!parsed.success) throw new SessionError("failed");
  return parsed.output;
}

function sharedFile({ result, body }: Served): SharedFile {
  const head = v.safeParse(sharedFileHead, result);
  if (!head.success || body === null || head.output.size !== body.length) {
    throw new SessionError("failed");
  }
  const { name, mediaType } = head.output;
  return { name, mediaType, content: body };
}

async function receiveParts(
  receive: () => Promise<Bytes>,
  parts: unknown,
): Promise<Bytes> {
  if (!v.is(partCount, parts)) {
    throw new Error("Ravenpass announced an unreadable number of parts.");
  }
  const frames: Bytes[] = [];
  for (let part = 0; part < parts; part += 1) frames.push(await receive());
  const joined = new Uint8Array(
    frames.reduce((length, frame) => length + frame.length, 0),
  );
  let offset = 0;
  for (const frame of frames) {
    joined.set(frame, offset);
    offset += frame.length;
  }
  return joined;
}

function encodeJson(value: unknown): Bytes {
  return encoder.encode(JSON.stringify(value));
}

function decodeJson(bytes: Bytes): unknown {
  return JSON.parse(decoder.decode(bytes));
}
