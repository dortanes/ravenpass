// Package breaches asks the Pwned Passwords range service how often a password appears in known breaches. Only the
// first five hexadecimal characters of the password's SHA-1 hash leave the device; the rest is matched here.
package breaches

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

const (
	// rangeURL is the service's range endpoint; a hash prefix follows it.
	rangeURL     = "https://api.pwnedpasswords.com/range/"
	userAgent    = "Ravenpass"
	fetchTimeout = 10 * time.Second
	prefixLength = 5
	// maxAnswerBytes bounds one answer; a padded answer holds about a thousand 40-byte lines.
	maxAnswerBytes = 1 << 20
)

var (
	// ErrUnanswered means the service could not be reached or did not answer with a range.
	ErrUnanswered = errors.New("the breach service did not answer")
	// ErrMalformedAnswer means the service answered with something other than suffixes and counts.
	ErrMalformedAnswer = errors.New("the breach service's answer is malformed")
)

// Digest is a password's SHA-1 hash, the form the service matches.
func Digest(password string) [sha1.Size]byte {
	return sha1.Sum([]byte(password))
}

// Client asks the service for the hash suffixes sharing a prefix and keeps each answer in memory until Forget.
type Client struct {
	endpoint string
	http     *http.Client
	flight   singleflight.Group

	mu sync.Mutex
	// answers maps a requested prefix to the count of each suffix the service listed.
	answers map[string]map[string]int
	// generation counts Forget calls; an answer fetched across one is not kept.
	generation uint64
}

// New returns a Client of the Pwned Passwords service.
func New() *Client {
	return newClient(rangeURL, http.DefaultTransport)
}

func newClient(endpoint string, transport http.RoundTripper) *Client {
	return &Client{
		endpoint: endpoint,
		http:     &http.Client{Transport: transport, Timeout: fetchTimeout},
		answers:  map[string]map[string]int{},
	}
}

// Count reports how many times the password with digest appears in known breaches; zero when it is not known.
func (c *Client) Count(ctx context.Context, digest [sha1.Size]byte) (int, error) {
	hash := strings.ToUpper(hex.EncodeToString(digest[:]))
	answer, err := c.answer(ctx, hash[:prefixLength])
	if err != nil {
		return 0, err
	}
	return answer[hash[prefixLength:]], nil
}

// Forget drops every kept answer, and any answer still on its way when it arrives.
func (c *Client) Forget() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.answers = map[string]map[string]int{}
	c.generation++
}

// answer is the kept answer for prefix, asked once however many callers want it at the same time. The shared request
// outlives the caller that started it, within the client's timeout, so one caller giving up fails no other.
func (c *Client) answer(ctx context.Context, prefix string) (map[string]int, error) {
	c.mu.Lock()
	kept, found := c.answers[prefix]
	generation := c.generation
	c.mu.Unlock()
	if found {
		return kept, nil
	}
	shared := c.flight.DoChan(prefix, func() (any, error) {
		answer, err := c.fetch(context.WithoutCancel(ctx), prefix)
		if err != nil {
			return nil, err
		}
		c.mu.Lock()
		if c.generation == generation {
			c.answers[prefix] = answer
		}
		c.mu.Unlock()
		return answer, nil
	})
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("%w: %w", ErrUnanswered, ctx.Err())
	case result := <-shared:
		if result.Err != nil {
			return nil, result.Err
		}
		return result.Val.(map[string]int), nil
	}
}

// fetch asks for the suffixes of prefix with padding, whose zero-count lines it drops.
func (c *Client) fetch(ctx context.Context, prefix string) (map[string]int, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint+prefix, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Add-Padding", "true")
	request.Header.Set("User-Agent", userAgent)
	response, err := c.http.Do(request)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUnanswered, err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d", ErrUnanswered, response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxAnswerBytes+1))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUnanswered, err)
	}
	if len(body) > maxAnswerBytes {
		return nil, ErrMalformedAnswer
	}
	return parse(string(body))
}

// parse reads "SUFFIX:COUNT" lines: a 35-character upper-case hexadecimal suffix and a decimal count.
func parse(body string) (map[string]int, error) {
	answer := map[string]int{}
	for line := range strings.Lines(body) {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		suffix, number, found := strings.Cut(line, ":")
		count, err := strconv.Atoi(number)
		if !found || err != nil || count < 0 || !validSuffix(suffix) {
			return nil, ErrMalformedAnswer
		}
		if count > 0 {
			answer[suffix] = count
		}
	}
	return answer, nil
}

func validSuffix(suffix string) bool {
	if len(suffix) != 2*sha1.Size-prefixLength {
		return false
	}
	for _, character := range suffix {
		if !('0' <= character && character <= '9' || 'A' <= character && character <= 'F') {
			return false
		}
	}
	return true
}
