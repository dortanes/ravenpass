import assert from "node:assert/strict";
import test from "node:test";
import { type ExtensionPageSender, ExtensionPages } from "./extension-pages.ts";

const extensionId = "test-extension";

function pages(base: string, id = extensionId) {
  return new ExtensionPages({ id, getURL: (path) => `${base}${path}` });
}

for (const protocol of [
  "chrome-extension:",
  "webkit-extension:",
  "safari-web-extension:",
  "browser-addon:",
]) {
  const base = `${protocol}//${extensionId}/`;
  const policy = pages(base);

  test(`${protocol} authenticates packaged pages independently of path and tab metadata`, () => {
    for (const path of [
      "",
      "action/index.html",
      "action/index.yab-action.html",
      "rebuilt/popup.html",
      "pages/color-scheme.html",
      "pages/menu.html#session",
      "popup/index.html?mode=link#view",
    ]) {
      for (const tab of [undefined, { id: 1, url: "https://example.test/" }]) {
        assert.equal(
          policy.allows({ id: extensionId, url: `${base}${path}`, tab }),
          true,
        );
      }
    }
  });

  test(`${protocol} rejects incomplete identities, sites and different extension addresses`, () => {
    const rejected: ExtensionPageSender[] = [
      {},
      { id: extensionId },
      { url: `${base}action/index.html` },
      { id: "other-extension", url: `${base}action/index.html` },
      ...[
        "https://example.test/action/index.html",
        "data:text/html,popup",
        "null",
        "not a url",
        "/action/index.html",
        `${protocol}//other-extension/action/index.html`,
        `${protocol}//${extensionId}.example.test/action/index.html`,
        `${protocol}//${extensionId}:42/action/index.html`,
        `https://${extensionId}/action/index.html`,
        `${protocol}//user@${extensionId}/action/index.html`,
        `${protocol}//user:password@${extensionId}/action/index.html`,
        `blob:${base}document`,
      ].map((url) => ({ id: extensionId, url })),
    ];
    for (const sender of rejected) {
      for (const tab of [
        undefined,
        { id: 1, url: `${base}action/index.html` },
      ]) {
        assert.equal(policy.allows({ ...sender, tab }), false);
      }
    }
  });
}

test("the observed Yab path mismatch does not require reading the manifest", () => {
  const base = `chrome-extension://${extensionId}/`;
  const runtime = {
    id: extensionId,
    getURL: (path: string) => `${base}${path}`,
    getManifest: () => {
      assert.fail("manifest paths must not determine page identity");
    },
  };
  assert.equal(
    new ExtensionPages(runtime).allows({
      id: extensionId,
      url: `${base}action/index.html`,
      tab: { id: 1 },
    }),
    true,
  );
});

test("opaque origins do not authorize another host or protocol", () => {
  const own = `webkit-extension://${extensionId}/action/index.html`;
  const otherHost = "webkit-extension://other-extension/action/index.html";
  const otherProtocol = `other-extension://${extensionId}/action/index.html`;
  const policy = pages(`webkit-extension://${extensionId}/`);
  assert.equal(new URL(own).origin, "null");
  assert.equal(new URL(otherHost).origin, "null");
  assert.equal(new URL(otherProtocol).origin, "null");
  assert.equal(policy.allows({ id: extensionId, url: own }), true);
  assert.equal(policy.allows({ id: extensionId, url: otherHost }), false);
  assert.equal(policy.allows({ id: extensionId, url: otherProtocol }), false);
});

test("the runtime ID need not equal its browser-assigned URL host", () => {
  const base = "webkit-extension://00000000-0000-4000-8000-000000000001/";
  const policy = pages(base);
  assert.equal(
    policy.allows({ id: extensionId, url: `${base}popup.html` }),
    true,
  );
  assert.equal(
    policy.allows({ id: "other-extension", url: `${base}popup.html` }),
    false,
  );
});

test("missing runtime identity cannot authorize a sender", () => {
  const base = `webkit-extension://${extensionId}/`;
  assert.equal(
    pages(base, "").allows({ id: "", url: `${base}popup.html` }),
    false,
  );
  assert.equal(
    pages("file:///").allows({ id: extensionId, url: "file:///popup.html" }),
    false,
  );
});
