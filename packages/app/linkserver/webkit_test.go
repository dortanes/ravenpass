package linkserver

import (
	"crypto/rand"
	"net/http"
	"testing"

	"github.com/coder/websocket"
	"github.com/dortanes/ravenpass/packages/app/linkproto"
)

func newP256Extension(t *testing.T) *extension {
	t.Helper()
	key, err := linkproto.GenerateP256KeyPair(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return &extension{key: key}
}

func TestWebKitLinksPersistAlongsideExistingChromeLinks(t *testing.T) {
	for _, origin := range []string{
		"webkit-extension://cceiadaelnccfbakmhcleifjfilkakag",
		"webkit-extension://12345678-1234-1234-1234-123456789abc",
		"safari-web-extension://12345678-1234-1234-1234-123456789ABC",
	} {
		t.Run(origin, func(t *testing.T) {
			path := recordsPath(t)
			server, _ := newServer(t, path)
			legacy := newExtension(t)
			legacy.linkWith(t, begin(t, server))
			key := begin(t, server)
			peer := newP256Extension(t)
			conn, _, err := dial(int(key.Port), linkproto.P256LinkPath, &websocket.DialOptions{HTTPHeader: http.Header{"Origin": {origin}}})
			if err != nil {
				t.Fatal(err)
			}
			message, err := peer.link(conn, key.Secret[:], "WebKit")
			if err != nil || message != `{"type":"linked","language":"en","signIn":"card"}` {
				t.Fatalf("link: %q, %v", message, err)
			}
			wantStatus(t, closeStatus(t, conn), linkproto.CloseFinished)
			if err := server.Close(); err != nil {
				t.Fatal(err)
			}
			restarted, vault := newServer(t, path)
			if err := restarted.Start(); err != nil {
				t.Fatal(err)
			}
			vault.unlocked.Store(true)
			for _, linked := range []*extension{legacy, peer} {
				conn, transport := linked.openSession(t, int(key.Port))
				if got := request(t, conn, transport, `{"id":1,"type":"status"}`); got != `{"id":1,"result":{"vault":"unlocked"}}` {
					t.Fatal(got)
				}
				conn.CloseNow()
			}
			record, found, err := restarted.store.Find(peer.key.Public)
			if err != nil || !found {
				t.Fatalf("missing peer: %v", err)
			}
			if err := restarted.Unlink(record.ID); err != nil {
				t.Fatal(err)
			}
			conn = connect(t, int(key.Port), linkproto.P256SessionPath)
			if _, err := peer.session(conn); err == nil {
				t.Fatal("revoked peer opened a session")
			} else {
				wantStatus(t, websocket.CloseStatus(err), linkproto.CloseNotLinked)
			}
		})
	}
}

func TestP256RefusesWrongSecretsAndOtherProtocolMessagesWithoutSpendingTheKey(t *testing.T) {
	server, _ := newServer(t, recordsPath(t))
	key := begin(t, server)
	peer := newP256Extension(t)
	wrong := key.Secret
	wrong[0] ^= 1
	conn := connect(t, int(key.Port), linkproto.P256LinkPath)
	if _, err := peer.link(conn, wrong[:], "WebKit"); err == nil {
		t.Fatal("wrong secret linked")
	} else {
		wantStatus(t, websocket.CloseStatus(err), linkproto.CloseUnauthenticated)
	}
	conn = connect(t, int(key.Port), linkproto.P256LinkPath)
	if _, err := newExtension(t).link(conn, key.Secret[:], "WebKit"); err == nil {
		t.Fatal("v1 handshake linked on v2 endpoint")
	}
	conn.CloseNow()
	if _, waiting := server.Waiting(); !waiting {
		t.Fatal("refused peer spent the key")
	}
	peer.linkWith(t, key)
}
