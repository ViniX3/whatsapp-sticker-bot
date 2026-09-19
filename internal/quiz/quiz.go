package quiz

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"
)

// ==========================================================
// CONFIGURAÇÕES
// ==========================================================

const (
	NormalTimeLimit = 25 * time.Second
	InsaneTimeLimit = 15 * time.Second
)

// ==========================================================
// DIFICULDADES
// ==========================================================

type Difficulty string

const (
	DifficultySuperEasy Difficulty = "SUPER FÁCIL"
	DifficultyEasy      Difficulty = "FÁCIL"
	DifficultyMedium    Difficulty = "MÉDIO"
	DifficultyHard      Difficulty = "DIFÍCIL"
	DifficultySuperHard Difficulty = "SUPER DIFÍCIL"
	DifficultyInsane    Difficulty = "INSANO"
)

// ==========================================================
// ERROS
// ==========================================================

var (
	ErrQuizActive = errors.New(
		"já existe um quiz ativo neste grupo",
	)

	ErrNoActiveQuiz = errors.New(
		"não existe quiz ativo neste grupo",
	)

	ErrNotQuizOwner = errors.New(
		"somente quem iniciou o quiz pode responder",
	)

	ErrInvalidAnswer = errors.New(
		"resposta inválida",
	)

	ErrQuizExpired = errors.New(
		"quiz expirado",
	)
)

// ==========================================================
// PERGUNTA
// ==========================================================

type Question struct {
	ID       string `json:"id"`
	Category string `json:"category"`

	Text string `json:"question"`

	Options [4]string `json:"options"`

	// 0 = A
	// 1 = B
	// 2 = C
	// 3 = D
	CorrectAnswer int `json:"correct"`
}

// ==========================================================
// SESSÃO
// ==========================================================

type Session struct {
	ID uint64

	GroupJID string
	OwnerJID string

	Difficulty Difficulty

	Question Question

	Reward int

	StartedAt time.Time
	ExpiresAt time.Time
}

// ==========================================================
// RESULTADO DA RESPOSTA
// ==========================================================

type AnswerResult struct {
	Correct bool

	SelectedAnswer string
	CorrectAnswer  string

	Reward int

	Difficulty Difficulty

	Question Question
}

// ==========================================================
// ESTADO
// ==========================================================
//
// Existe no máximo UM quiz ativo por grupo.
//
// Não existe cooldown entre quizzes.
//
// Assim que:
//   - alguém responder;
//   - o tempo acabar;
//   - o quiz for cancelado;
//
// outro quiz poderá começar imediatamente.
//
// ==========================================================

var quizState = struct {
	sync.Mutex

	active map[string]*Session

	nextID uint64
}{
	active: make(
		map[string]*Session,
	),
}

// ==========================================================
// INICIAR QUIZ
// ==========================================================

func Start(
	groupJID string,
	ownerJID string,
) (*Session, error) {

	quizState.Lock()
	defer quizState.Unlock()

	now :=
		time.Now()

	// ======================================================
	// VERIFICAR QUIZ ATIVO
	// ======================================================

	if current, exists :=
		quizState.active[groupJID]; exists {

		// Caso o quiz já tenha expirado mas o timer ainda
		// não tenha removido a sessão, removemos aqui.
		if !current.ExpiresAt.After(now) {
			delete(
				quizState.active,
				groupJID,
			)
		} else {
			return nil, ErrQuizActive
		}
	}

	// ======================================================
	// DIFICULDADE
	// ======================================================

	difficulty, err :=
		drawDifficulty()

	if err != nil {
		return nil, err
	}

	// ======================================================
	// PERGUNTA
	// ======================================================

	recentIDs :=
		recentQuestionIDs(
			groupJID,
		)

	question, err :=
		randomQuestion(
			difficulty,
			recentIDs,
		)

	// ======================================================
	// PRÊMIO
	// ======================================================

	minReward,
		maxReward :=
		rewardRange(
			difficulty,
		)

	reward, err :=
		randomInt(
			minReward,
			maxReward,
		)

	if err != nil {
		return nil, err
	}

	// ======================================================
	// TEMPO
	// ======================================================

	timeLimit :=
		NormalTimeLimit

	if difficulty ==
		DifficultyInsane {

		timeLimit =
			InsaneTimeLimit
	}

	// ======================================================
	// ID
	// ======================================================

	quizState.nextID++

	session := &Session{
		ID: quizState.nextID,

		GroupJID: groupJID,
		OwnerJID: ownerJID,

		Difficulty: difficulty,

		Question: question,

		Reward: reward,

		StartedAt: now,

		ExpiresAt: now.Add(
			timeLimit,
		),
	}

	quizState.active[groupJID] =
		session

	rememberQuestion(
		groupJID,
		question.ID,
	)

	sessionCopy :=
		*session

	return &sessionCopy, nil
}

// ==========================================================
// CONSULTAR QUIZ ATIVO
// ==========================================================

func GetActive(
	groupJID string,
) (*Session, bool) {

	quizState.Lock()
	defer quizState.Unlock()

	session, exists :=
		quizState.active[groupJID]

	if !exists {
		return nil, false
	}

	now :=
		time.Now()

	if !session.ExpiresAt.After(now) {
		delete(
			quizState.active,
			groupJID,
		)

		return nil, false
	}

	copySession :=
		*session

	return &copySession, true
}

// ==========================================================
// RESPONDER QUIZ
// ==========================================================
//
// Apenas quem iniciou o quiz pode responder.
//
// Respostas aceitas:
//
// A
// B
// C
// D
//
// Se acertar:
//   - quiz termina;
//   - prêmio poderá ser creditado.
//
// Se errar:
//   - quiz também termina.
//
// Se enviar algo inválido:
//   - quiz continua ativo.
//
// ==========================================================

func Answer(
	groupJID string,
	userJID string,
	answer string,
) (*AnswerResult, error) {

	quizState.Lock()
	defer quizState.Unlock()

	session, exists :=
		quizState.active[groupJID]

	if !exists {
		return nil, ErrNoActiveQuiz
	}

	now :=
		time.Now()

	// ======================================================
	// TEMPO ESGOTADO
	// ======================================================

	if !session.ExpiresAt.After(now) {
		delete(
			quizState.active,
			groupJID,
		)

		return nil, ErrQuizExpired
	}

	// ======================================================
	// VERIFICAR DONO
	// ======================================================

	if session.OwnerJID != userJID {
		return nil, ErrNotQuizOwner
	}

	normalized :=
		normalizeAnswer(
			answer,
		)

	if normalized == "" {
		return nil, ErrInvalidAnswer
	}

	selectedIndex :=
		answerIndex(
			normalized,
		)

	if selectedIndex < 0 {
		return nil, ErrInvalidAnswer
	}

	correct :=
		selectedIndex ==
			session.Question.CorrectAnswer

	correctAnswer :=
		answerLetter(
			session.Question.CorrectAnswer,
		)

	// Uma resposta válida encerra a rodada,
	// independentemente de estar certa ou errada.
	delete(
		quizState.active,
		groupJID,
	)

	return &AnswerResult{
		Correct: correct,

		SelectedAnswer: normalized,

		CorrectAnswer: correctAnswer,

		Reward: session.Reward,

		Difficulty: session.Difficulty,

		Question: session.Question,
	}, nil
}

// ==========================================================
// EXPIRAÇÃO AUTOMÁTICA
// ==========================================================
//
// O sessionID impede um timer antigo de remover
// acidentalmente um quiz novo.
//
// ==========================================================

func Expire(
	groupJID string,
	sessionID uint64,
) bool {

	quizState.Lock()
	defer quizState.Unlock()

	session, exists :=
		quizState.active[groupJID]

	if !exists {
		return false
	}

	if session.ID !=
		sessionID {

		return false
	}

	now :=
		time.Now()

	if session.ExpiresAt.After(now) {
		return false
	}

	delete(
		quizState.active,
		groupJID,
	)

	return true
}

// ==========================================================
// CANCELAR
// ==========================================================

func Cancel(
	groupJID string,
) bool {

	quizState.Lock()
	defer quizState.Unlock()

	_, exists :=
		quizState.active[groupJID]

	if !exists {
		return false
	}

	delete(
		quizState.active,
		groupJID,
	)

	return true
}

// ==========================================================
// SORTEAR DIFICULDADE
// ==========================================================
//
// Escala: 0–9999
//
// SUPER FÁCIL
// 30%
// 0–2999
//
// FÁCIL
// 30%
// 3000–5999
//
// MÉDIO
// 20%
// 6000–7999
//
// DIFÍCIL
// 12%
// 8000–9199
//
// SUPER DIFÍCIL
// 7,5%
// 9200–9949
//
// INSANO
// 0,5%
// 9950–9999
//
// ==========================================================

func drawDifficulty() (
	Difficulty,
	error,
) {

	n, err := rand.Int(
		rand.Reader,
		big.NewInt(10000),
	)

	if err != nil {
		return "",
			fmt.Errorf(
				"erro ao sortear dificuldade do quiz: %w",
				err,
			)
	}

	draw :=
		n.Int64()

	switch {
	case draw < 3000:
		return DifficultySuperEasy, nil

	case draw < 6000:
		return DifficultyEasy, nil

	case draw < 8000:
		return DifficultyMedium, nil

	case draw < 9200:
		return DifficultyHard, nil

	case draw < 9950:
		return DifficultySuperHard, nil

	default:
		return DifficultyInsane, nil
	}
}

// ==========================================================
// PRÊMIOS
// ==========================================================

func rewardRange(
	difficulty Difficulty,
) (int, int) {

	switch difficulty {
	case DifficultySuperEasy:
		return 500, 900

	case DifficultyEasy:
		return 1000, 1800

	case DifficultyMedium:
		return 2200, 3500

	case DifficultyHard:
		return 4500, 7000

	case DifficultySuperHard:
		return 9000, 15000

	case DifficultyInsane:
		return 25000, 50000

	default:
		return 500, 900
	}
}

// ==========================================================
// RANDOM
// ==========================================================

func randomInt(
	min int,
	max int,
) (int, error) {

	if min > max {
		return 0,
			fmt.Errorf(
				"faixa inválida: %d-%d",
				min,
				max,
			)
	}

	rangeSize :=
		int64(
			max - min + 1,
		)

	n, err := rand.Int(
		rand.Reader,
		big.NewInt(
			rangeSize,
		),
	)

	if err != nil {
		return 0,
			fmt.Errorf(
				"erro ao realizar sorteio: %w",
				err,
			)
	}

	return min +
			int(
				n.Int64(),
			),
		nil
}

// ==========================================================
// RESPOSTAS
// ==========================================================

func normalizeAnswer(
	answer string,
) string {

	answer =
		strings.TrimSpace(
			strings.ToUpper(
				answer,
			),
		)

	switch answer {
	case "A", "B", "C", "D":
		return answer

	default:
		return ""
	}
}

func answerIndex(
	answer string,
) int {

	switch answer {
	case "A":
		return 0

	case "B":
		return 1

	case "C":
		return 2

	case "D":
		return 3

	default:
		return -1
	}
}

func answerLetter(
	index int,
) string {

	switch index {
	case 0:
		return "A"

	case 1:
		return "B"

	case 2:
		return "C"

	case 3:
		return "D"

	default:
		return "?"
	}
}
