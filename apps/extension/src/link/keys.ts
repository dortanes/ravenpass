export type LinkCurve = "P256" | "25519";

export function linkCurve(key: CryptoKey): LinkCurve {
  if (key.algorithm.name === "X25519") return "25519";
  if (
    key.algorithm.name === "ECDH" &&
    (key.algorithm as EcKeyAlgorithm).namedCurve === "P-256"
  ) {
    return "P256";
  }
  throw new Error("The link key uses an unsupported algorithm.");
}

export function keyAlgorithm(
  curve: LinkCurve,
): AlgorithmIdentifier | EcKeyImportParams {
  return curve === "25519"
    ? { name: "X25519" }
    : { name: "ECDH", namedCurve: "P-256" };
}

/** Supplies a new static pair or a stored link's matching ephemeral pair. */
export interface KeySource {
  generate(curve?: LinkCurve): Promise<CryptoKeyPair>;
}

/** P-256 CryptoKeys survive WebKit's structured clone; X25519 keeps existing links working. */
export class WebCryptoKeys implements KeySource {
  async generate(curve: LinkCurve = "P256"): Promise<CryptoKeyPair> {
    return crypto.subtle.generateKey(keyAlgorithm(curve), false, [
      "deriveBits",
    ]) as Promise<CryptoKeyPair>;
  }
}
