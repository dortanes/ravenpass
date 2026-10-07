export const seed = {
  "seed.untitled": "Untitled seed",
  "seed.search.label": "Search seeds",
  "seed.search.placeholder": "Search names and wallets",

  "seed.list.label": "Saved seeds",
  "seed.list.empty.all": "Seeds you add appear here.",
  "seed.list.empty.recent": "Seeds you open appear here.",
  "seed.list.empty.pinned": "Seeds you add to Favorites appear here.",
  "seed.list.empty.search": "No seeds match your search.",

  "seed.loading": "Loading seeds…",
  "seed.detail.loading": "Opening seed…",
  "seed.empty.select.title": "Select a seed",
  "seed.empty.select.detail": "Choose a seed from the list to see its details.",
  "seed.empty.first.title": "Add your first seed",
  "seed.empty.first.detail":
    "Seed phrases, private keys and backup codes you add will appear in the list.",
  "seed.empty.first.action": "Add seed",

  "seed.summary.words": "{count}-word phrase",
  "seed.summary.codes":
    "{left} of {total, plural, one {# code} other {# codes}} left",
  "seed.summary.key": "Private key",
  "seed.subtitle.phrase": "{count}-word seed phrase",
  "seed.subtitle.codes": "Backup codes",

  "seed.format.label": "Format",
  "seed.format.phrase": "Seed phrase",
  "seed.format.key": "Private key",
  "seed.format.codes": "Backup codes",

  "seed.checksum.valid": "Checksum valid",
  "seed.checksum.invalid": "Checksum does not match",
  "seed.checksum.unknown": "Not a BIP-39 phrase",
  "seed.checked.on": "Checked {date}",
  "seed.checked.never": "Never checked",

  "seed.block.phrase": "Phrase",
  "seed.block.key": "Key",
  "seed.block.codes": "Codes",
  "seed.block.wallet": "Wallet",
  "seed.block.addresses": "Addresses",

  "seed.phrase.reveal": "Show every word for 30 seconds",
  "seed.phrase.conceal": "Hide the words",
  "seed.phrase.hold": "Hold to show word {number}",
  "seed.phrase.left":
    "{seconds, plural, one {# second left} other {# seconds left}}",

  "seed.field.label": "Name",
  "seed.field.label.placeholder": "Main wallet",
  "seed.field.phrase": "Phrase",
  "seed.field.passphrase": "Passphrase",
  "seed.field.path": "Derivation path",
  "seed.field.path.note":
    "The derivation path tells a wallet which accounts to open from the phrase. Leave it empty if your wallet never showed one.",
  "seed.field.key": "Key",
  "seed.field.codes": "Codes",
  "seed.field.wallet": "Wallet",
  "seed.field.wallet.placeholder": "Hardware wallet",
  "seed.field.address-label": "Label",
  "seed.field.address": "Address",
  "seed.address.untitled": "Address {number}",

  "seed.passphrase.reveal": "Show passphrase",
  "seed.passphrase.conceal": "Hide passphrase",
  "seed.key.reveal": "Show key",
  "seed.key.conceal": "Hide key",

  "seed.copy.phrase": "Copy phrase",
  "seed.copy.passphrase": "Copy passphrase",
  "seed.copy.path": "Copy path",
  "seed.copy.key": "Copy key",
  "seed.copy.address": "Copy address",
  "seed.copy.next-code": "Copy next unused code",
  "seed.copied.phrase": "Seed phrase copied.",
  "seed.copied.passphrase": "Passphrase copied.",
  "seed.copied.path": "Path copied.",
  "seed.copied.key": "Private key copied.",
  "seed.copied.address": "Address copied.",
  "seed.copied.code": "Code copied and marked as used.",

  "seed.copy-phrase.title": "Copy the whole phrase?",
  "seed.copy-phrase.detail":
    "Anyone who sees this phrase can access the funds it protects. Other apps can read the clipboard until it clears.",
  "seed.copy-phrase.confirm": "Copy phrase",

  "seed.code.use": "Copy code {number} and mark it as used",
  "seed.code.used": "Used",

  "seed.check.open": "Check my backup",
  "seed.check.detail":
    "Type the words at these positions from your written copy.",
  "seed.check.word": "Word {number}",
  "seed.check.empty": "Type this word.",
  "seed.check.mismatch":
    "These words do not match. Compare your written copy with the phrase and try again.",
  "seed.check.submit": "Check",
  "seed.check.done": "Backup check complete. Verification date saved.",

  "seed.editor.new.title": "New seed",
  "seed.editor.edit.title": "Edit seed",
  "seed.editor.create": "Add seed",
  "seed.editor.requirement.phrase": "A name and the phrase are required",
  "seed.editor.requirement.key": "A name and the key are required",
  "seed.editor.requirement.codes": "A name and at least one code are required",
  "seed.editor.unfinished": "Fix the highlighted values to save.",
  "seed.editor.phrase.placeholder":
    "Paste or type the words, separated by spaces",
  "seed.editor.phrase.unknown": "Words outside the BIP-39 list: {positions}",
  "seed.editor.phrase.too-many":
    "A phrase holds at most {limit, plural, one {# word} other {# words}}.",
  "seed.editor.phrase.too-long":
    "A word holds at most {limit, plural, one {# character} other {# characters}}.",
  "seed.editor.phrase.suggestions": "Suggested words",
  "seed.editor.codes.placeholder": "One code per line",
  "seed.editor.codes.count": "{count, plural, one {# code} other {# codes}}",
  "seed.editor.codes.too-many":
    "{limit, plural, one {Only # code fits.} other {At most # codes fit.}}",
  "seed.editor.codes.too-long":
    "A code holds at most {limit, plural, one {# character} other {# characters}}.",
  "seed.editor.codes.used": "Used codes",
  "seed.editor.codes.used.detail": "Select the codes you have already used.",
  "seed.editor.address.add": "Address",
  "seed.editor.address.remove": "Remove address",
  "seed.editor.address.missing": "Enter each address, or remove its row.",

  "seed.error.list": "Ravenpass could not load your seeds. Restart the app.",
  "seed.error.close":
    "Ravenpass could not close the selected seed. Lock the vault.",
  "seed.error.open":
    "This seed could not be opened. Select it again or reopen Ravenpass.",
  "seed.error.copy":
    "Ravenpass could not copy that value. Open the seed and try again.",
  "seed.error.pin":
    "Ravenpass could not add this seed to Favorites. Try again.",
  "seed.error.unpin":
    "Ravenpass could not remove this seed from Favorites. Try again.",
  "seed.error.save":
    "Ravenpass could not save this seed. Check what you entered and try again.",
  "seed.error.saved-partly": "Seed saved, but {detail}",
  "seed.error.delete": "Ravenpass could not delete this seed. Try again.",
  "seed.error.duplicate": "Ravenpass could not duplicate this seed. Try again.",
  "seed.error.deleted-partly": "Seed deleted, but {detail}",
  "seed.error.use-code":
    "Ravenpass could not mark that code as used. Open the seed and try again.",
  "seed.error.check":
    "Ravenpass could not save the verification date. Open the seed and try again.",
  "seed.error.wordlist":
    "Ravenpass could not load the BIP-39 word list. Type each word in full.",
};
