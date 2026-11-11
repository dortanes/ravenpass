import assert from "node:assert/strict";
import test, { type TestContext } from "node:test";
import { RequestFailure } from "./request-failure.ts";
import {
  isClosedPortError,
  sendFailure,
  sendIgnoringClosedPort,
} from "./send.ts";

function watchErrors(t: TestContext) {
  return t.mock.method(console, "error", () => {});
}

test("a message whose receiver is gone resolves undefined silently", async (t) => {
  const errors = watchErrors(t);
  for (const message of [
    "Could not establish connection. Receiving end does not exist.",
    "The message port closed before a response was received.",
    "A listener indicated an asynchronous response by returning true, but the message channel closed before a response was received",
    "No tab with id: 12.",
    "Extension context invalidated.",
  ]) {
    assert.equal(isClosedPortError(new Error(message)), true, message);
    assert.equal(
      await sendIgnoringClosedPort(Promise.reject(new Error(message))),
      undefined,
    );
  }
  assert.equal(errors.mock.callCount(), 0);
});

test("a message answers what its receiver answered", async (t) => {
  const errors = watchErrors(t);
  assert.deepEqual(
    await sendIgnoringClosedPort(Promise.resolve({ ok: true })),
    { ok: true },
  );
  assert.equal(errors.mock.callCount(), 0);
});

test("any other failure resolves undefined and logs its name once", async (t) => {
  const errors = watchErrors(t);
  const refused = new TypeError(
    "The service worker could not serve the menu-close request for https://example.com.",
  );
  assert.equal(
    await sendIgnoringClosedPort(Promise.reject(refused)),
    undefined,
  );
  assert.equal(errors.mock.callCount(), 1);
  assert.deepEqual(errors.mock.calls[0]?.arguments, [sendFailure, "TypeError"]);
});

test("a failure that is not an Error logs its type, never its value", async (t) => {
  const errors = watchErrors(t);
  assert.equal(
    await sendIgnoringClosedPort(
      Promise.reject("Receiving end does not exist."),
    ),
    undefined,
  );
  assert.deepEqual(
    errors.mock.calls.map((call) => call.arguments),
    [[sendFailure, "string"]],
  );
  assert.equal(isClosedPortError(null), false);
});

test("only explicitly refused menu housekeeping stays quiet", async (t) => {
  const errors = watchErrors(t);
  for (const kind of ["menu-size", "menu-close", "menu-focus"] as const) {
    assert.equal(
      await sendIgnoringClosedPort(
        Promise.reject(new RequestFailure(kind, "refused")),
      ),
      undefined,
    );
  }
  assert.equal(errors.mock.callCount(), 0);
  for (const [kind, reason] of [
    ["menu-size", "failed"],
    ["menu-close", "transport"],
    ["menu-focus", "failed"],
    ["menu-fill", "refused"],
    ["menu", "refused"],
  ] as const) {
    await sendIgnoringClosedPort(
      Promise.reject(new RequestFailure(kind, reason)),
    );
  }
  assert.deepEqual(
    errors.mock.calls.map((call) => call.arguments),
    [
      [sendFailure, "RequestFailure:menu-size:failed"],
      [sendFailure, "RequestFailure:menu-close:transport"],
      [sendFailure, "RequestFailure:menu-focus:failed"],
      [sendFailure, "RequestFailure:menu-fill:refused"],
      [sendFailure, "RequestFailure:menu:refused"],
    ],
  );
});

test("wrapped transport errors retain closed-port handling and hide other exception data", async (t) => {
  const errors = watchErrors(t);
  for (const message of [
    "Receiving end does not exist.",
    "The message port closed before a response was received.",
  ]) {
    const failure = new RequestFailure("menu-close", "transport", {
      cause: new Error(message),
    });
    assert.equal(isClosedPortError(failure), true);
    await sendIgnoringClosedPort(Promise.reject(failure));
  }
  assert.equal(errors.mock.callCount(), 0);
  const failure = new RequestFailure("menu-close", "transport", {
    cause: new Error("synthetic private payload https://example.com/secret"),
  });
  await sendIgnoringClosedPort(Promise.reject(failure));
  assert.deepEqual(errors.mock.calls[0]?.arguments, [
    sendFailure,
    "RequestFailure:menu-close:transport",
  ]);
});
