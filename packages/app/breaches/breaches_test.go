package breaches

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// "password" hashes to 5BAA61E4C9B93F3F0682250B6CF8331B7EE68FD8.
const (
	knownPrefix = "5BAA6"
	knownSuffix = "1E4C9B93F3F0682250B6CF8331B7EE68FD8"
)

// rangeServer answers every prefix with body and records each request.
type rangeServer struct {
	mu       sync.Mutex
	requests []*http.Request
	status   int
	body     string
}

func (s *rangeServer) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	s.mu.Lock()
	s.requests = append(s.requests, request)
	status, body := s.status, s.body
	s.mu.Unlock()
	if status != 0 {
		writer.WriteHeader(status)
	}
	fmt.Fprint(writer, body)
}

func (s *rangeServer) asked() []*http.Request {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]*http.Request(nil), s.requests...)
}

func serve(t *testing.T, server *rangeServer) *Client {
	t.Helper()
	listener := httptest.NewServer(server)
	t.Cleanup(listener.Close)
	return newClient(listener.URL+"/range/", listener.Client().Transport)
}

func TestCountSendsOnlyThePrefixAndAsksForPadding(t *testing.T) {
	server := &rangeServer{body: "0018A45C4D1DEF81644B54AB7F969B88D65:1\r\n" + knownSuffix + ":10434004\r\n00D4F6E8FA6EECAD2A3AA415EEC418D38EC:0\r\n"}
	client := serve(t, server)
	count, err := client.Count(context.Background(), Digest("password"))
	if err != nil || count != 10434004 {
		t.Fatalf("count = %d, %v", count, err)
	}
	asked := server.asked()
	if len(asked) != 1 {
		t.Fatalf("requests = %d", len(asked))
	}
	request := asked[0]
	if request.URL.Path != "/range/"+knownPrefix || request.URL.RawQuery != "" {
		t.Fatalf("asked for %s", request.URL)
	}
	if request.Header.Get("Add-Padding") != "true" || request.Header.Get("User-Agent") != userAgent {
		t.Fatalf("headers = %v", request.Header)
	}
	if strings.Contains(request.URL.String(), knownSuffix) {
		t.Fatal("the request carried the hash suffix")
	}
}

func TestAPaddedOrUnlistedSuffixCountsZero(t *testing.T) {
	server := &rangeServer{body: knownSuffix + ":0\n"}
	client := serve(t, server)
	if count, err := client.Count(context.Background(), Digest("password")); err != nil || count != 0 {
		t.Fatalf("a padding line = %d, %v", count, err)
	}
	if count, err := client.Count(context.Background(), Digest("another password entirely")); err != nil || count != 0 {
		t.Fatalf("an unlisted suffix = %d, %v", count, err)
	}
}

func TestAnswersAreKeptUntilForgotten(t *testing.T) {
	server := &rangeServer{body: knownSuffix + ":3\n"}
	client := serve(t, server)
	for range 3 {
		if _, err := client.Count(context.Background(), Digest("password")); err != nil {
			t.Fatal(err)
		}
	}
	if asked := len(server.asked()); asked != 1 {
		t.Fatalf("a kept answer was asked %d times", asked)
	}
	client.Forget()
	if _, err := client.Count(context.Background(), Digest("password")); err != nil {
		t.Fatal(err)
	}
	if asked := len(server.asked()); asked != 2 {
		t.Fatalf("a forgotten answer was asked %d times in all", asked)
	}
}

// heldServer answers once release is closed, telling started when a request arrives.
type heldServer struct {
	started chan struct{}
	release chan struct{}
}

func (s *heldServer) ServeHTTP(writer http.ResponseWriter, _ *http.Request) {
	s.started <- struct{}{}
	<-s.release
	fmt.Fprint(writer, knownSuffix+":3\n")
}

func TestAnAnswerArrivingAfterForgetIsNotKept(t *testing.T) {
	server := &heldServer{started: make(chan struct{}, 2), release: make(chan struct{})}
	listener := httptest.NewServer(server)
	t.Cleanup(listener.Close)
	client := newClient(listener.URL+"/range/", listener.Client().Transport)
	done := make(chan error, 1)
	go func() {
		_, err := client.Count(context.Background(), Digest("password"))
		done <- err
	}()
	<-server.started
	client.Forget()
	close(server.release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	client.mu.Lock()
	kept := len(client.answers)
	client.mu.Unlock()
	if kept != 0 {
		t.Fatalf("%d answers kept after Forget", kept)
	}
}

func TestACancelledCallerFailsAloneAndTheSharedRequestCompletes(t *testing.T) {
	server := &heldServer{started: make(chan struct{}, 2), release: make(chan struct{})}
	listener := httptest.NewServer(server)
	t.Cleanup(listener.Close)
	client := newClient(listener.URL+"/range/", listener.Client().Transport)
	ctx, cancel := context.WithCancel(context.Background())
	first := make(chan error, 1)
	go func() {
		_, err := client.Count(ctx, Digest("password"))
		first <- err
	}()
	<-server.started
	second := make(chan int, 1)
	go func() {
		count, _ := client.Count(context.Background(), Digest("password"))
		second <- count
	}()
	cancel()
	if err := <-first; !errors.Is(err, ErrUnanswered) {
		t.Fatalf("the cancelled caller = %v", err)
	}
	close(server.release)
	if count := <-second; count != 3 {
		t.Fatalf("the waiting caller counted %d", count)
	}
}

func TestAMalformedOrFailedAnswerIsAnError(t *testing.T) {
	for name, test := range map[string]struct {
		server *rangeServer
		want   error
	}{
		"a server error":     {&rangeServer{status: http.StatusServiceUnavailable}, ErrUnanswered},
		"no colon":           {&rangeServer{body: knownSuffix + "\n"}, ErrMalformedAnswer},
		"a short suffix":     {&rangeServer{body: "ABC:1\n"}, ErrMalformedAnswer},
		"a lower-case hash":  {&rangeServer{body: strings.ToLower(knownSuffix) + ":1\n"}, ErrMalformedAnswer},
		"a negative count":   {&rangeServer{body: knownSuffix + ":-1\n"}, ErrMalformedAnswer},
		"an oversize answer": {&rangeServer{body: strings.Repeat(knownSuffix+":1\n", maxAnswerBytes/38+1)}, ErrMalformedAnswer},
	} {
		client := serve(t, test.server)
		if _, err := client.Count(context.Background(), Digest("password")); !errors.Is(err, test.want) {
			t.Fatalf("%s: got %v, want %v", name, err, test.want)
		}
	}
}
