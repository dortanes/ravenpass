# Third-party notices and the license allowlist. Included by each Makefile after it sets ROOT.

NOTICES := $(ROOT)/notices
NOTICES_RULE := ================================================================================
NOTICES_ITEM_RULE := --------------------------------------------------------------------------------

comma := ,
space := $(subst ,, )
ALLOWED_LICENSES = $(shell node -p 'require("$(NOTICES)/allowed-licenses.json").allowedLicenses.map((l) => l.moduleLicense).join(" ")')

# $(call go-licenses,target environment,command) analyses the target platform's package graph with a host-built tool.
go-licenses = go run -exec 'env $(1) GOFLAGS=-tags=production' github.com/google/go-licenses/v2@v2.0.1 $(2) \
	--logtostderr=false --stderrthreshold=ERROR --log_file=/dev/null --ignore github.com/dortanes/ravenpass

# $(call go-licenses-check,target environment) fails on a Go dependency outside the allowlist.
go-licenses-check = $(call go-licenses,$(1),check) --allowed_licenses=$(subst $(space),$(comma),$(ALLOWED_LICENSES)) .

# $(call notices-start,file,product) starts the notices file for product with Ravenpass's own license.
define notices-start
printf '%s\n%s\n\n' 'Ravenpass is licensed under GPL-3.0-or-later with the additional permission that follows the license.' \
	'Source code: https://github.com/dortanes/ravenpass' > $(1)
cat $(ROOT)/LICENSE >> $(1)
printf '\n' >> $(1)
cat $(ROOT)/LICENSE-EXCEPTION >> $(1)
$(call notices-section,$(1),$(2) $(VERSION): third-party notices)
printf '\n%s\n' '$(2) includes the third-party software below, each under the license reproduced with it.' >> $(1)
endef

# $(call notices-section,file,title) appends a section heading.
define notices-section
printf '\n%s\n%s\n%s\n' '$(NOTICES_RULE)' '$(2)' '$(NOTICES_RULE)' >> $(1)
endef

# $(call notices-item,file,name,license,text files) appends one entry.
define notices-item
printf '\n%s\n%s\nLicense: %s\n\n' '$(NOTICES_ITEM_RULE)' "$(2)" '$(3)' >> $(1)
cat $(4) >> $(1)
endef

# $(call go-notices,file,target environment) appends the Go modules, the standard library and the Public Suffix List.
define go-notices
$(call notices-section,$(1),Go modules)
$(call go-licenses,$(2),report) --template $(NOTICES)/go.tmpl . >> $(1)
saved="$$(mktemp -d)" && $(call go-licenses,$(2),save) --save_path "$$saved/modules" . && \
	for notice in $$(cd "$$saved/modules" && find . -type f -name 'NOTICE*' | sort); do \
		printf '\n%s\n%s\n\n' '$(NOTICES_ITEM_RULE)' "$${notice#./}"; cat "$$saved/modules/$$notice"; \
	done >> $(1) && rm -rf "$$saved"
$(call notices-item,$(1),Go standard library and runtime $$(go env GOVERSION),BSD-3-Clause,$(NOTICES)/licenses/go.txt)
$(call notices-item,$(1),Public Suffix List,MPL-2.0,$(NOTICES)/overrides/public-suffix-list.txt $(NOTICES)/licenses/MPL-2.0.txt)
endef

# $(call npm-notices,file,package.json files) appends the production npm packages of the given workspaces.
define npm-notices
$(call notices-section,$(1),npm packages)
printf '\n' >> $(1)
npm_notices="$$(mktemp)" && $(ROOT)/node_modules/.bin/generate-license-file --ci --no-spinner --overwrite \
	--config $(NOTICES)/generate-license-file.json --input $(2) --output "$$npm_notices" && \
	cat "$$npm_notices" >> $(1) && rm "$$npm_notices"
endef

NPM_DISALLOWED = :root .prod:not(.workspace):not($(subst $(space),$(comma),$(patsubst %,[license=%],$(ALLOWED_LICENSES))))

# Fails on a production npm dependency of any workspace whose declared license is outside the allowlist.
define npm-licenses-check
@packages="$$(cd $(ROOT) && npm query '$(NPM_DISALLOWED)' | node -p 'JSON.parse(require("fs").readFileSync(0, "utf8")).map((p) => `$${p.name}@$${p.version} $${p.license}`).join("\n")')" \
	&& test -z "$$packages" \
	|| { echo "npm production packages with a license outside notices/allowed-licenses.json:"; echo "$$packages"; exit 1; }
endef
