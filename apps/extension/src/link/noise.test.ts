import assert from "node:assert/strict";
import test from "node:test";
import { type LinkCurve, WebCryptoKeys } from "./keys.ts";
import { HandshakeState } from "./noise.ts";
import {
  bytes,
  FixedKeys,
  type HandshakeVector,
  importKeyPair,
  linkProtocol,
  p256LinkProtocol,
  p256SessionProtocol,
  sessionProtocol,
  vectors,
} from "./vectors.test-support.ts";

async function initiator(vector: HandshakeVector): Promise<HandshakeState> {
  const keys = {
    staticKeys: await importKeyPair(vector.initiatorStatic),
    ephemeralKeys: new FixedKeys([
      await importKeyPair(vector.initiatorEphemeral),
    ]),
  };
  switch (vector.protocol) {
    case linkProtocol:
    case p256LinkProtocol:
      return HandshakeState.link(keys, bytes(vector.psk ?? ""));
    case sessionProtocol:
    case p256SessionProtocol:
      return HandshakeState.session(keys, bytes(vector.responderStatic.public));
    default:
      throw new Error(`No handshake runs ${vector.protocol}.`);
  }
}

test("the vectors hold a transcript of each handshake", () => {
  assert.deepEqual(
    vectors.handshakes.map((vector) => vector.protocol).sort(),
    [
      sessionProtocol,
      linkProtocol,
      p256LinkProtocol,
      p256SessionProtocol,
    ].sort(),
  );
});

test("a stored X25519 link asks for matching ephemeral keys after the update", async () => {
  const vector = vectors.handshakes.find((v) => v.protocol === sessionProtocol);
  assert.ok(vector);
  let requested: LinkCurve | undefined;
  const handshake = await HandshakeState.session(
    {
      staticKeys: await importKeyPair(vector.initiatorStatic),
      ephemeralKeys: {
        generate: async (curve) => {
          requested = curve;
          return new WebCryptoKeys().generate(curve);
        },
      },
    },
    bytes(vector.responderStatic.public),
  );
  await handshake.writeMessage(new Uint8Array(0));
  assert.equal(requested, "25519");
});

for (const vector of vectors.handshakes) {
  test(`${vector.protocol} reproduces the desktop app's transcript`, async () => {
    const handshake = await initiator(vector);
    for (const [index, message] of vector.messages.entries()) {
      const label = `message ${index + 1}`;
      const payload = bytes(message.payload);
      const ciphertext = bytes(message.ciphertext);
      const handshaking = index < vector.handshakeMessages;
      if (message.sender === "initiator") {
        const written = handshaking
          ? await handshake.writeMessage(payload)
          : await handshake.transport().seal(payload);
        assert.deepEqual(written, ciphertext, label);
      } else {
        const read = handshaking
          ? await handshake.readMessage(ciphertext)
          : await handshake.transport().open(ciphertext);
        assert.deepEqual(read, payload, label);
      }
      if (index === vector.handshakeMessages - 1) {
        assert.deepEqual(
          handshake.handshakeHash(),
          bytes(vector.handshakeHash),
        );
        assert.deepEqual(
          handshake.responderStatic(),
          bytes(vector.responderStatic.public),
        );
      }
    }
  });

  test(`${vector.protocol} rejects an altered responder message`, async () => {
    const handshake = await initiator(vector);
    const [first, second] = vector.messages;
    assert.ok(first && second);
    await handshake.writeMessage(bytes(first.payload));
    const altered = bytes(second.ciphertext);
    const last = altered.length - 1;
    altered[last] = (altered[last] ?? 0) ^ 0x01;
    await assert.rejects(handshake.readMessage(altered));
  });

  test(`${vector.protocol} rejects a responder message cut short`, async () => {
    const handshake = await initiator(vector);
    const [first, second] = vector.messages;
    assert.ok(first && second);
    await handshake.writeMessage(bytes(first.payload));
    await assert.rejects(
      handshake.readMessage(bytes(second.ciphertext).slice(0, 16)),
    );
  });

  test(`${vector.protocol} rejects an altered transport message after authentication`, async () => {
    const handshake = await initiator(vector);
    for (const message of vector.messages.slice(0, vector.handshakeMessages)) {
      if (message.sender === "initiator")
        await handshake.writeMessage(bytes(message.payload));
      else await handshake.readMessage(bytes(message.ciphertext));
    }
    const message = vector.messages
      .slice(vector.handshakeMessages)
      .find((message) => message.sender === "responder");
    assert.ok(message);
    const altered = bytes(message.ciphertext);
    const last = altered.length - 1;
    altered[last] = (altered[last] ?? 0) ^ 1;
    await assert.rejects(handshake.transport().open(altered));
  });
}
