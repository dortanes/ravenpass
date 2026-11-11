/** Bounded references only: browser exception messages and connection data must not cross this boundary. */
export const pairingDiagnostics = [
  "RP-00",
  "RP-01",
  "RP-02",
  "RP-03",
  "RP-04",
  "RP-05",
  "RP-06",
  "RP-07",
  "RP-08",
  "RP-09",
  "RP-10",
  "RP-11",
  "RP-12",
  "RP-13",
  "RP-14",
  "RP-15",
] as const;

export type PairingDiagnostic = (typeof pairingDiagnostics)[number];

export function isPairingDiagnostic(
  value: unknown,
): value is PairingDiagnostic {
  return pairingDiagnostics.some((diagnostic) => diagnostic === value);
}
