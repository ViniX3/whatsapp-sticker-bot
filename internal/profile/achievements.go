package profile

type Achievement struct {
	Icon        string
	Name        string
	Description string
	Progress    int
	Target      int
	Unlocked    bool
}

func EvaluateAchievements(
	player *PlayerProfile,
) []Achievement {
	if player == nil {
		return nil
	}

	level := LevelFromXP(player.XP).Level

	achievements := []Achievement{
		newAchievement(
			"🌱",
			"Primeiros Passos",
			"Alcance 100 XP.",
			player.XP,
			100,
		),
		newAchievement(
			"🧠",
			"Estudioso",
			"Acerte 10 perguntas no Quiz.",
			player.QuizCorrect,
			10,
		),
		newAchievement(
			"🎓",
			"Mestre do Quiz",
			"Acerte 50 perguntas no Quiz.",
			player.QuizCorrect,
			50,
		),
		newAchievement(
			"💀",
			"Sobreviveu ao Insano",
			"Acerte uma pergunta de dificuldade Insano.",
			player.QuizInsaneWins,
			1,
		),
		newAchievement(
			"🎲",
			"Apostador",
			"Complete 25 apostas no !bet.",
			player.BetsPlayed,
			25,
		),
		newAchievement(
			"🍀",
			"Pé Quente",
			"Vença 10 apostas no !bet.",
			player.BetsWon,
			10,
		),
		newAchievement(
			"🎰",
			"Frequentador do Cassino",
			"Jogue 25 partidas de Slots.",
			player.SlotsPlayed,
			25,
		),
		newAchievement(
			"💎",
			"Rei dos Slots",
			"Vença 10 partidas de Slots.",
			player.SlotsWon,
			10,
		),
		newAchievement(
			"🦹",
			"Mão Leve",
			"Realize 10 roubos com sucesso.",
			player.RobberiesSuccess,
			10,
		),
		newAchievement(
			"⚔️",
			"Gladiador",
			"Vença 10 duelos.",
			player.DuelsWon,
			10,
		),
		newAchievement(
			"⭐",
			"Veterano",
			"Alcance o nível 10.",
			level,
			10,
		),
		newAchievement(
			"👑",
			"Magnata",
			"Tenha 50.000 Gold no grupo.",
			player.Gold,
			50000,
		),
	}

	return achievements
}

func CountUnlockedAchievements(
	achievements []Achievement,
) int {
	unlocked := 0

	for _, achievement := range achievements {
		if achievement.Unlocked {
			unlocked++
		}
	}

	return unlocked
}

func newAchievement(
	icon string,
	name string,
	description string,
	progress int,
	target int,
) Achievement {
	if progress < 0 {
		progress = 0
	}

	if target < 1 {
		target = 1
	}

	return Achievement{
		Icon:        icon,
		Name:        name,
		Description: description,
		Progress:    progress,
		Target:      target,
		Unlocked:    progress >= target,
	}
}
