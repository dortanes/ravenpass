/**
 * Each `system.reason.*` message follows "Ravenpass is trying to". Each `system.prompt.*` message is a full
 * sentence under its `.title`, for a prompt that shows a title and a description.
 */
export const system = {
  "system.dialog.choose-vault-location": "Choose where to keep your vault",
  "system.dialog.select-vault-file": "Select a vault file",
  "system.dialog.save-export": "Save an encrypted copy of your vault",
  "system.dialog.save-recovery-key": "Save your recovery key",
  "system.dialog.choose-photo": "Choose a photo",
  "system.dialog.choose-scan": "Choose a scan of the document",
  "system.dialog.choose-qr-code": "Choose a picture of the QR code",
  "system.dialog.save-scan": "Save an unencrypted copy of the scan",
  "system.dialog.select-import": "Select the exported file",
  "system.dialog.choose-backup-folder": "Choose a folder for automatic backups",
  "system.filter.vault": "Ravenpass vault",
  "system.filter.text": "Text file",
  "system.filter.photo": "Image",
  "system.filter.scan": "Image or PDF",
  "system.filter.import": "Exported vault",
  "system.file.vault": "vault.rpv",
  "system.file.export": "Ravenpass export.rpv",
  "system.file.recovery-key": "Ravenpass recovery key.txt",
  "system.reason.share-file": "share “{file}” from {identity} with {site}",
  "system.reason.save-passkey": "save a passkey for {site}",
  "system.reason.sign-in-passkey":
    "sign in to {site} as {account} with a passkey",
  "system.reason.fill-sign-in": "fill the sign-in for {account} on {site}",
  "system.reason.fill-card": "fill the card “{card}” on {site}",
  "system.reason.change-unlock": "change how your vault unlocks",
  "system.reason.create-vault": "create a new vault",
  "system.reason.open-vault": "open another vault file",
  "system.reason.delete-vault": "delete the vault “{vault}”",
  "system.reason.reveal-item": "show “{item}”",
  "system.reason.unlock-vault": "unlock your vault",
  "system.prompt.share-file.title": "Share a file",
  "system.prompt.share-file":
    "Confirm it's you to share “{file}” from {identity} with {site}.",
  "system.prompt.save-passkey.title": "Save a passkey",
  "system.prompt.save-passkey":
    "Confirm it's you to save a passkey for {site}.",
  "system.prompt.sign-in-passkey.title": "Sign in with a passkey",
  "system.prompt.sign-in-passkey":
    "Confirm it's you to sign in to {site} as {account}.",
  "system.prompt.fill-sign-in.title": "Fill a sign-in",
  "system.prompt.fill-sign-in":
    "Confirm it's you to fill the sign-in for {account} on {site}.",
  "system.prompt.fill-card.title": "Fill a card",
  "system.prompt.fill-card":
    "Confirm it's you to fill the card “{card}” on {site}.",
  "system.prompt.change-unlock.title": "Change unlock settings",
  "system.prompt.change-unlock":
    "Confirm it's you to change how your vault unlocks.",
  "system.prompt.create-vault.title": "Create a vault",
  "system.prompt.create-vault": "Confirm it's you to create a new vault.",
  "system.prompt.open-vault.title": "Open a vault file",
  "system.prompt.open-vault": "Confirm it's you to open another vault file.",
  "system.prompt.reveal-item.title": "Show a secret",
  "system.prompt.reveal-item": "Confirm it's you to show “{item}”.",
  "system.prompt.delete-vault.title": "Delete a vault",
  "system.prompt.delete-vault":
    "Confirm it's you to delete the vault “{vault}”.",
  "system.prompt.unlock-vault.title": "Unlock Ravenpass",
  "system.prompt.unlock-vault": "Confirm it's you to unlock your vault.",
  "system.place.on-this-device": "On this device",
  "system.recovery-file.title": "Ravenpass recovery key",
  "system.recovery-file.notice":
    "This key unlocks an existing encrypted Ravenpass vault or export. It does not contain your passwords or a copy of the vault. This file is not encrypted. Keep it private.",
  "system.recovery-sheet.printed": "Printed on {date}",
  "system.recovery-sheet.keep": "Keep this sheet somewhere safe and private.",
  "system.recovery-sheet.warning":
    "Anyone with these words can open your vault.",
  "system.menu.open": "Open Ravenpass",
  "system.menu.lock": "Lock vault",
  "system.menu.quit": "Quit Ravenpass",
};
