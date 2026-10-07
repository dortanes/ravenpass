import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import type { VaultApi } from "../vault-api.ts";
import { queryKeys } from "./keys.ts";

/** How long typing pauses before a typed password is checked. */
const typingPause = 600;

/** What the last vault-wide check found. */
export interface BreachReport {
  /** How many passwords were checked. */
  checked: number;
  /** How many times each breached password was seen, by credential id. */
  counts: ReadonlyMap<string, number>;
  /** Why the check failed, null when it ran; a failed check knows nothing. */
  failure: unknown;
}

/** Where the vault-wide check stands while checks are on. */
export interface BreachStatus {
  /** Null until the first check ends. */
  report: BreachReport | null;
  /** Set while a check runs. */
  checking: boolean;
}

export function useBreachChecks(host: Pick<VaultApi, "breachChecks">) {
  return (
    useQuery({
      queryKey: queryKeys.breachChecks,
      queryFn: () => host.breachChecks(),
      meta: { failure: "app.error.setting-read" },
    }).data?.enabled ?? false
  );
}

/**
 * useBreachReport checks the vault's passwords while checks are on, again whenever `version` changes, and waits while
 * the passwords are unread (`version` null); null while checks are off. The last report stays shown while the next
 * check runs.
 */
export function useBreachReport(
  host: Pick<VaultApi, "breachChecks" | "checkBreaches">,
  version: number | null,
): BreachStatus | null {
  const checksOn = useBreachChecks(host);
  const report = useQuery({
    queryKey: queryKeys.breaches(version ?? 0),
    queryFn: async (): Promise<BreachReport> => {
      try {
        const found = await host.checkBreaches();
        return {
          checked: found.checked,
          counts: new Map(
            found.breaches.map((breach) => [breach.id, breach.count]),
          ),
          failure: null,
        };
      } catch (cause) {
        return { checked: 0, counts: new Map(), failure: cause };
      }
    },
    enabled: checksOn && version !== null,
    placeholderData: keepPreviousData,
  });
  if (!checksOn) return null;
  return {
    report: report.data ?? null,
    checking: version === null || report.isFetching,
  };
}

/** breachedFirst moves the breached passwords ahead of the rest, keeping the order within each part. */
export function breachedFirst<Entry extends { id: string }>(
  entries: Entry[],
  counts: ReadonlyMap<string, number>,
): Entry[] {
  if (!counts.size) return entries;
  return [
    ...entries.filter((entry) => counts.has(entry.id)),
    ...entries.filter((entry) => !counts.has(entry.id)),
  ];
}

/** useBreachCount is how many times a typed password was seen in breaches; zero while unknown. */
export function useBreachCount(
  host: Pick<VaultApi, "breachChecks" | "checkPassword">,
  password: string,
): number {
  const enabled = useBreachChecks(host);
  const [found, setFound] = useState({ password: "", count: 0 });
  useEffect(() => {
    if (!enabled || !password) return;
    let current = true;
    const timer = setTimeout(() => {
      host.checkPassword(password).then(
        (count) => {
          if (current) setFound({ password, count });
        },
        // An unanswered check warns of nothing; the saved password is checked again with the vault.
        () => undefined,
      );
    }, typingPause);
    return () => {
      current = false;
      clearTimeout(timer);
    };
  }, [enabled, password, host]);
  return enabled && found.password === password ? found.count : 0;
}
