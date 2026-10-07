package api

import (
	"context"
	"crypto/sha1"
	"errors"

	"golang.org/x/sync/errgroup"

	"github.com/dortanes/ravenpass/packages/app/breaches"
)

// breachRequests bounds the range requests one check runs at once.
const breachRequests = 6

// breachCounter tells how often a password digest appears in known breaches and forgets what it kept.
type breachCounter interface {
	Count(ctx context.Context, digest [sha1.Size]byte) (int, error)
	Forget()
}

// BreachChecks reports whether passwords are checked against known breaches.
type BreachChecks struct {
	Enabled bool `json:"enabled"`
}

// Breach is a credential whose password appears in known breaches, and how many times.
type Breach struct {
	ID    string `json:"id"`
	Count int    `json:"count"`
}

// GetBreachChecks reports whether passwords are checked against known breaches.
func (s *Service) GetBreachChecks() (BreachChecks, error) {
	return BreachChecks{Enabled: s.preferences.BreachChecks()}, nil
}

// SetBreachChecks records whether passwords are checked against known breaches; turning checks off forgets every
// answer.
func (s *Service) SetBreachChecks(enabled bool) error {
	if err := s.preferences.SetBreachChecks(enabled); err != nil {
		return present(err)
	}
	if !enabled {
		s.breaches.Forget()
	}
	return nil
}

// BreachReport is what one check of the vault's passwords found.
type BreachReport struct {
	// Checked counts the passwords checked: every non-empty one outside the trash.
	Checked  int      `json:"checked"`
	Breaches []Breach `json:"breaches"`
}

// CheckBreaches checks every password of the vault against known breaches. It fails with breach-checks-off while
// checks are off and breach-check-unreachable when the service cannot be reached.
func (s *Service) CheckBreaches(ctx context.Context) (BreachReport, error) {
	if !s.preferences.BreachChecks() {
		return BreachReport{}, fail(failureBreachChecksOff)
	}
	digests, err := s.vault.PasswordDigests()
	if err != nil {
		return BreachReport{}, present(err)
	}
	report := BreachReport{Checked: len(digests), Breaches: make([]Breach, 0)}
	counts := make(chan Breach, len(digests))
	group, groupCtx := errgroup.WithContext(ctx)
	group.SetLimit(breachRequests)
	for id, digest := range digests {
		group.Go(func() error {
			count, err := s.breaches.Count(groupCtx, digest)
			if err == nil && count > 0 {
				counts <- Breach{ID: id.String(), Count: count}
			}
			return err
		})
	}
	err = group.Wait()
	close(counts)
	if err != nil {
		return BreachReport{}, presentBreach(err)
	}
	for breach := range counts {
		report.Breaches = append(report.Breaches, breach)
	}
	return report, nil
}

// CheckPassword reports how many times password appears in known breaches, zero when it is empty or not known. It
// fails as CheckBreaches does.
func (s *Service) CheckPassword(ctx context.Context, password string) (int, error) {
	if !s.preferences.BreachChecks() {
		return 0, fail(failureBreachChecksOff)
	}
	if password == "" {
		return 0, nil
	}
	count, err := s.breaches.Count(ctx, breaches.Digest(password))
	if err != nil {
		return 0, presentBreach(err)
	}
	return count, nil
}

// presentBreach names a failed exchange with the breach service and presents the rest as present does.
func presentBreach(err error) error {
	if errors.Is(err, breaches.ErrUnanswered) || errors.Is(err, breaches.ErrMalformedAnswer) {
		return fail(failureBreachCheckUnreachable)
	}
	return present(err)
}
