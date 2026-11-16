import assert from "node:assert/strict";
import test from "node:test";
import { checkNewPin } from "./new-pin.ts";

test("a PIN shorter than the vault takes is not kept, whatever its repeat", () => {
  assert.equal(checkNewPin("", "", 6), "short");
  assert.equal(checkNewPin("12345", "12345", 6), "short");
  assert.equal(checkNewPin("12345", "99999", 6), "short");
});

test("a PIN long enough is kept once its repeat is the same", () => {
  assert.equal(checkNewPin("123456", "123456", 6), "matched");
  assert.equal(checkNewPin("123456789012", "123456789012", 6), "matched");
  assert.equal(checkNewPin("Pass123", "Pass123", 6), "matched");
  assert.equal(checkNewPin("Pass123", "pass123", 6), "mismatched");
});

test("a repeat shorter than the PIN is still being typed", () => {
  assert.equal(checkNewPin("123456", "", 6), "repeating");
  assert.equal(checkNewPin("123456", "12345", 6), "repeating");
  assert.equal(checkNewPin("123456", "99", 6), "repeating");
});

test("a repeat as long as the PIN or longer that differs is a mismatch", () => {
  assert.equal(checkNewPin("123456", "123457", 6), "mismatched");
  assert.equal(checkNewPin("123456", "1234567", 6), "mismatched");
});
