import type { TagLimits } from "../vault-api.ts";

/** The most known tags the field suggests at once, fewer before anything is typed. */
const suggestionLimit = 6;
const untypedSuggestionLimit = 3;

/** withTag adds a typed tag, tidied as the vault keeps it, unless the vault refuses it, it is held already or past the limits. */
export function withTag(
  tags: readonly string[],
  typed: string,
  limits: TagLimits | null,
): string[] {
  const tag = typed.split(/\s+/).filter(Boolean).join(" ");
  const key = tag.toLocaleLowerCase();
  if (
    !tag ||
    /\p{Cc}/u.test(tag) ||
    tags.some((held) => held.toLocaleLowerCase() === key) ||
    (limits !== null &&
      ([...tag].length > limits.tag || tags.length >= limits.tags))
  ) {
    return [...tags];
  }
  return [...tags, tag];
}

/** suggestedTags are the known tags not chosen yet that begin with what is typed, ignoring case. */
export function suggestedTags(
  known: readonly string[],
  chosen: readonly string[],
  typed: string,
): string[] {
  const term = typed.trim().toLocaleLowerCase();
  const held = new Set(chosen.map((tag) => tag.toLocaleLowerCase()));
  return known
    .filter((tag) => {
      const key = tag.toLocaleLowerCase();
      return !held.has(key) && key.startsWith(term) && key !== term;
    })
    .slice(0, term ? suggestionLimit : untypedSuggestionLimit);
}
