package handler

import (
	"testing"

	"whatsapp-sticker-bot/internal/rpg"
)

func TestParseRPGPVEBatchAmount(t *testing.T) {
	region, battles, isBatch, valid :=
		parseRPGPVEBatchArguments(
			[]string{"!pve", "100"},
		)

	if !isBatch || !valid {
		t.Fatalf("!pve 100 deveria ser lote válido")
	}

	if region != "" {
		t.Fatalf("região deveria ser aleatória, recebida %s", region)
	}

	if battles != 100 {
		t.Fatalf("esperadas 100 batalhas, recebidas %d", battles)
	}
}

func TestParseRPGPVEBatchRegion(t *testing.T) {
	region, battles, isBatch, valid :=
		parseRPGPVEBatchArguments(
			[]string{"!pve", "mina", "500"},
		)

	if !isBatch || !valid {
		t.Fatalf("!pve mina 500 deveria ser lote válido")
	}

	if region != rpg.GatheringMine {
		t.Fatalf("esperada região MINA, recebida %s", region)
	}

	if battles != 500 {
		t.Fatalf("esperadas 500 batalhas, recebidas %d", battles)
	}
}

func TestParseRPGPVEBatchKeepsNormalRegionCommand(t *testing.T) {
	_, _, isBatch, _ :=
		parseRPGPVEBatchArguments(
			[]string{"!pve", "mina"},
		)

	if isBatch {
		t.Fatalf("!pve mina deve continuar sendo PvE comum")
	}
}

func TestParseRPGPVEBatchRejectsAboveLimit(t *testing.T) {
	_, battles, isBatch, valid :=
		parseRPGPVEBatchArguments(
			[]string{"!pve", "1001"},
		)

	if !isBatch {
		t.Fatalf("quantidade numérica deve ser reconhecida como lote")
	}

	if valid {
		t.Fatalf("1001 batalhas deveria ser inválido")
	}

	if battles != rpg.PVEBatchMaxBattles+1 {
		t.Fatalf("quantidade inesperada: %d", battles)
	}
}
