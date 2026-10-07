export const breaches = {
  "settings.privacy.passwords": "Passwords",
  "settings.breach-checks": "Check for breached passwords",
  "settings.breach-checks.detail":
    "Ravenpass sends Have I Been Pwned the first 5 characters of each password's SHA-1 hash, never the password, and marks passwords found in known breaches.",
  "settings.breach-checks.error":
    "Ravenpass could not change breach checks. Try again.",
  "breach.mark": "Breached",
  "breach.list.checking": "Checking passwords for breaches…",
  "breach.list.found":
    "{count, plural, one {# of {checked} passwords was found in breaches. Change it first.} other {# of {checked} passwords were found in breaches. Change them first.}}",
  "breach.list.clean":
    "{count, plural, one {# password checked. None found in breaches.} other {# passwords checked. None found in breaches.}}",
  "breach.list.failed":
    "Ravenpass could not check passwords for breaches. It tries again when the vault changes or unlocks.",
  "breach.detail":
    "{count, plural, one {This password appeared in a known breach once. Change it on the website, then here.} other {This password appeared in known breaches # times. Change it on the website, then here.}}",
  "breach.unreachable":
    "Ravenpass could not check this password for breaches. It tries again when the vault changes or unlocks.",
  "breach.editor":
    "{count, plural, one {This password appeared in a known breach. Choose another.} other {This password appeared in known breaches # times. Choose another.}}",
};
