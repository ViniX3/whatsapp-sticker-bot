package quiz

import "sync"

const RecentQuestionLimit = 20

var recentQuestions = struct {
	sync.Mutex
	byGroup map[string][]string
}{
	byGroup: make(map[string][]string),
}

// recentQuestionIDs retorna uma cópia dos IDs recentes
// daquele grupo.
func recentQuestionIDs(
	groupJID string,
) []string {

	recentQuestions.Lock()
	defer recentQuestions.Unlock()

	current :=
		recentQuestions.byGroup[groupJID]

	result :=
		make(
			[]string,
			len(current),
		)

	copy(
		result,
		current,
	)

	return result
}

// rememberQuestion registra uma pergunta utilizada.
//
// Apenas as últimas RecentQuestionLimit perguntas
// permanecem no histórico.
func rememberQuestion(
	groupJID string,
	questionID string,
) {

	if questionID == "" {
		return
	}

	recentQuestions.Lock()
	defer recentQuestions.Unlock()

	current :=
		recentQuestions.byGroup[groupJID]

	current =
		append(
			current,
			questionID,
		)

	if len(current) >
		RecentQuestionLimit {

		current =
			current[len(current)-
				RecentQuestionLimit:]
	}

	recentQuestions.byGroup[groupJID] =
		current
}
