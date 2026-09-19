package quiz

import (
	"crypto/rand"
	"embed"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
)

//go:embed questions/*.json
var questionsFS embed.FS

var questionBank map[Difficulty][]Question

var questionFiles = map[Difficulty]string{
	DifficultySuperEasy: "questions/super_easy.json",
	DifficultyEasy:      "questions/easy.json",
	DifficultyMedium:    "questions/medium.json",
	DifficultyHard:      "questions/hard.json",
	DifficultySuperHard: "questions/super_hard.json",
	DifficultyInsane:    "questions/insane.json",
}

func init() {
	bank, err := loadQuestionBank()
	if err != nil {
		panic(fmt.Sprintf(
			"erro carregando banco de perguntas: %v",
			err,
		))
	}

	questionBank = bank
}

func loadQuestionBank() (map[Difficulty][]Question, error) {
	bank := make(map[Difficulty][]Question)
	seenIDs := make(map[string]struct{})

	for difficulty, filename := range questionFiles {
		data, err := questionsFS.ReadFile(filename)
		if err != nil {
			return nil, fmt.Errorf(
				"erro lendo %s: %w",
				filename,
				err,
			)
		}

		var questions []Question

		if err := json.Unmarshal(data, &questions); err != nil {
			return nil, fmt.Errorf(
				"JSON inválido em %s: %w",
				filename,
				err,
			)
		}

		if len(questions) == 0 {
			return nil, fmt.Errorf(
				"arquivo %s não possui perguntas",
				filename,
			)
		}

		for index := range questions {
			if err := validateQuestion(
				&questions[index],
				difficulty,
				filename,
				index,
				seenIDs,
			); err != nil {
				return nil, err
			}
		}

		bank[difficulty] = questions
	}

	return bank, nil
}

func validateQuestion(
	question *Question,
	difficulty Difficulty,
	filename string,
	index int,
	seenIDs map[string]struct{},
) error {
	position := index + 1

	question.ID = strings.TrimSpace(question.ID)
	question.Category = strings.TrimSpace(question.Category)
	question.Text = strings.TrimSpace(question.Text)

	if question.ID == "" {
		return fmt.Errorf(
			"%s: pergunta %d sem ID",
			filename,
			position,
		)
	}

	if _, exists := seenIDs[question.ID]; exists {
		return fmt.Errorf(
			"%s: ID duplicado %q",
			filename,
			question.ID,
		)
	}

	seenIDs[question.ID] = struct{}{}

	if question.Category == "" {
		return fmt.Errorf(
			"%s: pergunta %s sem categoria",
			filename,
			question.ID,
		)
	}

	if question.Text == "" {
		return fmt.Errorf(
			"%s: pergunta %s sem enunciado",
			filename,
			question.ID,
		)
	}

	if question.CorrectAnswer < 0 || question.CorrectAnswer > 3 {
		return fmt.Errorf(
			"%s: pergunta %s possui resposta inválida %d",
			filename,
			question.ID,
			question.CorrectAnswer,
		)
	}

	seenOptions := make(map[string]struct{})

	for optionIndex := range question.Options {
		option := strings.TrimSpace(
			question.Options[optionIndex],
		)

		if option == "" {
			return fmt.Errorf(
				"%s: pergunta %s possui alternativa vazia",
				filename,
				question.ID,
			)
		}

		question.Options[optionIndex] = option

		normalized := strings.ToLower(option)

		if _, exists := seenOptions[normalized]; exists {
			return fmt.Errorf(
				"%s: pergunta %s possui alternativas duplicadas",
				filename,
				question.ID,
			)
		}

		seenOptions[normalized] = struct{}{}
	}

	if !validQuestionPrefix(
		question.ID,
		difficulty,
	) {
		return fmt.Errorf(
			"%s: ID %q não corresponde à dificuldade %s",
			filename,
			question.ID,
			difficulty,
		)
	}

	return nil
}

func validQuestionPrefix(
	id string,
	difficulty Difficulty,
) bool {
	var prefix string

	switch difficulty {
	case DifficultySuperEasy:
		prefix = "super_easy_"

	case DifficultyEasy:
		prefix = "easy_"

	case DifficultyMedium:
		prefix = "medium_"

	case DifficultyHard:
		prefix = "hard_"

	case DifficultySuperHard:
		prefix = "super_hard_"

	case DifficultyInsane:
		prefix = "insane_"

	default:
		return false
	}

	return strings.HasPrefix(
		id,
		prefix,
	)
}

func randomQuestion(
	difficulty Difficulty,
	excludedIDs []string,
) (Question, error) {
	questions, exists := questionBank[difficulty]

	if !exists || len(questions) == 0 {
		return Question{}, fmt.Errorf(
			"nenhuma pergunta disponível para dificuldade %s",
			difficulty,
		)
	}

	// IDs das últimas perguntas utilizadas naquele grupo.
	excluded := make(
		map[string]struct{},
		len(excludedIDs),
	)

	for _, id := range excludedIDs {
		excluded[id] = struct{}{}
	}

	// Perguntas disponíveis depois de remover as recentes.
	candidates := make(
		[]Question,
		0,
		len(questions),
	)

	for _, question := range questions {
		if _, blocked := excluded[question.ID]; blocked {
			continue
		}

		candidates = append(
			candidates,
			question,
		)
	}

	// Enquanto ainda temos apenas poucas perguntas por nível,
	// pode acontecer de todas estarem nas últimas 20.
	//
	// Nesse caso permitimos novamente todas daquela
	// dificuldade para que o Quiz continue funcionando.
	if len(candidates) == 0 {
		candidates = questions
	}

	n, err := rand.Int(
		rand.Reader,
		big.NewInt(
			int64(len(candidates)),
		),
	)

	if err != nil {
		return Question{}, fmt.Errorf(
			"erro ao sortear pergunta: %w",
			err,
		)
	}

	index := int(
		n.Int64(),
	)

	return candidates[index], nil
}

func QuestionCount(
	difficulty Difficulty,
) int {
	return len(
		questionBank[difficulty],
	)
}
