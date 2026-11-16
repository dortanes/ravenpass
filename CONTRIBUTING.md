# Contributing

Thanks for helping. Open an issue before large changes so the approach can be agreed first. Report security problems privately as described in [SECURITY.md](SECURITY.md).

## Setup

Install the tools listed under **Build from source** in the [README](README.md), then:

```sh
make deps
make check
```

`make check` runs every gate: `gofmt`, `go mod tidy -diff`, `go vet` and `go test -race` for each Go module, the dependency license allowlist, Biome, TypeScript and the Node test runner for the interface and the extension, the macOS build with its Swift AutoFill extension, and the Android build with Android lint. `make check-core`, `make check-ui`, `make desktop-check`, `make android-check` and `make extension-check` run one area; the macOS and Android checks need their SDKs.

`make -C apps/desktop test-secure-enclave` exercises the Secure Enclave and runs only on an Apple silicon Mac.

## Code

- Go is formatted with `gofmt`; TypeScript with Biome (`npm --prefix packages/ui run format` and `npm --prefix apps/extension run format`).
- Shared behaviour lives once: vault logic in `packages/vault`, application services in `packages/app`, the interface in `packages/ui`. Apps add only platform adapters.
- Use a maintained library for standard formats and algorithms rather than writing your own.
- Vault data is encrypted and authenticated before it reaches disk or the network. Never log or persist credentials, keys, one-time-code secrets or recovery keys.
- New dependencies must use a permissive license (MIT, BSD-2-Clause, BSD-3-Clause, 0BSD, ISC, Apache-2.0), including their transitive dependencies.
- `make licenses-check` fails when a shipped Go or npm dependency has a license outside the allowlist in `notices/allowed-licenses.json`, or an Android library one outside `notices/allowed-licenses-android.json`.
- Interface text comes from the English catalog in `packages/ui/src/i18n/en`; each translation lives beside it.

### Generated files

These are committed and must not be edited by hand; CI fails when they are stale:

| Files | Regenerate with |
| --- | --- |
| `packages/ui/src/bindings/**` | `make -C apps/desktop bindings` |
| `apps/mobile/build/android/app/src/main/res/values*/strings.xml` | `make -C apps/mobile strings` |
| `packages/app/messages/catalog.generated.json` | `make messages` |

`make generate` regenerates all of them.

## Commits and pull requests

Commit messages and pull request titles follow [Conventional Commits](https://www.conventionalcommits.org/): `feat:`, `fix:`, `perf:`, `security:`, `revert:`, `deps:`, `docs:`, `refactor:`, `test:`, `build:`, `ci:`, `chore:`. Pull requests are squash-merged, so the title becomes the commit on `main` and the changelog entry. Mark a breaking change with `!` (`feat!: …`) and describe it in the body.

A pull request describes the change, its risks and how it was verified, and includes screenshots for interface changes, with no real secrets in them.

By contributing, you license your contribution under GPL-3.0-or-later with the app store permission in [`LICENSE-EXCEPTION`](LICENSE-EXCEPTION).

## Releases

Releases are automated with [release-please](https://github.com/googleapis/release-please). Every push to `main` updates a release pull request that bumps `version.txt` and the extension manifest and writes `CHANGELOG.md` from the commits since the last release. Merging it tags a draft release, builds and signs the apps, attaches them and publishes the release:

- the notarized macOS disk image,
- the signed Android APK,
- the Chrome extension zip, which is also submitted to the Chrome Web Store once that is configured,
- a third-party notices file for each app, `SHA256SUMS`, and build provenance attestations.

File names carry no version, so `releases/latest/download/Ravenpass-macos-arm64.dmg` and `releases/latest/download/Ravenpass-android-arm64.apk` always point at the newest build.

A failed build leaves the release as a draft, so the latest published release always has its files.

No version number is edited by hand.

### Homebrew

This repository also serves as the Homebrew tap. After publishing a release, CI updates `Casks/ravenpass.rb` on `main` with the version and SHA-256 of the signed macOS artifact. Homebrew's cask updater, style check, and online audit run before the commit is pushed. This uses `GITHUB_TOKEN`; no additional secret or repository is needed.

Release-please remains the source of the version. The cask is excluded from its `extra-files` because updating it in the release pull request would pair a new version with the previous artifact's checksum before the new DMG is published.

If the Homebrew job fails, rerun it in the release workflow after fixing the failure. Rerunning a job for a release that is no longer the latest does not change the cask. A successful rerun with an unchanged cask does not create a commit.

To check cask edits locally, run `brew style Casks/ravenpass.rb`. CI also audits the cask from the checked-out revision.

### Release configuration

Repository secrets:

| Secret | Contents |
| --- | --- |
| `MACOS_CERTIFICATE_P12`, `MACOS_CERTIFICATE_PASSWORD` | Developer ID Application certificate with its private key, base64-encoded `.p12`, and its password |
| `MACOS_PROFILE_APP`, `MACOS_PROFILE_AUTOFILL` | Developer ID provisioning profiles for the app and its AutoFill extension, base64-encoded |
| `APPLE_API_KEY_P8`, `APPLE_API_KEY_ID`, `APPLE_API_ISSUER_ID` | App Store Connect API key for notarization, the `.p8` base64-encoded |
| `ANDROID_KEYSTORE`, `ANDROID_KEYSTORE_PASSWORD`, `ANDROID_KEY_ALIAS`, `ANDROID_KEY_PASSWORD` | Android release keystore, base64-encoded, and its credentials |
| `CHROME_EXTENSION_ID`, `CHROME_PUBLISHER_ID`, `CHROME_CLIENT_ID`, `CHROME_CLIENT_SECRET`, `CHROME_REFRESH_TOKEN` | Chrome Web Store API access; without them the store submission is skipped |
| `RELEASE_PLEASE_TOKEN` | Optional fine-grained token with contents and pull-request write access, so CI runs on the release pull request |

The store submission runs in the `chrome-web-store` environment, so the `CHROME_*` secrets can live there and its protection rules, such as required reviewers, gate each submission. With none of them set the submission is skipped; with only some set the release fails.

Repository variable `APPLE_TEAM_ID` overrides the Apple team the macOS app is signed for.

The Chrome Web Store needs one manual upload to create the listing. Its public key (Developer Dashboard → Package → View public key) then replaces `key` in `apps/extension/src/manifest.json`, and the resulting extension ID replaces the one in `packages/app/linkproto`, so unpacked builds and the store build share one ID. Add the `CHROME_*` secrets only after that change is released; until then the store build cannot reach the Mac app.

### Signed builds on your Mac

- **macOS:** `make -C apps/desktop release PROFILES=<folder>` needs:
  - the Developer ID Application identity in the login keychain;
  - `Ravenpass_Developer_ID.provisionprofile` and `Ravenpass_Autofill_Developer_ID.provisionprofile` in `PROFILES`, plus `Ravenpass_Development.provisionprofile` and `Ravenpass_Autofill_Development.provisionprofile` for `package-development`;
  - notarization credentials, stored once with `xcrun notarytool store-credentials ravenpass-notary`, or passed as `NOTARY_ARGS`;
  - `TEAM_ID=<your team>` when signing for a team other than the project's.
- **Mac App Store:** `make -C apps/desktop app-store PROFILES=<folder>` signs the app for the App Sandbox, builds `bin/app-store/Ravenpass.pkg` and validates it with App Store Connect; `make -C apps/desktop app-store-upload` uploads it. They need:
  - the Apple Distribution and Mac Installer Distribution identities in the login keychain;
  - `Ravenpass_App_Store.provisionprofile` and `Ravenpass_Autofill_App_Store.provisionprofile` in `PROFILES`;
  - `APPLE_API_KEY_ID` and `APPLE_API_ISSUER_ID` in the environment, with the key's `AuthKey_<id>.p8` in `~/.appstoreconnect/private_keys`.

  `make -C apps/desktop package-development APP_ENTITLEMENTS=RavenpassAppStore DEVELOPMENT_DIR=bin/sandbox` builds a development app that runs in the App Sandbox as the store build does. It keeps its files in its own container, apart from the regular build's.
- **Android:** `make -C apps/mobile release` reads the keystore from the `RAVENPASS_ANDROID_KEYSTORE`, `RAVENPASS_ANDROID_KEYSTORE_PASSWORD`, `RAVENPASS_ANDROID_KEY_ALIAS` and `RAVENPASS_ANDROID_KEY_PASSWORD` environment variables.
