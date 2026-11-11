import { CardFrames } from "./card-frames.ts";
import { CornerCard } from "./corner-card.ts";
import { FieldMenus } from "./field-menus.ts";
import { FileMenus } from "./file-menus.ts";
import { MenuFrame } from "./menu-frame.ts";
import { SignInForms } from "./sign-in-forms.ts";
import { SubmissionWatcher } from "./submissions.ts";

// Extension.js 4 mounts a content script through its default export, which returns the unmount.
export default function main(): () => void {
  const onVisibility = (
    message: unknown,
    sender: chrome.runtime.MessageSender,
    respond: (answer: { visible: boolean }) => void,
  ) => {
    if (
      sender.id !== chrome.runtime.id ||
      !message ||
      typeof message !== "object" ||
      !("kind" in message) ||
      message.kind !== "menu-visibility" ||
      !("token" in message) ||
      typeof message.token !== "string"
    )
      return;
    respond({ visible: MenuFrame.visible(message.token) });
  };
  chrome.runtime.onMessage.addListener(onVisibility);
  const fieldMenus = new FieldMenus(document);
  const fileMenus = new FileMenus(document);
  const cardFrames = new CardFrames(document);
  const submissions = new SubmissionWatcher(document);
  const signInForms = new SignInForms(document, {
    typing: (inputs) => fieldMenus.dismiss(inputs),
  });
  // The card shows in the tab's top frame only, for a form of that frame or of a frame of its origin or site.
  const cornerCard = window === window.top ? new CornerCard(document) : null;
  fieldMenus.start();
  fileMenus.start();
  cardFrames.start();
  submissions.start();
  signInForms.start();
  cornerCard?.start();
  return () => {
    chrome.runtime.onMessage.removeListener(onVisibility);
    fieldMenus.stop();
    fileMenus.stop();
    cardFrames.stop();
    submissions.stop();
    signInForms.stop();
    cornerCard?.stop();
  };
}
