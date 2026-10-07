# Ravenpass

<img src="apps/desktop/build/appicon.png" alt="Ravenpass logo" width="128" align="left">

<p><strong>The password manager that stays yours.</strong></p>
<p>Passwords, passkeys and two-factor codes in one encrypted vault on your computer and phone, filled in wherever you sign in. No account needed, and you decide where your vault lives.</p>
<p>
  <a href="https://github.com/dortanes/ravenpass/releases/latest"><img src="https://img.shields.io/github/v/release/dortanes/ravenpass?logo=github&amp;color=blue" alt="Latest release"></a>
  <a href="https://chromewebstore.google.com/detail/cceiadaelnccfbakmhcleifjfilkakag"><img src="https://img.shields.io/chrome-web-store/v/cceiadaelnccfbakmhcleifjfilkakag?logo=googlechrome&amp;logoColor=white&amp;label=Chrome%20Web%20Store&amp;color=blue" alt="Chrome Web Store"></a>
  <a href="https://github.com/dortanes/ravenpass/actions/workflows/ci.yml"><img src="https://github.com/dortanes/ravenpass/actions/workflows/ci.yml/badge.svg" alt="CI status"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-GPL--3.0--or--later-blue.svg" alt="GPL-3.0-or-later license"></a>
</p>
<p>
  <a href="https://ravenpass.org">Website</a> ·
  <a href="https://www.youtube.com/watch?v=7Vlkf9ZTwo4">Trailer</a> ·
  <a href="#features">Features</a> ·
  <a href="#download">Download</a> ·
  <a href="#security">Security</a> ·
  <a href="#build-from-source">Build from source</a> ·
  <a href="CONTRIBUTING.md">Contributing</a> ·
  <a href="#support-ravenpass">Support</a>
</p>

<br clear="left">

<p align="center"><img src=".github/assets/screenshot.png" alt="Ravenpass on a Mac and an Android phone" width="100%"></p>

> [!NOTE]
> Ravenpass is in beta. Keep a backup of anything you store in it.

## Features

- **Native autofill, everywhere.** Built into your system's own autofill, so passwords, passkeys and two-factor codes fill in any app or browser. Chrome also has its own extension, which fills payment cards at checkout after your fingerprint or PIN and offers to save a card you type.
- **Your vault, your choice.** Keep it on your device, or in a cloud drive folder to open the same vault on your computer and phone.
- **Everything in one place.** Passwords, passkeys, two-factor codes, payment cards, identities with document scans, secure notes and crypto wallet seeds.
- **Strong passwords on demand.** Generate a password or a passphrase while you edit an item or on its own, with an optional history of what you generated.
- **Breach alerts.** Turn on breach checks to see which of your passwords appear in known data breaches. Only the first five characters of each password's hash leave your device.
- **Quick to unlock.** Your fingerprint or your device's screen lock, or a PIN.
- **Hard to lose.** Deleted items wait in the trash for 30 days, or a period you choose. Automatic encrypted backups go to a folder you choose, and a 24-word recovery key can be printed or saved.
- **Easy to switch.** Bring your passwords over from other password managers in a few clicks, then merge duplicates into one item.
- **No tracking.** No analytics or telemetry. Ravenpass goes online only to fetch icons from the sites you save, on Android to check which apps a site trusts, and, if you turn on breach checks, to look up your passwords' hash prefixes in [Have I Been Pwned](https://haveibeenpwned.com/Passwords).

## Download

Get the latest version from [Releases](https://github.com/dortanes/ravenpass/releases/latest).

| Platform | File | Requirements |
| --- | --- | --- |
| macOS | `.dmg` | Apple silicon, macOS 13 or later; AutoFill needs macOS 15 |
| Android | `.apk` | Android 12 or later; passkeys need Android 14 |
| Chrome | [Chrome Web Store](https://chromewebstore.google.com/detail/cceiadaelnccfbakmhcleifjfilkakag) | Chrome 133 or later and Ravenpass on your computer |
| Windows, Linux, iOS | | Coming soon |

- **macOS:** open the disk image and drag Ravenpass to Applications. On macOS 15 and later, turn on Ravenpass in **System Settings → General → AutoFill & Passwords**.
- **Android:** open the APK and allow the install, then choose Ravenpass as your autofill service and, on Android 14 and later, as your passkey provider.
- **Chrome:** add Ravenpass from the [Chrome Web Store](https://chromewebstore.google.com/detail/cceiadaelnccfbakmhcleifjfilkakag). The extension then shows how to link it to Ravenpass on your computer.

On macOS, you can also install with Homebrew:

```sh
brew tap dortanes/ravenpass https://github.com/dortanes/ravenpass
brew install --cask dortanes/ravenpass/ravenpass
```

To update, run `brew update` and `brew upgrade --cask dortanes/ravenpass/ravenpass`.

On Android, you can also install with [Obtainium](https://obtainium.imranr.dev/), which keeps Ravenpass up to date from this repository's releases:

<a href="https://apps.obtainium.imranr.dev/redirect?r=obtainium://add/https://github.com/dortanes/ravenpass"><img src=".github/assets/badge_obtainium.png" alt="Get it on Obtainium" width="161"></a>

> [!IMPORTANT]
> Nobody can reset your vault for you. Keep backups on and keep a copy of your recovery key: it unlocks a vault file you still have, but it cannot bring back a file that is gone.

## Security

Every item is encrypted with XChaCha20-Poly1305 before it is written anywhere, and unlocking is bound to your device's secure hardware, such as the Secure Enclave or Android Keystore. The Chrome extension stores no passwords: it asks Ravenpass on your computer over an end-to-end encrypted local connection.

Found a vulnerability? Report it privately as described in [SECURITY.md](SECURITY.md).

## Build from source

You need Go 1.27 and Node.js 24. The macOS app also needs Xcode or its Command Line Tools, version 16 or later, because the AutoFill extension builds in Swift 6 mode. The Android app needs JDK 17 or later, the Android SDK and NDK 29.0.14206865.

```sh
git clone https://github.com/dortanes/ravenpass.git
cd ravenpass
make deps
make desktop     # macOS app in apps/desktop/bin
make android     # debug APK in apps/mobile/build/android/app/build/outputs/apk/debug
make extension   # Chrome extension in apps/extension/dist
```

`make help` lists every target. [CONTRIBUTING.md](CONTRIBUTING.md) covers the checks, signed builds and releases.

## Community

Ask questions and share ideas in [Discussions](https://github.com/dortanes/ravenpass/discussions). Report bugs in [Issues](https://github.com/dortanes/ravenpass/issues).

## Support Ravenpass

Ravenpass is free and built in spare time. If it is useful to you, a donation helps keep it going.

<a href="https://ko-fi.com/dortanes"><img src="https://img.shields.io/badge/Ko--fi-Support%20Ravenpass-FF5E5B?logo=kofi&amp;logoColor=white" alt="Support Ravenpass on Ko-fi"></a>

<details>
<summary>Donate USDT</summary>

| Network | Address |
| --- | --- |
| TON | `UQDdOJ-e67sTvdkw3QbstGXiWa3KmB7_QUs98JZ-s53IPsyw` |
| TRON (TRC-20) | `TNdEYdDKnfFt84wervgywr1iNg3ununuYw` |
| Ethereum (ERC-20) | `0x464e564580C76D252CC8A9aF1994dA4eCacEEaC7` |
| Solana | `GfcK6LRhTWGYHQYSGoqTaYRssk4xGV26XywpCwkA9BPJ` |

Send only USDT, and only on the network listed next to the address.

</details>

## License

Ravenpass is free software under the [GNU General Public License v3.0 or later](LICENSE).

---

<p align="center">
  If Ravenpass keeps your passwords safe, please <a href="https://github.com/dortanes/ravenpass">⭐ star the repository</a>.<br>
  It helps others find the project and keeps development going.
</p>
