package handler

import (
	"errors"
	"fmt"

	"whatsapp-sticker-bot/internal/database"
	"whatsapp-sticker-bot/internal/rpg"
)

var (
	errRobberyRobberWalletMissing = errors.New(
		"ladrão não possui carteira Gold",
	)

	errRobberyTargetWalletMissing = errors.New(
		"alvo não possui carteira Gold",
	)
)

// robberyCombatContext contém os dados de combate
// calculados antes da tentativa de roubo.
type robberyCombatContext struct {
	RobberPower int
	TargetPower int

	SuccessChance int
}

// prepareRobberyCombat garante que os dois participantes
// estejam aptos a participar do sistema RPG.
//
// Regras:
//
//   - ambos precisam possuir carteira Gold no grupo;
//   - carteiras antigas sem RPG recebem o Set do Recruta;
//   - nenhum Gold inicial é concedido aqui;
//   - o Poder de Combate é calculado pelos equipamentos atuais;
//   - a chance final usa a fórmula central de combate RPG.
func prepareRobberyCombat(
	groupJID string,
	robberJID string,
	targetJID string,
) (*robberyCombatContext, error) {
	catalog, _, err :=
		getRPGCatalogs()

	if err != nil {
		return nil, fmt.Errorf(
			"erro carregando catálogo RPG: %w",
			err,
		)
	}

	if err := ensureRobberyRPGParticipant(
		groupJID,
		robberJID,
		catalog,
		errRobberyRobberWalletMissing,
	); err != nil {
		return nil, err
	}

	if err := ensureRobberyRPGParticipant(
		groupJID,
		targetJID,
		catalog,
		errRobberyTargetWalletMissing,
	); err != nil {
		return nil, err
	}

	robberSummary, err :=
		rpg.GetEquipmentSummary(
			groupJID,
			robberJID,
			catalog,
		)

	if err != nil {
		return nil, fmt.Errorf(
			"erro consultando equipamentos do ladrão: %w",
			err,
		)
	}

	targetSummary, err :=
		rpg.GetEquipmentSummary(
			groupJID,
			targetJID,
			catalog,
		)

	if err != nil {
		return nil, fmt.Errorf(
			"erro consultando equipamentos da vítima: %w",
			err,
		)
	}

	comparison :=
		rpg.CompareCombatPower(
			robberSummary.CombatPower,
			targetSummary.CombatPower,
		)

	return &robberyCombatContext{
		RobberPower: comparison.AttackerPower,

		TargetPower: comparison.DefenderPower,

		SuccessChance: comparison.SuccessChance,
	}, nil
}

// ensureRobberyRPGParticipant garante que um participante
// possui carteira e personagem RPG.
//
// IMPORTANTE:
//
// Esta função NÃO executa ClaimInitialGold.
//
// Dessa forma, mencionar uma pessoa em !roubar nunca cria
// uma carteira com 3000 Gold para ela.
//
// Se o jogador possuir uma carteira antiga, mas ainda não
// tiver ativado o RPG, ele recebe somente o Set do Recruta.
func ensureRobberyRPGParticipant(
	groupJID string,
	jid string,
	catalog *rpg.Catalog,
	walletMissingError error,
) error {
	wallet, err :=
		database.GetWallet(
			groupJID,
			jid,
		)

	if err != nil {
		return fmt.Errorf(
			"erro consultando carteira do participante: %w",
			err,
		)
	}

	if wallet == nil {
		return walletMissingError
	}

	_, err =
		rpg.ClaimStarterSet(
			groupJID,
			jid,
			catalog,
		)

	switch {
	case err == nil:
		// Carteira antiga sem RPG.
		// Starter criado e equipado agora.
		return nil

	case errors.Is(
		err,
		rpg.ErrStarterAlreadyClaimed,
	):
		// RPG já estava inicializado.
		return nil

	default:
		return fmt.Errorf(
			"erro inicializando participante RPG: %w",
			err,
		)
	}
}
