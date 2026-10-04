module github.com/dortanes/ravenpass/packages/app

go 1.27.1

require (
	github.com/PuerkitoBio/goquery v1.13.0
	github.com/coder/websocket v1.8.15
	github.com/dortanes/ravenpass/packages/authenticator v0.0.0-00010101000000-000000000000
	github.com/dortanes/ravenpass/packages/importers v0.0.0-00010101000000-000000000000
	github.com/dortanes/ravenpass/packages/vault v0.0.0-00010101000000-000000000000
	github.com/flynn/noise v1.1.0
	github.com/mat/besticon/v3 v3.23.0
	github.com/mazznoer/csscolorparser v0.1.8
	github.com/sergeymakinen/go-ico v1.0.0
	github.com/wailsapp/wails/v3 v3.0.0-beta.23
	github.com/xgfone/go-imagex v0.6.0
	golang.org/x/image v0.46.0
	golang.org/x/sync v0.23.0
	golang.org/x/sys v0.48.0
	golang.org/x/text v0.42.0
)

require (
	github.com/adrg/xdg v0.5.3 // indirect
	github.com/andybalholm/cascadia v1.3.4 // indirect
	github.com/boombuler/barcode v1.0.1-0.20190219062509-6c824513bacc // indirect
	github.com/fxamacker/cbor/v2 v2.9.4 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/godbus/dbus/v5 v5.2.2 // indirect
	github.com/golang/groupcache v0.0.0-20241129210726-2c02b8208cf8 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/kslamph/bip39-hdwallet v1.1.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/pquerna/otp v1.5.0 // indirect
	github.com/sergeymakinen/go-bmp v1.0.0 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	golang.org/x/crypto v0.57.0 // indirect
	golang.org/x/net v0.59.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace (
	github.com/dortanes/ravenpass/packages/authenticator => ../authenticator
	github.com/dortanes/ravenpass/packages/importers => ../importers
	github.com/dortanes/ravenpass/packages/vault => ../vault
)
