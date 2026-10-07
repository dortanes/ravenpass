import { sameSite } from "./background/same-site.ts";

/**
 * `strong`: exact host on a secure page; `same-site`: `https` page of the matched website's registrable domain;
 * `other-site`: any other secure page matching by registrable domain only.
 */
export type MatchStrength =
  | "strong"
  | "same-site"
  | "other-site"
  | "insecure-page";

/** `site` is the saved website the vault matched the page's origin with. */
export function matchStrength(
  exact: boolean,
  site: string,
  origin: string,
): MatchStrength {
  if (!isSecure(origin)) return "insecure-page";
  if (exact) return "strong";
  return sameSite(origin, `https://${site}`) ? "same-site" : "other-site";
}

/** An `https` origin, or an `http` one on `localhost` or a `.localhost` host. */
export function isSecure(origin: string): boolean {
  if (!URL.canParse(origin)) return false;
  const { protocol, hostname } = new URL(origin);
  if (protocol === "https:") return true;
  return (
    protocol === "http:" &&
    (hostname === "localhost" || hostname.endsWith(".localhost"))
  );
}
