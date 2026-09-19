package duel

import (
	"errors"
	"sync"
	"time"
)

const ChallengeTimeout = 60 * time.Second

var (
	ErrChallengeExists  = errors.New("já existe um duelo pendente neste grupo")
	ErrChallengeMissing = errors.New("não existe duelo pendente neste grupo")
	ErrChallengeExpired = errors.New("o desafio de duelo expirou")
	ErrNotTarget        = errors.New("somente o jogador desafiado pode responder")
)

type Challenge struct {
	GroupJID       string
	ChallengerJID  string
	ChallengerName string
	TargetJID      string
	TargetName     string
	Amount         int
	CreatedAt      time.Time
	ExpiresAt      time.Time
}

var challenges = struct {
	sync.Mutex
	byGroup map[string]Challenge
}{
	byGroup: make(map[string]Challenge),
}

func Create(challenge Challenge) error {
	challenges.Lock()
	defer challenges.Unlock()

	now := time.Now()

	if current, exists := challenges.byGroup[challenge.GroupJID]; exists {
		if now.Before(current.ExpiresAt) {
			return ErrChallengeExists
		}

		delete(challenges.byGroup, challenge.GroupJID)
	}

	challenge.CreatedAt = now
	challenge.ExpiresAt = now.Add(ChallengeTimeout)

	challenges.byGroup[challenge.GroupJID] = challenge

	return nil
}

func Accept(
	groupJID string,
	targetJID string,
) (*Challenge, error) {
	return take(
		groupJID,
		targetJID,
	)
}

func Reject(
	groupJID string,
	targetJID string,
) (*Challenge, error) {
	return take(
		groupJID,
		targetJID,
	)
}

func take(
	groupJID string,
	targetJID string,
) (*Challenge, error) {
	challenges.Lock()
	defer challenges.Unlock()

	challenge, exists :=
		challenges.byGroup[groupJID]

	if !exists {
		return nil, ErrChallengeMissing
	}

	if !time.Now().Before(
		challenge.ExpiresAt,
	) {
		delete(
			challenges.byGroup,
			groupJID,
		)

		return nil, ErrChallengeExpired
	}

	if challenge.TargetJID != targetJID {
		return nil, ErrNotTarget
	}

	delete(
		challenges.byGroup,
		groupJID,
	)

	copy := challenge

	return &copy, nil
}
