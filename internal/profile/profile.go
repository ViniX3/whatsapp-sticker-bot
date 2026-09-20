package profile

import (
	"database/sql"
	"fmt"

	"whatsapp-sticker-bot/internal/database"
)

// PlayerProfile concentra os dados exibidos pelo comando !perfil.
// O nível é sempre derivado do XP total, evitando inconsistência
// entre duas colunas diferentes no banco.
type PlayerProfile struct {
	GroupJID string
	JID      string
	Name     string
	Gold     int
	GoldRank int
	Players  int

	XP int

	QuizCorrect    int
	QuizWrong      int
	QuizInsaneWins int

	BetsPlayed int
	BetsWon    int

	RobberiesSuccess int
	RobberiesFailed  int

	SlotsPlayed       int
	SlotsWon          int
	SlotsBiggestPrize int

	DuelsWon  int
	DuelsLost int
}

// LevelInfo representa a progressão atual do jogador.
type LevelInfo struct {
	Level           int
	TotalXP         int
	CurrentLevelXP  int
	RequiredLevelXP int
	ProgressPercent int
}

// XPResult é retornado quando XP é concedido.
// Ele permite identificar facilmente um level up.
type XPResult struct {
	Before LevelInfo
	After  LevelInfo
	Gained int
}

func Init() error {
	if database.DB == nil {
		return fmt.Errorf("banco de dados não inicializado")
	}

	_, err := database.DB.Exec(`
		CREATE TABLE IF NOT EXISTS player_profiles (
			group_jid TEXT NOT NULL,
			jid TEXT NOT NULL,

			xp INTEGER NOT NULL DEFAULT 0
				CHECK (xp >= 0),

			quiz_correct INTEGER NOT NULL DEFAULT 0
				CHECK (quiz_correct >= 0),
			quiz_wrong INTEGER NOT NULL DEFAULT 0
				CHECK (quiz_wrong >= 0),
			quiz_insane_wins INTEGER NOT NULL DEFAULT 0
				CHECK (quiz_insane_wins >= 0),

			bets_played INTEGER NOT NULL DEFAULT 0
				CHECK (bets_played >= 0),
			bets_won INTEGER NOT NULL DEFAULT 0
				CHECK (bets_won >= 0),

			robberies_success INTEGER NOT NULL DEFAULT 0
				CHECK (robberies_success >= 0),
			robberies_failed INTEGER NOT NULL DEFAULT 0
				CHECK (robberies_failed >= 0),

			slots_played INTEGER NOT NULL DEFAULT 0
				CHECK (slots_played >= 0),
			slots_won INTEGER NOT NULL DEFAULT 0
				CHECK (slots_won >= 0),
			slots_biggest_prize INTEGER NOT NULL DEFAULT 0
				CHECK (slots_biggest_prize >= 0),

			duels_won INTEGER NOT NULL DEFAULT 0
				CHECK (duels_won >= 0),
			duels_lost INTEGER NOT NULL DEFAULT 0
				CHECK (duels_lost >= 0),

			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

			PRIMARY KEY (
				group_jid,
				jid
			),

			FOREIGN KEY (
				group_jid,
				jid
			)
				REFERENCES group_wallets(
					group_jid,
					jid
				)
				ON DELETE CASCADE
		);
	`)
	if err != nil {
		return fmt.Errorf(
			"erro ao criar tabela player_profiles: %w",
			err,
		)
	}

	_, err = database.DB.Exec(`
		CREATE INDEX IF NOT EXISTS idx_player_profiles_group_xp
		ON player_profiles (
			group_jid,
			xp DESC
		);
	`)
	if err != nil {
		return fmt.Errorf(
			"erro ao criar índice de XP: %w",
			err,
		)
	}

	if err := initCoinflipSchema(); err != nil {
		return err
	}

	return nil
}

// Ensure cria o registro de progressão caso ainda não exista.
// A carteira Gold do jogador precisa existir primeiro.
func Ensure(
	groupJID string,
	jid string,
) error {
	_, err := database.DB.Exec(`
		INSERT INTO player_profiles (
			group_jid,
			jid
		)
		VALUES (?, ?)
		ON CONFLICT(group_jid, jid) DO NOTHING
	`,
		groupJID,
		jid,
	)

	if err != nil {
		return fmt.Errorf(
			"erro ao garantir perfil do jogador: %w",
			err,
		)
	}

	return nil
}

// Get retorna o perfil completo do jogador.
func Get(
	groupJID string,
	jid string,
) (*PlayerProfile, error) {
	if err := Ensure(
		groupJID,
		jid,
	); err != nil {
		return nil, err
	}

	player := &PlayerProfile{
		GroupJID: groupJID,
		JID:      jid,
	}

	err := database.DB.QueryRow(`
		SELECT
			u.name,
			gw.gold,
			pp.xp,
			pp.quiz_correct,
			pp.quiz_wrong,
			pp.quiz_insane_wins,
			pp.bets_played,
			pp.bets_won,
			pp.robberies_success,
			pp.robberies_failed,
			pp.slots_played,
			pp.slots_won,
			pp.slots_biggest_prize,
			pp.duels_won,
			pp.duels_lost
		FROM player_profiles pp
		INNER JOIN users u
			ON u.jid = pp.jid
		INNER JOIN group_wallets gw
			ON gw.group_jid = pp.group_jid
			AND gw.jid = pp.jid
		WHERE pp.group_jid = ?
		  AND pp.jid = ?
	`,
		groupJID,
		jid,
	).Scan(
		&player.Name,
		&player.Gold,
		&player.XP,
		&player.QuizCorrect,
		&player.QuizWrong,
		&player.QuizInsaneWins,
		&player.BetsPlayed,
		&player.BetsWon,
		&player.RobberiesSuccess,
		&player.RobberiesFailed,
		&player.SlotsPlayed,
		&player.SlotsWon,
		&player.SlotsBiggestPrize,
		&player.DuelsWon,
		&player.DuelsLost,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf(
			"erro ao consultar perfil: %w",
			err,
		)
	}

	err = database.DB.QueryRow(`
		SELECT COUNT(*) + 1
		FROM group_wallets
		WHERE group_jid = ?
		  AND gold > ?
	`,
		groupJID,
		player.Gold,
	).Scan(
		&player.GoldRank,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"erro ao consultar posição Gold: %w",
			err,
		)
	}

	err = database.DB.QueryRow(`
		SELECT COUNT(*)
		FROM group_wallets
		WHERE group_jid = ?
	`,
		groupJID,
	).Scan(
		&player.Players,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"erro ao consultar quantidade de jogadores: %w",
			err,
		)
	}

	return player, nil
}

// AddXP adiciona experiência e retorna o estado
// anterior e posterior do jogador.
func AddXP(
	groupJID string,
	jid string,
	amount int,
) (*XPResult, error) {
	if amount <= 0 {
		return nil,
			fmt.Errorf(
				"quantidade de XP deve ser maior que zero",
			)
	}

	if err := Ensure(
		groupJID,
		jid,
	); err != nil {
		return nil, err
	}

	var beforeXP int

	err := database.DB.QueryRow(`
		SELECT xp
		FROM player_profiles
		WHERE group_jid = ?
		  AND jid = ?
	`,
		groupJID,
		jid,
	).Scan(
		&beforeXP,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"erro ao consultar XP atual: %w",
			err,
		)
	}

	_, err = database.DB.Exec(`
		UPDATE player_profiles
		SET
			xp = xp + ?,
			updated_at = CURRENT_TIMESTAMP
		WHERE group_jid = ?
		  AND jid = ?
	`,
		amount,
		groupJID,
		jid,
	)

	if err != nil {
		return nil, fmt.Errorf(
			"erro ao adicionar XP: %w",
			err,
		)
	}

	return &XPResult{
		Before: LevelFromXP(beforeXP),
		After:  LevelFromXP(beforeXP + amount),
		Gained: amount,
	}, nil
}

// LevelFromXP converte XP total em nível.
//
// Progressão:
//
// Nível 1 -> 2: 100 XP
// Nível 2 -> 3: 150 XP
// Nível 3 -> 4: 200 XP
// Nível 4 -> 5: 250 XP
//
// A cada nível, o próximo avanço exige 50 XP a mais.
func LevelFromXP(
	xp int,
) LevelInfo {
	if xp < 0 {
		xp = 0
	}

	level := 1

	for xp >= levelThreshold(level+1) {
		level++
	}

	currentThreshold :=
		levelThreshold(level)

	nextThreshold :=
		levelThreshold(level + 1)

	currentLevelXP :=
		xp - currentThreshold

	requiredLevelXP :=
		nextThreshold - currentThreshold

	progress := 0

	if requiredLevelXP > 0 {
		progress =
			currentLevelXP *
				100 /
				requiredLevelXP
	}

	if progress > 100 {
		progress = 100
	}

	return LevelInfo{
		Level:           level,
		TotalXP:         xp,
		CurrentLevelXP:  currentLevelXP,
		RequiredLevelXP: requiredLevelXP,
		ProgressPercent: progress,
	}
}

func levelThreshold(
	level int,
) int {
	if level <= 1 {
		return 0
	}

	return 25 *
		(level - 1) *
		(level + 2)
}
