import assert from "node:assert/strict";
import { test } from "node:test";
import { ask } from "../messages.ts";
import {
  isRefusedReply,
  RefusedRequest,
  refusedReply,
  requestFailureReply,
} from "./request-failure.ts";

test("only actual request refusals get the bounded refusal reply", () => {
  assert.deepEqual(
    requestFailureReply(new RefusedRequest("menu-close")),
    refusedReply,
  );
  for (const failure of [
    new Error("synthetic secret"),
    { name: "RefusedRequest", message: "synthetic secret" },
    "synthetic secret",
    null,
  ])
    assert.equal(requestFailureReply(failure), null);
  for (const reply of [null, undefined, {}, { requestFailure: "unknown" }]) {
    assert.equal(isRefusedReply(reply), false);
  }
});

test("ask distinguishes handler refusal, handler failure and transport without request data", async (t) => {
  const previous = Object.getOwnPropertyDescriptor(globalThis, "chrome");
  let reply: unknown = { ok: true };
  let failure: Error | null = null;
  const requests: unknown[] = [];
  Object.defineProperty(globalThis, "chrome", {
    configurable: true,
    value: {
      runtime: {
        sendMessage: async (request: unknown) => {
          requests.push(request);
          if (failure) throw failure;
          return reply;
        },
      },
    },
  });
  t.after(() => {
    if (previous) Object.defineProperty(globalThis, "chrome", previous);
    else Reflect.deleteProperty(globalThis, "chrome");
  });
  const request = {
    kind: "menu-close",
    token: "synthetic-secret-token",
  } as const;
  assert.deepEqual(await ask(request), { ok: true });
  assert.deepEqual(requests[0], request);
  reply = refusedReply;
  await assert.rejects(ask(request), {
    name: "RequestFailure:menu-close:refused",
  });
  reply = null;
  await assert.rejects(ask(request), {
    name: "RequestFailure:menu-close:failed",
  });
  failure = new Error("synthetic-private-exception");
  await assert.rejects(ask(request), {
    name: "RequestFailure:menu-close:transport",
    cause: failure,
  });
});
