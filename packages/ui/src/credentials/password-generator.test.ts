import assert from "node:assert/strict";
import test from "node:test";
import {
  defaultGeneratorOptions,
  PasswordGenerator,
  type RandomIndex,
  randomIndex,
  readGeneratorOptions,
} from "./password-generator.ts";

const words = ["orbit", "velvet", "canyon", "ladder"];

/** A source that returns the given draws in turn, each below its bound. */
function scripted(...draws: number[]): RandomIndex {
  let next = 0;
  return (bound) => {
    const value = draws[next++ % draws.length] ?? 0;
    assert.ok(value < bound, `draw ${value} is not below ${bound}`);
    return value;
  };
}

test("randomIndex stays below its bound and refuses an empty range", () => {
  for (const bound of [1, 2, 7, 2048]) {
    for (let i = 0; i < 200; i++) {
      const value = randomIndex(bound);
      assert.ok(Number.isInteger(value) && value >= 0 && value < bound);
    }
  }
  assert.throws(() => randomIndex(0), RangeError);
  assert.throws(() => randomIndex(1.5), RangeError);
});

test("the default is five capitalized words joined by hyphens with one digit", () => {
  // Words 0, 1, 2, 3, 0; the digit goes after word 2 and is 7.
  const generator = new PasswordGenerator(words, scripted(0, 1, 2, 3, 0, 2, 7));
  const { password, bits } = generator.generate(defaultGeneratorOptions);
  assert.equal(password, "Orbit-Velvet-Canyon7-Ladder-Orbit");
  assert.equal(bits, 5 * Math.log2(4) + Math.log2(10 * 5));
});

test("word options change the count, the separator and the capitals", () => {
  const generator = new PasswordGenerator(words, scripted(3, 2, 1));
  const { password, bits } = generator.generate({
    ...defaultGeneratorOptions,
    words: 3,
    separator: " ",
    capitalize: false,
    digit: false,
  });
  assert.equal(password, "ladder canyon velvet");
  assert.equal(bits, 3 * Math.log2(4));
});

test("an out-of-bounds count is brought within bounds", () => {
  const generator = new PasswordGenerator(words);
  const short = generator.generate({
    ...defaultGeneratorOptions,
    words: 1,
    digit: false,
  });
  assert.equal(short.password.split("-").length, 3);
  const long = generator.generate({
    ...defaultGeneratorOptions,
    mode: "characters",
    length: 500,
  });
  assert.equal(long.password.length, 64);
});

test("character passwords use every chosen class and no other", () => {
  const generator = new PasswordGenerator(words);
  for (let i = 0; i < 50; i++) {
    const { password } = generator.generate({
      ...defaultGeneratorOptions,
      mode: "characters",
      length: 8,
    });
    assert.match(password, /[a-z]/);
    assert.match(password, /[A-Z]/);
    assert.match(password, /[0-9]/);
    assert.match(password, /[^a-zA-Z0-9]/);
  }
  const { password, bits } = generator.generate({
    ...defaultGeneratorOptions,
    mode: "characters",
    length: 20,
    uppercase: false,
    digits: false,
    symbols: false,
  });
  assert.match(password, /^[a-z]{20}$/);
  assert.equal(bits, 20 * Math.log2(26));
});

test("strength follows the entropy", () => {
  assert.equal(PasswordGenerator.strength(39.9), "weak");
  assert.equal(PasswordGenerator.strength(40), "fair");
  assert.equal(PasswordGenerator.strength(61), "strong");
  assert.equal(PasswordGenerator.strength(80), "very-strong");
});

test("stored options keep what is valid and fall back for the rest", () => {
  assert.deepEqual(readGeneratorOptions(null), defaultGeneratorOptions);
  assert.deepEqual(readGeneratorOptions("{"), defaultGeneratorOptions);
  assert.deepEqual(readGeneratorOptions("7"), defaultGeneratorOptions);
  assert.deepEqual(
    readGeneratorOptions(
      JSON.stringify({
        mode: "characters",
        words: 11,
        separator: "/",
        capitalize: false,
        length: 32,
        symbols: "yes",
      }),
    ),
    {
      ...defaultGeneratorOptions,
      mode: "characters",
      capitalize: false,
      length: 32,
    },
  );
});
