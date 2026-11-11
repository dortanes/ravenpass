type ExtensionRuntime = Pick<typeof chrome.runtime, "id" | "getURL">;

/** Browser-supplied identity, never fields copied from the request payload. */
export interface ExtensionPageSender {
  readonly id?: string;
  readonly url?: string;
  readonly tab?: unknown;
}

/** Authenticates packaged extension pages independently of their browser-assigned paths. */
export class ExtensionPages {
  private readonly extensionId: string;
  private readonly baseURL: URL;

  constructor(runtime: ExtensionRuntime) {
    this.extensionId = runtime.id;
    this.baseURL = new URL(runtime.getURL(""));
  }

  allows(sender: ExtensionPageSender): boolean {
    if (
      !this.extensionId ||
      !this.baseURL.host ||
      sender.id !== this.extensionId ||
      !sender.url
    ) {
      return false;
    }
    try {
      const url = new URL(sender.url);
      // URL.origin can be "null" for unrelated custom-scheme URLs. Compare
      // the browser's protocol and host; paths do not establish identity.
      return (
        url.protocol === this.baseURL.protocol &&
        url.host === this.baseURL.host &&
        !url.username &&
        !url.password
      );
    } catch {
      return false;
    }
  }
}
