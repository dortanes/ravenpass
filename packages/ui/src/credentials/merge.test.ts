import assert from "node:assert/strict";
import test from "node:test";
import type { Credential, CredentialSummary } from "../vault-api.ts";
import { emptyCredential } from "./credential.ts";
import { CredentialMerge, mergeCandidates } from "./merge.ts";

const bounds = { websites: 3, tags: 8, notes: 40 };

function credential(fields: Partial<Credential>): Credential {
  return {
    ...emptyCredential,
    id: "kept",
    groups: [],
    site: "",
    passkeys: [],
    ...fields,
  };
}

function summary(fields: Partial<CredentialSummary>): CredentialSummary {
  return {
    id: "",
    label: "",
    login: "",
    pinned: false,
    lastUsedAt: 0,
    groups: [],
    tags: [],
    site: "",
    sites: [],
    email: "",
    oneTimeCode: false,
    passkeys: 0,
    ...fields,
  };
}

test("equal and one-sided fields need no choice; differing ones do", () => {
  const merge = new CredentialMerge(
    credential({ label: "Mail", login: "alex", password: "current" }),
    credential({
      id: "other",
      label: "Mail",
      email: "alex@example.com",
      password: "old",
      totp: "otpauth://totp/x?secret=JBSWY3DPEHPK3PXP",
    }),
    bounds,
  );
  assert.deepEqual(merge.conflicts, ["password"]);
  const kept = merge.input({});
  assert.equal(kept.password, "current");
  assert.equal(kept.login, "alex");
  assert.equal(kept.email, "alex@example.com");
  assert.equal(kept.totp, "otpauth://totp/x?secret=JBSWY3DPEHPK3PXP");
  assert.equal(merge.input({ password: "other" }).password, "old");
});

test("lists combine without repeats, the kept password's first, within bounds", () => {
  const merge = new CredentialMerge(
    credential({
      websites: ["mail.example.com", "example.com"],
      tags: ["Work"],
      groups: ["g1"],
      apps: [{ package: "com.example.mail", signer: "ab", name: "Mail" }],
    }),
    credential({
      id: "other",
      websites: ["EXAMPLE.com", "login.example.com", "old.example.com"],
      tags: ["work", "Old"],
      groups: ["g2", "g1"],
      apps: [
        { package: "com.example.mail", signer: "ab", name: "Mail" },
        { package: "com.example.chat", signer: "cd", name: "Chat" },
      ],
    }),
    bounds,
  );
  const input = merge.input({});
  assert.deepEqual(input.websites, [
    "mail.example.com",
    "example.com",
    "login.example.com",
  ]);
  assert.deepEqual(merge.dropped, { websites: 1, tags: 0 });
  assert.deepEqual(input.tags, ["Work", "Old"]);
  assert.deepEqual(merge.groups(), ["g1", "g2"]);
  assert.deepEqual(
    input.apps.map((app) => app.package),
    ["com.example.mail", "com.example.chat"],
  );
});

test("notes keep one side, or both when they fit together", () => {
  const merge = new CredentialMerge(
    credential({ notes: "PIN 1234" }),
    credential({ id: "other", notes: "Recovery code" }),
    bounds,
  );
  assert.deepEqual(merge.conflicts, ["notes"]);
  assert.equal(merge.input({}).notes, "PIN 1234");
  assert.equal(merge.input({ notes: "other" }).notes, "Recovery code");
  assert.equal(
    merge.input({ notes: "both" }).notes,
    "PIN 1234\n\nRecovery code",
  );
  const long = new CredentialMerge(
    credential({ notes: "x".repeat(30) }),
    credential({ id: "other", notes: "y".repeat(10) }),
    bounds,
  );
  assert.equal(long.notesFitTogether, false);
  assert.equal(long.input({ notes: "both" }).notes, "x".repeat(30));
});

test("candidates sharing a site come first, then a shared account, then the rest", () => {
  const open = summary({
    id: "open",
    label: "Mail",
    login: "alex",
    sites: ["example.com"],
  });
  const all = [
    open,
    summary({ id: "b", label: "Bank" }),
    summary({ id: "a", label: "Account", login: "ALEX" }),
    summary({ id: "m", label: "Mail old", sites: ["example.com"] }),
  ];
  assert.deepEqual(
    mergeCandidates(open, all, "").map((item) => item.id),
    ["m", "a", "b"],
  );
  assert.deepEqual(
    mergeCandidates(open, all, "bank").map((item) => item.id),
    ["b"],
  );
});
