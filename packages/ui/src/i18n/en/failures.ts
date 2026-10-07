/** One message per failure code the desktop service reports. */
export const failures = {
  "failure.general":
    "Ravenpass could not complete this action. Try again, or reopen the vault if the problem continues.",
  "failure.window-unavailable":
    "Ravenpass cannot reach its window. Reopen the app and try again.",
  "failure.authentication-canceled":
    "Authentication was canceled. Try again when you're ready.",
  "failure.authentication-failed":
    "Your identity could not be verified. Try again.",
  "failure.unlock-unavailable":
    "Secure device unlock is unavailable. Check your device's security settings and try again.",
  "failure.unlock-key-missing":
    "Biometrics can no longer unlock this vault on this device. Use your PIN or recovery key, then turn biometrics on again in Settings.",
  "failure.vault-locked": "The vault is locked. Unlock it and try again.",
  "failure.vault-exists":
    "A vault already exists there. Choose another file for the new vault.",
  "failure.setup-in-progress":
    "Setup is already running. Finish or cancel it first.",
  "failure.setup-not-active":
    "Vault setup is no longer active. Start creating your vault again.",
  "failure.no-local-vault":
    "No vault was found there. Select an encrypted copy to restore it.",
  "failure.recovery-replaced-key":
    "This copy uses a recovery key that has since been replaced. Confirm that you restored it yourself before opening it.",
  "failure.key-change-unfinished":
    "The recovery key changed, but Ravenpass could not finish saving the change on this device. Use the new recovery key from now on, and unlock the vault again.",
  "failure.key-change-uncertain":
    "Ravenpass could not confirm whether the recovery key changed. Keep both the old and the new recovery key until the vault opens again.",
  "failure.recovery-needs-confirmation":
    "This copy may be older than your latest vault. Confirm the possible loss of changes before restoring it.",
  "failure.recovery-changed":
    "The vault changed during recovery. Start the recovery again.",
  "failure.recovery-phrase-invalid":
    "That recovery key does not match. Check the words and try again.",
  "failure.recovery-key-mismatch":
    "The recovery key does not match this vault. Check the words and try again.",
  "failure.recovery-key-not-saved":
    "No file was saved. Choose a location to save your recovery key.",
  "failure.recovery-key-save-failed":
    "The recovery key could not be saved. Choose another location and try again.",
  "failure.recovery-key-copy-failed":
    "The recovery key could not be copied. Save it to a private file or write it down.",
  "failure.recovery-key-print-failed":
    "The recovery key could not be printed. Check your printer and try again, or write the words down.",
  "failure.vault-changed":
    "The vault changed since it was opened. Lock Ravenpass and reopen it before editing.",
  "failure.vault-missing":
    "The vault file is no longer where Ravenpass opened it. It may have been moved on another device. Open the vault file from its new place.",
  "failure.save-unconfirmed":
    "The save could not be confirmed. Reopen Ravenpass before making more changes.",
  "failure.authentication-invalid":
    "The vault could not be opened. Check the recovery key or use a copy you know is good.",
  "failure.unsupported-format":
    "This vault file was written in a format this version of Ravenpass cannot open. Restore it from an encrypted copy with your recovery key, or start a new vault.",
  "failure.malformed-vault":
    "This vault file is not readable. Restore it from an encrypted copy with your recovery key.",
  "failure.invalid-item": "Some fields are invalid. Check them and try again.",
  "failure.invalid-code":
    "That one-time code setup cannot produce a code. Paste the setup key the site shows, or its one-time code link.",
  "failure.no-code":
    "This password has no one-time code set up. Add one to the item, then try again.",
  "failure.code-setup-expired":
    "This one-time code setup expired. Open its link or scan its QR code again.",
  "failure.qr-code-missing":
    "No QR code was found. Choose or copy a picture that shows the whole code.",
  "failure.qr-code-not-setup":
    "This QR code does not set up one-time codes. Use the QR code the site shows for an authenticator app.",
  "failure.qr-code-ambiguous":
    "The picture shows more than one setup code. Use a picture with only one.",
  "failure.resource-limit":
    "This item or vault exceeds the size limit. Shorten the item and try again.",
  "failure.passkeys-full":
    "Together these passwords hold more passkeys than one can. Remove a passkey from one of them, then merge.",
  "failure.breach-checks-off":
    "Breach checks are off. Turn them on in Privacy settings.",
  "failure.breach-check-unreachable":
    "Ravenpass could not reach the breach check service. Check your connection and try again.",
  "failure.permission-denied":
    "Ravenpass cannot use that file or folder. Choose another location and try again.",
  "failure.file-size-invalid":
    "That file is empty or too large. Select a valid Ravenpass file.",
  "failure.file-unverified":
    "The file could not be verified. Choose another location and try again.",
  "failure.file-not-selected": "No file was selected. Choose one to continue.",
  "failure.location-not-selected":
    "The location could not be selected. Try again.",
  "failure.item-unreadable":
    "This item could not be opened. Select it again from the list.",
  "failure.photo-unsupported":
    "That picture could not be read. Choose a JPEG, PNG or WebP photo.",
  "failure.photo-too-large":
    "That picture is too large. Choose one under 32 MB with fewer than 50 megapixels.",
  "failure.scan-unsupported":
    "That file could not be read as a scan. Choose a JPEG, PNG or WebP picture, or a PDF.",
  "failure.scan-too-large":
    "That scan is too large to keep. Choose a smaller picture, or a PDF under 960 KB.",
  "failure.scan-limit":
    "This document already has 4 scans. Remove one before attaching another.",
  "failure.field-not-copyable":
    "That field cannot be copied. Choose another field.",
  "failure.field-empty":
    "That field is empty. Add a value to it before copying.",
  "failure.copy-failed":
    "The value could not be copied. Open the item and copy it by hand.",
  "failure.website-missing":
    "This password has no website saved. Add the website's address to the item, then try again.",
  "failure.website-unsupported":
    "That website address is not a web address Ravenpass can open. Change it to an address that starts with https://, or copy it and open it yourself.",
  "failure.website-open-failed":
    "That website could not be opened. Copy the address and open it yourself.",
  "failure.export-canceled":
    "Export canceled. Choose a file location to save a copy.",
  "failure.export-stale":
    "The vault changed before the export. Reopen your vault and export a new copy.",
  "failure.export-unconfirmed":
    "An encrypted copy was saved, but Ravenpass could not confirm it is the latest. Reopen the vault and export again.",
  "failure.export-unrecorded":
    "An encrypted copy was saved, but Ravenpass could not record it. Export again so Settings can report the latest copy.",
  "failure.storage-unavailable":
    "Ravenpass cannot reach the folder that holds your vault. Reconnect it, or select your vault file.",
  "failure.storage-selection-invalid":
    "Ravenpass could not read where your vault is kept. Select your vault file to continue.",
  "failure.storage-occupied":
    "A file already exists there. Choose another name or folder.",
  "failure.storage-same-location":
    "Your vault is already kept there. Choose another folder to move it.",
  "failure.storage-kind-unsupported":
    "This version keeps your vault in a file. Choose a file location.",
  "failure.storage-path-invalid":
    "That location cannot hold a vault file. Choose another name or folder.",
  "failure.move-unverified":
    "The vault could not be verified at the new location. It stays where it was.",
  "failure.second-copy":
    "That file is another copy of a vault you already have. Use one file as your vault and keep the other as an encrypted backup.",
  "failure.older-copy":
    "This file is older than the vault used on this device. Wait for the latest version to sync, or switch back to the vault file you were using.",
  "failure.vault-diverged":
    "This vault was changed on another device at the same time. Open Ravenpass and unlock it there to continue with that version.",
  "failure.vault-key-replaced":
    "This vault's recovery key was changed on another device. Unlock it once with the new recovery key.",
  "failure.witness-missing":
    "This device has no record of that vault. Restore it with your recovery key.",
  "failure.vault-unknown":
    "Ravenpass does not know that vault. Reopen the app and try again.",
  "failure.no-other-vault":
    "There is no other vault to go back to. Finish creating this one, or open a vault file.",
  "failure.group-unknown":
    "Ravenpass does not know that group. Reopen the vault and try again.",
  "failure.group-name-invalid":
    "That group name is empty, too long, or already taken. Choose another one.",
  "failure.language-unsupported":
    "Ravenpass does not offer that language. Choose one from the list.",
  "failure.clear-delay-unsupported":
    "Ravenpass does not offer that delay. Choose one from the list.",
  "failure.appearance-unsupported":
    "Ravenpass does not offer that appearance. Choose one from the list.",
  "failure.sign-in-style-unsupported":
    "Ravenpass does not offer that way to sign in. Choose one from the list.",
  "failure.auto-lock-delay-unsupported":
    "Ravenpass does not offer that delay. Choose one from the list.",
  "failure.backup-choice-unsupported":
    "Ravenpass does not offer that backup frequency or count. Choose one from the list.",
  "failure.backup-folder-missing":
    "No backup folder is chosen. Choose a folder, then turn on automatic backups.",
  "failure.shortcut-invalid":
    "Ravenpass cannot use those keys as a shortcut. Choose other keys.",
  "failure.pin-invalid": "A PIN is 6 to 12 digits. Enter one and try again.",
  "failure.pin-wrong": "That PIN is wrong. Wait a moment and try again.",
  "failure.pin-too-soon":
    "Too many wrong PINs in a row. Wait a few seconds, then try again.",
  "failure.pin-removed":
    "Too many incorrect PIN attempts, so your PIN was removed. Unlock another way, or restore access with your recovery key.",
  "failure.pin-missing":
    "This vault has no PIN on this device. Unlock another way, or set a PIN in Settings.",
  "failure.biometry-unavailable":
    "This device cannot verify who you are. Turn on its screen lock in your device's settings, or use a PIN.",
  "failure.biometry-disabled":
    "Biometrics are turned off for this vault. Use your PIN, or turn them back on in Settings.",
  "failure.unlock-method-required":
    "Keep one way to unlock this vault. Set the other one up first.",
  "failure.unlock-record-unreadable":
    "Ravenpass cannot read how this vault unlocks on this device. Restore it with your recovery key.",
  "failure.owner-unverified":
    "Your identity wasn't confirmed, so nothing changed. Try again.",
  "failure.import-source-unknown":
    "Ravenpass cannot import from this app. Choose one of the apps in the list.",
  "failure.import-unrecognized":
    "Ravenpass cannot read this file as a {source} export. Choose the file {source} saved.",
  "failure.import-too-large":
    "This file is too large to import. Export fewer items into a smaller file and try again.",
  "failure.import-account-bound":
    "This file opens only in {source}, with your account. Export it again with a file password, or unencrypted.",
  "failure.import-password-wrong":
    "That password doesn't open this file. Enter the password you set when exporting it.",
  "failure.import-encryption-unsupported":
    "This file is encrypted in a way Ravenpass cannot open. Export it again from {source}.",
  "failure.import-not-active": "Choose the file again to import it.",
  "failure.import-empty":
    "Nothing to import: every item is already in Ravenpass or cannot be imported.",
  "failure.import-limit":
    "Your vault cannot hold everything in this file. Turn off grouping by folder, or remove items from the export and try again.",
  "failure.import-trash-failed":
    "Ravenpass could not move the file to the Trash. Delete it yourself.",
  "failure.import-reveal-failed":
    "Ravenpass could not open the folder that holds the file. Find the file and delete it.",
  "failure.link-expired":
    "This connection key expired. Create a new one to link an extension.",
  "failure.link-canceled":
    "Linking was canceled. Create a new key to link an extension.",
  "failure.link-unavailable":
    "Ravenpass cannot open its connection for extensions. Another app may be using it. Try again later.",
  "failure.extension-not-found":
    "That extension is already unlinked. Reopen Settings to see the extensions still linked.",
  "failure.extension-name-invalid":
    "That name is empty or too long. Use up to 64 characters.",
  "failure.confirmation-ended":
    "This request has ended. Start it again from your browser or the app that asked.",
};
