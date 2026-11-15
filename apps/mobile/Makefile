# Ravenpass for Android (arm64, Android 12 or later). Run `make help` for the targets.

.DEFAULT_GOAL := help
.PHONY: help deps frontend strings lib lib-release notices licenses-check apk release install run dev \
	dev-prepare logs check clean

ROOT := $(abspath ../..)
VERSION := $(shell tr -d '[:space:]' < $(ROOT)/version.txt)

include $(ROOT)/notices/notices.mk

APP_ID := com.dortanes.ravenpass
ACTIVITY := com.wails.app.MainActivity

# wails3 dev exports this port to every step it starts; the desktop dev server uses 9245.
WAILS_VITE_PORT ?= 9246

ifeq ($(shell uname -s),Darwin)
ANDROID_HOME ?= $(HOME)/Library/Android/sdk
else
ANDROID_HOME ?= $(HOME)/Android/Sdk
endif
export ANDROID_HOME
NDK_VERSION ?= 29.0.14206865
NDK := $(ANDROID_HOME)/ndk/$(NDK_VERSION)
# The NDK ships one x86_64-named toolchain per host OS, also used on Apple silicon.
NDK_HOST := $(shell uname -s | tr '[:upper:]' '[:lower:]')-x86_64
NDK_CC := $(NDK)/toolchains/llvm/prebuilt/$(NDK_HOST)/bin/aarch64-linux-android31-clang
GO_ANDROID := GOOS=android GOARCH=arm64 CGO_ENABLED=1 CC=$(NDK_CC)
JNI_LIB := build/android/app/src/main/jniLibs/arm64-v8a/libwails.so
ADB := $(ANDROID_HOME)/platform-tools/adb
GRADLE := cd build/android && ./gradlew
APK_DIR := build/android/app/build/outputs/apk
RELEASE_APK := $(APK_DIR)/release/Ravenpass-android-arm64.apk
NOTICES_FILE := build/android/app/src/main/assets/THIRD_PARTY_NOTICES.txt
NOTICES_DRAFT := build/android/app/build/THIRD_PARTY_NOTICES.txt
LICENSE_REPORT := build/android/app/build/reports/dependency-license/THIRD-PARTY-NOTICES.txt

WAILS := go run github.com/wailsapp/wails/v3/cmd/wails3

export RAVENPASS_HOST := mobile
export RAVENPASS_OUT_DIR := $(CURDIR)/frontend/dist
export RAVENPASS_AUTOFILL_OUT_DIR := $(CURDIR)/build/android/app/src/main/assets/autofill

UI := npm --prefix $(ROOT)/packages/ui

help:
	@echo "Ravenpass for Android $(VERSION)"
	@echo
	@echo "  make deps            install npm dependencies for the interface"
	@echo "  make dev             run on the connected phone with live reload"
	@echo "  make run             build, install and start the debug app on the connected phone"
	@echo "  make check           vet and test, then build and lint the debug APK"
	@echo "  make apk             debug APK in $(APK_DIR)/debug"
	@echo "  make install         build and install the debug app on the connected phone"
	@echo "  make release         signed release APK (needs the RAVENPASS_ANDROID_* signing variables)"
	@echo "  make frontend        build the interface into frontend"
	@echo "  make lib             debug Go library for the APK"
	@echo "  make lib-release     release Go library for the APK"
	@echo "  make notices         third-party notices packaged into the APK"
	@echo "  make licenses-check  fail on a Go or Android dependency outside the license allowlist"
	@echo "  make strings         regenerate the native strings from the interface catalogs"
	@echo "  make logs            follow the app's log on the connected phone"
	@echo "  make clean           remove build output"

deps:
	@$(MAKE) -s --no-print-directory -C $(ROOT) node_modules

frontend: deps
	$(UI) run build

# The native strings are the interface catalogs' messages that strings.json names.
strings: deps
	node $(ROOT)/packages/ui/scripts/export-messages.ts strings.json | go run ./cmd/androidstrings -res build/android/app/src/main/res

# Under `make dev`, wails3 sets FRONTEND_DEVSERVER_URL and the library loads the interface from it.
lib: frontend
	$(GO_ANDROID) go build -buildmode=c-shared -buildvcs=false \
		-ldflags "-X main.frontendDevServerURL=$(FRONTEND_DEVSERVER_URL)" -o $(JNI_LIB) .

lib-release: frontend
	$(GO_ANDROID) go build -buildmode=c-shared -buildvcs=false -tags production -trimpath \
		-ldflags "-s -w" -o $(JNI_LIB) .

notices: deps
	$(GRADLE) generateLicenseReport
	$(call notices-start,$(NOTICES_DRAFT),Ravenpass for Android)
	$(call go-notices,$(NOTICES_DRAFT),$(GO_ANDROID))
	$(call npm-notices,$(NOTICES_DRAFT),$(ROOT)/packages/ui/package.json)
	$(call notices-section,$(NOTICES_DRAFT),Android libraries)
	grep -v -e '^Dependency License Report for' -e '^This report was generated at' $(LICENSE_REPORT) >> $(NOTICES_DRAFT)
	$(call notices-item,$(NOTICES_DRAFT),Android libraries,Apache-2.0,$(NOTICES)/overrides/android-libraries.txt $(NOTICES)/licenses/Apache-2.0.txt)
	$(call notices-item,$(NOTICES_DRAFT),Wails Android template,MIT,$(NOTICES)/overrides/wails-android-template.txt $(NOTICES)/licenses/wails.txt)
	mkdir -p $(dir $(NOTICES_FILE))
	mv $(NOTICES_DRAFT) $(NOTICES_FILE)

licenses-check:
	$(call go-licenses-check,$(GO_ANDROID))
	$(GRADLE) checkLicense

apk: lib strings notices
	$(GRADLE) assembleDebug

release: lib-release strings notices
	$(GRADLE) assembleRelease
	cp $(APK_DIR)/release/app-release.apk $(RELEASE_APK)
	cp $(NOTICES_FILE) $(APK_DIR)/release/Ravenpass-android-arm64-notices.txt

install: lib strings notices
	$(GRADLE) installDebug
	$(ADB) shell am start -S -W -n $(APP_ID)/$(ACTIVITY)

run: install

dev: deps
	$(WAILS) dev -config build/config.yml -port $(WAILS_VITE_PORT)

dev-prepare: frontend
	$(ADB) reverse tcp:$(WAILS_VITE_PORT) tcp:$(WAILS_VITE_PORT)

logs:
	$(ADB) logcat -v color --pid=$$($(ADB) shell pidof -s $(APP_ID))

check: frontend
	go vet ./...
	go test -race ./...
	$(GO_ANDROID) go vet ./... $(ROOT)/packages/app/...
	$(GO_ANDROID) go vet -tags production ./... $(ROOT)/packages/app/...
	$(MAKE) apk
	$(GRADLE) lint

clean:
	rm -rf frontend $(JNI_LIB) build/android/app/src/main/assets/autofill $(NOTICES_FILE)
	$(GRADLE) clean
