import assert from "node:assert/strict";
import test from "node:test";
import { SiteNaming } from "./site-naming.ts";

test("a website is looked up once, and not again once cut to its domain", () => {
  const naming = new SiteNaming(" example.com ");
  assert.equal(naming.begin("example.com"), "");
  assert.equal(naming.begin("  "), "");
  assert.equal(
    naming.begin(" https://login.example.org/a "),
    "https://login.example.org/a",
  );
  naming.cut("login.example.org");
  assert.equal(naming.begin("login.example.org"), "");
});

test("a site names a credential only while its name is empty or the one the last site gave", () => {
  const naming = new SiteNaming("");
  assert.equal(naming.names(" "), true);
  assert.equal(naming.names("Mail"), false);
  naming.filled("Example");
  assert.equal(naming.names("Example"), true);
  assert.equal(naming.names("Example mail"), false);
});
