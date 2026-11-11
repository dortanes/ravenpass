# Ravenpass for Chromium and WebKit. Run `make help` for the targets.

.DEFAULT_GOAL := help
.PHONY: help deps dev check build notices package clean

ROOT := $(abspath ../..)
VERSION := $(shell tr -d '[:space:]' < $(ROOT)/version.txt)
PACKAGE := dist/Ravenpass-$(VERSION)-chrome.zip
STORE_PACKAGE := dist/Ravenpass-$(VERSION)-chrome-web-store.zip
RELEASE_NOTICES := dist/Ravenpass-$(VERSION)-chrome-notices.txt
NOTICES_FILE := dist/chrome/THIRD_PARTY_NOTICES.txt

include $(ROOT)/notices/notices.mk

help:
	@echo "Ravenpass for Chrome $(VERSION)"
	@echo
	@echo "  make deps     install npm dependencies"
	@echo "  make dev      run Chrome with the extension and live reload"
	@echo "  make check    lint, type-check, test and build"
	@echo "  make build    one Chromium/WebKit extension in dist/chrome"
	@echo "  make notices  third-party notices in $(NOTICES_FILE)"
	@echo "  make package  zips for loading unpacked and for the Chrome Web Store"
	@echo "  make clean    remove build output"

deps:
	@$(MAKE) -s --no-print-directory -C $(ROOT) node_modules

dev: deps
	npm run dev

check: deps
	npm run typecheck
	npm run check
	npm test
	npm run build

build: deps
	npm run build

notices: build
	$(call notices-start,dist/THIRD_PARTY_NOTICES.txt,Ravenpass for Chrome)
	$(call npm-notices,dist/THIRD_PARTY_NOTICES.txt,$(CURDIR)/package.json $(ROOT)/packages/ui/package.json)
	mv dist/THIRD_PARTY_NOTICES.txt $(NOTICES_FILE)

# The store package omits `key`: the Chrome Web Store holds the item's key itself.
package: build notices
	@test "$$(node -p 'require("./dist/chrome/manifest.json").version')" = "$(VERSION)" \
		|| { echo "src/manifest.json version differs from version.txt"; exit 1; }
	rm -f $(PACKAGE) $(STORE_PACKAGE)
	cp $(NOTICES_FILE) $(RELEASE_NOTICES)
	cd dist/chrome && zip -qrX ../$(notdir $(PACKAGE)) .
	rm -rf dist/store && cp -R dist/chrome dist/store
	node -e 'const f="dist/store/manifest.json",fs=require("fs"),m=JSON.parse(fs.readFileSync(f));delete m.key;fs.writeFileSync(f,JSON.stringify(m))'
	cd dist/store && zip -qrX ../$(notdir $(STORE_PACKAGE)) .
	rm -rf dist/store

clean:
	rm -rf dist .extension-js
