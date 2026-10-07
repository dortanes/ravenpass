import assert from "node:assert/strict";
import test from "node:test";
import type { SignInStyle } from "@ravenpass/ui/extensions/sign-in-style.ts";
import type { Language } from "@ravenpass/ui/i18n/language.ts";
import { base64urlnopad } from "@scure/base";
import type { DesktopLanguageStore } from "../language.ts";
import type { CreateOptions, GetOptions } from "../passkeys/requests.ts";
import type { SignInStyleStore } from "../sign-in-style.ts";
import {
  type IdentityFiles,
  LinkClient,
  LinkError,
  SessionError,
  type SessionFailure,
  type ShareProgress,
} from "./client.ts";
import { ConnectionKey } from "./key.ts";
import { HandshakeState, Transport } from "./noise.ts";
import {
  SocketClosed,
  type SocketFactory,
  SocketTimeout,
  SocketUnavailable,
} from "./socket.ts";
import type { LinkRecord, LinkStore } from "./store.ts";
import {
  bytes,
  FixedKeys,
  type HandshakeVector,
  handshakeVector,
  importKeyPair,
  linkProtocol,
  sessionProtocol,
} from "./vectors.test-support.ts";

type Bytes = Uint8Array<ArrayBuffer>;
type Reply = Bytes | { readonly close: number } | typeof silence;

const silence = Symbol("silence");

const link = handshakeVector(linkProtocol);
const session = handshakeVector(sessionProtocol);
const port = 53117;
const linkedAt = 1_750_000_000_000;
const name = "Chrome · macOS";

class MemoryStore implements LinkStore {
  record: LinkRecord | undefined;

  constructor(record?: LinkRecord) {
    this.record = record;
  }

  async load(): Promise<LinkRecord | undefined> {
    return this.record;
  }

  async save(record: LinkRecord): Promise<void> {
    this.record = record;
  }

  async clear(): Promise<void> {
    this.record = undefined;
  }
}

class MemoryLanguage implements DesktopLanguageStore {
  language: Language | undefined;

  constructor(language?: Language) {
    this.language = language;
  }

  async keepDesktopLanguage(language: Language): Promise<void> {
    this.language = language;
  }

  async forgetDesktopLanguage(): Promise<void> {
    this.language = undefined;
  }
}

class MemorySignInStyle implements SignInStyleStore {
  style: SignInStyle | undefined;

  constructor(style?: SignInStyle) {
    this.style = style;
  }

  async keepSignInStyle(style: SignInStyle): Promise<void> {
    this.style = style;
  }

  async forgetSignInStyle(): Promise<void> {
    this.style = undefined;
  }
}

class ScriptedDesktop {
  readonly urls: string[] = [];
  readonly sent: Bytes[] = [];
  closed = 0;
  private readonly replies: Reply[];

  constructor(replies: readonly Reply[]) {
    this.replies = [...replies];
  }

  readonly connect: SocketFactory = async (url) => {
    this.urls.push(url);
    return {
      send: (frame) => {
        this.sent.push(frame);
      },
      receive: async () => {
        const reply = this.replies.shift();
        if (!reply) throw new Error("The script has no reply left.");
        if (reply === silence) throw new SocketTimeout();
        if (reply instanceof Uint8Array) return reply;
        throw new SocketClosed(reply.close);
      },
      close: () => {
        this.closed += 1;
      },
    };
  };
}

const refused: SocketFactory = async () => {
  throw new SocketUnavailable();
};

function message(vector: HandshakeVector, index: number) {
  const found = vector.messages[index];
  assert.ok(found, `the vector has message ${index}`);
  return found;
}

function frame(vector: HandshakeVector, index: number): Bytes {
  return bytes(message(vector, index).ciphertext);
}

function sentFrame(desktop: ScriptedDesktop, index: number): Bytes {
  const sent = desktop.sent[index];
  assert.ok(sent, `the extension sent frame ${index}`);
  return sent;
}

function altered(original: Bytes): Bytes {
  const copy = original.slice();
  const last = copy.length - 1;
  copy[last] = (copy[last] ?? 0) ^ 0x01;
  return copy;
}

async function connectionKey(): Promise<ConnectionKey> {
  const raw = new Uint8Array(42);
  raw.set([0x9a, 0x5c, 0x13, 1]);
  new DataView(raw.buffer).setUint16(4, port);
  raw.set(bytes(link.psk ?? ""), 6);
  const sum = await crypto.subtle.digest("SHA-256", raw.subarray(0, 38));
  raw.set(new Uint8Array(sum, 0, 4), 38);
  return ConnectionKey.parse(base64urlnopad.encode(raw));
}

async function linkingClient(connect: SocketFactory) {
  const store = new MemoryStore();
  const language = new MemoryLanguage();
  const signIn = new MemorySignInStyle();
  const staticKeys = await importKeyPair(link.initiatorStatic);
  const client = new LinkClient({
    store,
    language,
    signIn,
    connect,
    keys: new FixedKeys([
      staticKeys,
      await importKeyPair(link.initiatorEphemeral),
    ]),
  });
  return {
    store,
    language,
    signIn,
    staticKeys,
    client,
    key: await connectionKey(),
  };
}

// Starts with English and "field"; the session vector reports Russian and "card".
async function linkedClient(connect: SocketFactory) {
  const store = new MemoryStore({
    port,
    desktopPublicKey: bytes(session.responderStatic.public),
    staticKeys: await importKeyPair(session.initiatorStatic),
    linkedAt,
  });
  const language = new MemoryLanguage("en");
  const signIn = new MemorySignInStyle("field");
  const client = new LinkClient({
    store,
    language,
    signIn,
    connect,
    keys: new FixedKeys([await importKeyPair(session.initiatorEphemeral)]),
  });
  return { store, language, signIn, client };
}

function unlinkedClient(connect: SocketFactory) {
  return new LinkClient({
    store: new MemoryStore(),
    language: new MemoryLanguage(),
    signIn: new MemorySignInStyle(),
    connect,
    keys: new FixedKeys([]),
  });
}

async function desktopTransport(): Promise<Transport> {
  const handshake = await HandshakeState.session(
    {
      staticKeys: await importKeyPair(session.initiatorStatic),
      ephemeralKeys: new FixedKeys([
        await importKeyPair(session.initiatorEphemeral),
      ]),
    },
    bytes(session.responderStatic.public),
  );
  await handshake.writeMessage(new Uint8Array(0));
  await handshake.readMessage(frame(session, 1));
  const extension = handshake.transport();
  return new Transport(extension.receive, extension.send);
}

function failedFor(reason: LinkError["reason"]) {
  return (error: unknown) =>
    error instanceof LinkError && error.reason === reason;
}

function refusedFor(reason: SessionFailure) {
  return (error: unknown) =>
    error instanceof SessionError && error.reason === reason;
}

// Each reply is JSON text or raw bytes, sealed in turn, or a silence.
async function serving(
  ...replies: readonly (string | Bytes | typeof silence)[]
) {
  const transport = await desktopTransport();
  const frames: Reply[] = [frame(session, 1)];
  for (const reply of replies) {
    frames.push(
      reply === silence
        ? reply
        : await transport.seal(
            typeof reply === "string" ? new TextEncoder().encode(reply) : reply,
          ),
    );
  }
  const desktop = new ScriptedDesktop(frames);
  const { store, language, signIn, client } = await linkedClient(
    desktop.connect,
  );
  const sent = async () =>
    new TextDecoder().decode(await transport.open(sentFrame(desktop, 1)));
  return { desktop, store, language, signIn, client, sent };
}

test("linking keeps the link, Ravenpass's language and its sign-in style once Ravenpass confirms it", async () => {
  const desktop = new ScriptedDesktop([frame(link, 1), frame(link, 3)]);
  const { store, language, signIn, staticKeys, client, key } =
    await linkingClient(desktop.connect);
  const before = Date.now();

  await client.link(key, name);

  assert.deepEqual(desktop.urls, [`ws://127.0.0.1:${port}/v1/link`]);
  assert.deepEqual(desktop.sent, [frame(link, 0), frame(link, 2)]);
  assert.equal(desktop.closed, 1);
  assert.equal(store.record?.port, port);
  assert.deepEqual(
    store.record?.desktopPublicKey,
    bytes(link.responderStatic.public),
  );
  assert.equal(store.record?.staticKeys, staticKeys);
  const kept = store.record?.linkedAt ?? 0;
  assert.ok(kept >= before && kept <= Date.now());
  assert.equal(language.language, "ru");
  assert.equal(signIn.style, "field");
});

test("linking keeps nothing until Ravenpass confirms the link", async () => {
  for (const confirmation of [{ close: 1000 }, altered(frame(link, 3))]) {
    const desktop = new ScriptedDesktop([frame(link, 1), confirmation]);
    const { store, language, signIn, client, key } = await linkingClient(
      desktop.connect,
    );
    await assert.rejects(client.link(key, name), failedFor("failed"));
    assert.equal(store.record, undefined);
    assert.equal(language.language, undefined);
    assert.equal(signIn.style, undefined);
  }
});

test("a link confirmation that names no sign-in style keeps nothing", async () => {
  const handshake = await HandshakeState.link(
    {
      staticKeys: await importKeyPair(link.initiatorStatic),
      ephemeralKeys: new FixedKeys([
        await importKeyPair(link.initiatorEphemeral),
      ]),
    },
    bytes(link.psk ?? ""),
  );
  await handshake.writeMessage(bytes(message(link, 0).payload));
  await handshake.readMessage(frame(link, 1));
  await handshake.writeMessage(bytes(message(link, 2).payload));
  const extension = handshake.transport();
  const desktopSide = new Transport(extension.receive, extension.send);
  const confirmation = await desktopSide.seal(
    new TextEncoder().encode('{"type":"linked","language":"ru"}'),
  );
  const desktop = new ScriptedDesktop([frame(link, 1), confirmation]);
  const { store, language, signIn, client, key } = await linkingClient(
    desktop.connect,
  );

  await assert.rejects(client.link(key, name), failedFor("failed"));
  assert.equal(store.record, undefined);
  assert.equal(language.language, undefined);
  assert.equal(signIn.style, undefined);
});

test("a key that expired, was spent or was replaced keeps nothing", async () => {
  for (const replies of [
    [{ close: 4404 }],
    [frame(link, 1), { close: 4403 }],
  ]) {
    const desktop = new ScriptedDesktop(replies);
    const { store, client, key } = await linkingClient(desktop.connect);
    await assert.rejects(client.link(key, name), failedFor("expired"));
    assert.equal(store.record, undefined);
  }
});

test("linking while Ravenpass is not open keeps nothing", async () => {
  const { store, client, key } = await linkingClient(refused);
  await assert.rejects(client.link(key, name), failedFor("not-open"));
  assert.equal(store.record, undefined);
});

test("status reports Ravenpass unlocked and keeps the language and the sign-in style its session names", async () => {
  const desktop = new ScriptedDesktop([frame(session, 1), frame(session, 3)]);
  const { language, signIn, client } = await linkedClient(desktop.connect);

  assert.deepEqual(await client.status(), {
    linked: true,
    desktop: "unlocked",
    linkedAt,
  });
  assert.deepEqual(desktop.urls, [`ws://127.0.0.1:${port}/v1/session`]);
  assert.deepEqual(desktop.sent, [frame(session, 0), frame(session, 2)]);
  assert.equal(desktop.closed, 1);
  assert.equal(language.language, "ru");
  assert.equal(signIn.style, "card");
});

test("a request Ravenpass refuses still keeps the language and the sign-in style its session names", async () => {
  const { language, signIn, client } = await serving(
    '{"id":1,"error":"locked"}',
  );

  await assert.rejects(
    client.suggest("https://github.com", "sign-in"),
    refusedFor("locked"),
  );
  assert.equal(language.language, "ru");
  assert.equal(signIn.style, "card");
});

test("status reports Ravenpass locked", async () => {
  const transport = await desktopTransport();
  const locked = new TextEncoder().encode(
    '{"id":1,"result":{"vault":"locked"}}',
  );
  const desktop = new ScriptedDesktop([
    frame(session, 1),
    await transport.seal(locked),
  ]);
  const { client } = await linkedClient(desktop.connect);

  assert.deepEqual(await client.status(), {
    linked: true,
    desktop: "locked",
    linkedAt,
  });
});

test("Ravenpass no longer knowing the extension ends the link and forgets its language and sign-in style", async () => {
  const desktop = new ScriptedDesktop([{ close: 4401 }]);
  const { store, language, signIn, client } = await linkedClient(
    desktop.connect,
  );

  assert.deepEqual(await client.status(), {
    linked: false,
    unlinkedInRavenpass: true,
  });
  assert.equal(store.record, undefined);
  assert.equal(language.language, undefined);
  assert.equal(signIn.style, undefined);
});

test("a peer that fails the handshake reads as Ravenpass not open, and the link stays", async () => {
  for (const connect of [
    refused,
    new ScriptedDesktop([{ close: 4403 }]).connect,
    new ScriptedDesktop([altered(frame(session, 1))]).connect,
    new ScriptedDesktop([frame(session, 1), { close: 1011 }]).connect,
  ]) {
    const { store, client } = await linkedClient(connect);
    assert.deepEqual(await client.status(), {
      linked: true,
      desktop: "not-open",
      linkedAt,
    });
    assert.equal(store.record?.linkedAt, linkedAt);
  }
});

test("status without a link asks Ravenpass nothing", async () => {
  const desktop = new ScriptedDesktop([]);
  const client = unlinkedClient(desktop.connect);

  assert.deepEqual(await client.status(), {
    linked: false,
    unlinkedInRavenpass: false,
  });
  assert.deepEqual(desktop.urls, []);
});

test("unlinking asks Ravenpass to forget the extension and forgets the link, its language and its sign-in style", async () => {
  const transport = await desktopTransport();
  const done = new TextEncoder().encode('{"id":1,"result":{}}');
  const desktop = new ScriptedDesktop([
    frame(session, 1),
    await transport.seal(done),
  ]);
  const { store, language, signIn, client } = await linkedClient(
    desktop.connect,
  );

  await client.unlink();

  assert.equal(store.record, undefined);
  assert.equal(language.language, undefined);
  assert.equal(signIn.style, undefined);
  assert.deepEqual(desktop.sent[0], frame(session, 0));
  assert.equal(
    new TextDecoder().decode(await transport.open(sentFrame(desktop, 1))),
    '{"id":1,"type":"unlink"}',
  );
});

test("unlinking forgets the link, its language and its sign-in style when Ravenpass is not open", async () => {
  const { store, language, signIn, client } = await linkedClient(refused);

  await client.unlink();

  assert.equal(store.record, undefined);
  assert.equal(language.language, undefined);
  assert.equal(signIn.style, undefined);
});

test("suggest asks for the credentials matching the origin", async () => {
  const { desktop, client, sent } = await serving(
    JSON.stringify({
      id: 1,
      result: {
        credentials: [
          {
            id: "a1",
            label: "GitHub",
            account: "alex",
            site: "github.com",
            exact: true,
            tags: ["Work"],
          },
          {
            id: "b2",
            label: "",
            account: "",
            site: "github.com",
            exact: false,
          },
        ],
      },
    }),
  );

  assert.deepEqual(await client.suggest("https://gist.github.com", "sign-in"), [
    {
      id: "a1",
      label: "GitHub",
      account: "alex",
      site: "github.com",
      exact: true,
      tags: ["Work"],
    },
    // An older Ravenpass sends no tags.
    {
      id: "b2",
      label: "",
      account: "",
      site: "github.com",
      exact: false,
      tags: [],
    },
  ]);
  assert.deepEqual(desktop.urls, [`ws://127.0.0.1:${port}/v1/session`]);
  assert.deepEqual(desktop.sent[0], frame(session, 0));
  assert.equal(
    await sent(),
    '{"id":1,"type":"suggest","origin":"https://gist.github.com","purpose":"sign-in"}',
  );
  assert.equal(desktop.closed, 1);
});

test("suggest asks for the credentials with a one-time code for a code field", async () => {
  const { client, sent } = await serving(
    '{"id":1,"result":{"credentials":[{"id":"a1","label":"GitHub","account":"alex","site":"github.com","exact":true,"digits":6,"period":30}]}}',
  );

  assert.deepEqual(await client.suggest("https://github.com", "code"), [
    {
      id: "a1",
      label: "GitHub",
      account: "alex",
      site: "github.com",
      exact: true,
      tags: [],
      digits: 6,
      period: 30,
    },
  ]);
  assert.equal(
    await sent(),
    '{"id":1,"type":"suggest","origin":"https://github.com","purpose":"code"}',
  );
});

test("a code suggestion without a whole number of digits and seconds is refused", async () => {
  for (const face of [
    "",
    ',"digits":6',
    ',"period":30',
    ',"digits":0,"period":30',
    ',"digits":6,"period":0.5',
    ',"digits":"6","period":30',
  ]) {
    const { client } = await serving(
      `{"id":1,"result":{"credentials":[{"id":"a1","label":"GitHub","account":"alex","site":"github.com","exact":true${face}}]}}`,
    );
    await assert.rejects(
      client.suggest("https://github.com", "code"),
      refusedFor("failed"),
    );
  }
});

test("fill asks for one credential's values with the origin", async () => {
  const { client, sent } = await serving(
    '{"id":1,"result":{"login":"alex","email":"alex@example.com","password":"correct horse"}}',
  );

  assert.deepEqual(await client.fill("a1", "https://github.com"), {
    login: "alex",
    email: "alex@example.com",
    password: "correct horse",
  });
  assert.equal(
    await sent(),
    '{"id":1,"type":"fill","credential":"a1","origin":"https://github.com"}',
  );
});

test("code asks for one credential's current one-time code with the origin", async () => {
  const { client, sent } = await serving(
    '{"id":1,"result":{"code":"492039","digits":6,"period":30,"expiresAt":1750000030000}}',
  );

  assert.deepEqual(await client.code("a1", "https://github.com"), {
    code: "492039",
    digits: 6,
    period: 30,
    expiresAt: 1_750_000_030_000,
  });
  assert.equal(
    await sent(),
    '{"id":1,"type":"code","credential":"a1","origin":"https://github.com"}',
  );
});

test("addWebsite asks Ravenpass to add the origin to one credential", async () => {
  const { client, sent } = await serving('{"id":1,"result":{}}');

  await client.addWebsite("a1", "https://secure.github.com");

  assert.equal(
    await sent(),
    '{"id":1,"type":"add-website","credential":"a1","origin":"https://secure.github.com"}',
  );
});

test("a fill or a code waiting for the person reports each step, then answers", async () => {
  const fillProgress = progressLog();
  const filling = await serving(
    '{"id":1,"progress":"confirm-on-device"}',
    '{"id":1,"result":{"login":"alex","email":"","password":"correct horse"}}',
  );
  assert.deepEqual(
    await filling.client.fill("a1", "https://github.com", fillProgress.listen),
    { login: "alex", email: "", password: "correct horse" },
  );
  assert.deepEqual(fillProgress.steps, ["confirm-on-device"]);

  const codeProgress = progressLog();
  const coding = await serving(
    '{"id":1,"progress":"confirm-in-ravenpass"}',
    '{"id":1,"result":{"code":"492039","digits":6,"period":30,"expiresAt":1750000030000}}',
  );
  assert.equal(
    (await coding.client.code("a1", "https://github.com", codeProgress.listen))
      .code,
    "492039",
  );
  assert.deepEqual(codeProgress.steps, ["confirm-in-ravenpass"]);
});

test("a fill or a code the person does not confirm rejects with its reason", async () => {
  for (const code of ["declined", "unverifiable"] as const) {
    const filling = await serving(
      '{"id":1,"progress":"confirm-on-device"}',
      `{"id":1,"error":"${code}"}`,
    );
    await assert.rejects(
      filling.client.fill("a1", "https://github.com", () => {}),
      refusedFor(code),
    );
    const coding = await serving(`{"id":1,"error":"${code}"}`);
    await assert.rejects(
      coding.client.code("a1", "https://github.com", () => {}),
      refusedFor(code),
    );
  }
});

test("icon asks for a site's icon", async () => {
  const { client, sent } = await serving(
    '{"id":1,"result":{"image":"iVBORw0KGgo=","tint":"#24292f"}}',
  );

  assert.deepEqual(await client.icon("github.com"), {
    image: "iVBORw0KGgo=",
    tint: "#24292f",
  });
  assert.equal(await sent(), '{"id":1,"type":"icon","site":"github.com"}');
});

test("unlock asks Ravenpass to bring its window forward", async () => {
  const { client, sent } = await serving('{"id":1,"result":{}}');

  await client.unlock();

  assert.equal(await sent(), '{"id":1,"type":"unlock"}');
});

test("each of Ravenpass's error codes rejects with its reason, and the link stays", async () => {
  for (const code of [
    "locked",
    "no-match",
    "not-found",
    "invalid-origin",
    "unknown-request",
  ] as const) {
    const { store, client } = await serving(`{"id":1,"error":"${code}"}`);
    await assert.rejects(
      client.fill("a1", "https://github.com"),
      refusedFor(code),
    );
    assert.equal(store.record?.linkedAt, linkedAt);
  }
});

test("a code request rejects with each reason Ravenpass gives for it", async () => {
  for (const code of [
    "locked",
    "no-match",
    "not-found",
    "no-code",
    "invalid-origin",
  ] as const) {
    const { client } = await serving(`{"id":1,"error":"${code}"}`);
    await assert.rejects(
      client.code("a1", "https://github.com"),
      refusedFor(code),
    );
  }
});

test("a purpose Ravenpass does not know rejects the suggestions", async () => {
  const { client } = await serving('{"id":1,"error":"invalid-purpose"}');
  await assert.rejects(
    client.suggest("https://github.com", "code"),
    refusedFor("invalid-purpose"),
  );
});

test("an unknown error code or an unreadable result rejects as failed", async () => {
  const cases: [string, (client: LinkClient) => Promise<unknown>][] = [
    ['{"id":1,"error":"out-of-cheese"}', (client) => client.unlock()],
    [
      '{"id":1,"result":{"credentials":[{"id":"a1","label":"GitHub"}]}}',
      (client) => client.suggest("https://github.com", "sign-in"),
    ],
    [
      '{"id":1,"result":{}}',
      (client) => client.suggest("https://github.com", "sign-in"),
    ],
    [
      '{"id":1,"result":{"login":"alex","email":""}}',
      (client) => client.fill("a1", "https://github.com"),
    ],
    [
      '{"id":1,"result":{"code":"492039","digits":6,"period":30}}',
      (client) => client.code("a1", "https://github.com"),
    ],
    [
      '{"id":1,"result":{"code":492039,"digits":6,"period":30,"expiresAt":1750000030000}}',
      (client) => client.code("a1", "https://github.com"),
    ],
    ['{"id":1,"result":{"image":""}}', (client) => client.icon("github.com")],
  ];
  for (const [reply, call] of cases) {
    const { client } = await serving(reply);
    await assert.rejects(call(client), refusedFor("failed"));
  }
});

const origin = "https://example.com";

const identity: IdentityFiles = {
  id: "i1",
  label: "Алекс",
  thumbnail: "/9j/4AAQ",
  files: [
    {
      id: "photo",
      kind: "photo",
      name: "Алекс.jpg",
      mediaType: "image/jpeg",
      thumbnail: "/9j/4AAR",
    },
    {
      id: "s1",
      kind: "scan",
      name: "passport.pdf",
      mediaType: "application/pdf",
      document: { type: "passport", label: "" },
      thumbnail: "",
    },
  ],
};

function progressLog() {
  const steps: ShareProgress[] = [];
  return { steps, listen: (step: ShareProgress) => steps.push(step) };
}

test("identities asks for the identities whose files the origin may receive", async () => {
  const { client, sent } = await serving(
    JSON.stringify({ id: 1, result: { identities: [identity] } }),
  );

  assert.deepEqual(await client.identities(origin), [identity]);
  assert.equal(
    await sent(),
    '{"id":1,"type":"identities","origin":"https://example.com"}',
  );
});

test("a result too large for one message is joined from its parts before it is read", async () => {
  const json = new TextEncoder().encode(
    JSON.stringify({ identities: [identity] }),
  );
  // The split falls inside a two-byte character.
  const split = json.indexOf(0xd0) + 1;

  const { client } = await serving(
    '{"id":1,"parts":2}',
    json.slice(0, split),
    json.slice(split),
  );

  assert.deepEqual(await client.identities(origin), [identity]);
});

test("share reports each step Ravenpass asks of the person, then receives the file in parts", async () => {
  const progress = progressLog();
  const { client, sent } = await serving(
    '{"id":1,"progress":"confirm-on-device"}',
    '{"id":1,"progress":"confirm-on-device"}',
    '{"id":1,"result":{"name":"passport.jpg","mediaType":"image/jpeg","size":5},"parts":2}',
    new Uint8Array([0xff, 0xd8, 0xff]),
    new Uint8Array([0xd9, 0x00]),
  );

  assert.deepEqual(await client.share("i1", "s1", origin, progress.listen), {
    name: "passport.jpg",
    mediaType: "image/jpeg",
    content: new Uint8Array([0xff, 0xd8, 0xff, 0xd9, 0x00]),
  });
  assert.deepEqual(progress.steps, ["confirm-on-device", "confirm-on-device"]);
  assert.equal(
    await sent(),
    '{"id":1,"type":"share","identity":"i1","file":"s1","origin":"https://example.com"}',
  );
});

test("a share Ravenpass refuses after asking the person rejects with its reason", async () => {
  const progress = progressLog();
  const { client } = await serving(
    '{"id":1,"progress":"confirm-in-ravenpass"}',
    '{"id":1,"error":"declined"}',
  );

  await assert.rejects(
    client.share("i1", "photo", origin, progress.listen),
    refusedFor("declined"),
  );
  assert.deepEqual(progress.steps, ["confirm-in-ravenpass"]);
});

test("a share rejects with each reason Ravenpass gives for it, and the link stays", async () => {
  for (const code of [
    "locked",
    "not-found",
    "declined",
    "unverifiable",
    "invalid-origin",
  ] as const) {
    const { store, client } = await serving(`{"id":1,"error":"${code}"}`);
    await assert.rejects(
      client.share("i1", "s1", origin, () => {}),
      refusedFor(code),
    );
    assert.equal(store.record?.linkedAt, linkedAt);
  }
});

test("an identities request rejects with each reason Ravenpass gives for it", async () => {
  for (const code of ["locked", "invalid-origin"] as const) {
    const { client } = await serving(`{"id":1,"error":"${code}"}`);
    await assert.rejects(client.identities(origin), refusedFor(code));
  }
});

test("a frame that does not arrive in time between progress frames rejects as not open", async () => {
  const progress = progressLog();
  const { store, client } = await serving(
    '{"id":1,"progress":"confirm-on-device"}',
    silence,
  );

  await assert.rejects(
    client.share("i1", "s1", origin, progress.listen),
    refusedFor("not-open"),
  );
  assert.deepEqual(progress.steps, ["confirm-on-device"]);
  assert.equal(store.record?.linkedAt, linkedAt);
});

test("a file whose parts do not add up to its size, or that comes without parts, rejects as failed", async () => {
  for (const replies of [
    [
      '{"id":1,"result":{"name":"a.jpg","mediaType":"image/jpeg","size":4},"parts":1}',
      new Uint8Array([1, 2, 3]),
    ],
    ['{"id":1,"result":{"name":"a.jpg","mediaType":"image/jpeg","size":0}}'],
  ]) {
    const { client } = await serving(...replies);
    await assert.rejects(
      client.share("i1", "s1", origin, () => {}),
      refusedFor("failed"),
    );
  }
});

test("an unreadable identity listing rejects as failed", async () => {
  const [photo, scan] = identity.files;
  for (const files of [
    [{ ...scan, document: undefined }],
    [{ ...scan, document: { type: "visa", label: "" } }],
    [{ ...photo, document: { type: "passport", label: "" } }],
    [{ ...photo, kind: "video" }],
    [{ ...photo, thumbnail: null }],
  ]) {
    const { client } = await serving(
      JSON.stringify({
        id: 1,
        result: { identities: [{ ...identity, files }] },
      }),
    );
    await assert.rejects(client.identities(origin), refusedFor("failed"));
  }
});

test("a head frame announcing an unreadable number of parts rejects as not open", async () => {
  for (const parts of ["-1", "1.5", '"2"']) {
    const { client } = await serving(`{"id":1,"parts":${parts}}`);
    await assert.rejects(client.identities(origin), refusedFor("not-open"));
  }
});

test("a request Ravenpass answers with close 4401 ends the link and forgets its language", async () => {
  const desktop = new ScriptedDesktop([{ close: 4401 }]);
  const { store, language, client } = await linkedClient(desktop.connect);

  await assert.rejects(
    client.suggest("https://github.com", "sign-in"),
    refusedFor("unlinked"),
  );
  assert.equal(store.record, undefined);
  assert.equal(language.language, undefined);
});

test("a request Ravenpass does not answer rejects as not open, and the link stays", async () => {
  for (const connect of [
    refused,
    new ScriptedDesktop([{ close: 4403 }]).connect,
    new ScriptedDesktop([altered(frame(session, 1))]).connect,
    new ScriptedDesktop([frame(session, 1), { close: 4400 }]).connect,
  ]) {
    const { store, client } = await linkedClient(connect);
    await assert.rejects(
      client.suggest("https://github.com", "sign-in"),
      refusedFor("not-open"),
    );
    assert.equal(store.record?.linkedAt, linkedAt);
  }
});

test("a request without a link asks Ravenpass nothing", async () => {
  const desktop = new ScriptedDesktop([]);
  const client = unlinkedClient(desktop.connect);

  await assert.rejects(client.unlock(), refusedFor("unlinked"));
  assert.deepEqual(desktop.urls, []);
});

const readyOffer = {
  kind: "password",
  state: "ready",
  pending: "p1",
  site: "github.com",
  account: "alex",
  name: "github.com",
  card: null,
  targets: [
    {
      credential: "a1",
      label: "GitHub",
      account: "alex",
      action: "update",
      tags: ["Work"],
    },
    {
      credential: "b2",
      label: "GitLab",
      account: "alex@example.com",
      action: "add-site",
      tags: [],
    },
  ],
  suggested: "a1",
};

test("capture sends the typed password with the frame's origin and reads the offer", async () => {
  const { client, sent } = await serving(
    JSON.stringify({ id: 1, result: readyOffer }),
  );

  assert.deepEqual(
    await client.capture("https://github.com", {
      account: "alex",
      password: "correct horse",
      current: "old horse",
    }),
    readyOffer,
  );
  assert.equal(
    await sent(),
    '{"id":1,"type":"capture","origin":"https://github.com","account":"alex","password":"correct horse","current":"old horse"}',
  );
});

test("capture without a current password sends none, and reads nothing to offer", async () => {
  const { client, sent } = await serving(
    '{"id":1,"result":{"state":"none","site":"github.com","account":"","name":"github.com","targets":[],"suggested":""}}',
  );

  assert.deepEqual(
    await client.capture("https://github.com", {
      account: "",
      password: "correct horse",
    }),
    { state: "none" },
  );
  assert.equal(
    await sent(),
    '{"id":1,"type":"capture","origin":"https://github.com","account":"","password":"correct horse"}',
  );
});

test("a capture Ravenpass keeps while locked answers locked with its pending id", async () => {
  const { client } = await serving(
    '{"id":1,"result":{"kind":"password","state":"locked","pending":"p1","site":"github.com","account":"alex","name":"github.com","targets":[],"suggested":""}}',
  );

  assert.deepEqual(
    await client.capture("https://github.com", {
      account: "alex",
      password: "correct horse",
    }),
    {
      kind: "password",
      state: "locked",
      pending: "p1",
      site: "github.com",
      account: "alex",
      name: "github.com",
      card: null,
      targets: [],
      suggested: "",
    },
  );
});

test("review asks what Ravenpass offers now for a pending capture", async () => {
  const { client, sent } = await serving(
    JSON.stringify({ id: 1, result: readyOffer }),
  );

  assert.deepEqual(await client.review("p1"), readyOffer);
  assert.equal(await sent(), '{"id":1,"type":"review","pending":"p1"}');
});

test("an offer from an older Ravenpass reads as an untagged password offer", async () => {
  const { kind: _, card: __, ...older } = readyOffer;
  const targets = older.targets.map(({ tags: _, ...target }) => target);
  const { client } = await serving(
    JSON.stringify({ id: 1, result: { ...older, targets } }),
  );

  assert.deepEqual(await client.review("p1"), {
    ...readyOffer,
    targets: targets.map((target) => ({ ...target, tags: [] })),
  });
});

test("a card capture sends the typed card and reads the card's offer", async () => {
  const cardOffer = {
    kind: "card",
    state: "ready",
    pending: "p2",
    site: "shop.example.com",
    account: "",
    name: "",
    card: { network: "visa", lastFour: "1111" },
    targets: [
      {
        credential: "c3",
        label: "Travel",
        account: "",
        action: "update",
        tags: [],
      },
    ],
    suggested: "c3",
  };
  const { client, sent } = await serving(
    JSON.stringify({ id: 1, result: cardOffer }),
  );
  const card = {
    holder: "Alex Example",
    number: "4111111111111111",
    expiry: "2032-01",
    securityCode: "739",
    network: "visa",
  };
  assert.deepEqual(await client.captureCard(origin, card), cardOffer);
  assert.deepEqual(JSON.parse(await sent()), {
    id: 1,
    type: "card-capture",
    origin,
    ...card,
  });

  const unknown = await serving(
    JSON.stringify({
      id: 1,
      result: { ...cardOffer, card: { network: "amex", lastFour: "1111" } },
    }),
  );
  await assert.rejects(unknown.client.captureCard(origin, card), {
    name: "SessionError",
    reason: "failed",
  });
});

test("cards list the index faces and a card fill reads its values", async () => {
  const option = {
    id: "c3",
    label: "Travel",
    bankName: "Example Bank",
    site: "bank.example",
    network: "visa",
    lastFour: "1111",
    color: "#00a0e1",
    expiresOn: "2029-08-31",
  };
  const listing = await serving(
    JSON.stringify({ id: 1, result: { cards: [option] } }),
  );
  assert.deepEqual(await listing.client.cards(origin), [option]);
  assert.equal(
    await listing.sent(),
    `{"id":1,"type":"cards","origin":"${origin}"}`,
  );

  const values = {
    holder: "Alex Example",
    number: "4111111111111111",
    expiry: "2029-08",
    securityCode: "739",
    billing: {
      street: "1 Example Street",
      city: "Springfield",
      region: "",
      postalCode: "12345",
      country: "US",
    },
  };
  const fill = await serving(
    JSON.stringify({ id: 1, result: { ...values, pin: "4829" } }),
  );
  assert.deepEqual(await fill.client.fillCard("c3", origin, () => {}), values);
  assert.equal(
    await fill.sent(),
    `{"id":1,"type":"card-fill","card":"c3","origin":"${origin}"}`,
  );
});

test("save sends where the person chose, with the account and name of a new credential", async () => {
  for (const [choice, saved, request] of [
    [
      { target: "", account: "alex@example.com", name: "GitHub" },
      "created",
      '{"id":1,"type":"save","pending":"p1","target":"","account":"alex@example.com","name":"GitHub"}',
    ],
    [
      { target: "a1", account: "alex", name: "github.com" },
      "updated",
      '{"id":1,"type":"save","pending":"p1","target":"a1","account":"alex","name":"github.com"}',
    ],
  ] as const) {
    const { client, sent } = await serving(
      `{"id":1,"result":{"saved":"${saved}"}}`,
    );
    assert.equal(await client.save("p1", choice), saved);
    assert.equal(await sent(), request);
  }
});

test("discard asks Ravenpass to forget a pending capture", async () => {
  const { client, sent } = await serving('{"id":1,"result":{}}');

  await client.discard("p1");

  assert.equal(await sent(), '{"id":1,"type":"discard","pending":"p1"}');
});

test("the requests about a capture reject with each reason Ravenpass gives for them", async () => {
  const choice = { target: "", account: "alex", name: "github.com" };
  const cases: [string, (client: LinkClient) => Promise<unknown>][] = [
    [
      "invalid-origin",
      (client) =>
        client.capture("chrome://settings", { account: "", password: "x" }),
    ],
    ["not-found", (client) => client.review("p1")],
    ["locked", (client) => client.save("p1", choice)],
    ["not-found", (client) => client.save("p1", choice)],
    ["invalid-origin", (client) => client.save("p1", choice)],
    ["invalid-account", (client) => client.save("p1", choice)],
    ["invalid-name", (client) => client.save("p1", choice)],
  ];
  for (const [code, call] of cases) {
    const { store, client } = await serving(`{"id":1,"error":"${code}"}`);
    await assert.rejects(call(client), refusedFor(code as SessionFailure));
    assert.equal(store.record?.linkedAt, linkedAt);
  }
});

test("an unreadable offer or save rejects as failed", async () => {
  const { pending: _, ...withoutPending } = readyOffer;
  const [update] = readyOffer.targets;
  for (const result of [
    { ...readyOffer, state: "maybe" },
    withoutPending,
    { ...readyOffer, name: undefined },
    { ...readyOffer, targets: undefined },
    { ...readyOffer, suggested: "c3" },
    { ...readyOffer, targets: [{ ...update, action: "replace" }] },
    { ...readyOffer, targets: [{ ...update, label: null }] },
    { state: "none" },
  ]) {
    const { client } = await serving(JSON.stringify({ id: 1, result }));
    await assert.rejects(client.review("p1"), refusedFor("failed"));
  }
  for (const reply of [
    '{"id":1,"result":{"saved":"moved"}}',
    '{"id":1,"result":{}}',
  ]) {
    const { client } = await serving(reply);
    await assert.rejects(
      client.save("p1", { target: "", account: "", name: "github.com" }),
      refusedFor("failed"),
    );
  }
});

test("a capture without a link asks Ravenpass nothing", async () => {
  const desktop = new ScriptedDesktop([]);
  const client = unlinkedClient(desktop.connect);

  await assert.rejects(
    client.capture("https://github.com", { account: "", password: "x" }),
    refusedFor("unlinked"),
  );
  assert.deepEqual(desktop.urls, []);
});

const getOptions: GetOptions = {
  rpId: "example.com",
  challenge: "AQI",
  timeout: null,
  allow: ["AQ"],
  userVerification: "discouraged",
  hints: [],
};

const createOptions: CreateOptions = {
  rpId: null,
  rpName: "Example",
  user: { id: "dXNlcg", name: "alex@example.com", displayName: "Alex" },
  challenge: "AQI",
  algorithms: [-7],
  timeout: null,
  exclude: ["CQk"],
  attachment: null,
  userVerification: "preferred",
  hints: [],
  credProps: false,
};

const passkeyOption = {
  credential: "c1",
  credentialId: "AQID",
  account: "alex@example.com",
  label: "Example",
};

const createdResult = {
  credential: "c1",
  credentialId: "AQID",
  clientDataJSON: "e30",
  attestationObject: "oA",
  authenticatorData: "BAUG",
  publicKey: "MFkwEw",
  publicKeyAlgorithm: -7,
};

const signedResult = {
  credentialId: "AQID",
  clientDataJSON: "e30",
  authenticatorData: "BAUG",
  signature: "MEUCIQ",
  userHandle: "dXNlcg",
};

test("linked tells whether a link is kept, asking Ravenpass nothing", async () => {
  const desktop = new ScriptedDesktop([]);
  const { client } = await linkedClient(desktop.connect);
  assert.equal(await client.linked(), true);
  assert.equal(await unlinkedClient(desktop.connect).linked(), false);
  assert.deepEqual(desktop.urls, []);
});

test("passkeys asks for the passkeys a sign-in at the origin can use", async () => {
  const { client, sent } = await serving(
    JSON.stringify({
      id: 1,
      result: { rpId: "example.com", passkeys: [passkeyOption] },
    }),
  );

  assert.deepEqual(await client.passkeys(origin, getOptions), [passkeyOption]);
  assert.equal(
    await sent(),
    '{"id":1,"type":"passkeys","origin":"https://example.com","mode":"get","rpId":"example.com","allow":["AQ"]}',
  );
});

test("passkeyTargets asks where a new passkey for the page's account can go", async () => {
  const targets = [
    { credential: "c1", label: "Example", account: "alex", tags: ["Work"] },
    // An older Ravenpass sends no tags.
    { credential: "c2", label: "Example", account: "sam" },
  ];
  const { client, sent } = await serving(
    JSON.stringify({
      id: 1,
      result: { rpId: "example.com", targets, excluded: true },
    }),
  );

  assert.deepEqual(await client.passkeyTargets(origin, createOptions), {
    targets: [targets[0], { ...targets[1], tags: [] }],
    excluded: true,
  });
  assert.equal(
    await sent(),
    '{"id":1,"type":"passkeys","origin":"https://example.com","mode":"create","rpId":"","account":"alex@example.com","exclude":["CQk"]}',
  );
});

test("createPasskey reports each step Ravenpass asks of the person, then answers the new passkey without the credential holding it", async () => {
  const progress = progressLog();
  const { client, sent } = await serving(
    '{"id":1,"progress":"confirm-on-device"}',
    '{"id":1,"progress":"confirm-in-ravenpass"}',
    JSON.stringify({ id: 1, result: createdResult }),
  );

  const { credential: _credential, ...created } = createdResult;
  assert.deepEqual(
    await client.createPasskey(origin, createOptions, "new", progress.listen),
    created,
  );
  assert.deepEqual(progress.steps, [
    "confirm-on-device",
    "confirm-in-ravenpass",
  ]);
  assert.equal(
    await sent(),
    '{"id":1,"type":"passkey-create","origin":"https://example.com","rpId":"","rpName":"Example","user":{"id":"dXNlcg","name":"alex@example.com","displayName":"Alex"},"challenge":"AQI","verify":true,"target":"new","exclude":["CQk"],"algorithms":[-7]}',
  );
});

test("signPasskey reports each step Ravenpass asks of the person, then answers the signed sign-in", async () => {
  const progress = progressLog();
  const { client, sent } = await serving(
    '{"id":1,"progress":"confirm-in-ravenpass"}',
    JSON.stringify({ id: 1, result: signedResult }),
  );

  assert.deepEqual(
    await client.signPasskey(
      origin,
      getOptions,
      { credential: "c1", credentialId: "AQID" },
      progress.listen,
    ),
    signedResult,
  );
  assert.deepEqual(progress.steps, ["confirm-in-ravenpass"]);
  assert.equal(
    await sent(),
    '{"id":1,"type":"passkey-sign","origin":"https://example.com","rpId":"example.com","challenge":"AQI","verify":false,"credential":"c1","credentialId":"AQID"}',
  );
});

test("each passkey request rejects with each reason Ravenpass gives, and the link stays", async () => {
  const calls: ((client: LinkClient) => Promise<unknown>)[] = [
    (client) => client.passkeys(origin, getOptions),
    (client) => client.passkeyTargets(origin, createOptions),
    (client) => client.createPasskey(origin, createOptions, "c1", () => {}),
    (client) =>
      client.signPasskey(
        origin,
        getOptions,
        { credential: "c1", credentialId: "AQID" },
        () => {},
      ),
  ];
  for (const code of [
    "locked",
    "not-found",
    "no-match",
    "declined",
    "unverifiable",
    "invalid-origin",
    "invalid-rp",
    "invalid-request",
    "excluded",
    "full",
  ] as const) {
    for (const call of calls) {
      const { store, client } = await serving(
        '{"id":1,"progress":"confirm-on-device"}',
        `{"id":1,"error":"${code}"}`,
      );
      await assert.rejects(call(client), refusedFor(code));
      assert.equal(store.record?.linkedAt, linkedAt);
    }
  }
});

test("an unreadable passkey result rejects as failed", async () => {
  const cases: [unknown, (client: LinkClient) => Promise<unknown>][] = [
    [
      { passkeys: [{ ...passkeyOption, credentialId: "?" }] },
      (client) => client.passkeys(origin, getOptions),
    ],
    [{ passkeys: null }, (client) => client.passkeys(origin, getOptions)],
    [
      { targets: [], excluded: "no" },
      (client) => client.passkeyTargets(origin, createOptions),
    ],
    [
      { targets: [{ credential: "c1" }], excluded: false },
      (client) => client.passkeyTargets(origin, createOptions),
    ],
    [
      { ...createdResult, publicKeyAlgorithm: null },
      (client) => client.createPasskey(origin, createOptions, "new", () => {}),
    ],
    [
      { ...signedResult, signature: undefined },
      (client) =>
        client.signPasskey(
          origin,
          getOptions,
          { credential: "c1", credentialId: "AQID" },
          () => {},
        ),
    ],
  ];
  for (const [result, call] of cases) {
    const { client } = await serving(JSON.stringify({ id: 1, result }));
    await assert.rejects(call(client), refusedFor("failed"));
  }
});

test("results keep only the fields the extension reads", async () => {
  const fill = await serving(
    '{"id":1,"result":{"login":"alex","email":"","password":"x","totp":"secret"}}',
  );
  assert.deepEqual(await fill.client.fill("a1", origin), {
    login: "alex",
    email: "",
    password: "x",
  });

  const listing = await serving(
    JSON.stringify({
      id: 1,
      result: {
        identities: [
          {
            ...identity,
            notes: "private",
            files: identity.files.map((file) => ({ ...file, size: 9 })),
          },
        ],
      },
    }),
  );
  assert.deepEqual(await listing.client.identities(origin), [identity]);

  const offer = await serving(
    JSON.stringify({ id: 1, result: { ...readyOffer, password: "secret" } }),
  );
  assert.deepEqual(await offer.client.review("p1"), readyOffer);

  const none = await serving(
    '{"id":1,"result":{"state":"none","pending":"p1","site":"github.com","account":"","name":"github.com","targets":[],"suggested":"c3"}}',
  );
  assert.deepEqual(await none.client.review("p1"), { state: "none" });
});

test("one malformed item rejects the whole list", async () => {
  const good = {
    id: "a1",
    label: "GitHub",
    account: "alex",
    site: "github.com",
    exact: true,
  };
  const cases: [unknown, (client: LinkClient) => Promise<unknown>][] = [
    [
      { credentials: [good, { ...good, exact: "true" }] },
      (client) => client.suggest(origin, "sign-in"),
    ],
    [
      { passkeys: [passkeyOption, { ...passkeyOption, account: null }] },
      (client) => client.passkeys(origin, getOptions),
    ],
    [
      {
        targets: [{ credential: "c1", label: "", account: "" }, {}],
        excluded: false,
      },
      (client) => client.passkeyTargets(origin, createOptions),
    ],
    [
      {
        identities: [
          identity,
          { ...identity, files: [...identity.files, { id: "s2" }] },
        ],
      },
      (client) => client.identities(origin),
    ],
    [
      {
        state: "none",
        site: "github.com",
        account: "",
        name: "github.com",
        targets: [{ ...readyOffer.targets[0], action: "move" }],
        suggested: "",
      },
      (client) => client.review("p1"),
    ],
  ];
  for (const [result, call] of cases) {
    const { client } = await serving(JSON.stringify({ id: 1, result }));
    await assert.rejects(call(client), refusedFor("failed"));
  }
});

test("a missing, null or mistyped field rejects the result", async () => {
  const cases: [string, (client: LinkClient) => Promise<unknown>][] = [
    [
      '{"id":1,"result":{"login":null,"email":"","password":"x"}}',
      (client) => client.fill("a1", origin),
    ],
    [
      '{"id":1,"result":{"image":"","tint":null}}',
      (client) => client.icon("github.com"),
    ],
    [
      '{"id":1,"result":{"targets":[]}}',
      (client) => client.passkeyTargets(origin, createOptions),
    ],
    [
      '{"id":1,"result":{"credentials":[{"id":"a1","label":"","account":"","site":"","exact":true,"digits":1e999,"period":30}]}}',
      (client) => client.suggest(origin, "code"),
    ],
    [
      JSON.stringify({
        id: 1,
        result: {
          identities: [
            { ...identity, files: [{ ...identity.files[0], document: null }] },
          ],
        },
      }),
      (client) => client.identities(origin),
    ],
    [
      '{"id":1,"result":{"state":"ready","pending":null,"site":"","account":"","name":"","targets":[],"suggested":""}}',
      (client) => client.review("p1"),
    ],
  ];
  for (const [reply, call] of cases) {
    const { client } = await serving(reply);
    await assert.rejects(call(client), refusedFor("failed"));
  }
  const { client } = await serving(
    '{"id":1,"result":{"name":"a.jpg","mediaType":"image/jpeg","size":"1"},"parts":1}',
    new Uint8Array([1]),
  );
  await assert.rejects(
    client.share("i1", "s1", origin, () => {}),
    refusedFor("failed"),
  );
});

test("progress Ravenpass does not name is skipped, and the request goes on", async () => {
  const progress = progressLog();
  const { client } = await serving(
    '{"id":1,"progress":"confirm-on-watch"}',
    '{"id":1,"progress":null,"error":"locked"}',
    '{"id":1,"error":7,"result":{"name":"a.jpg","mediaType":"image/jpeg","size":1},"parts":1}',
    new Uint8Array([1]),
  );

  assert.deepEqual(await client.share("i1", "s1", origin, progress.listen), {
    name: "a.jpg",
    mediaType: "image/jpeg",
    content: new Uint8Array([1]),
  });
  assert.deepEqual(progress.steps, []);
});

test("a frame for another request, or without an answer, rejects as not open", async () => {
  for (const reply of [
    '{"id":2,"result":{}}',
    '{"id":"1","result":{}}',
    '{"result":{}}',
    '{"id":1}',
    "[]",
  ]) {
    const { client } = await serving(reply);
    await assert.rejects(client.unlock(), refusedFor("not-open"));
  }
});

test("a status Ravenpass cannot name reads as Ravenpass not open", async () => {
  for (const reply of [
    '{"id":1,"result":{"vault":"open"}}',
    '{"id":1,"result":{}}',
    '{"id":1,"error":"locked"}',
  ]) {
    const { client } = await serving(reply);
    assert.deepEqual(await client.status(), {
      linked: true,
      desktop: "not-open",
      linkedAt,
    });
  }
});

test("a passkey request without a link asks Ravenpass nothing", async () => {
  const desktop = new ScriptedDesktop([]);
  const client = unlinkedClient(desktop.connect);

  await assert.rejects(
    client.passkeys(origin, getOptions),
    refusedFor("unlinked"),
  );
  assert.deepEqual(desktop.urls, []);
});
