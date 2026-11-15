/** Words joined into a phrase, or characters drawn one by one. */
export type GeneratorMode = "words" | "characters";

export type WordSeparator = "-" | "." | "_" | " ";

export const wordSeparators: readonly WordSeparator[] = ["-", ".", "_", " "];

export interface GeneratorOptions {
  mode: GeneratorMode;
  words: number;
  separator: WordSeparator;
  /** Starts every word with a capital. */
  capitalize: boolean;
  /** Adds one digit to the end of one word. */
  digit: boolean;
  length: number;
  uppercase: boolean;
  digits: boolean;
  symbols: boolean;
  /** The fewest digits a password holds while digits are on. */
  fewestDigits: number;
  /** The fewest symbols a password holds while symbols are on. */
  fewestSymbols: number;
  /** Leaves out I, O, l, 0 and 1, which read alike in many fonts. */
  skipLookAlikes: boolean;
}

export const generatorBounds = {
  words: { min: 3, max: 10 },
  length: { min: 8, max: 64 },
  fewest: { min: 1, max: 9 },
} as const;

export const defaultGeneratorOptions: GeneratorOptions = {
  mode: "words",
  words: 5,
  separator: "-",
  capitalize: true,
  digit: true,
  length: 20,
  uppercase: true,
  digits: true,
  symbols: true,
  fewestDigits: 1,
  fewestSymbols: 1,
  skipLookAlikes: false,
};

/** How hard a password is to guess, from its entropy. */
export type Strength = "weak" | "fair" | "strong" | "very-strong";

export interface GeneratedPassword {
  password: string;
  /** Entropy in bits, given the options and the list it was drawn from. */
  bits: number;
}

const lowercase = "abcdefghijklmnopqrstuvwxyz";
const uppercase = lowercase.toUpperCase();
const digits = "0123456789";
const symbols = "!#$%&*+-=?@^_";
const lookAlikes = new Set("IOl01");

function withoutLookAlikes(set: string): string {
  return [...set].filter((character) => !lookAlikes.has(character)).join("");
}

/** Draws an integer from 0 up to bound, exclusive, uniformly. */
export type RandomIndex = (bound: number) => number;

const uint32Range = 2 ** 32;

/** randomIndex draws with the platform's cryptographic source, rejecting draws past the last whole multiple of bound. */
export function randomIndex(bound: number): number {
  if (!Number.isInteger(bound) || bound < 1 || bound > uint32Range) {
    throw new RangeError(`cannot draw below ${bound}`);
  }
  const limit = uint32Range - (uint32Range % bound);
  const draw = new Uint32Array(1);
  for (;;) {
    crypto.getRandomValues(draw);
    const value = draw[0] ?? 0;
    if (value < limit) return value % bound;
  }
}

/** PasswordGenerator makes passwords from a word list or from character classes. */
export class PasswordGenerator {
  readonly #words: readonly string[];
  readonly #random: RandomIndex;

  constructor(words: readonly string[], random: RandomIndex = randomIndex) {
    if (words.length < 2) throw new RangeError("the word list is too short");
    this.#words = words;
    this.#random = random;
  }

  generate(options: GeneratorOptions): GeneratedPassword {
    return options.mode === "words"
      ? this.#phrase(options)
      : this.#characters(options);
  }

  static strength(bits: number): Strength {
    if (bits < 40) return "weak";
    if (bits < 60) return "fair";
    if (bits < 80) return "strong";
    return "very-strong";
  }

  #phrase(options: GeneratorOptions): GeneratedPassword {
    const count = clamp(options.words, generatorBounds.words);
    const words = Array.from({ length: count }, () => {
      const word = this.#pick(this.#words);
      return options.capitalize ? capitalized(word) : word;
    });
    let bits = count * Math.log2(this.#words.length);
    if (options.digit) {
      const at = this.#random(count);
      words[at] = `${words[at]}${this.#pick(digits)}`;
      bits += Math.log2(digits.length * count);
    }
    return { password: words.join(options.separator), bits };
  }

  // Random positions take the characters the options require, each drawn from its own class; the other positions
  // draw from every chosen class. The bits count each draw from the set it was made from, never the positions.
  #characters(options: GeneratorOptions): GeneratedPassword {
    const fitted = fitCharacterOptions(options, "length");
    const sets = characterSets(fitted);
    const pool = sets.chosen.join("");
    const positions = this.#shuffled(fitted.length);
    const drawn: string[] = Array.from({ length: fitted.length });
    // Draws per set size, so equal sets add up as one product.
    const draws = new Map<number, number>();
    positions.forEach((position, index) => {
      const set = sets.required[index] ?? pool;
      drawn[position] = this.#pick(set);
      draws.set(set.length, (draws.get(set.length) ?? 0) + 1);
    });
    let bits = 0;
    for (const [size, count] of draws) bits += count * Math.log2(size);
    return { password: drawn.join(""), bits };
  }

  /** The positions 0 up to count in a uniformly random order. */
  #shuffled(count: number): number[] {
    const order = Array.from({ length: count }, (_, index) => index);
    for (let last = count - 1; last > 0; last--) {
      const other = this.#random(last + 1);
      [order[last], order[other]] = [order[other] ?? 0, order[last] ?? 0];
    }
    return order;
  }

  #pick(from: string | readonly string[]): string {
    return from[this.#random(from.length)] ?? "";
  }
}

/** The classes options choose, and one set per character a password must hold. */
function characterSets(options: GeneratorOptions): {
  chosen: string[];
  required: string[];
} {
  const set = (characters: string) =>
    options.skipLookAlikes ? withoutLookAlikes(characters) : characters;
  const chosen = [set(lowercase)];
  const required = [set(lowercase)];
  if (options.uppercase) {
    chosen.push(set(uppercase));
    required.push(set(uppercase));
  }
  if (options.digits) {
    chosen.push(set(digits));
    for (let i = 0; i < options.fewestDigits; i++) required.push(set(digits));
  }
  if (options.symbols) {
    chosen.push(symbols);
    for (let i = 0; i < options.fewestSymbols; i++) required.push(symbols);
  }
  return { chosen, required };
}

function requiredCount(options: GeneratorOptions): number {
  return characterSets(options).required.length;
}

/**
 * fitCharacterOptions bounds every count and keeps the required characters within the length: a raised fewest count
 * lengthens the password, and a shortened length lowers the larger fewest count until the password holds them.
 */
export function fitCharacterOptions(
  options: GeneratorOptions,
  changed: "length" | "fewest",
): GeneratorOptions {
  const fitted: GeneratorOptions = {
    ...options,
    length: clamp(options.length, generatorBounds.length),
    fewestDigits: clamp(options.fewestDigits, generatorBounds.fewest),
    fewestSymbols: clamp(options.fewestSymbols, generatorBounds.fewest),
  };
  if (changed === "fewest") {
    fitted.length = Math.max(fitted.length, requiredCount(fitted));
    return fitted;
  }
  while (requiredCount(fitted) > fitted.length) {
    const digitsCanDrop = fitted.digits && fitted.fewestDigits > 1;
    const symbolsCanDrop = fitted.symbols && fitted.fewestSymbols > 1;
    if (
      symbolsCanDrop &&
      (!digitsCanDrop || fitted.fewestSymbols >= fitted.fewestDigits)
    ) {
      fitted.fewestSymbols--;
    } else if (digitsCanDrop) {
      fitted.fewestDigits--;
    } else {
      break;
    }
  }
  return fitted;
}

/** One stretch of a shown password and whether it is tinted as digits or symbols. */
export interface PasswordPart {
  text: string;
  tint: "digit" | "symbol" | null;
}

/** passwordParts splits a password for showing: digits tinted always, symbols only in a character password. */
export function passwordParts(
  password: string,
  mode: GeneratorMode,
): PasswordPart[] {
  const parts: PasswordPart[] = [];
  for (const character of password) {
    const tint = digits.includes(character)
      ? "digit"
      : mode === "characters" && symbols.includes(character)
        ? "symbol"
        : null;
    const last = parts.at(-1);
    if (last && last.tint === tint) last.text += character;
    else parts.push({ text: character, tint });
  }
  return parts;
}

function capitalized(word: string): string {
  return word.charAt(0).toUpperCase() + word.slice(1);
}

function clamp(value: number, bounds: { min: number; max: number }): number {
  return Math.min(bounds.max, Math.max(bounds.min, Math.round(value)));
}

/** readGeneratorOptions reads stored options, keeping each valid one and the default for the rest. */
export function readGeneratorOptions(stored: string | null): GeneratorOptions {
  let parsed: unknown;
  try {
    parsed = JSON.parse(stored ?? "null");
  } catch {
    return defaultGeneratorOptions;
  }
  if (typeof parsed !== "object" || parsed === null) {
    return defaultGeneratorOptions;
  }
  const value = parsed as Record<string, unknown>;
  const defaults = defaultGeneratorOptions;
  const flag = (key: keyof GeneratorOptions, fallback: boolean) =>
    typeof value[key] === "boolean" ? value[key] : fallback;
  const count = (
    key: "words" | "length" | "fewestDigits" | "fewestSymbols",
    bounds: { min: number; max: number },
  ) => {
    const number = value[key];
    return typeof number === "number" &&
      Number.isInteger(number) &&
      number >= bounds.min &&
      number <= bounds.max
      ? number
      : defaults[key];
  };
  const separator = wordSeparators.find((item) => item === value.separator);
  return fitCharacterOptions(
    {
      mode:
        value.mode === "words" || value.mode === "characters"
          ? value.mode
          : defaults.mode,
      words: count("words", generatorBounds.words),
      separator: separator ?? defaults.separator,
      capitalize: flag("capitalize", defaults.capitalize),
      digit: flag("digit", defaults.digit),
      length: count("length", generatorBounds.length),
      uppercase: flag("uppercase", defaults.uppercase),
      digits: flag("digits", defaults.digits),
      symbols: flag("symbols", defaults.symbols),
      fewestDigits: count("fewestDigits", generatorBounds.fewest),
      fewestSymbols: count("fewestSymbols", generatorBounds.fewest),
      skipLookAlikes: flag("skipLookAlikes", defaults.skipLookAlikes),
    },
    "fewest",
  );
}
