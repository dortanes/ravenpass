import assert from "node:assert/strict";
import test from "node:test";
import type { CardValues } from "../link/client.ts";
import { CardFrames, sharedWith } from "./card-frames.ts";
import { MemoryArea } from "./test-doubles.test-support.ts";

const values: CardValues = {
  holder: "Alex Example",
  number: "4111111111111111",
  expiry: "2029-08",
  securityCode: "739",
  billing: null,
};

const shop = "https://shop.example.com";
const provider = "https://js.payments.example";
const tabId = 7;

test("a frame of the chosen frame's origin takes every value; the top frame's all but the secrets", () => {
  assert.equal(sharedWith(provider, provider, shop, values), values);
  assert.deepEqual(sharedWith(shop, provider, shop, values), {
    ...values,
    number: "",
    securityCode: "",
  });
  assert.equal(sharedWith("https://ads.example", provider, shop, values), null);
});

test("a fill reaches the tab's other reported frames as their origins allow", async () => {
  const frames = new CardFrames({ area: new MemoryArea() });
  const chosen = { tabId, documentId: "number", origin: provider };
  await frames.report(chosen);
  await frames.report({ tabId, documentId: "expiry", origin: provider });
  await frames.report({ tabId, documentId: "page", origin: shop });
  await frames.report({
    tabId,
    documentId: "ad",
    origin: "https://ads.example",
  });
  await frames.report({ tabId: 8, documentId: "other-tab", origin: provider });

  assert.deepEqual(await frames.deliveries(chosen, shop, values), [
    { frame: { tabId, documentId: "expiry" }, values },
    {
      frame: { tabId, documentId: "page" },
      values: { ...values, number: "", securityCode: "" },
    },
  ]);

  await frames.forget(tabId);
  assert.deepEqual(await frames.deliveries(chosen, shop, values), []);
});

test("a tab keeps the frames that reported last", async () => {
  const frames = new CardFrames({ area: new MemoryArea() });
  for (let frame = 0; frame < 20; frame += 1) {
    await frames.report({
      tabId,
      documentId: `frame-${frame}`,
      origin: provider,
    });
  }
  await frames.report({ tabId, documentId: "frame-5", origin: provider });
  const chosen = { tabId, documentId: "chosen", origin: provider };
  const reached = (await frames.deliveries(chosen, shop, values)).map(
    ({ frame }) => frame.documentId,
  );
  assert.equal(reached.length, 16);
  assert.ok(!reached.includes("frame-3") && reached.includes("frame-4"));
  assert.equal(reached.at(-1), "frame-5");
});
