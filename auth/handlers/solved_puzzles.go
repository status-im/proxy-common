package handlers

import (
	"sync"
	"time"
)

// solvedPuzzlesSweepInterval is how often expired puzzles are forgotten, and
// how long past its expiry a puzzle is still remembered.
const solvedPuzzlesSweepInterval = time.Minute

// solvedPuzzles remembers the puzzles a token was already issued for, so that
// one solution cannot be exchanged for a second token.
//
// A puzzle is forgotten after it expires: an expired puzzle is rejected before
// it is looked up here, so there is nothing left to protect.
type solvedPuzzles struct {
	mu        sync.Mutex
	expiresAt map[string]time.Time
	nextSweep time.Time
	now       func() time.Time
}

func newSolvedPuzzles() *solvedPuzzles {
	return &solvedPuzzles{
		expiresAt: make(map[string]time.Time),
		now:       time.Now,
	}
}

// claim records that a token is issued for the puzzle with the given challenge
// and reports whether it is the first one. Of several simultaneous submissions
// of one solution exactly one gets true.
func (s *solvedPuzzles) claim(challenge string, expiresAt time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.now()
	if !now.Before(s.nextSweep) {
		s.forgetExpired(now)
		s.nextSweep = now.Add(solvedPuzzlesSweepInterval)
	}

	if _, solved := s.expiresAt[challenge]; solved {
		return false
	}
	s.expiresAt[challenge] = expiresAt
	return true
}

// forgetExpired drops the puzzles that expired more than a sweep interval ago.
// The margin keeps a puzzle remembered for as long as a request that was let
// through just before its expiry may still be on its way here.
func (s *solvedPuzzles) forgetExpired(now time.Time) {
	for challenge, expiresAt := range s.expiresAt {
		if now.Sub(expiresAt) > solvedPuzzlesSweepInterval {
			delete(s.expiresAt, challenge)
		}
	}
}
