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
}

export const generatorBounds = {
  words: { min: 3, max: 10 },
  length: { min: 8, max: 64 },
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

  // Drawn again until every chosen class appears, which a short password can miss.
  #characters(options: GeneratorOptions): GeneratedPassword {
    const length = clamp(options.length, generatorBounds.length);
    const classes = [lowercase];
    if (options.uppercase) classes.push(uppercase);
    if (options.digits) classes.push(digits);
    if (options.symbols) classes.push(symbols);
    const pool = classes.join("");
    for (;;) {
      const password = Array.from({ length }, () => this.#pick(pool)).join("");
      if (classes.every((set) => [...set].some((c) => password.includes(c)))) {
        return { password, bits: length * Math.log2(pool.length) };
      }
    }
  }

  #pick(from: string | readonly string[]): string {
    return from[this.#random(from.length)] ?? "";
  }
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
    key: "words" | "length",
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
  return {
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
  };
}
