/**
 * SiteNaming decides when a credential's first website is looked up and whether the site's name may replace the
 * credential's name: only while that name is empty or still the one the last site gave.
 */
export class SiteNaming {
  #lookedUp: string;
  #filled = "";

  constructor(firstWebsite: string) {
    this.#lookedUp = firstWebsite.trim();
  }

  /** The website to look up, empty when it is blank or the one looked up last. */
  begin(website: string): string {
    const site = website.trim();
    if (!site || site === this.#lookedUp) return "";
    this.#lookedUp = site;
    return site;
  }

  /** Records the address a lookup cut the website to, which needs no lookup of its own. */
  cut(website: string): void {
    this.#lookedUp = website;
  }

  names(label: string): boolean {
    return !label.trim() || label === this.#filled;
  }

  /** Records a name a site gave. */
  filled(name: string): void {
    this.#filled = name;
  }
}
