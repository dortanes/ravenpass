export const access = {
  "wizard.steps.label": "Setup progress",
  "wizard.steps.storage": "Vault file",
  "wizard.steps.phrase": "Recovery key",
  "wizard.steps.confirm": "Key check",
  "wizard.steps.unlock": "Unlock",
  "wizard.storage.title": "Choose where to save your vault",
  "wizard.storage.title-fixed": "Your vault stays on this device",
  "wizard.storage.description":
    "Your Ravenpass data is stored in one encrypted file.",
  "wizard.storage.type": "Storage type",
  "wizard.storage.location": "Location",
  "wizard.storage.file-empty": "Checking…",
  "wizard.storage.change": "Change…",
  "wizard.storage.change-busy": "Choosing…",
  "wizard.storage.fact-encrypted":
    "Encrypted on this device before anything is written.",
  "wizard.storage.fact-saved": "Every change is saved the moment you make it.",
  "wizard.storage.fact-backup":
    "You can export a separate encrypted copy in Settings.",
  "wizard.storage.unrestricted":
    "This location cannot limit who may read the file.",
  "wizard.phrase.title": "Save your recovery key",
  "wizard.phrase.description":
    "To restore your vault, you'll need this key and an encrypted copy.",
  "wizard.phrase.list": "24-word recovery key",
  "wizard.phrase.copy": "Copy key",
  "wizard.phrase.copy-busy": "Copying…",
  "wizard.phrase.copy-done":
    "Copied. Clear your clipboard after storing the key.",
  "wizard.phrase.save": "Save to file…",
  "wizard.phrase.save-busy": "Saving…",
  "wizard.phrase.save-done":
    "Saved. Keep this key file separate from your encrypted vault.",
  "wizard.phrase.print": "Print…",
  "wizard.phrase.print-busy": "Printing…",
  "wizard.phrase.note":
    "A saved key file is not encrypted. Anyone with the key and an encrypted copy can open your vault.",
  "wizard.phrase.note-words":
    "Anyone with this key and an encrypted copy can open your vault.",
  "wizard.phrase.acknowledgment":
    "I have stored my recovery key somewhere safe.",
  "wizard.confirm.title": "Check your recovery key",
  "wizard.confirm.description":
    "Enter three words from the key you just saved.",
  "wizard.confirm.words": "Words from your recovery key",
  "wizard.confirm.hint": "Type a space to move to the next word.",
  "phrase.word": "Word {number}",
  "wizard.unlock.title": "Choose how to open this vault",
  "wizard.unlock.description":
    "This applies to this device only. You can change it in Settings.",
  "wizard.unlock.create": "Create vault",
  "wizard.unlock.creating": "Creating your vault",
  "wizard.actions.back": "Back",
  "wizard.actions.cancel": "Cancel",
  "wizard.actions.continue": "Continue",
  "wizard.actions.preparing": "Preparing…",
  "wizard.errors.storage-unreadable":
    "Ravenpass could not read where your vault will be kept. Restart the app.",
  "wizard.errors.location-rejected":
    "Ravenpass could not use that location. Choose another folder and try again.",
  "wizard.errors.begin-failed": "Ravenpass could not start setup. Try again.",
  "wizard.errors.cancel-failed":
    "Ravenpass could not close setup. Restart the app.",
  "wizard.errors.leave-failed":
    "Ravenpass could not return to your other vault. Try again.",
  "wizard.errors.copy-failed":
    "Ravenpass could not copy the key. Try again or save it to a file.",
  "wizard.errors.copy-failed-no-file":
    "Ravenpass could not copy the key. Try again, or write the words down.",
  "wizard.errors.save-failed":
    "Ravenpass could not save the key. Choose another location and try again.",
  "wizard.errors.print-failed":
    "Ravenpass could not print the key. Try again, or write the words down.",
  "wizard.errors.words-mismatch":
    "Those words do not match. Check your recovery key and try again.",
  "wizard.errors.finish-failed": "Ravenpass could not finish setup. Try again.",

  "unlock-methods.biometry.title": "Biometrics",
  "unlock-methods.biometry.description":
    "Your device's biometrics or its password.",
  "unlock-methods.biometry.unavailable":
    "Biometrics are unavailable on this device.",
  "unlock-methods.biometry.creating":
    "Creating a key in your device's secure hardware. This can take up to 20 seconds.",
  "unlock-methods.biometry.last":
    "This is the only way to unlock on this device. Set a PIN to turn off biometrics.",
  "unlock-methods.pin.last":
    "This is the only way to unlock on this device. Turn on biometrics to turn off the PIN.",
  "unlock-methods.pin.only":
    "The PIN is the only way to unlock that works on this device.",
  "unlock-methods.pin.title": "PIN",
  "unlock-methods.pin.description":
    "{min} to {max, plural, one {# letter or digit} other {# letters or digits}}.",
  "unlock-methods.pin.weaker": "Avoid a PIN that is easy to guess.",
  "unlock-methods.pin.set": "Set a PIN",
  "unlock-methods.pin.change": "Change PIN",
  "unlock-methods.pin.current": "Current PIN",
  "unlock-methods.pin.new": "New PIN",
  "unlock-methods.pin.repeat": "Repeat PIN",
  "unlock-methods.pin.save": "Save PIN",
  "unlock-methods.pin.saving": "Saving…",
  "unlock-methods.pin.cancel": "Cancel",
  "unlock-methods.pin.mismatch": "The PINs don't match.",
  "unlock-methods.pin.saved": "PIN saved.",
  "unlock-methods.pin.removed": "PIN removed.",
  "unlock-methods.biometry.on": "Biometrics turned on.",
  "unlock-methods.biometry.off": "Biometrics turned off.",
  "unlock-methods.confirm.title": "Enter your PIN",
  "unlock-methods.confirm.description":
    "Confirm it's you to change how this vault unlocks.",
  "unlock-methods.confirm.action": "Continue",
  "unlock-methods.confirm-key.title": "Enter your recovery key",
  "unlock-methods.confirm-key.description":
    "This device cannot confirm who you are with a PIN or biometrics. Enter the vault's recovery key to change how it unlocks.",
  "unlock-methods.errors.save-failed":
    "Ravenpass could not save that change. Try again.",
  "unlock-methods.errors.unreadable":
    "Ravenpass could not read how this vault can open. Restart the app.",
  "unlock-methods.choose": "Choose at least one way to continue.",

  "unlock.title": "Unlock your vault",
  "unlock.action": "Unlock vault",
  "unlock.action-busy": "Unlocking…",
  "unlock.biometry-action": "Use biometrics",
  "unlock.back": "Back to start",
  "unlock.restore.title": "Restore access to this vault",
  "unlock.restore.reason": "This device has no way to unlock this vault.",
  "unlock.restore.next":
    "Enter your recovery key and set up a new way to unlock.",
  "unlock.missing.title": "Vault file not found",
  "unlock.missing.reason":
    "The vault file is no longer at {location}. It may have been moved on another device.",
  "unlock.missing.next": "Open it from its new place to unlock it here.",
  "unlock.missing.action": "Open vault file…",
  "unlock.pin.label": "PIN",
  "unlock.pin.placeholder": "Your PIN",
  "unlock.pin.action": "Unlock",
  "unlock.pin.busy": "Checking…",
  "unlock.pin.attempts":
    "{count, plural, one {# attempt} other {# attempts}} left before the PIN is removed.",
  "unlock.none.title": "No way to open this vault on this device",
  "unlock.none.description":
    "Use your recovery key to set up a new way to unlock.",
  "unlock.recovery.title": "Restore access",
  "unlock.recovery.description":
    "Enter your 24-word recovery key to open this vault.",
  "unlock.recovery.action": "Use recovery key",
  "unlock.errors.failed":
    "Ravenpass could not unlock this vault. Try again, or use your recovery key.",
  "unlock.errors.switch-failed":
    "Ravenpass could not open that vault. Choose another one and try again.",
  "unlock.errors.back-failed":
    "Ravenpass could not close this vault. Try again.",
  "unlock.errors.adopt-failed":
    "Ravenpass could not open that version. Unlock the vault and try again.",
  "unlock.diverged.title": "Changed on another device",
  "unlock.diverged.description":
    "This vault was changed on another device at the same time. This device's latest changes were replaced by that version.",
  "unlock.diverged.confirm": "Open this version",
  "unlock.diverged.cancel": "Not now",

  "recovery.phrase.label": "Recovery key",
  "recovery.phrase.show": "Show recovery key",
  "recovery.phrase.hide": "Hide recovery key",
  "recovery.preview.older.title": "This copy may be older",
  "recovery.preview.older.description":
    "Restoring it could remove items added after this copy was saved.",
  "recovery.preview.accept-loss": "I understand that newer items may be lost.",
  "recovery.preview.replaced-key.title":
    "This copy uses a replaced recovery key",
  "recovery.preview.replaced-key.description":
    "This vault's recovery key has since been replaced. Anyone who has the old key could have written this copy. Open it only if you restored it yourself.",
  "recovery.preview.accept-replaced-key": "I restored this copy myself.",
  "recovery.actions.back": "Back",
  "recovery.actions.continue": "Continue",
  "recovery.actions.checking": "Checking…",
  "recovery.unlock.title": "Choose how to open this vault",
  "recovery.unlock.description":
    "Choose how to unlock on this device. You can change it later in Settings.",
  "recovery.actions.restore": "Restore access",
  "recovery.restoring": "Restoring access to your vault",
  "recovery.errors.phrase-missing": "Enter your 24-word recovery key first.",
  "recovery.errors.local-failed":
    "Ravenpass could not open the vault. Try again.",
  "recovery.errors.confirm-failed":
    "Ravenpass could not finish recovery. Restart the app before trying again.",
  "recovery.errors.cancel-failed":
    "Ravenpass could not close recovery. Restart the app.",

  "storage-unavailable.title": "Your vault's storage is unavailable",
  "storage-unavailable.description-folder":
    "Ravenpass cannot reach {place}. Reconnect it, or select your vault file.",
  "storage-unavailable.description":
    "Ravenpass cannot reach the folder that holds your vault.",
  "storage-unavailable.note":
    "Ravenpass has no second copy on this device. Reconnect the storage that holds the vault file, or select the file where it is now.",
  "storage-unavailable.actions.locate": "Select vault file…",
  "storage-unavailable.actions.retry": "Try again",
  "storage-unavailable.actions.retry-busy": "Checking…",
  "storage-unavailable.errors.unreadable":
    "Ravenpass could not read where your vault is kept. Restart the app.",
  "storage-unavailable.errors.unreachable":
    "Ravenpass still cannot reach that location. Reconnect it, or select your vault file.",
  "storage-unavailable.errors.switch-failed":
    "Ravenpass could not open that vault. Choose another one and try again.",
  "storage-unavailable.errors.create-failed":
    "Ravenpass could not start a new vault. Try again.",
  "storage-unavailable.errors.open-failed":
    "That file could not be opened. Select a Ravenpass vault file.",
};
