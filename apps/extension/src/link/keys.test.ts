import assert from "node:assert/strict";
import test from "node:test";
import { linkCurve, WebCryptoKeys } from "./keys.ts";

for (const curve of ["P256", "25519"] as const) {
  test(`${curve} keys remain non-extractable and usable after structured clone`, async () => {
    const source = new WebCryptoKeys();
    const pair = structuredClone(await source.generate(curve));
    assert.equal(linkCurve(pair.privateKey), curve);
    assert.equal(pair.privateKey.extractable, false);
    await assert.rejects(crypto.subtle.exportKey("pkcs8", pair.privateKey));
    const peer = await source.generate(curve);
    const agreement = (key: CryptoKey, publicKey: CryptoKey) =>
      crypto.subtle.deriveBits(
        { name: key.algorithm.name, public: publicKey },
        key,
        256,
      );
    assert.deepEqual(
      await agreement(pair.privateKey, peer.publicKey),
      await agreement(peer.privateKey, pair.publicKey),
    );
  });
}

test("new links default to P-256", async () => {
  const pair = await new WebCryptoKeys().generate();
  assert.equal(linkCurve(pair.privateKey), "P256");
});

test("unknown key algorithms are rejected", async () => {
  const key = await crypto.subtle.generateKey(
    { name: "AES-GCM", length: 256 },
    false,
    ["encrypt"],
  );
  assert.throws(() => linkCurve(key));
});
