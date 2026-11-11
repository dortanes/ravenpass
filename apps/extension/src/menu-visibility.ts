/** Whether this engine can report occlusion across the embedding frame boundary. */
export function tracksMenuVisibility(): boolean {
  return (
    typeof IntersectionObserverEntry !== "undefined" &&
    "isVisible" in IntersectionObserverEntry.prototype &&
    typeof IntersectionObserver !== "undefined" &&
    "trackVisibility" in IntersectionObserver.prototype
  );
}

/** These operations may release secrets or authorize an account action. */
export function needsMenuVisibility(kind: string): boolean {
  return [
    "menu-fill",
    "menu-code",
    "menu-fill-code",
    "menu-fill-card",
    "menu-share",
    "passkey-sign",
    "passkey-save",
  ].includes(kind);
}
