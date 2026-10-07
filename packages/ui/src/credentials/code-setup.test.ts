import assert from "node:assert/strict";
import test from "node:test";
import type { CredentialSummary } from "../vault-api.ts";
import { codeSetupChoices, namesCredential } from "./code-setup.ts";

function credential(
  id: string,
  label: string,
  login = "",
  email = "",
): CredentialSummary {
  return {
    id,
    label,
    login,
    email,
    pinned: false,
    lastUsedAt: 0,
    groups: [],
    tags: [],
    site: "",
    sites: [],
    oneTimeCode: false,
    passkeys: 0,
  };
}

const credentials = [
  credential("1", "GitHub", "alex"),
  credential("2", "Work mail", "alice", "alice@example.com"),
  credential("3", "Example", "bob"),
  credential("4", "Bank"),
];

function ids(found: CredentialSummary[]): string[] {
  return found.map((entry) => entry.id);
}

test("before a search, the setup's issuer or account finds its credentials by name", () => {
  assert.deepEqual(
    ids(
      codeSetupChoices(
        credentials,
        { issuer: "example", account: "alice@example.com" },
        "",
      ),
    ),
    ["3", "2"],
  );
  assert.deepEqual(
    ids(
      codeSetupChoices(
        credentials,
        { issuer: "GitHub", account: "nobody" },
        "",
      ),
    ),
    ["1"],
  );
});

test("a credential found by both the issuer and the account is offered once", () => {
  assert.deepEqual(
    ids(
      codeSetupChoices(credentials, { issuer: "Work", account: "alice" }, " "),
    ),
    ["2"],
  );
});

test("a search looks through every credential", () => {
  assert.deepEqual(
    ids(
      codeSetupChoices(
        credentials,
        { issuer: "Example", account: "alice" },
        "bank",
      ),
    ),
    ["4"],
  );
});

test("a setup that names no one offers every credential", () => {
  assert.deepEqual(
    ids(codeSetupChoices(credentials, { issuer: "", account: "" }, "")),
    ["4", "3", "1", "2"],
  );
});

test("a new credential needs the setup's issuer or account for its name", () => {
  assert.ok(namesCredential({ issuer: "Example", account: "" }));
  assert.ok(namesCredential({ issuer: "", account: "alice" }));
  assert.ok(!namesCredential({ issuer: "", account: "" }));
});
