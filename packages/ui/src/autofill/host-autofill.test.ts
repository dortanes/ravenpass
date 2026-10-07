import assert from "node:assert/strict";
import test from "node:test";
import type { AutofillWait } from "./autofill-api.ts";
import { AutofillChannel } from "./channel.ts";
import { HostAutofill } from "./host-autofill.ts";

/** A host that answers each request with the next of its answers, recording the requests. */
function phone(...answers: unknown[]) {
  const requests: unknown[] = [];
  const channel: AutofillChannel = new AutofillChannel({
    post(id, request) {
      requests.push(JSON.parse(request));
      if (id > 0) {
        queueMicrotask(() =>
          channel.answer(id, JSON.stringify(answers.shift())),
        );
      }
    },
  });
  return { api: new HostAutofill(channel), channel, requests };
}

test("the search screen opens on the host's results and passkeys, read defensively", async () => {
  const { api } = phone({
    status: "ok",
    screen: "search",
    site: "example.com",
    code: false,
    scope: "matches",
    results: [
      {
        id: "a1",
        label: "Mail",
        account: "alex",
        site: "example.com",
        matches: true,
        tags: ["Work", 4, ""],
      },
      { id: "b2", label: 7, matches: "yes", tags: "Work" },
    ],
    passkeys: [
      { key: "a2V5", label: "Mail", account: "alex", site: "example.com" },
      { key: 3 },
    ],
  });
  assert.deepEqual(await api.open(), {
    screen: "search",
    site: "example.com",
    code: false,
    passkey: false,
    scope: "matches",
    credentials: [
      {
        id: "a1",
        label: "Mail",
        account: "alex",
        site: "example.com",
        matches: true,
        tags: ["Work"],
      },
      {
        id: "b2",
        label: "",
        account: "",
        site: "",
        matches: false,
        tags: [],
      },
    ],
    passkeys: [
      { key: "a2V5", label: "Mail", account: "alex", site: "example.com" },
      { key: "", label: "", account: "", site: "" },
    ],
  });
});

test("a search screen of a host that lists no passkeys has none", async () => {
  const { api } = phone({ status: "ok", screen: "search", results: [] });
  const opening = await api.open();
  assert.equal(opening.screen, "search");
  assert.deepEqual(opening.screen === "search" && opening.passkeys, []);
});

test("the unlock screen opens on the ways in the phone reports", async () => {
  const { api } = phone({
    status: "ok",
    screen: "unlock",
    biometry: true,
    pin: true,
    attemptsLeft: 4,
    pinMin: 6,
    pinMax: 12,
  });
  assert.deepEqual(await api.open(), {
    screen: "unlock",
    methods: {
      biometryAvailable: true,
      biometryEnabled: true,
      pinSet: true,
      pinAttemptsLeft: 4,
      pinMinLength: 6,
      pinMaxLength: 12,
    },
    verify: false,
  });
});

test("the unlock screen confirms the owner where the vault is already open", async () => {
  const { api } = phone({
    status: "ok",
    screen: "unlock",
    pin: true,
    open: true,
  });
  const opening = await api.open();
  assert.equal(opening.screen === "unlock" && opening.verify, true);
});

test("the save review opens on the offer, leaving out targets it cannot name", async () => {
  const { api } = phone({
    status: "ok",
    screen: "save",
    offer: {
      name: "example.com",
      site: "example.com",
      account: "alex",
      suggested: "a1",
      targets: [
        {
          id: "a1",
          label: "Example",
          account: "alex",
          action: "update",
          tags: ["Work", 4, ""],
        },
        { id: "b2", label: "Other", account: "alex", action: "delete" },
      ],
    },
  });
  assert.deepEqual(await api.open(), {
    screen: "save",
    offer: {
      site: "example.com",
      account: "alex",
      name: "example.com",
      suggested: "a1",
      targets: [
        {
          credential: "a1",
          label: "Example",
          account: "alex",
          action: "update",
          tags: ["Work"],
        },
      ],
    },
  });
});

test("an opening that failed or names no known screen is refused", async () => {
  await assert.rejects(phone({ status: "failed" }).api.open());
  await assert.rejects(phone({ status: "ok", screen: "settings" }).api.open());
});

test("fills, unlocks and saves read how they ended", async () => {
  const { api, requests } = phone(
    { status: "not-added" },
    { status: "locked" },
    { status: "wrong-pin", attemptsLeft: 3 },
    { status: "ok", created: true },
    { status: "account-refused" },
  );
  const credential = {
    id: "b2",
    label: "Bank",
    account: "",
    site: "",
    matches: false,
    tags: [],
  };
  assert.equal(await api.fill(credential, true), "not-added");
  assert.equal(await api.fill(credential, false), "failed");
  assert.deepEqual(await api.unlock("000000"), {
    kind: "wrong-pin",
    attemptsLeft: 3,
  });
  const choice = { target: "", account: "alex", name: "Example" };
  assert.deepEqual(await api.save(choice), { kind: "saved", created: true });
  assert.deepEqual(await api.save(choice), { kind: "account-refused" });
  assert.deepEqual(requests[0], {
    id: "b2",
    label: "Bank",
    add: true,
    op: "fill",
  });
  assert.deepEqual(requests[3], { ...choice, op: "save" });
});

test("an unlock tried too soon after wrong PINs reads as such, with or without attempts", async () => {
  const { api } = phone(
    { status: "too-soon" },
    { status: "too-soon", attemptsLeft: 2 },
    { status: "wrong-pin" },
  );
  assert.deepEqual(await api.unlock("000000"), { kind: "too-soon" });
  assert.deepEqual(await api.unlock("000000"), { kind: "too-soon" });
  assert.deepEqual(await api.unlock("000000"), {
    kind: "wrong-pin",
    attemptsLeft: 0,
  });
});

test("a passkey signs in by its key alone, and its outcome is read", async () => {
  const { api, requests } = phone(
    { status: "ok" },
    { status: "unverifiable" },
    { status: "outdated" },
    { status: "declined" },
  );
  const passkey = {
    key: "a2V5",
    label: "Mail",
    account: "alex",
    site: "example.com",
  };
  assert.equal(await api.signIn(passkey), "signed");
  assert.equal(await api.signIn(passkey), "unverifiable");
  assert.equal(await api.signIn(passkey), "outdated");
  assert.equal(await api.signIn(passkey), "failed");
  assert.deepEqual(requests[0], { key: "a2V5", op: "sign-in" });
});

test("a fill reads a credential gone from the vault, a host that did not answer and one updated while open", async () => {
  const { api } = phone(
    { status: "not-found" },
    { status: "unreachable" },
    { status: "outdated" },
  );
  const credential = {
    id: "a1",
    label: "Mail",
    account: "alex",
    site: "example.com",
    matches: true,
    tags: [],
  };
  assert.equal(await api.fill(credential, false), "not-found");
  assert.equal(await api.fill(credential, false), "unreachable");
  assert.equal(await api.fill(credential, false), "outdated");
});

test("waits the host reports reach the watcher until it stops watching", () => {
  const { api, channel } = phone();
  const seen: (AutofillWait | null)[] = [];
  const stop = api.watchWait((wait) => seen.push(wait));
  channel.notice('{"status":"wait","wait":"unlocking"}');
  channel.notice('{"status":"wait","wait":"failed","failure":"unsupported"}');
  channel.notice('{"status":"wait","wait":"failed","failure":"outdated"}');
  channel.notice('{"status":"wait","wait":"failed","failure":"melted"}');
  channel.notice('{"status":"other","wait":"opening"}');
  channel.notice('{"status":"wait","wait":""}');
  stop();
  channel.notice('{"status":"wait","wait":"verifying"}');
  assert.deepEqual(seen, [
    { kind: "unlocking" },
    { kind: "failed", failure: "unsupported" },
    { kind: "failed", failure: "outdated" },
    { kind: "failed", failure: "failed" },
    null,
  ]);
});

test("the interface size is the host's, or the natural one where it keeps none", async () => {
  const { api } = phone(
    { status: "ok", percent: 115 },
    { status: "failed" },
    { status: "ok", percent: "large" },
  );
  assert.equal(await api.interfaceSize(), 115);
  assert.equal(await api.interfaceSize(), 100);
  assert.equal(await api.interfaceSize(), 100);
});

test("the appearance is the host's, or the system's where it keeps none", async () => {
  const { api } = phone(
    { status: "ok", appearance: "light" },
    { status: "failed" },
    { status: "ok", appearance: "sepia" },
  );
  assert.equal(await api.appearance(), "light");
  assert.equal(await api.appearance(), "system");
  assert.equal(await api.appearance(), "system");
});

test("an icon the phone cannot give reads as none", async () => {
  const { api } = phone({ status: "failed" }, { status: "ok", image: "iVBOR" });
  assert.deepEqual(await api.siteIcon("example.com"), { image: "", tint: "" });
  assert.deepEqual(await api.siteIcon("example.com"), {
    image: "iVBOR",
    tint: "",
  });
});
