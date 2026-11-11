import { readFile } from "node:fs/promises";
import { base64urlnopad, hex } from "@scure/base";
import type { KeyRejection } from "./key.ts";
import type { KeySource } from "./keys.ts";

// The desktop app's protocol tests write these vectors; both sides must agree byte for byte.

interface KeyPairVector {
  readonly private: string;
  readonly public: string;
}

export interface HandshakeVector {
  readonly protocol: string;
  readonly psk: string | null;
  readonly initiatorStatic: KeyPairVector;
  readonly initiatorEphemeral: KeyPairVector;
  readonly responderStatic: KeyPairVector;
  readonly handshakeMessages: number;
  readonly handshakeHash: string;
  readonly messages: readonly {
    readonly sender: "initiator" | "responder";
    readonly payload: string;
    readonly ciphertext: string;
  }[];
}

interface Vectors {
  readonly keys: {
    readonly valid: readonly {
      readonly key: string;
      readonly port: number;
      readonly secret: string;
    }[];
    readonly invalid: readonly {
      readonly key: string;
      readonly reason: KeyRejection;
    }[];
  };
  readonly handshakes: readonly HandshakeVector[];
}

export const linkProtocol = "Noise_XXpsk3_25519_AESGCM_SHA256";
export const sessionProtocol = "Noise_IK_25519_AESGCM_SHA256";
export const p256LinkProtocol = "Noise_XXpsk3_P256_AESGCM_SHA256";
export const p256SessionProtocol = "Noise_IK_P256_AESGCM_SHA256";

const vectorsFile = new URL(
  "../../../../packages/app/linkproto/testdata/vectors.json",
  import.meta.url,
);

export const vectors: Vectors = JSON.parse(await readFile(vectorsFile, "utf8"));

export function handshakeVector(protocol: string): HandshakeVector {
  const vector = vectors.handshakes.find(
    (candidate) => candidate.protocol === protocol,
  );
  if (!vector) throw new Error(`The vectors hold no ${protocol} transcript.`);
  return vector;
}

export function bytes(text: string): Uint8Array<ArrayBuffer> {
  return Uint8Array.from(hex.decode(text));
}

export async function importKeyPair(
  pair: KeyPairVector,
): Promise<CryptoKeyPair> {
  const p256 = bytes(pair.public).length === 65;
  const algorithm = p256
    ? { name: "ECDH", namedCurve: "P-256" }
    : { name: "X25519" };
  const privateKey = await crypto.subtle.importKey(
    "jwk",
    {
      kty: p256 ? "EC" : "OKP",
      crv: p256 ? "P-256" : "X25519",
      d: base64urlnopad.encode(bytes(pair.private)),
      x: base64urlnopad.encode(
        p256 ? bytes(pair.public).slice(1, 33) : bytes(pair.public),
      ),
      ...(p256
        ? { y: base64urlnopad.encode(bytes(pair.public).slice(33)) }
        : {}),
    },
    algorithm,
    false,
    ["deriveBits"],
  );
  const publicKey = await crypto.subtle.importKey(
    "raw",
    bytes(pair.public),
    algorithm,
    true,
    [],
  );
  return { privateKey, publicKey };
}

/** Hands out fixed key pairs in order, as a handshake with a vector's keys asks for them. */
export class FixedKeys implements KeySource {
  private readonly pairs: CryptoKeyPair[];

  constructor(pairs: readonly CryptoKeyPair[]) {
    this.pairs = [...pairs];
  }

  async generate(): Promise<CryptoKeyPair> {
    const pair = this.pairs.shift();
    if (!pair) throw new Error("No fixed key pair is left.");
    return pair;
  }
}
