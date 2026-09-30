package handler

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"whatsapp-sticker-bot/internal/logger"
	"whatsapp-sticker-bot/internal/rpg"
	"whatsapp-sticker-bot/internal/whatsapp"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

func handleVaultCommand(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	text string,
) bool {
	parts :=
		strings.Fields(text)

	if len(parts) == 0 ||
		!strings.EqualFold(
			parts[0],
			"!cofre",
		) {

		return false
	}

	if !requireGoldGroup(
		client,
		msgEvent,
		"!cofre",
	) {
		return true
	}

	groupJID :=
		msgEvent.Info.Chat.String()

	jid :=
		canonicalSenderJID(
			msgEvent,
		)

	// Garante o RPG sem conceder Gold extra.
	catalog,
		_,
		err :=
		getRPGCatalogs()

	if err != nil {
		sendRPGInternalError(
			client,
			msgEvent,
			"carregando RPG para cofre",
			err,
		)

		return true
	}

	_, err =
		rpg.ClaimStarterSet(
			groupJID,
			jid,
			catalog,
		)

	switch {
	case err == nil:

	case errors.Is(
		err,
		rpg.ErrStarterAlreadyClaimed,
	):

	case errors.Is(
		err,
		rpg.ErrWalletNotFound,
	):
		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			"❌ Você ainda não possui uma carteira neste grupo.\n\n"+
				"Use *!gold* ou *!rpg* antes de acessar o cofre.",
		)

		return true

	default:
		sendRPGInternalError(
			client,
			msgEvent,
			"preparando cofre",
			err,
		)

		return true
	}

	// ======================================================
	// !cofre
	// ======================================================

	if len(parts) == 1 {
		snapshot, err :=
			rpg.GetVaultSnapshot(
				groupJID,
				jid,
			)

		if err != nil {
			sendVaultError(
				client,
				msgEvent,
				err,
			)

			return true
		}

		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			renderVault(
				snapshot,
			),
		)

		return true
	}

	// ======================================================
	// !cofre tipos
	// ======================================================

	if len(parts) == 2 {

		action,
			ok :=
			canonicalAction(
				parts[1],
			)

		if ok &&
			action == actionTypes {

			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				renderVaultTiers(),
			)

			return true
		}
	}

	// ======================================================
	// guardar / retirar
	// ======================================================

	if len(parts) < 4 {
		sendVaultUsage(
			client,
			msgEvent,
		)

		return true
	}

	action,
		actionOK :=
		canonicalAction(
			parts[1],
		)

	if !actionOK ||
		action != actionDeposit &&
			action != actionWithdraw {

		sendVaultUsage(
			client,
			msgEvent,
		)

		return true
	}

	resourceText :=
		strings.Join(
			parts[2:len(parts)-1],
			" ",
		)

	resource,
		ok :=
		canonicalVaultResource(
			resourceText,
		)

	if !ok {
		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			"❌ Recurso inválido.\n\n"+
				"Use: *gold*, *cristais* ou *pedras*.",
		)

		return true
	}

	amount, err :=
		strconv.Atoi(
			parts[len(parts)-1],
		)

	if err != nil ||
		amount <= 0 {

		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			"❌ A quantidade precisa ser um número inteiro maior que zero.",
		)

		return true
	}

	var result *rpg.VaultTransferResult

	switch action {
	case actionDeposit:

		result, err =
			rpg.DepositVault(
				groupJID,
				jid,
				resource,
				amount,
			)

	case actionWithdraw:

		result, err =
			rpg.WithdrawVault(
				groupJID,
				jid,
				resource,
				amount,
			)

	default:
		sendVaultUsage(
			client,
			msgEvent,
		)

		return true
	}

	if err != nil {
		sendVaultError(
			client,
			msgEvent,
			err,
		)

		return true
	}

	_ = whatsapp.SendText(
		client,
		msgEvent.Info.Chat,
		renderVaultTransfer(
			result,
		),
	)

	return true
}

func renderVault(
	snapshot *rpg.VaultSnapshot,
) string {
	capacity :=
		vaultCapacityText(
			snapshot.Tier,
		)

	return fmt.Sprintf(
		"🏦 *%s*\n"+
			"💰 %s / %s\n"+
			"💎 %s / %s\n"+
			"🌠 %s / %s\n\n"+
			"👝 Fora: *%s Gold*",
		snapshot.Tier.Name,
		formatVaultNumber(
			snapshot.Vault.Gold,
		),
		capacity,
		formatVaultNumber(
			snapshot.Vault.MagicCrystals,
		),
		capacity,
		formatVaultNumber(
			snapshot.Vault.StellarStones,
		),
		capacity,
		formatGoldAmount(
			snapshot.WalletGold,
		),
	)
}

func renderVaultTransfer(
	result *rpg.VaultTransferResult,
) string {
	stored :=
		vaultStoredFromSnapshot(
			&result.Snapshot,
			result.Resource,
		)

	outside :=
		vaultOutsideFromSnapshot(
			&result.Snapshot,
			result.Resource,
		)

	action := "🔒 Guardado"

	if result.Action != "DEPOSIT" {
		action = "🔓 Retirado"
	}

	return fmt.Sprintf(
		"%s\n%s *%s*\n🏦 %s\n👝 %s",
		action,
		result.Resource.Icon(),
		formatVaultNumber(
			result.Amount,
		),
		formatVaultNumber(
			stored,
		),
		formatVaultNumber(
			outside,
		),
	)
}

func vaultStoredFromSnapshot(
	snapshot *rpg.VaultSnapshot,
	resource rpg.VaultResource,
) int {
	switch resource {
	case rpg.VaultResourceGold:
		return snapshot.Vault.Gold

	case rpg.VaultResourceMagicCrystals:
		return snapshot.Vault.
			MagicCrystals

	case rpg.VaultResourceStellarStones:
		return snapshot.Vault.
			StellarStones

	default:
		return 0
	}
}

func vaultOutsideFromSnapshot(
	snapshot *rpg.VaultSnapshot,
	resource rpg.VaultResource,
) int {
	switch resource {
	case rpg.VaultResourceGold:
		return snapshot.WalletGold

	case rpg.VaultResourceMagicCrystals:
		return snapshot.
			WalletMagicCrystals

	case rpg.VaultResourceStellarStones:
		return snapshot.
			WalletStellarStones

	default:
		return 0
	}
}

func renderVaultTiers() string {
	var builder strings.Builder

	builder.WriteString(
		"🏦 *COFRES DO REINO*\n\n",
	)

	for index, tier := range rpg.VaultTiers() {

		if tier.Infinite {
			fmt.Fprintf(
				&builder,
				"🌌 *%s*\n"+
					"Capacidade: *∞*\n"+
					"☠️ Drop exclusivo: *Deus do Vazio*\n",
				tier.Name,
			)

			continue
		}

		fmt.Fprintf(
			&builder,
			"%d. *%s* — %s por recurso\n",
			index+1,
			tier.Name,
			formatVaultNumber(
				tier.Capacity,
			),
		)
	}

	builder.WriteString(
		"\n🛒 Os upgrades serão liberados futuramente na *!loja*.",
	)

	return builder.String()
}

func vaultCapacityText(
	tier rpg.VaultTier,
) string {
	if tier.Infinite {
		return "∞"
	}

	return formatVaultNumber(
		tier.Capacity,
	)
}

func formatVaultNumber(
	value int,
) string {
	raw :=
		strconv.Itoa(value)

	if len(raw) <= 3 {
		return raw
	}

	var builder strings.Builder

	first :=
		len(raw) % 3

	if first == 0 {
		first = 3
	}

	builder.WriteString(
		raw[:first],
	)

	for index :=
		first; index < len(raw); index += 3 {

		builder.WriteString(".")

		builder.WriteString(
			raw[index : index+3],
		)
	}

	return builder.String()
}

func sendVaultUsage(
	client *whatsmeow.Client,
	msgEvent *events.Message,
) {
	_ = whatsapp.SendText(
		client,
		msgEvent.Info.Chat,
		"🏦 *COFRE*\n\n"+
			"🔎 Consultar:\n"+
			"*!cofre*\n\n"+
			"🔒 Guardar:\n"+
			"*!cofre guardar gold 10000*\n"+
			"*!cofre depositar cristais mágicos 100*\n"+
			"*!cofre colocar pedras estelares 10*\n\n"+
			"🔓 Retirar:\n"+
			"*!cofre retirar gold 10000*\n"+
			"*!cofre sacar cristais 100*\n"+
			"*!cofre pegar pedras estelares 10*\n\n"+
			"💡 *Ações equivalentes*\n"+
			"Guardar: *guardar • depositar • colocar • armazenar*\n"+
			"Retirar: *retirar • sacar • pegar • remover*\n\n"+
			"💰 *Recursos aceitos*\n"+
			"*Gold/Ouro • Cristais Mágicos • Pedras Estelares*\n\n"+
			"🏦 Ver tipos de cofre:\n"+
			"*!cofre tipos*",
	)
}

func sendVaultError(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	err error,
) {
	var capacityErr *rpg.VaultCapacityError

	if errors.As(
		err,
		&capacityErr,
	) {
		remaining :=
			capacityErr.Tier.Capacity -
				capacityErr.Current

		if remaining < 0 {
			remaining = 0
		}

		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			fmt.Sprintf(
				"🏦 *COFRE CHEIO*\n\n"+
					"%s Capacidade para %s: *%s*\n"+
					"🔐 Guardado atualmente: *%s*\n"+
					"📦 Espaço restante: *%s*",
				capacityErr.Resource.Icon(),
				capacityErr.Resource.Name(),
				vaultCapacityText(
					capacityErr.Tier,
				),
				formatVaultNumber(
					capacityErr.Current,
				),
				formatVaultNumber(
					remaining,
				),
			),
		)

		return
	}

	var insufficientErr *rpg.VaultInsufficientError

	if errors.As(
		err,
		&insufficientErr,
	) {
		location :=
			"disponível fora do cofre"

		if insufficientErr.Stored {
			location =
				"guardado no cofre"
		}

		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			fmt.Sprintf(
				"❌ Você não possui %s suficiente.\n\n"+
					"%s Saldo %s: *%s*\n"+
					"📦 Solicitado: *%s*",
				insufficientErr.Resource.Name(),
				insufficientErr.Resource.Icon(),
				location,
				formatVaultNumber(
					insufficientErr.Available,
				),
				formatVaultNumber(
					insufficientErr.Requested,
				),
			),
		)

		return
	}

	if errors.Is(
		err,
		rpg.ErrVaultInvalidAmount,
	) {
		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			"❌ Quantidade inválida.",
		)

		return
	}

	logger.Error(
		"Erro no !cofre:",
		err,
	)

	_ = whatsapp.SendText(
		client,
		msgEvent.Info.Chat,
		"❌ Não foi possível acessar o cofre agora.",
	)
}
