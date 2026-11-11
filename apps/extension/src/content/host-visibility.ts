/** A conservative fallback for engines without cross-frame occlusion tracking. */
export class HostVisibility {
  private readonly host: HTMLElement;
  private readonly children: readonly Element[];

  constructor(host: HTMLElement, children: readonly Element[]) {
    this.host = host;
    this.children = children;
  }

  visible(): boolean {
    try {
      if (!this.host.matches(":popover-open") || !this.clear(this.host))
        return false;
      if (this.children.some((element) => !this.clear(element))) return false;
      const view = this.host.ownerDocument.defaultView;
      // Geometric hit testing in a subframe cannot prove that its embedding iframe is unobscured.
      if (!view || view !== view.top || !this.uncovered(this.host))
        return false;
      for (
        let ancestor: Element | null = this.host;
        ancestor;
        ancestor = ancestor.parentElement
      ) {
        if (!this.clear(ancestor) || this.pseudo(ancestor)) return false;
      }
      return true;
    } catch {
      return false;
    }
  }

  private clear(element: Element): boolean {
    const view = element.ownerDocument.defaultView;
    if (
      !view ||
      !element.isConnected ||
      element.ownerDocument.visibilityState !== "visible"
    )
      return false;
    if (
      !element.checkVisibility({ checkOpacity: true, checkVisibilityCSS: true })
    )
      return false;
    const style = view.getComputedStyle(element);
    return (
      Number(style.opacity) === 1 &&
      style.filter === "none" &&
      (!style.backdropFilter || style.backdropFilter === "none") &&
      style.transform === "none" &&
      style.mixBlendMode === "normal" &&
      style.clipPath === "none" &&
      (!style.maskImage || style.maskImage === "none")
    );
  }

  private pseudo(element: Element): boolean {
    const view = element.ownerDocument.defaultView;
    if (!view) return true;
    return ["::before", "::after"].some((pseudo) => {
      const style = view.getComputedStyle(element, pseudo);
      return (
        !["none", "normal", ""].includes(style.content) &&
        style.display !== "none" &&
        style.visibility === "visible" &&
        Number(style.opacity) > 0
      );
    });
  }

  private uncovered(element: Element): boolean {
    const document = element.ownerDocument;
    const view = document.defaultView;
    if (!view) return false;
    const box = element.getBoundingClientRect();
    if (
      box.width <= 0 ||
      box.height <= 0 ||
      box.left < 0 ||
      box.top < 0 ||
      box.right > view.innerWidth ||
      box.bottom > view.innerHeight
    )
      return false;
    for (const other of document.querySelectorAll(
      ":popover-open, dialog:modal",
    )) {
      if (other === element || other.contains(element)) continue;
      // Another top-layer surface can cover us through its viewport-wide ::backdrop,
      // including when the surface itself is elsewhere or ignores pointer events.
      return false;
    }
    if (
      document.getAnimations().some((animation) => {
        const effect = animation.effect;
        return (
          effect &&
          "pseudoElement" in effect &&
          typeof effect.pseudoElement === "string" &&
          effect.pseudoElement.startsWith("::view-transition")
        );
      })
    )
      return false;
    return [0.1, 0.5, 0.9].every((x) =>
      [0.1, 0.5, 0.9].every((y) => {
        const hit = document.elementFromPoint(
          box.left + box.width * x,
          box.top + box.height * y,
        );
        return hit === element || (hit !== null && element.contains(hit));
      }),
    );
  }
}
