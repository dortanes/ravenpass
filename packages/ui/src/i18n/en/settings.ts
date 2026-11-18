export const settings = {
  "settings.nav.label": "Settings sections",
  "settings.nav.heading": "Settings",
  "settings.general.heading": "General",
  "settings.general.summary": "Language and appearance.",
  "settings.general.language": "Language",
  "settings.general.interface": "Interface",
  "settings.general.macos": "macOS",
  "settings.interface-size": "Interface size",
  "settings.interface-size.natural": "{size} (default)",
  "settings.appearance": "Appearance",
  "settings.appearance.system": "Match system",
  "settings.appearance.light": "Light",
  "settings.appearance.dark": "Dark",
  "settings.launch-at-login": "Open at login",
  "settings.launch-at-login.blocked":
    "Blocked in System Settings. Allow Ravenpass under Login Items.",
  "settings.launch-at-login.open": "Open Login Items",
  "settings.launch-at-login.hidden": "Start in the menu bar only",
  "settings.dock-icon": "Hide Dock icon when the window is closed",
  "settings.dock-icon.detail": "Ravenpass stays in the menu bar.",
  "settings.unlock.heading": "Unlock this vault",
  "settings.locking.heading": "Automatic locking",
  "settings.auto-lock": "Lock when device is idle",
  "settings.auto-lock.hidden": "Lock after leaving the app",
  "settings.auto-lock.delay": "After",
  "settings.delay.immediately": "Immediately",
  "settings.auto-lock.note":
    "Ravenpass also locks when this device goes to sleep.",
  "settings.auto-lock.note.hidden":
    "Ravenpass also locks when this device sleeps. After you unlock it for autofill, it stays open for at least 3 minutes after the last fill, or longer if your auto-lock delay allows.",
  "settings.auto-lock.off.title": "Turn off automatic locking?",
  "settings.auto-lock.off.detail":
    "Your vault will stay open until you lock it, close Ravenpass, or this device goes to sleep.",
  "settings.auto-lock.off.confirm": "Turn off",
  "settings.auto-lock.off.cancel": "Cancel",
  "settings.recovery-key.heading": "Recovery key",
  "settings.recovery-key.title": "Change recovery key",
  "settings.recovery-key.detail":
    "Your other devices ask for the new key once. Earlier backups open only with the current key.",
  "settings.recovery-key.change": "Change…",
  "recovery-key-change.steps.verify": "Confirm",
  "recovery-key-change.steps.phrase": "New key",
  "recovery-key-change.steps.confirm": "Key check",
  "recovery-key-change.verify.title": "Change your recovery key",
  "recovery-key-change.verify.description":
    "Ravenpass makes a new recovery key and encrypts your vault with it.",
  "recovery-key-change.verify.devices":
    "If this vault is open on other devices, you'll need to enter the new key there.",
  "recovery-key-change.verify.backups":
    "Backups made before the key change open only with the old key. Keep it.",
  "recovery-key-change.verify.this-device.both":
    "You won't need the new key on this device: your PIN and biometrics keep working.",
  "recovery-key-change.verify.this-device.pin":
    "You won't need the new key on this device: your PIN keeps working.",
  "recovery-key-change.verify.this-device.biometry":
    "You won't need the new key on this device: biometrics keep working.",
  "recovery-key-change.verify.this-device.none":
    "On this device, you'll open the vault with the new key.",
  "recovery-key-change.verify.pin": "Enter your PIN to continue.",
  "recovery-key-change.verify.current":
    "This device cannot confirm it's you with a PIN or biometrics, so enter your current recovery key to continue.",
  "recovery-key-change.verify.pin-then-biometry":
    "Enter your PIN so it keeps working after the key change. Then confirm it's you with biometrics.",
  "recovery-key-change.verify.busy": "Checking…",
  "recovery-key-change.phrase.title": "Save your new recovery key",
  "recovery-key-change.phrase.description":
    "Your vault switches to this key once you check it.",
  "recovery-key-change.confirm.title": "Check your new recovery key",
  "recovery-key-change.confirm.description":
    "Enter three words from the key you just saved.",
  "recovery-key-change.saving": "Switching your vault to the new key",
  "recovery-key-change.done": "Recovery key changed.",
  "recovery-key-change.errors.begin-failed":
    "Ravenpass could not make a new recovery key. Try again.",
  "recovery-key-change.errors.finish-failed":
    "Ravenpass could not change your recovery key. Try again.",
  "recovery-key-change.errors.cancel-failed":
    "Ravenpass could not discard the new recovery key. Your current key stays in use. Restart the app.",
  "settings.clipboard.clear": "Clear copied data",
  "settings.clipboard.clear.detail":
    "Clears the clipboard only while it still contains data copied from Ravenpass.",
  "settings.clipboard.delay": "Clear after",
  "settings.clipboard.note": "Locking the vault also clears copied data.",
  "settings.clipboard.note.hidden":
    "Locking the vault clears the clipboard, except when it locks because you switched apps.",
  "settings.screenshots": "Allow screenshots",
  "settings.screenshots.detail":
    "Your vault also shows in recent apps. Autofill screens stay hidden.",
  "settings.screenshots.error":
    "Ravenpass could not change screenshots. Try again.",
  "settings.site-icons": "Website icon",
  "settings.site-icons.detail":
    "Load icons directly from the websites saved in your vault.",
  "settings.site-icons.note":
    "Turning this off removes downloaded icons from this device.",
  "settings.bank-details": "Bank name and color",
  "settings.bank-details.detail":
    "Load them from the bank website saved in each card.",
  "settings.groups.heading": "Groups",
  "settings.groups.summary": "Organize your items into tabs.",
  "settings.groups.count": "{count, plural, one {# item} other {# items}}",
  "settings.groups.empty": "Groups you add appear here.",
  "settings.groups.new": "New group…",
  "settings.groups.rename": "Rename…",
  "settings.groups.delete": "Delete group…",
  "settings.groups.name": "Name",
  "settings.groups.name.placeholder": "Work",
  "settings.groups.create.title": "New group",
  "settings.groups.rename.title": "Rename group",
  "settings.groups.save": "Save",
  "settings.groups.cancel": "Cancel",
  "settings.groups.delete.title": "Delete {name}?",
  "settings.groups.delete.detail":
    "The items in this group stay in your vault.",
  "settings.groups.delete.confirm": "Delete group",
  "settings.groups.default.label": "Default group",
  "settings.groups.default.none": "No group",
  "settings.shortcuts.heading": "Shortcuts",
  "settings.shortcuts.summary":
    "Keyboard shortcuts available while your vault is open.",
  "settings.shortcuts.palette": "Search all items",
  "settings.shortcuts.search": "Search current list",
  "settings.shortcuts.change": "Change the keys for {action}",
  "settings.shortcuts.reset": "Restore the default keys for {action}",
  "settings.shortcuts.recording": "Press keys…",
  "settings.shortcuts.note":
    "Select a shortcut and press the new keys. Esc cancels; Delete restores the default.",
  "settings.shortcuts.bare": "Add a modifier key, so typing never sets it off.",
  "settings.shortcuts.reserved":
    "{keys} is reserved by the system or text editor. Choose another shortcut.",
  "settings.shortcuts.taken":
    "{keys} is already used for “{action}”. Choose another shortcut.",
  "settings.security.heading": "Unlock",
  "settings.security.summary": "Ways to unlock and automatic locking.",
  "settings.autofill.heading": "Autofill",
  "settings.autofill.summary":
    "Fill in passwords, passkeys and codes in apps and browsers.",
  "settings.autofill.system.autofill": "Autofill service",
  "settings.autofill.system.autofill.detail":
    "Fills in passwords and codes in apps and browsers.",
  "settings.autofill.system.passkeys": "Passkey provider",
  "settings.autofill.system.passkeys.detail":
    "Signs you in with passkeys in apps and browsers.",
  "settings.autofill.system.on": "On",
  "settings.autofill.system.off": "Off",
  "settings.autofill.system.open": "Open Android settings",
  "settings.autofill.system.chrome":
    "In Chrome, also turn on “Autofill using another service”.",
  "settings.autofill.system.error":
    "Ravenpass could not open Android settings. Try again.",
  "settings.autofill.suggestions.title": "System AutoFill",
  "settings.autofill.suggestions": "Suggest saved accounts",
  "settings.autofill.suggestions.detail":
    "Show accounts below sign-in fields without opening Ravenpass.",
  "settings.autofill.suggestions.note":
    "macOS stores website addresses and usernames outside your encrypted vault. Passwords, codes, and passkey private keys stay in Ravenpass.",
  "settings.autofill.error.suggestions":
    "Ravenpass could not change account suggestions. Try again.",
  "settings.extensions.title": "Linked browser extensions",
  "settings.extensions.linked": "Linked {date}",
  "settings.extensions.unlink": "Unlink…",
  "settings.extensions.unlink.action": "Unlink {name}, linked {date}",
  "settings.extensions.unlink.title": "Unlink {name}?",
  "settings.extensions.unlink.detail":
    "This extension, linked {date}, will stop filling passwords from Ravenpass. Other linked extensions will keep working.",
  "settings.extensions.unlink.confirm": "Unlink",
  "settings.extensions.unlink.cancel": "Cancel",
  "settings.extensions.rename": "Rename…",
  "settings.extensions.rename.action": "Rename {name}",
  "settings.extensions.rename.title": "Rename extension",
  "settings.extensions.rename.save": "Save",
  "settings.extensions.rename.cancel": "Cancel",
  "settings.extensions.name": "Name",
  "settings.extensions.link": "Link extension…",
  "settings.extensions.empty": "Browser extensions you link appear here.",
  "settings.extensions.sign-in": "Sign-in prompt",
  "settings.extensions.sign-in.card": "“Sign in as…” card",
  "settings.extensions.sign-in.field": "Menu below the field",
  "settings.extensions.confirm-fills": "Require confirmation to fill passwords",
  "settings.extensions.confirm-fills.detail":
    "Ask before an extension fills a password or one-time code.",
  "settings.extensions.confirm-fills.unavailable":
    "Set a PIN or turn on biometrics to use this.",
  "settings.extensions.unreachable":
    "Linked extensions cannot reach Ravenpass right now. Another app may be using its connection.",
  "settings.extensions.dialog.title": "Link browser extension",
  "settings.extensions.dialog.step.open":
    "Open the Ravenpass extension in your browser.",
  "settings.extensions.dialog.step.paste": "Paste this key.",
  "settings.extensions.dialog.step.finish":
    "The link finishes here on its own.",
  "settings.extensions.dialog.key": "One-time key",
  "settings.extensions.dialog.copy": "Copy",
  "settings.extensions.dialog.copy.label": "Copy key",
  "settings.extensions.dialog.copied": "Copied",
  "settings.extensions.dialog.expires": "Expires in {time}",
  "settings.extensions.dialog.expired": "This key expired.",
  "settings.extensions.dialog.renew": "Get a new key",
  "settings.extensions.dialog.cancel": "Cancel",
  "settings.extensions.linked-notice": "{name} linked.",
  "settings.extensions.error.link":
    "Ravenpass could not link the extension. Try again.",
  "settings.extensions.error.unlink":
    "Ravenpass could not unlink this extension. Try again.",
  "settings.extensions.error.copy":
    "Ravenpass could not copy the key. Select it and copy it yourself.",
  "settings.extensions.error.sign-in":
    "Ravenpass could not change how you sign in on websites. Try again.",
  "settings.extensions.error.confirm-fills":
    "Ravenpass could not change fill confirmation. Try again.",
  "settings.extensions.error.rename":
    "Ravenpass could not rename this extension. Try again.",
  "settings.extensions.error.load":
    "Ravenpass could not load linked extensions. Open this section again to retry.",
  "settings.storage.type": "Type",
  "settings.storage.file": "Vault file",
  "settings.storage.checking": "Checking…",
  "settings.storage.move": "Move…",
  "settings.storage.moving": "Moving…",
  "settings.storage.unrestricted":
    "This location cannot limit who may read the file.",
  "settings.storage.shared.title": "Move your vault?",
  "settings.storage.shared.detail":
    "Once the move succeeds, {location} is deleted. Other devices that open the vault from there lose access until you open the moved file on each of them.",
  "settings.storage.shared.confirm": "Move vault",
  "settings.storage.shared.cancel": "Cancel",
  "settings.vaults.heading": "Vault",
  "settings.vaults.summary": "Storage, backups, groups and import.",
  "settings.vaults.list": "Vaults on this device",
  "settings.vaults.current": "Open",
  "settings.vaults.actions": "Actions for {name}",
  "settings.vaults.forget": "Remove from list",
  "settings.vaults.delete": "Delete permanently…",
  "settings.vaults.note": "Removing a vault from the list keeps its file.",
  "settings.delete.title": "Delete {name}?",
  "settings.delete.description":
    "This erases {location} and this device's keys for it. Afterwards only an encrypted copy and its recovery key can bring its items back.",
  "settings.delete.cancel": "Cancel",
  "settings.delete.confirm": "Delete vault",
  "settings.backup.state.current": "Matches your latest change.",
  "settings.backup.state.stale":
    "Older than your latest change. Save a new copy.",
  "settings.backup.state.unknown": "No record of the last copy.",
  "settings.backup.export": "Save copy…",
  "settings.backup.exporting": "Saving…",
  "storage.type.local-file": "File on this device",
  "storage.type.local-file.detail": "An encrypted file in a folder you choose.",
  "storage.type.local-file.private":
    "An encrypted file only Ravenpass can read.",
  "storage.type.document": "A file you choose",
  "storage.type.document.detail":
    "Google Drive, Dropbox, or a folder on this device.",
  "storage.type.document.concurrent":
    "Changes made on two devices at the same time cannot be merged.",
  "settings.privacy.heading": "Privacy",
  "settings.privacy.summary": "Website data and copied information.",
  "settings.privacy.clipboard": "Clipboard",
  "settings.privacy.websites": "Website data",
  "settings.storage.heading": "Storage",
  "settings.backup.heading": "Backups",
  "settings.backup.summary": "Save an encrypted copy of your vault.",
  "settings.backup.copy": "Encrypted copy",
  "settings.backup.auto": "Automatic backups",
  "settings.backup.frequency": "Frequency",
  "settings.backup.frequency.daily": "Every day",
  "settings.backup.frequency.weekly": "Every week",
  "settings.backup.frequency.monthly": "Every month",
  "settings.backup.keep": "Keep",
  "settings.backup.keep.latest": "Only the latest",
  "settings.backup.keep.count": "Last {count}",
  "settings.backup.folder": "Backup folder",
  "settings.backup.folder.empty": "No folder selected",
  "settings.backup.folder.choose": "Choose…",
  "settings.backup.folder.change": "Change…",
  "settings.backup.note.failed":
    "The last backup could not be saved. Check that the backup folder is available, or choose another one.",
  "settings.backup.note.last": "Last backup: {date}",
  "settings.backup.note.open":
    "Automatic backups are saved while your vault is open.",
  "settings.backup.error":
    "Ravenpass could not change automatic backups. Try again.",
  "settings.about.heading": "About",
  "settings.about.summary": "Version, license and support.",
  "settings.about.version": "Version",
  "settings.about.build": "Build",
  "settings.about.source": "Source code",
  "settings.about.release-notes": "What's new in this version",
  "settings.about.report-problem": "Report a problem",
  "settings.about.report-problem.detail":
    "Opens GitHub with the app and system versions filled in.",
  "settings.about.report-vulnerability": "Report a vulnerability",
  "settings.about.report-vulnerability.detail":
    "Only the maintainers will see your report.",
  "settings.about.license": "License and third-party software",
  "settings.about.license.detail":
    "Ravenpass is free software under GPL-3.0-or-later.",
  "settings.about.license.missing":
    "The license text isn't available in this build.",
  "settings.about.license.online": "View license on GitHub",
  "settings.about.license.close": "Close",
  "settings.about.donate": "Support Ravenpass",
};
