// Noise XXpsk3/IK initiator: v1 X25519 and v2 P-256, with AESGCM/SHA256 and shared Go vectors.
import { type KeySource, keyAlgorithm, linkCurve } from "./keys.ts";

type Bytes = Uint8Array<ArrayBuffer>;
type Token = "e" | "s" | "ee" | "es" | "se" | "ss" | "psk";
type DhToken = Exclude<Token, "e" | "s" | "psk">;

/** A handshake pattern as its initiator runs it. Messages alternate, the initiator's first. */
interface HandshakePattern {
  readonly protocolName: string;
  readonly messages: readonly (readonly Token[])[];
}

// `psk` follows `s, se` in the third message, after both static keys are sent.
const linkPattern: HandshakePattern = {
  protocolName: "Noise_XXpsk3_25519_AESGCM_SHA256",
  messages: [["e"], ["e", "ee", "s", "es"], ["s", "se", "psk"]],
};

// The responder's static key is a pre-message (`<- s`): the initiator knows it from linking.
const sessionPattern: HandshakePattern = {
  protocolName: "Noise_IK_25519_AESGCM_SHA256",
  messages: [
    ["e", "es", "s", "ss"],
    ["e", "ee", "se"],
  ],
};

const hashLength = 32;
const sharedSecretLength = 32;
const tagLength = 16;
const nonceLength = 12;

/** Noise section 3: the bound on every handshake and transport message. */
const maxMessageBytes = 65535;

const empty: Bytes = new Uint8Array(0);
const prologue = new TextEncoder().encode("ravenpass-link/1");

function concat(...parts: readonly Bytes[]): Bytes {
  const joined = new Uint8Array(
    parts.reduce((length, part) => length + part.length, 0),
  );
  let offset = 0;
  for (const part of parts) {
    joined.set(part, offset);
    offset += part.length;
  }
  return joined;
}

async function sha256(data: Bytes): Promise<Bytes> {
  return new Uint8Array(await crypto.subtle.digest("SHA-256", data));
}

/** Noise section 4.3 HKDF is RFC 5869 HKDF-SHA256 with the chaining key as salt and empty info. */
function hkdf(
  chainingKey: Bytes,
  inputKeyMaterial: Bytes,
  outputs: 2,
): Promise<[Bytes, Bytes]>;
function hkdf(
  chainingKey: Bytes,
  inputKeyMaterial: Bytes,
  outputs: 3,
): Promise<[Bytes, Bytes, Bytes]>;
async function hkdf(
  chainingKey: Bytes,
  inputKeyMaterial: Bytes,
  outputs: 2 | 3,
): Promise<Bytes[]> {
  const key = await crypto.subtle.importKey(
    "raw",
    inputKeyMaterial,
    "HKDF",
    false,
    ["deriveBits"],
  );
  const bits = new Uint8Array(
    await crypto.subtle.deriveBits(
      { name: "HKDF", hash: "SHA-256", salt: chainingKey, info: empty },
      key,
      outputs * hashLength * 8,
    ),
  );
  return Array.from({ length: outputs }, (_, index) =>
    bits.slice(index * hashLength, (index + 1) * hashLength),
  );
}

async function publicBytes(key: CryptoKey): Promise<Bytes> {
  return new Uint8Array(await crypto.subtle.exportKey("raw", key));
}

/** Native ECDH validates the peer point; X25519 also rejects small-order points. */
async function dh(privateKey: CryptoKey, publicKey: Bytes): Promise<Bytes> {
  const peer = await crypto.subtle.importKey(
    "raw",
    publicKey,
    keyAlgorithm(linkCurve(privateKey)),
    true,
    [],
  );
  return new Uint8Array(
    await crypto.subtle.deriveBits(
      { name: privateKey.algorithm.name, public: peer },
      privateKey,
      sharedSecretLength * 8,
    ),
  );
}

/** AES-256-GCM; nonce is four zero bytes then a 64-bit big-endian counter; keyless passes through. */
class CipherState {
  private key: CryptoKey | null = null;
  private nonce = 0n;

  static async withKey(key: Bytes): Promise<CipherState> {
    const state = new CipherState();
    await state.initializeKey(key);
    return state;
  }

  async initializeKey(key: Bytes): Promise<void> {
    this.key = await crypto.subtle.importKey("raw", key, "AES-GCM", false, [
      "encrypt",
      "decrypt",
    ]);
    this.nonce = 0n;
  }

  hasKey(): boolean {
    return this.key !== null;
  }

  async encryptWithAd(ad: Bytes, plaintext: Bytes): Promise<Bytes> {
    if (!this.key) return plaintext;
    const iv = this.iv();
    this.nonce += 1n;
    return new Uint8Array(
      await crypto.subtle.encrypt(
        { name: "AES-GCM", iv, additionalData: ad },
        this.key,
        plaintext,
      ),
    );
  }

  /** Rejects a message that fails to authenticate and keeps the nonce for the next one. */
  async decryptWithAd(ad: Bytes, ciphertext: Bytes): Promise<Bytes> {
    if (!this.key) return ciphertext;
    const plaintext = await crypto.subtle.decrypt(
      { name: "AES-GCM", iv: this.iv(), additionalData: ad },
      this.key,
      ciphertext,
    );
    this.nonce += 1n;
    return new Uint8Array(plaintext);
  }

  private iv(): Bytes {
    const iv = new Uint8Array(nonceLength);
    new DataView(iv.buffer).setBigUint64(nonceLength - 8, this.nonce);
    return iv;
  }
}

/** A Noise SymmetricState. */
class SymmetricState {
  private readonly cipher = new CipherState();
  private chainingKey: Bytes;
  private hash: Bytes;

  private constructor(hash: Bytes) {
    this.hash = hash;
    this.chainingKey = hash;
  }

  static async initialize(protocolName: string): Promise<SymmetricState> {
    const name = new TextEncoder().encode(protocolName);
    if (name.length > hashLength) return new SymmetricState(await sha256(name));
    const padded = new Uint8Array(hashLength);
    padded.set(name);
    return new SymmetricState(padded);
  }

  hasKey(): boolean {
    return this.cipher.hasKey();
  }

  handshakeHash(): Bytes {
    return this.hash.slice();
  }

  async mixKey(inputKeyMaterial: Bytes): Promise<void> {
    const [chainingKey, key] = await hkdf(
      this.chainingKey,
      inputKeyMaterial,
      2,
    );
    this.chainingKey = chainingKey;
    await this.cipher.initializeKey(key);
  }

  async mixHash(data: Bytes): Promise<void> {
    this.hash = await sha256(concat(this.hash, data));
  }

  async mixKeyAndHash(inputKeyMaterial: Bytes): Promise<void> {
    const [chainingKey, hash, key] = await hkdf(
      this.chainingKey,
      inputKeyMaterial,
      3,
    );
    this.chainingKey = chainingKey;
    await this.mixHash(hash);
    await this.cipher.initializeKey(key);
  }

  async encryptAndHash(plaintext: Bytes): Promise<Bytes> {
    const ciphertext = await this.cipher.encryptWithAd(this.hash, plaintext);
    await this.mixHash(ciphertext);
    return ciphertext;
  }

  async decryptAndHash(ciphertext: Bytes): Promise<Bytes> {
    const plaintext = await this.cipher.decryptWithAd(this.hash, ciphertext);
    await this.mixHash(ciphertext);
    return plaintext;
  }

  /** The initiator encrypts with the first cipher state and decrypts with the second. */
  async split(): Promise<Transport> {
    const [first, second] = await hkdf(this.chainingKey, empty, 2);
    return new Transport(
      await CipherState.withKey(first),
      await CipherState.withKey(second),
    );
  }
}

/** Each direction's messages must be opened in the order they were sealed. */
export class Transport {
  readonly send: CipherState;
  readonly receive: CipherState;

  constructor(send: CipherState, receive: CipherState) {
    this.send = send;
    this.receive = receive;
  }

  async seal(plaintext: Bytes): Promise<Bytes> {
    if (plaintext.length > maxMessageBytes - tagLength) {
      throw new Error("The message is too large for Noise.");
    }
    return this.send.encryptWithAd(empty, plaintext);
  }

  async open(message: Bytes): Promise<Bytes> {
    if (message.length > maxMessageBytes) {
      throw new Error("The message is too large for Noise.");
    }
    return this.receive.decryptWithAd(empty, message);
  }
}

interface InitiatorKeys {
  readonly staticKeys: CryptoKeyPair;
  readonly ephemeralKeys: KeySource;
}

/** A Noise HandshakeState on the initiator's side. */
export class HandshakeState {
  private readonly pattern: HandshakePattern;
  private readonly symmetric: SymmetricState;
  private readonly keys: InitiatorKeys;
  private readonly staticPublic: Bytes;
  private readonly psk: Bytes;
  private ephemeral: CryptoKeyPair | null = null;
  private remoteStatic: Bytes | null;
  private remoteEphemeral: Bytes | null = null;
  private step = 0;
  private finished: Transport | null = null;

  private constructor(
    pattern: HandshakePattern,
    symmetric: SymmetricState,
    keys: InitiatorKeys,
    staticPublic: Bytes,
    remoteStatic: Bytes | null,
    psk: Bytes,
  ) {
    this.pattern = pattern;
    this.symmetric = symmetric;
    this.keys = keys;
    this.staticPublic = staticPublic;
    this.remoteStatic = remoteStatic;
    this.psk = psk;
  }

  /** Starts XXpsk3 with a connection key's secret and the static key's curve. */
  static link(keys: InitiatorKeys, psk: Bytes): Promise<HandshakeState> {
    return HandshakeState.start(linkPattern, keys, null, psk);
  }

  /** Starts IK towards the desktop app's static public key on the stored curve. */
  static session(
    keys: InitiatorKeys,
    responderStatic: Bytes,
  ): Promise<HandshakeState> {
    return HandshakeState.start(sessionPattern, keys, responderStatic, empty);
  }

  private static async start(
    pattern: HandshakePattern,
    keys: InitiatorKeys,
    responderStatic: Bytes | null,
    psk: Bytes,
  ): Promise<HandshakeState> {
    const curve = linkCurve(keys.staticKeys.privateKey);
    const selected =
      curve === "25519"
        ? pattern
        : {
            ...pattern,
            protocolName: pattern.protocolName.replace("_25519_", "_P256_"),
          };
    const symmetric = await SymmetricState.initialize(selected.protocolName);
    await symmetric.mixHash(
      curve === "25519"
        ? prologue
        : new TextEncoder().encode("ravenpass-link/2"),
    );
    if (responderStatic) await symmetric.mixHash(responderStatic);
    return new HandshakeState(
      selected,
      symmetric,
      keys,
      await publicBytes(keys.staticKeys.publicKey),
      responderStatic,
      psk,
    );
  }

  async writeMessage(payload: Bytes): Promise<Bytes> {
    const parts: Bytes[] = [];
    for (const token of this.turn(true)) {
      if (token === "e") {
        this.ephemeral = await this.keys.ephemeralKeys.generate(
          linkCurve(this.keys.staticKeys.privateKey),
        );
        const ephemeralPublic = await publicBytes(this.ephemeral.publicKey);
        parts.push(ephemeralPublic);
        await this.mixEphemeral(ephemeralPublic);
      } else if (token === "s") {
        parts.push(await this.symmetric.encryptAndHash(this.staticPublic));
      } else if (token === "psk") {
        await this.symmetric.mixKeyAndHash(this.psk);
      } else {
        await this.mixDh(token);
      }
    }
    parts.push(await this.symmetric.encryptAndHash(payload));
    const message = concat(...parts);
    if (message.length > maxMessageBytes) {
      throw new Error("The handshake message is too large for Noise.");
    }
    await this.advance();
    return message;
  }

  /** Rejects a message that is too short or fails to authenticate. */
  async readMessage(message: Bytes): Promise<Bytes> {
    if (message.length > maxMessageBytes) {
      throw new Error("The handshake message is too large for Noise.");
    }
    let offset = 0;
    const take = (length: number): Bytes => {
      if (offset + length > message.length) {
        throw new Error("The handshake message is too short.");
      }
      offset += length;
      return message.slice(offset - length, offset);
    };
    for (const token of this.turn(false)) {
      if (token === "e") {
        this.remoteEphemeral = take(this.staticPublic.length);
        await this.mixEphemeral(this.remoteEphemeral);
      } else if (token === "s") {
        const length = this.symmetric.hasKey()
          ? this.staticPublic.length + tagLength
          : this.staticPublic.length;
        this.remoteStatic = await this.symmetric.decryptAndHash(take(length));
      } else if (token === "psk") {
        await this.symmetric.mixKeyAndHash(this.psk);
      } else {
        await this.mixDh(token);
      }
    }
    const payload = await this.symmetric.decryptAndHash(message.slice(offset));
    await this.advance();
    return payload;
  }

  /** The transport, once the last handshake message has been written or read. */
  transport(): Transport {
    if (!this.finished) throw new Error("The Noise handshake is not complete.");
    return this.finished;
  }

  /** The responder's static public key, once known. */
  responderStatic(): Bytes {
    if (!this.remoteStatic) {
      throw new Error("The responder has not sent its static key.");
    }
    return this.remoteStatic.slice();
  }

  handshakeHash(): Bytes {
    return this.symmetric.handshakeHash();
  }

  private turn(writing: boolean): readonly Token[] {
    const tokens = this.pattern.messages[this.step];
    if (!tokens || (this.step % 2 === 0) !== writing) {
      throw new Error("A Noise handshake message came out of turn.");
    }
    return tokens;
  }

  /** Noise section 9.2: with a `psk` token, every ephemeral public key also keys the cipher. */
  private async mixEphemeral(ephemeralPublic: Bytes): Promise<void> {
    await this.symmetric.mixHash(ephemeralPublic);
    if (this.pattern.messages.some((tokens) => tokens.includes("psk"))) {
      await this.symmetric.mixKey(ephemeralPublic);
    }
  }

  private async mixDh(token: DhToken): Promise<void> {
    const local =
      token === "ee" || token === "es"
        ? this.ephemeral?.privateKey
        : this.keys.staticKeys.privateKey;
    const remote =
      token === "ee" || token === "se"
        ? this.remoteEphemeral
        : this.remoteStatic;
    if (!local || !remote) {
      throw new Error(`The Noise ${token} token came before its keys.`);
    }
    await this.symmetric.mixKey(await dh(local, remote));
  }

  private async advance(): Promise<void> {
    this.step += 1;
    if (this.step === this.pattern.messages.length) {
      this.finished = await this.symmetric.split();
    }
  }
}
