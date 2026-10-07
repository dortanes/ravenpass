# Ravenpass for macOS (Apple silicon, macOS 13 or later). Run `make help` for the targets.

.DEFAULT_GOAL := help
.PHONY: help deps dev dev-prepare dev-build frontend bindings bindings-check runtime-version \
	secure-enclave app autofill check test-secure-enclave notices licenses-check package \
	package-development release release-sign release-staple signing-profiles clean

ROOT := $(abspath ../..)
VERSION := $(shell tr -d '[:space:]' < $(ROOT)/version.txt)

include $(ROOT)/notices/notices.mk

export RAVENPASS_HOST := desktop
export RAVENPASS_OUT_DIR := $(CURDIR)/frontend/dist
export RAVENPASS_AUTOFILL_OUT_DIR := $(CURDIR)/frontend/autofill

UI := npm --prefix $(ROOT)/packages/ui
WAILS := go run github.com/wailsapp/wails/v3/cmd/wails3

GO_DARWIN := GOOS=darwin GOARCH=arm64 CGO_ENABLED=1 MACOSX_DEPLOYMENT_TARGET=13.0 \
	CGO_CFLAGS=-mmacosx-version-min=13.0 CGO_LDFLAGS=-mmacosx-version-min=13.0
SWIFT_TARGET := arm64-apple-macosx13.0
# A build no provisioning profile signs carries the adhoc tag, which keeps unlock keys in the login keychain; signed
# targets refuse such a binary.
GO_TAGS := production

# The AutoFill extension needs macOS 15 and a provisioning profile; only signed packages carry it.
AUTOFILL_NAME := RavenpassAutoFill
AUTOFILL_TARGET := arm64-apple-macosx15.0
AUTOFILL_SOURCES := $(wildcard autofill/*.swift)
AUTOFILL_BINARY := bin/$(AUTOFILL_NAME)
AUTOFILL := bin/$(AUTOFILL_NAME).appex

SECURE_ENCLAVE_SOURCE := internal/secureenclave/secureenclave.swift
SECURE_ENCLAVE_OBJECT := internal/secureenclave/secureenclave_darwin_arm64.syso

# Signing material stays outside the repository: identities in a keychain, profiles in PROFILES.
# An empty TEAM_ID from the environment or command line would sign entitlements that no profile matches.
ifeq ($(strip $(TEAM_ID)),)
override TEAM_ID := VRR562LFAB
endif
PROFILES ?=
DEVELOPMENT_IDENTITY ?= Apple Development
RELEASE_IDENTITY ?= Developer ID Application
NOTARY_ARGS ?= --keychain-profile ravenpass-notary
# The sandboxed AutoFill extension can read the app's files to check its signature only under /Applications.
DEVELOPMENT_DIR ?= /Applications
RELEASE_DIR := bin/release
RELEASE_DMG := $(RELEASE_DIR)/Ravenpass-$(VERSION)-macos-arm64.dmg
NOTICES_FILE := bin/THIRD_PARTY_NOTICES.txt

help:
	@echo "Ravenpass for macOS $(VERSION)"
	@echo
	@echo "  make deps                 install npm dependencies for the interface"
	@echo "  make dev                  run the app with live reload"
	@echo "  make check                build, vet and test (no Secure Enclave needed)"
	@echo "  make test-secure-enclave  create and use keys in this Mac's Secure Enclave (needs PROFILES)"
	@echo "  make package              ad-hoc signed bin/Ravenpass.app, without AutoFill, unlock keys in the login keychain"
	@echo "  make package-development  development-signed app installed into $(DEVELOPMENT_DIR)"
	@echo "  make release              Developer ID signed, notarized $(RELEASE_DMG)"
	@echo "  make release-sign         sign and notarize the already built app into the disk image"
	@echo "  make release-staple       staple a notarization Apple accepted after release stopped waiting"
	@echo "  make app                  unsigned bin/Ravenpass binary with the built interface"
	@echo "  make autofill             unsigned AutoFill extension in $(AUTOFILL)"
	@echo "  make frontend             build the interface into frontend"
	@echo "  make secure-enclave       compile the Secure Enclave bridge"
	@echo "  make notices              third-party notices in $(NOTICES_FILE)"
	@echo "  make licenses-check       fail on a Go dependency outside the license allowlist"
	@echo "  make runtime-version      fail when @wailsio/runtime differs from the Wails version in go.mod"
	@echo "  make bindings             regenerate the interface's Go bindings"
	@echo "  make bindings-check       fail when the committed bindings are stale"
	@echo "  make clean                remove build output"
	@echo
	@echo "Signed targets read the provisioning profiles from PROFILES and sign for TEAM_ID ($(TEAM_ID))."

deps:
	@$(MAKE) -s --no-print-directory -C $(ROOT) node_modules

$(SECURE_ENCLAVE_OBJECT): $(SECURE_ENCLAVE_SOURCE)
	swiftc -parse-as-library -emit-object -O -module-name RavenpassSecureEnclave -target $(SWIFT_TARGET) $< -o $@

secure-enclave: $(SECURE_ENCLAVE_OBJECT)

# The bindings import @wailsio/runtime, which the interface bundles.
bindings:
	$(WAILS) generate bindings -ts -i -d $(ROOT)/packages/ui/src/bindings

bindings-check: bindings
	@test -z "$$(git -C $(ROOT) status --porcelain --untracked-files=all -- packages/ui/src/bindings)" || { \
		git -C $(ROOT) status --short --untracked-files=all -- packages/ui/src/bindings; \
		git -C $(ROOT) diff -- packages/ui/src/bindings; \
		echo "The committed bindings are stale; run make bindings and commit the result"; exit 1; }

runtime-version:
	@test "$$(node -p 'require("$(ROOT)/packages/ui/package.json").dependencies["@wailsio/runtime"]')" = "$$(go list -m -f '{{slice .Version 1}}' github.com/wailsapp/wails/v3)" \
		|| { echo "packages/ui must depend on @wailsio/runtime at the Wails version in go.mod"; exit 1; }

frontend: deps bindings
	$(UI) run build

app: frontend secure-enclave
	mkdir -p bin
	$(GO_DARWIN) go build -tags $(GO_TAGS) -trimpath -ldflags="-s -w" -o bin/Ravenpass .

$(AUTOFILL_BINARY): $(AUTOFILL_SOURCES)
	mkdir -p bin
	swiftc -application-extension -parse-as-library -swift-version 6 -O -target $(AUTOFILL_TARGET) \
		-module-name $(AUTOFILL_NAME) -Xlinker -e -Xlinker _NSExtensionMain \
		$(AUTOFILL_SOURCES) -o $@

# Apple requires an app extension's version to match its app's.
$(AUTOFILL): $(AUTOFILL_BINARY) autofill/Info.plist $(ROOT)/version.txt frontend
	rm -rf $@
	mkdir -p $@/Contents/MacOS $@/Contents/Resources
	cp $(AUTOFILL_BINARY) $@/Contents/MacOS/$(AUTOFILL_NAME)
	cp autofill/Info.plist $@/Contents/Info.plist
	plutil -insert CFBundleShortVersionString -string "$(VERSION)" $@/Contents/Info.plist
	plutil -insert CFBundleVersion -string "$(VERSION)" $@/Contents/Info.plist
	ditto $(RAVENPASS_AUTOFILL_OUT_DIR) $@/Contents/Resources/autofill

autofill: $(AUTOFILL)

check: runtime-version frontend secure-enclave autofill
	$(GO_DARWIN) go vet ./... $(ROOT)/packages/app/...
	$(GO_DARWIN) go test -race ./... $(ROOT)/packages/app/...

# Needs Secure Enclave hardware; hosted CI runners do not have it. Each key's keychain item needs the application
# identifier, which macOS grants only to a bundle signed with its provisioning profile, so the tests run as one.
SECURE_ENCLAVE_TESTS := bin/SecureEnclaveTests.app

test-secure-enclave: signing-profiles secure-enclave
	rm -rf $(SECURE_ENCLAVE_TESTS)
	mkdir -p $(SECURE_ENCLAVE_TESTS)/Contents/MacOS
	$(GO_DARWIN) go test -c -tags secureenclave -o $(SECURE_ENCLAVE_TESTS)/Contents/MacOS/SecureEnclaveTests ./internal/secureenclave
	cp build/darwin/Info.plist $(SECURE_ENCLAVE_TESTS)/Contents/Info.plist
	plutil -replace CFBundleExecutable -string SecureEnclaveTests $(SECURE_ENCLAVE_TESTS)/Contents/Info.plist
	cp "$(PROFILES)/Ravenpass_Development.provisionprofile" $(SECURE_ENCLAVE_TESTS)/Contents/embedded.provisionprofile
	$(call entitlements,Ravenpass,com.dortanes.ravenpass)
	codesign --force --options runtime --timestamp=none --entitlements bin/Ravenpass.entitlements \
		--sign "$(DEVELOPMENT_IDENTITY)" $(SECURE_ENCLAVE_TESTS)
	$(SECURE_ENCLAVE_TESTS)/Contents/MacOS/SecureEnclaveTests -test.count=1

dev: deps
	$(WAILS) dev -config build/config.yml

dev-prepare: deps bindings
	$(UI) run build

dev-build: bindings secure-enclave
	mkdir -p bin
	$(GO_DARWIN) go build -tags adhoc -o bin/Ravenpass-dev .

notices: deps
	mkdir -p bin
	$(call notices-start,$(NOTICES_FILE).tmp,Ravenpass for macOS)
	$(call go-notices,$(NOTICES_FILE).tmp,$(GO_DARWIN))
	$(call npm-notices,$(NOTICES_FILE).tmp,$(ROOT)/packages/ui/package.json)
	mv $(NOTICES_FILE).tmp $(NOTICES_FILE)

licenses-check:
	$(call go-licenses-check,$(GO_DARWIN))

signing-profiles:
	@test -n "$(PROFILES)" || { echo "Set PROFILES to the folder that holds the provisioning profiles"; exit 1; }

# $(call bundle,directory) assembles Ravenpass.app from the built binary in directory.
define bundle
rm -rf "$(1)/Ravenpass.app"
mkdir -p "$(1)/Ravenpass.app/Contents/MacOS" "$(1)/Ravenpass.app/Contents/Resources"
cp bin/Ravenpass "$(1)/Ravenpass.app/Contents/MacOS/Ravenpass"
cp build/darwin/Info.plist "$(1)/Ravenpass.app/Contents/Info.plist"
plutil -insert CFBundleShortVersionString -string "$(VERSION)" "$(1)/Ravenpass.app/Contents/Info.plist"
plutil -insert CFBundleVersion -string "$(VERSION)" "$(1)/Ravenpass.app/Contents/Info.plist"
cp build/darwin/icons.icns "$(1)/Ravenpass.app/Contents/Resources/icons.icns"
cp $(NOTICES_FILE) "$(1)/Ravenpass.app/Contents/Resources/THIRD_PARTY_NOTICES.txt"
endef

# $(call entitlements,source,bundle id) writes source's entitlements for TEAM_ID into bin/.
define entitlements
mkdir -p bin
cp build/darwin/$(1).entitlements bin/$(1).entitlements
/usr/libexec/PlistBuddy -c "Add :com.apple.application-identifier string $(TEAM_ID).$(2)" bin/$(1).entitlements
/usr/libexec/PlistBuddy -c "Add :com.apple.developer.team-identifier string $(TEAM_ID)" bin/$(1).entitlements
endef

# $(call profile-match,bundle) fails when bundle is signed for an app ID its provisioning profile does not grant.
define profile-match
codesign -d --entitlements - --xml "$(1)" > bin/signed.plist 2>/dev/null
security cms -D -i "$(1)/Contents/embedded.provisionprofile" > bin/profile.plist
test "$$(/usr/libexec/PlistBuddy -c 'Print :com.apple.application-identifier' bin/signed.plist)" = \
	"$$(/usr/libexec/PlistBuddy -c 'Print :Entitlements:com.apple.application-identifier' bin/profile.plist)" \
	|| { echo "$(1) is signed for an app ID its provisioning profile does not grant"; exit 1; }
endef

# $(call signed-app,directory,identity,app profile,extension profile,codesign options) embeds each
# bundle's provisioning profile and signs inside out with the hardened runtime.
define signed-app
! go version -m bin/Ravenpass | grep -Eq -- '-tags=([^[:space:]]*,)?adhoc(,|[[:space:]]|$$)' \
	|| { echo "bin/Ravenpass was built for ad-hoc signing; build it again without the adhoc tag"; exit 1; }
$(call bundle,$(1))
mkdir -p "$(1)/Ravenpass.app/Contents/PlugIns"
cp "$(PROFILES)/$(3)" "$(1)/Ravenpass.app/Contents/embedded.provisionprofile"
ditto $(AUTOFILL) "$(1)/Ravenpass.app/Contents/PlugIns/$(AUTOFILL_NAME).appex"
cp "$(PROFILES)/$(4)" "$(1)/Ravenpass.app/Contents/PlugIns/$(AUTOFILL_NAME).appex/Contents/embedded.provisionprofile"
$(call entitlements,$(AUTOFILL_NAME),com.dortanes.ravenpass.autofill)
$(call entitlements,Ravenpass,com.dortanes.ravenpass)
codesign --force --options runtime $(5) --entitlements bin/$(AUTOFILL_NAME).entitlements \
	--sign "$(2)" "$(1)/Ravenpass.app/Contents/PlugIns/$(AUTOFILL_NAME).appex"
codesign --force --options runtime $(5) --entitlements bin/Ravenpass.entitlements \
	--sign "$(2)" "$(1)/Ravenpass.app"
codesign --verify --deep --strict "$(1)/Ravenpass.app"
$(call profile-match,$(1)/Ravenpass.app/Contents/PlugIns/$(AUTOFILL_NAME).appex)
$(call profile-match,$(1)/Ravenpass.app)
endef

package: GO_TAGS := production,adhoc
package: app notices
	$(call bundle,bin)
	codesign --force --options runtime --sign - bin/Ravenpass.app

package-development: signing-profiles app $(AUTOFILL) notices
	$(call signed-app,$(DEVELOPMENT_DIR),$(DEVELOPMENT_IDENTITY),Ravenpass_Development.provisionprofile,Ravenpass_Autofill_Development.provisionprofile,--timestamp=none)

release: signing-profiles app $(AUTOFILL) notices
	$(MAKE) release-sign

# Signs what app, autofill and notices built, without running any build step.
release-sign: signing-profiles
	rm -rf $(RELEASE_DIR)
	mkdir -p $(RELEASE_DIR)/image
	cp $(NOTICES_FILE) $(RELEASE_DIR)/Ravenpass-$(VERSION)-macos-arm64-notices.txt
	$(call signed-app,$(RELEASE_DIR)/image,$(RELEASE_IDENTITY),Ravenpass_Developer_ID.provisionprofile,Ravenpass_Autofill_Developer_ID.provisionprofile,--timestamp)
	ln -s /Applications $(RELEASE_DIR)/image/Applications
	hdiutil create -volname Ravenpass -srcfolder $(RELEASE_DIR)/image -fs HFS+ -format UDZO -ov $(RELEASE_DMG)
	codesign --force --timestamp --sign "$(RELEASE_IDENTITY)" $(RELEASE_DMG)
	xcrun notarytool submit $(RELEASE_DMG) $(NOTARY_ARGS) --wait
	$(MAKE) release-staple

# Staples the ticket once Apple accepts the submission, including one still in progress when release stopped waiting.
release-staple:
	xcrun stapler staple $(RELEASE_DMG)
	spctl --assess --type open --context context:primary-signature --verbose=2 $(RELEASE_DMG)

clean:
	rm -rf bin frontend $(SECURE_ENCLAVE_OBJECT)
