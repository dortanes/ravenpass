package linkserver

import (
	"context"
	"encoding/json"
	"slices"
	"testing"
	"time"

	"github.com/dortanes/ravenpass/packages/app/linkproto"
)

const (
	cardID   = "c4000000000000000000000000000000"
	checkout = "https://shop.example.com"
)

func (v *fakeVault) Cards() ([]linkproto.CardOption, error) {
	if err := v.call("cards"); err != nil {
		return nil, err
	}
	return []linkproto.CardOption{{ID: cardID, Label: "Travel", Network: "visa", LastFour: "1111", ExpiresOn: "2029-08-31"}}, nil
}

func (v *fakeVault) FillCard(ctx context.Context, card, origin string, asked func(linkproto.Progress)) (linkproto.CardFill, error) {
	if err := v.call("card-fill " + card + " " + origin); err != nil {
		return linkproto.CardFill{}, err
	}
	if err := v.verify(ctx, asked); err != nil {
		return linkproto.CardFill{}, err
	}
	return linkproto.CardFill{Holder: "Alex Example", Number: "4111111111111111", Expiry: "2029-08", SecurityCode: "739"}, nil
}

// cardCaptureOffer answers locked, nothing for the travel card, else an update of the travel card.
func (v *fakeVault) cardCaptureOffer(capture Capture) (linkproto.CaptureOffer, error) {
	card := capture.Card
	if err := v.call("card-capture " + capture.Origin + " " + card.Number + " " + card.Expiry); err != nil {
		return linkproto.CaptureOffer{}, err
	}
	offer := linkproto.CaptureOffer{Kind: linkproto.CaptureCard, State: linkproto.CaptureLocked, Site: "shop.example.com", Card: &linkproto.CardFace{Network: card.Network, LastFour: "1111"}}
	switch {
	case !v.Unlocked():
	case card.Expiry == "2029-08":
		offer.State = linkproto.CaptureNone
	default:
		offer.State = linkproto.CaptureReady
		offer.Targets = []linkproto.SaveTarget{{Credential: cardID, Label: "Travel", Action: linkproto.SaveUpdate}}
		offer.Suggested = cardID
	}
	return offer, nil
}

func TestCardsAreListedAndFilledAfterThePersonConfirms(t *testing.T) {
	server, vault := newServer(t, recordsPath(t), func(s *Server) {
		s.progressInterval = 40 * time.Millisecond
		s.idleTimeout = 100 * time.Millisecond
	})
	vault.unlocked.Store(true)
	session := openCaptureSession(t, server)
	want := `{"id":1,"result":{"cards":[{"id":"` + cardID + `","label":"Travel","bankName":"","site":"","network":"visa","lastFour":"1111",` +
		`"color":"","expiresOn":"2029-08-31"}]}}`
	if reply := session.ask(map[string]any{"type": "cards", "origin": checkout}); reply != want {
		t.Fatalf("cards = %s", reply)
	}
	vault.verifyWith(verifiedAfter(linkproto.ProgressConfirmOnDevice, 150*time.Millisecond, nil))
	sendRequest(t, session.conn, session.transport, `{"id":7,"type":"card-fill","card":"`+cardID+`","origin":"`+checkout+`"}`)
	progress, final := progressThen(t, session.conn, session.transport)
	if len(progress) < 2 || slices.ContainsFunc(progress, func(p string) bool { return p != "confirm-on-device" }) {
		t.Fatalf("progress = %q", progress)
	}
	if want := `{"holder":"Alex Example","number":"4111111111111111","expiry":"2029-08","securityCode":"739","billing":null}`; final.ID != 7 || string(final.Result) != want {
		t.Fatalf("after the progress = %+v", final)
	}
	vault.verifyWith(verifiedAfter(linkproto.ProgressConfirmInRavenpass, 10*time.Millisecond, ErrDeclined))
	sendRequest(t, session.conn, session.transport, `{"id":7,"type":"card-fill","card":"`+cardID+`","origin":"`+checkout+`"}`)
	if progress, refusal := progressThen(t, session.conn, session.transport); !slices.Equal(progress, []string{"confirm-in-ravenpass"}) || refusal.Error != "declined" || refusal.Result != nil {
		t.Fatalf("progress %q, then %+v", progress, refusal)
	}
	vault.verifyWith(nil)
	if calls := vault.takeCalls(); !slices.Equal(calls, []string{"cards", "card-fill " + cardID + " " + checkout, "card-fill " + cardID + " " + checkout}) {
		t.Fatalf("the vault was asked %q", calls)
	}
}

func TestCardRequestsFromAnInsecurePageAreRefusedBeforeTheVault(t *testing.T) {
	server, vault := newServer(t, recordsPath(t))
	vault.unlocked.Store(true)
	session := openCaptureSession(t, server)
	for _, origin := range []string{"http://shop.example.com", "http://127.0.0.1:8080", "", "https://shop.example.com/pay"} {
		for _, fields := range []map[string]any{
			{"type": "cards", "origin": origin},
			{"type": "card-fill", "origin": origin, "card": cardID},
			{"type": "card-capture", "origin": origin, "number": "4111111111111111", "network": "visa"},
		} {
			if reply := session.ask(fields); reply != `{"id":1,"error":"invalid-origin"}` {
				t.Fatalf("%v answered %s", fields, reply)
			}
		}
	}
	if calls := vault.takeCalls(); len(calls) != 0 {
		t.Fatalf("insecure pages reached the vault: %q", calls)
	}
	if reply := session.ask(map[string]any{"type": "cards", "origin": "http://localhost:8080"}); reply == `{"id":1,"error":"invalid-origin"}` {
		t.Fatal("a localhost checkout was refused")
	}
}

func TestACardCaptureIsHeldReviewedAndSavedLikeAPassword(t *testing.T) {
	server, vault := newServer(t, recordsPath(t))
	vault.unlocked.Store(true)
	session := openCaptureSession(t, server)
	capture := func(expiry string) linkproto.CaptureOffer {
		t.Helper()
		reply := session.ask(map[string]any{
			"type": "card-capture", "origin": checkout, "holder": "Alex Example", "number": "4111111111111111",
			"expiry": expiry, "securityCode": "739", "network": "visa",
		})
		var answer struct {
			Result linkproto.CaptureOffer `json:"result"`
		}
		if err := json.Unmarshal([]byte(reply), &answer); err != nil {
			t.Fatalf("card capture answered %s", reply)
		}
		return answer.Result
	}
	if offer := capture("2029-08"); offer.State != linkproto.CaptureNone || offer.Pending != "" {
		t.Fatalf("a held card offered %+v", offer)
	}
	offer := capture("2032-01")
	if offer.Kind != linkproto.CaptureCard || offer.State != linkproto.CaptureReady || !pendingID.MatchString(offer.Pending) || offer.Suggested != cardID {
		t.Fatalf("a renewed card offered %+v", offer)
	}
	if reply := session.review(offer.Pending); reply == notFound {
		t.Fatal("the card capture was not held")
	}
	if reply := session.save(offer.Pending, cardID, "", ""); reply != `{"id":1,"result":{"saved":"updated"}}` {
		t.Fatalf("save = %s", reply)
	}
	want := []string{
		"card-capture " + checkout + " 4111111111111111 2029-08",
		"card-capture " + checkout + " 4111111111111111 2032-01",
		"card-capture " + checkout + " 4111111111111111 2032-01",
		`save 4111111111111111 to "` + cardID + `" as  `,
	}
	if calls := vault.takeCalls(); !slices.Equal(calls, want) {
		t.Fatalf("the vault was asked %q", calls)
	}
}
