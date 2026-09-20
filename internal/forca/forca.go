package forca

import (
	"crypto/rand"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"
	"unicode"
)

const (
	MaxErrors      = 6
	SessionTimeout = 10 * time.Minute
)

var (
	ErrSessionExists = errors.New(
		"já existe uma partida de forca neste grupo",
	)

	ErrSessionMissing = errors.New(
		"não existe partida de forca neste grupo",
	)

	ErrSessionExpired = errors.New(
		"a partida de forca expirou",
	)

	ErrInvalidLetter = errors.New(
		"letra inválida",
	)

	ErrInvalidWord = errors.New(
		"palavra inválida",
	)
)

type Word struct {
	Word     string `json:"word"`
	Category string `json:"category"`
}

type Session struct {
	GroupJID string

	Word       string
	Normalized string
	Category   string

	Guessed map[rune]bool
	Wrong   []rune

	Errors int

	StartedAt time.Time
	ExpiresAt time.Time
}

type GuessResult struct {
	Session *Session

	Correct  bool
	Repeated bool

	Won  bool
	Lost bool
}

//go:embed words.json
var wordsJSON []byte

var wordBank []Word

var sessions = struct {
	sync.Mutex
	byGroup map[string]*Session
}{
	byGroup: make(
		map[string]*Session,
	),
}

func init() {
	if err :=
		json.Unmarshal(
			wordsJSON,
			&wordBank,
		); err != nil {

		panic(
			fmt.Sprintf(
				"erro carregando palavras do Forca: %v",
				err,
			),
		)
	}

	if len(wordBank) == 0 {
		panic(
			"banco de palavras do Forca está vazio",
		)
	}
}

func Start(
	groupJID string,
) (*Session, error) {
	sessions.Lock()
	defer sessions.Unlock()

	if current, exists :=
		sessions.byGroup[groupJID]; exists {

		if time.Now().Before(
			current.ExpiresAt,
		) {
			return cloneSession(current),
				ErrSessionExists
		}

		delete(
			sessions.byGroup,
			groupJID,
		)
	}

	index, err :=
		randomIndex(
			len(wordBank),
		)

	if err != nil {
		return nil, err
	}

	selected :=
		wordBank[index]

	now := time.Now()

	session := &Session{
		GroupJID: groupJID,

		Word: selected.Word,

		Normalized: normalizeText(
			selected.Word,
		),

		Category: selected.Category,

		Guessed: make(
			map[rune]bool,
		),

		Wrong: make(
			[]rune,
			0,
		),

		Errors: 0,

		StartedAt: now,

		ExpiresAt: now.Add(
			SessionTimeout,
		),
	}

	sessions.byGroup[groupJID] =
		session

	return cloneSession(session), nil
}

func Status(
	groupJID string,
) (*Session, error) {
	sessions.Lock()
	defer sessions.Unlock()

	session, exists :=
		sessions.byGroup[groupJID]

	if !exists {
		return nil, ErrSessionMissing
	}

	if !time.Now().Before(
		session.ExpiresAt,
	) {
		delete(
			sessions.byGroup,
			groupJID,
		)

		return nil, ErrSessionExpired
	}

	return cloneSession(session), nil
}

func GuessLetter(
	groupJID string,
	input string,
) (*GuessResult, error) {
	normalized :=
		normalizeText(input)

	runes :=
		[]rune(normalized)

	if len(runes) != 1 ||
		!unicode.IsLetter(runes[0]) {

		return nil, ErrInvalidLetter
	}

	letter :=
		runes[0]

	sessions.Lock()
	defer sessions.Unlock()

	session, err :=
		activeSession(
			groupJID,
		)

	if err != nil {
		return nil, err
	}

	if session.Guessed[letter] ||
		containsRune(
			session.Wrong,
			letter,
		) {

		return &GuessResult{
			Session: cloneSession(session),

			Repeated: true,
		}, nil
	}

	if strings.ContainsRune(
		session.Normalized,
		letter,
	) {
		session.Guessed[letter] =
			true

		won :=
			isSolved(session)

		result := &GuessResult{
			Session: cloneSession(session),

			Correct: true,

			Won: won,
		}

		if won {
			delete(
				sessions.byGroup,
				groupJID,
			)
		}

		return result, nil
	}

	session.Wrong =
		append(
			session.Wrong,
			letter,
		)

	session.Errors++

	lost :=
		session.Errors >= MaxErrors

	result := &GuessResult{
		Session: cloneSession(session),

		Correct: false,

		Lost: lost,
	}

	if lost {
		delete(
			sessions.byGroup,
			groupJID,
		)
	}

	return result, nil
}

func GuessWord(
	groupJID string,
	input string,
) (*GuessResult, error) {
	guess :=
		normalizeText(input)

	if guess == "" {
		return nil, ErrInvalidWord
	}

	sessions.Lock()
	defer sessions.Unlock()

	session, err :=
		activeSession(
			groupJID,
		)

	if err != nil {
		return nil, err
	}

	if guess == session.Normalized {
		result := &GuessResult{
			Session: cloneSession(session),

			Correct: true,

			Won: true,
		}

		delete(
			sessions.byGroup,
			groupJID,
		)

		return result, nil
	}

	session.Errors++

	lost :=
		session.Errors >= MaxErrors

	result := &GuessResult{
		Session: cloneSession(session),

		Correct: false,

		Lost: lost,
	}

	if lost {
		delete(
			sessions.byGroup,
			groupJID,
		)
	}

	return result, nil
}

func Mask(
	session *Session,
) string {
	if session == nil {
		return ""
	}

	var parts []string

	for _, original := range []rune(session.Word) {

		if unicode.IsLetter(
			original,
		) {
			normalized :=
				normalizeRune(
					original,
				)

			if session.Guessed[normalized] {
				parts =
					append(
						parts,
						string(original),
					)
			} else {
				parts =
					append(
						parts,
						"_",
					)
			}

			continue
		}

		if unicode.IsSpace(
			original,
		) {
			parts =
				append(
					parts,
					"   ",
				)

			continue
		}

		parts =
			append(
				parts,
				string(original),
			)
	}

	return strings.Join(
		parts,
		" ",
	)
}

func RemainingAttempts(
	session *Session,
) int {
	if session == nil {
		return 0
	}

	remaining :=
		MaxErrors -
			session.Errors

	if remaining < 0 {
		return 0
	}

	return remaining
}

func activeSession(
	groupJID string,
) (*Session, error) {
	session, exists :=
		sessions.byGroup[groupJID]

	if !exists {
		return nil,
			ErrSessionMissing
	}

	if !time.Now().Before(
		session.ExpiresAt,
	) {
		delete(
			sessions.byGroup,
			groupJID,
		)

		return nil,
			ErrSessionExpired
	}

	return session, nil
}

func isSolved(
	session *Session,
) bool {
	for _, r := range []rune(
		session.Normalized,
	) {

		if !unicode.IsLetter(r) {
			continue
		}

		if !session.Guessed[r] {
			return false
		}
	}

	return true
}

func containsRune(
	list []rune,
	target rune,
) bool {
	for _, value := range list {

		if value == target {
			return true
		}
	}

	return false
}

func randomIndex(
	length int,
) (int, error) {
	if length <= 0 {
		return 0,
			fmt.Errorf(
				"banco de palavras vazio",
			)
	}

	value, err :=
		rand.Int(
			rand.Reader,
			big.NewInt(
				int64(length),
			),
		)

	if err != nil {
		return 0,
			fmt.Errorf(
				"erro sorteando palavra: %w",
				err,
			)
	}

	return int(
		value.Int64(),
	), nil
}

func cloneSession(
	session *Session,
) *Session {
	if session == nil {
		return nil
	}

	copySession :=
		*session

	copySession.Guessed =
		make(
			map[rune]bool,
			len(session.Guessed),
		)

	for letter, guessed := range session.Guessed {

		copySession.Guessed[letter] =
			guessed
	}

	copySession.Wrong =
		append(
			[]rune(nil),
			session.Wrong...,
		)

	return &copySession
}

func normalizeText(
	value string,
) string {
	fields :=
		strings.Fields(
			strings.ToLower(
				strings.TrimSpace(
					value,
				),
			),
		)

	value =
		strings.Join(
			fields,
			" ",
		)

	var builder strings.Builder

	for _, r := range value {

		builder.WriteRune(
			normalizeRune(r),
		)
	}

	return builder.String()
}

func normalizeRune(
	r rune,
) rune {
	r =
		unicode.ToLower(r)

	switch r {

	case 'á', 'à', 'ã', 'â', 'ä':
		return 'a'

	case 'é', 'è', 'ê', 'ë':
		return 'e'

	case 'í', 'ì', 'î', 'ï':
		return 'i'

	case 'ó', 'ò', 'õ', 'ô', 'ö':
		return 'o'

	case 'ú', 'ù', 'û', 'ü':
		return 'u'

	case 'ç':
		return 'c'
	}

	return r
}
