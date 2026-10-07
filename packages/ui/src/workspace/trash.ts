const dayMillis = 24 * 60 * 60 * 1000;

/** The retention periods the trash offers, in days. */
export const retentionChoices: readonly number[] = [7, 30, 90, 365];

/** The whole days left before the trash deletes an item for good; zero on its last day and after. */
export function daysLeft(
  deletedAt: number,
  retentionDays: number,
  now: number,
): number {
  return Math.max(
    0,
    Math.floor((deletedAt + retentionDays * dayMillis - now) / dayMillis),
  );
}

/** How many items a period of retentionDays deletes for good at once, as the vault purges: kept the whole period. */
export function dueAt(
  deletedAt: readonly number[],
  retentionDays: number,
  now: number,
): number {
  return deletedAt.filter((at) => at + retentionDays * dayMillis <= now).length;
}

/** The periods to offer, the vault's own among them when another device set one the trash does not offer. */
export function retentionOptions(current: number): number[] {
  return retentionChoices.includes(current)
    ? [...retentionChoices]
    : [...retentionChoices, current].sort((a, b) => a - b);
}
