module github.com/dortanes/ravenpass/packages/authenticator

go 1.27.1

require (
	github.com/dortanes/ravenpass/packages/vault v0.0.0-00010101000000-000000000000
	github.com/fxamacker/cbor/v2 v2.9.4
	github.com/go-webauthn/webauthn v0.18.2
	golang.org/x/net v0.59.0
)

replace github.com/dortanes/ravenpass/packages/vault => ../vault

require (
	github.com/boombuler/barcode v1.0.1-0.20190219062509-6c824513bacc // indirect
	github.com/go-viper/mapstructure/v2 v2.5.0 // indirect
	github.com/go-webauthn/x v0.3.1 // indirect
	github.com/golang-jwt/jwt/v5 v5.3.1 // indirect
	github.com/google/go-tpm v0.9.8 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/kslamph/bip39-hdwallet v1.1.0 // indirect
	github.com/philhofer/fwd v1.2.0 // indirect
	github.com/pquerna/otp v1.5.0 // indirect
	github.com/tinylib/msgp v1.6.4 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	golang.org/x/crypto v0.57.0 // indirect
	golang.org/x/image v0.46.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
	golang.org/x/text v0.42.0 // indirect
)
