package linkproto

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
)

func TestCaptureRequestsCarryTheirFields(t *testing.T) {
	tests := []struct {
		payload string
		want    Request
	}{
		{
			`{"id":1,"type":"capture","origin":"https://example.com","account":"alex","password":"new horse","current":"old horse"}`,
			Request{ID: 1, Type: RequestCapture, Origin: "https://example.com", Account: "alex", Password: "new horse", Current: "old horse"},
		},
		{
			`{"id":2,"type":"capture","origin":"https://example.com","account":"","password":"horse"}`,
			Request{ID: 2, Type: RequestCapture, Origin: "https://example.com", Password: "horse"},
		},
		{`{"id":3,"type":"review","pending":"p1"}`, Request{ID: 3, Type: RequestReview, Pending: "p1"}},
		{
			`{"id":4,"type":"save","pending":"p1","target":"","account":"alex@example.com","name":"Example"}`,
			Request{ID: 4, Type: RequestSave, Pending: "p1", Account: "alex@example.com", Name: "Example"},
		},
		{
			`{"id":5,"type":"save","pending":"p1","target":"0102","account":"alex","name":"example.com"}`,
			Request{ID: 5, Type: RequestSave, Pending: "p1", Target: "0102", Account: "alex", Name: "example.com"},
		},
		{`{"id":6,"type":"discard","pending":"p1"}`, Request{ID: 6, Type: RequestDiscard, Pending: "p1"}},
		{
			`{"id":7,"type":"card-capture","origin":"https://shop.example.com","holder":"Alex Example","number":"4111111111111111",` +
				`"expiry":"2029-08","securityCode":"739","network":"visa"}`,
			Request{ID: 7, Type: RequestCardCapture, Origin: "https://shop.example.com", Card: CardFields{
				Holder: "Alex Example", Number: "4111111111111111", Expiry: "2029-08", SecurityCode: "739", Network: "visa",
			}},
		},
		{
			`{"id":8,"type":"card-fill","origin":"https://shop.example.com","card":"0102"}`,
			Request{ID: 8, Type: RequestCardFill, Origin: "https://shop.example.com", Card: CardFields{Card: "0102"}},
		},
		{`{"id":9,"type":"cards","origin":"https://shop.example.com"}`, Request{ID: 9, Type: RequestCards, Origin: "https://shop.example.com"}},
	}
	for _, test := range tests {
		request, err := ParseRequest([]byte(test.payload))
		if err != nil || !reflect.DeepEqual(request, test.want) {
			t.Fatalf("%s parsed as %+v, error = %v", test.payload, request, err)
		}
	}
	for _, payload := range []string{
		`{"id":1,"type":"capture","origin":"https://example.com","password":["horse"]}`,
		`{"id":1,"type":"capture","origin":"https://example.com","password":"horse","current":1}`,
		`{"id":1,"type":"review","pending":{}}`,
		`{"id":1,"type":"save","pending":"p1","target":2}`,
		`{"id":1,"type":"save","pending":"p1","name":false}`,
		`{"id":1,"type":"card-capture","origin":"https://example.com","number":4111111111111111}`,
		`{"id":1,"type":"card-fill","origin":"https://example.com","card":["0102"]}`,
	} {
		if _, err := ParseRequest([]byte(payload)); !errors.Is(err, ErrMalformed) {
			t.Errorf("payload %q: got %v, want ErrMalformed", payload, err)
		}
	}
}

func TestCaptureResultsEncodeAsTheExtensionReadsThem(t *testing.T) {
	tests := []struct {
		result any
		want   string
	}{
		{
			CaptureOffer{
				Kind: CapturePassword, State: CaptureReady, Pending: "p1", Site: "github.com", Account: "alex", Name: "github.com",
				Targets: []SaveTarget{
					{Credential: "a1", Label: "GitHub", Account: "alex", Action: SaveUpdate},
					{Credential: "b2", Label: "Mail", Account: "alex@example.com", Action: SaveAddSite},
				},
				Suggested: "a1",
			},
			`{"kind":"password","state":"ready","pending":"p1","site":"github.com","account":"alex","name":"github.com",` +
				`"targets":[{"credential":"a1","label":"GitHub","account":"alex","action":"update"},` +
				`{"credential":"b2","label":"Mail","account":"alex@example.com","action":"add-site"}],"suggested":"a1"}`,
		},
		{
			CaptureOffer{Kind: CapturePassword, State: CaptureNone, Site: "github.com", Account: "alex", Name: "github.com", Targets: []SaveTarget{}},
			`{"kind":"password","state":"none","site":"github.com","account":"alex","name":"github.com","targets":[],"suggested":""}`,
		},
		{
			CaptureOffer{Kind: CapturePassword, State: CaptureLocked, Pending: "p2", Site: "github.com", Name: "github.com", Targets: []SaveTarget{}},
			`{"kind":"password","state":"locked","pending":"p2","site":"github.com","account":"","name":"github.com","targets":[],"suggested":""}`,
		},
		{
			CaptureOffer{
				Kind: CaptureCard, State: CaptureReady, Pending: "p3", Site: "shop.example.com", Card: &CardFace{Network: "visa", LastFour: "1111"},
				Targets: []SaveTarget{{Credential: "c3", Label: "Everyday", Action: SaveUpdate}}, Suggested: "c3",
			},
			`{"kind":"card","state":"ready","pending":"p3","site":"shop.example.com","account":"","name":"",` +
				`"card":{"network":"visa","lastFour":"1111"},` +
				`"targets":[{"credential":"c3","label":"Everyday","account":"","action":"update"}],"suggested":"c3"}`,
		},
		{
			Cards{Cards: []CardOption{{ID: "c3", Label: "Everyday", BankName: "Example Bank", Site: "bank.example", Network: "visa", LastFour: "1111", Color: "#00a0e1", ExpiresOn: "2029-08-31"}}},
			`{"cards":[{"id":"c3","label":"Everyday","bankName":"Example Bank","site":"bank.example","network":"visa",` +
				`"lastFour":"1111","color":"#00a0e1","expiresOn":"2029-08-31"}]}`,
		},
		{
			CardFill{Holder: "Alex Example", Number: "4111111111111111", Expiry: "2029-08", SecurityCode: "739",
				Billing: &BillingAddress{Street: "1 Example Street", City: "Springfield", Region: "", PostalCode: "12345", Country: "US"}},
			`{"holder":"Alex Example","number":"4111111111111111","expiry":"2029-08","securityCode":"739",` +
				`"billing":{"street":"1 Example Street","city":"Springfield","region":"","postalCode":"12345","country":"US"}}`,
		},
		{CardFill{Number: "4111111111111111"}, `{"holder":"","number":"4111111111111111","expiry":"","securityCode":"","billing":null}`},
		{Saved{Saved: SavedCreated}, `{"saved":"created"}`},
		{Saved{Saved: SavedUpdated}, `{"saved":"updated"}`},
	}
	for _, test := range tests {
		encoded, err := json.Marshal(test.result)
		if err != nil || string(encoded) != test.want {
			t.Fatalf("%+v encodes as %s, error = %v", test.result, encoded, err)
		}
	}
}

func TestOnlyAnOriginANetworkAttackerCannotServeIsSecure(t *testing.T) {
	for origin, want := range map[string]bool{
		"https://shop.example.com":     true,
		"https://shop.example.com:444": true,
		"http://localhost:8080":        true,
		"http://shop.localhost":        true,
		"http://shop.example.com":      false,
		"http://127.0.0.1:8080":        false,
		"http://localhost.example.com": false,
		"ftp://shop.example.com":       false,
		"not an origin":                false,
	} {
		if got := SecureOrigin(origin); got != want {
			t.Errorf("SecureOrigin(%q) = %v, want %v", origin, got, want)
		}
	}
}
