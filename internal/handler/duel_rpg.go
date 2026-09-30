package handler

import (
	"errors"
	"fmt"

	"whatsapp-sticker-bot/internal/database"
	"whatsapp-sticker-bot/internal/rpg"
)

var (
	errDuelChallengerWalletMissing = errors.New(
		"desafiante não possui carteira Gold",
	)

	errDuelTargetWalletMissing = errors.New(
		"desafiado não possui carteira Gold",
	)
)

type duelCombatContext struct {
	ChallengerPower int
	TargetPower     int

	ChallengerChance int
	TargetChance     int
}

// prepareDuelCombat calcula o combate usando os equipamentos
// existentes NO MOMENTO em que !aceitar é executado.
//
// Jogadores antigos com carteira Gold, mas sem RPG,
// recebem apenas o Set do Recruta.
//
// Nenhum Gold inicial é concedido aqui.
func prepareDuelCombat(
	groupJID string,
	challengerJID string,
	targetJID string,
) (*duelCombatContext, error) {
	catalog, _, err :=
		getRPGCatalogs()

	if err != nil {
		return nil, fmt.Errorf(
			"erro carregando catálogo RPG: %w",
			err,
		)
	}

	if err := ensureDuelRPGParticipant(
		groupJID,
		challengerJID,
		catalog,
		errDuelChallengerWalletMissing,
	); err != nil {
		return nil, err
	}

	if err := ensureDuelRPGParticipant(
		groupJID,
		targetJID,
		catalog,
		errDuelTargetWalletMissing,
	); err != nil {
		return nil, err
	}

	challengerSummary, err :=
		rpg.GetEquipmentSummary(
			groupJID,
			challengerJID,
			catalog,
		)

	if err != nil {
		return nil, fmt.Errorf(
			"erro consultando equipamentos do desafiante: %w",
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
			"erro consultando equipamentos do desafiado: %w",
			err,
		)
	}

	comparison :=
		rpg.CompareCombatPower(
			challengerSummary.CombatPower,
			targetSummary.CombatPower,
		)

	return &duelCombatContext{
		ChallengerPower: comparison.AttackerPower,

		TargetPower: comparison.DefenderPower,

		ChallengerChance: comparison.SuccessChance,

		TargetChance: 100 - comparison.SuccessChance,
	}, nil
}

func ensureDuelRPGParticipant(
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
		return nil

	case errors.Is(
		err,
		rpg.ErrStarterAlreadyClaimed,
	):
		return nil

	default:
		return fmt.Errorf(
			"erro inicializando participante RPG: %w",
			err,
		)
	}
}
