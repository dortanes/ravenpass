module github.com/dortanes/ravenpass/packages/importers

go 1.27.1

require (
	github.com/dortanes/ravenpass/packages/vault v0.0.0-00010101000000-000000000000
	github.com/google/uuid v1.6.0
	golang.org/x/crypto v0.57.0
)

require (
	github.com/boombuler/barcode v1.0.1-0.20190219062509-6c824513bacc // indirect
	github.com/fxamacker/cbor/v2 v2.9.4 // indirect
	github.com/kslamph/bip39-hdwallet v1.1.0 // indirect
	github.com/pquerna/otp v1.5.0 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/image v0.46.0 // indirect
	golang.org/x/net v0.59.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
	golang.org/x/text v0.42.0 // indirect
)

replace github.com/dortanes/ravenpass/packages/vault => ../vault
