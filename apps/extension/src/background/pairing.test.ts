import assert from "node:assert/strict";
import test from "node:test";
import { LinkError } from "../link/client.ts";
import { vectors } from "../link/vectors.test-support.ts";
import { ExtensionPages } from "./extension-pages.ts";
import { ExtensionPairing } from "./pairing.ts";

const key = vectors.keys.valid[0]?.key;
assert.ok(key);
const sensitive = "synthetic-connection-key-private-data";
const extensionId = "test-extension";
const popupUrl = `webkit-extension://${extensionId}/action/index.html`;
const sender = { id: extensionId, url: popupUrl };
const pages = new ExtensionPages({
  id: extensionId,
  getURL: (path) => `webkit-extension://${extensionId}/${path}`,
});

test("tab senders are refused before parsing or linking", async () => {
  const pairing = new ExtensionPairing(
    {
      link: async () => {
        assert.fail("unauthorized link");
      },
    },
    () => {
      assert.fail("unauthorized metadata read");
    },
    pages,
  );
  assert.deepEqual(await pairing.link(sensitive, { tab: { id: 1 } }), {
    ok: false,
    reason: "failed",
    diagnostic: "RP-11",
  });
});

test("a legitimate extension page can link", async () => {
  let called = false;
  const pairing = new ExtensionPairing(
    {
      link: async (_, name) => {
        called = true;
        assert.equal(name, "Test browser");
      },
    },
    () => "Test browser",
    pages,
  );
  assert.deepEqual(await pairing.link(key, sender), { ok: true });
  assert.equal(called, true);
});

test("background failure replies carry only bounded references", async () => {
  for (const error of [
    new Error(sensitive),
    new LinkError("failed", { cause: new Error(sensitive) }, "RP-09"),
  ]) {
    const pairing = new ExtensionPairing(
      {
        link: async () => {
          throw error;
        },
      },
      () => "Test browser",
      pages,
    );
    const answer = await pairing.link(key, sender);
    assert.deepEqual(answer, {
      ok: false,
      reason: "failed",
      diagnostic: error instanceof LinkError ? "RP-09" : "RP-14",
    });
    assert.equal(JSON.stringify(answer).includes(sensitive), false);
  }
});

test("metadata exceptions never leak their message", async () => {
  const pairing = new ExtensionPairing(
    {
      link: async () => {
        assert.fail("link must not start");
      },
    },
    () => {
      throw new Error(sensitive);
    },
    pages,
  );
  assert.deepEqual(await pairing.link(key, sender), {
    ok: false,
    reason: "failed",
    diagnostic: "RP-14",
  });
});

test("unexpected diagnostic values cannot expose arbitrary data", async () => {
  const error = new LinkError("failed");
  Object.defineProperty(error, "diagnostic", { value: sensitive });
  const pairing = new ExtensionPairing(
    {
      link: async () => {
        throw error;
      },
    },
    () => "Test browser",
    pages,
  );
  assert.deepEqual(await pairing.link(key, sender), {
    ok: false,
    reason: "failed",
    diagnostic: "RP-14",
  });
});

test("the authenticated WebKit popup can pair with tab metadata", async () => {
  let called = false;
  const pairing = new ExtensionPairing(
    {
      link: async () => {
        called = true;
      },
    },
    () => "Test browser",
    pages,
  );
  assert.deepEqual(await pairing.link(key, { ...sender, tab: { id: 1 } }), {
    ok: true,
  });
  assert.equal(called, true);
});

test("unauthenticated senders are denied before parsing even without a tab", async () => {
  const pairing = new ExtensionPairing(
    {
      link: async () => {
        assert.fail("unauthorized link");
      },
    },
    () => {
      assert.fail("unauthorized metadata read");
    },
    pages,
  );
  for (const rejected of [
    {},
    { id: extensionId },
    { url: popupUrl },
    { id: "other-extension", url: popupUrl },
    { id: extensionId, url: "https://example.test/action/index.html" },
    {
      id: extensionId,
      url: `webkit-extension://other-extension/pages/menu.html`,
    },
  ]) {
    assert.deepEqual(await pairing.link(sensitive, rejected), {
      ok: false,
      reason: "failed",
      diagnostic: "RP-11",
    });
  }
});

test("the observed Yab popup sender reaches key parsing despite the manifest wrapper", async () => {
  const runtime = {
    id: extensionId,
    getURL: (path: string) => `chrome-extension://${extensionId}/${path}`,
    getManifest: () => ({
      action: { default_popup: "action/index.yab-action.html" },
    }),
  };
  const pairing = new ExtensionPairing(
    {
      link: async () => {
        assert.fail("invalid key must not link");
      },
    },
    () => {
      assert.fail("invalid key must not read metadata");
    },
    new ExtensionPages(runtime),
  );
  assert.deepEqual(
    await pairing.link("(invalid diagnostic key)", {
      id: extensionId,
      url: `chrome-extension://${extensionId}/action/index.html`,
      tab: { id: 1 },
    }),
    { ok: false, reason: "not-key", diagnostic: "RP-00" },
  );
});
