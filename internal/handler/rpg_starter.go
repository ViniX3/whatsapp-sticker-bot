package handler

import (
	"errors"
	"fmt"
	"strings"

	"whatsapp-sticker-bot/internal/gold"
	"whatsapp-sticker-bot/internal/rpg"
	"whatsapp-sticker-bot/internal/whatsapp"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

type adventureOnboardingResult struct {
	GoldClaimed bool

	StarterClaimed bool

	Balance int

	CombatPower int

	Starter *rpg.StarterSetResult
}

// handleRPGStarterCommand é a entrada do onboarding
// quando o usuário executa !rpg.
//
// A validação de grupo já ocorre no dispatcher RPG.
func handleRPGStarterCommand(
	client *whatsmeow.Client,
	msgEvent *events.Message,
) {
	handleAdventureOnboarding(
		client,
		msgEvent,
		"!rpg",
	)
}

// handleGoldStarterCommand é a entrada do mesmo
// onboarding quando o jogador utiliza !gold.
//
// !gold e !rpg são duas portas para o mesmo jogador.
func handleGoldStarterCommand(
	client *whatsmeow.Client,
	msgEvent *events.Message,
) {
	if !requireGoldGroup(
		client,
		msgEvent,
		"!gold",
	) {
		return
	}

	handleAdventureOnboarding(
		client,
		msgEvent,
		"!gold",
	)
}

// handleAdventureOnboarding garante que o jogador
// possua:
//
//   - carteira Gold inicializada;
//   - personagem RPG inicializado;
//   - Set do Recruta recebido;
//   - equipamentos iniciais equipados.
//
// O processo é idempotente.
//
// Jogadores antigos que já possuem Gold não recebem
// Gold novamente. Eles recebem somente o RPG faltante.
//
// Jogadores que já possuem RPG também não recebem
// equipamentos duplicados.
func handleAdventureOnboarding(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	entryPoint string,
) {
	catalog, _, err :=
		getRPGCatalogs()

	if err != nil {
		sendRPGInternalError(
			client,
			msgEvent,
			"carregando catálogo RPG",
			err,
		)

		return
	}

	groupJID :=
		msgEvent.Info.Chat.String()

	jid :=
		canonicalSenderJID(
			msgEvent,
		)

	name :=
		strings.TrimSpace(
			msgEvent.Info.PushName,
		)

	result, err :=
		ensureAdventureInitialized(
			groupJID,
			jid,
			name,
			catalog,
		)

	if err != nil {
		sendRPGInternalError(
			client,
			msgEvent,
			"inicializando aventureiro",
			err,
		)

		return
	}

	response :=
		renderAdventureOnboarding(
			entryPoint,
			result,
		)

	if entryPoint == "!rpg" {

		playerState, stateErr :=
			rpg.GetPlayer(
				groupJID,
				jid,
			)

		if stateErr != nil {
			sendRPGInternalError(
				client,
				msgEvent,
				"consultando Cristais do aventureiro",
				stateErr,
			)

			return
		}

		response +=
			fmt.Sprintf(
				"\n💎 *%s Cristais*",
				formatRPGNumber(
					playerState.MagicCrystals,
				),
			)
	}

	_ = whatsapp.SendText(
		client,
		msgEvent.Info.Chat,
		response,
	)
}

// ensureAdventureInitialized é a função central
// compartilhada por !gold e !rpg.
//
// A ordem é intencional:
//
//  1. inicializa/verifica a carteira Gold;
//  2. inicializa/verifica o personagem RPG;
//  3. consulta saldo e Poder de Combate.
//
// ClaimInitialGold e ClaimStarterSet são idempotentes.
//
// Portanto, se uma etapa já tiver ocorrido anteriormente,
// somente a etapa faltante será executada.
func ensureAdventureInitialized(
	groupJID string,
	jid string,
	name string,
	catalog *rpg.Catalog,
) (*adventureOnboardingResult, error) {
	if catalog == nil {
		return nil,
			errors.New(
				"catálogo RPG não inicializado",
			)
	}

	_, goldClaimed, err :=
		gold.ClaimInitialGold(
			groupJID,
			jid,
			name,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro inicializando carteira Gold: %w",
				err,
			)
	}

	starterResult, starterErr :=
		rpg.ClaimStarterSet(
			groupJID,
			jid,
			catalog,
		)

	starterClaimed := false

	switch {
	case starterErr == nil:
		starterClaimed = true

	case errors.Is(
		starterErr,
		rpg.ErrStarterAlreadyClaimed,
	):
		// Estado válido.
		//
		// O personagem já possui seu kit.
		starterResult = nil

	default:
		return nil,
			fmt.Errorf(
				"erro inicializando personagem RPG: %w",
				starterErr,
			)
	}

	balance, err :=
		gold.GetBalance(
			groupJID,
			jid,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro consultando saldo após onboarding: %w",
				err,
			)
	}

	summary, err :=
		rpg.GetEquipmentSummary(
			groupJID,
			jid,
			catalog,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro consultando Poder de Combate após onboarding: %w",
				err,
			)
	}

	return &adventureOnboardingResult{
		GoldClaimed: goldClaimed,

		StarterClaimed: starterClaimed,

		Balance: balance,

		CombatPower: summary.CombatPower,

		Starter: starterResult,
	}, nil
}

func renderAdventureOnboarding(
	entryPoint string,
	result *adventureOnboardingResult,
) string {
	if result == nil {
		return "❌ Não foi possível carregar o aventureiro."
	}

	switch {

	case result.GoldClaimed &&
		result.StarterClaimed:

		return fmt.Sprintf(
			"⚔️ *Aventureiro registrado!*\n"+
				"💰 +%s • Saldo *%s*\n"+
				"🎒 Set do Recruta recebido • ⚡ *%s PC*\n"+
				"📖 Use *!menu* para começar.",
			formatGold(
				gold.InitialGold,
			),
			formatGold(
				result.Balance,
			),
			formatRPGNumber(
				result.CombatPower,
			),
		)

	case !result.GoldClaimed &&
		result.StarterClaimed:

		return fmt.Sprintf(
			"⚔️ *RPG ativado!*\n"+
				"🎒 Set do Recruta recebido\n"+
				"💰 *%s* • ⚡ *%s PC*\n"+
				"📖 Use *!menu* para continuar.",
			formatGold(
				result.Balance,
			),
			formatRPGNumber(
				result.CombatPower,
			),
		)

	case result.GoldClaimed &&
		!result.StarterClaimed:

		return fmt.Sprintf(
			"💰 *Carteira ativada!*\n"+
				"💰 +%s • Saldo *%s*\n"+
				"⚡ *%s PC*",
			formatGold(
				gold.InitialGold,
			),
			formatGold(
				result.Balance,
			),
			formatRPGNumber(
				result.CombatPower,
			),
		)

	default:

		if entryPoint == "!gold" {
			return fmt.Sprintf(
				"💰 *%s*\n⚡ *%s PC*",
				formatGold(
					result.Balance,
				),
				formatRPGNumber(
					result.CombatPower,
				),
			)
		}

		return fmt.Sprintf(
			"⚔️ *Aventureiro ativo*\n"+
				"⚡ *%s PC* • 💰 *%s*",
			formatRPGNumber(
				result.CombatPower,
			),
			formatGold(
				result.Balance,
			),
		)
	}
}

// Novo usuário:
//
// não possuía Gold
// +
// não possuía RPG.
func renderNewAdventure(
	result *adventureOnboardingResult,
) string {
	var builder strings.Builder

	builder.WriteString(
		"╔════════════════════╗\n",
	)

	builder.WriteString(
		"⚔️ *AVENTUREIRO REGISTRADO*\n",
	)

	builder.WriteString(
		"🏰 Sua jornada começa agora\n",
	)

	builder.WriteString(
		"╚════════════════════╝\n\n",
	)

	fmt.Fprintf(
		&builder,
		"💰 Você recebeu *%s Gold* iniciais.\n",
		formatGoldAmount(gold.InitialGold),
	)

	fmt.Fprintf(
		&builder,
		"💰 Saldo atual: *%s Gold*\n\n",
		formatGoldAmount(result.Balance),
	)

	builder.WriteString(
		"━━━━━━━━━━━━━━━━━━━━\n",
	)

	builder.WriteString(
		"🎁 *SET DO RECRUTA*\n",
	)

	builder.WriteString(
		"━━━━━━━━━━━━━━━━━━━━\n\n",
	)

	renderStarterSet(
		&builder,
		result.Starter,
	)

	builder.WriteString(
		"━━━━━━━━━━━━━━━━━━━━\n",
	)

	fmt.Fprintf(
		&builder,
		"🏆 *PODER DE COMBATE: %s*\n",
		formatRPGNumber(
			result.CombatPower,
		),
	)

	builder.WriteString(
		"━━━━━━━━━━━━━━━━━━━━\n\n",
	)

	builder.WriteString(
		"✨ Os equipamentos foram equipados automaticamente.\n\n",
	)

	builder.WriteString(
		"🎒 *!inventario*\n",
	)

	builder.WriteString(
		"🛡️ *!equipamentos*\n",
	)

	builder.WriteString(
		"💰 *!saldo*",
	)

	return builder.String()
}

// Jogador antigo:
//
// já tinha carteira Gold
// +
// ainda não tinha aderido ao RPG.
func renderLegacyGoldMigration(
	result *adventureOnboardingResult,
) string {
	var builder strings.Builder

	builder.WriteString(
		"╔════════════════════╗\n",
	)

	builder.WriteString(
		"⚔️ *JORNADA RPG ATIVADA*\n",
	)

	builder.WriteString(
		"🏰 Seu aventureiro entrou no reino\n",
	)

	builder.WriteString(
		"╚════════════════════╝\n\n",
	)

	builder.WriteString(
		"💰 Sua carteira Gold já existia.\n",
	)

	builder.WriteString(
		"*Nenhum Gold adicional foi concedido.*\n\n",
	)

	fmt.Fprintf(
		&builder,
		"💰 Saldo preservado: *%s Gold*\n\n",
		formatGoldAmount(result.Balance),
	)

	builder.WriteString(
		"━━━━━━━━━━━━━━━━━━━━\n",
	)

	builder.WriteString(
		"🎁 *SET DO RECRUTA RECEBIDO*\n",
	)

	builder.WriteString(
		"━━━━━━━━━━━━━━━━━━━━\n\n",
	)

	renderStarterSet(
		&builder,
		result.Starter,
	)

	builder.WriteString(
		"━━━━━━━━━━━━━━━━━━━━\n",
	)

	fmt.Fprintf(
		&builder,
		"🏆 *PODER DE COMBATE: %s*\n",
		formatRPGNumber(
			result.CombatPower,
		),
	)

	builder.WriteString(
		"━━━━━━━━━━━━━━━━━━━━\n\n",
	)

	builder.WriteString(
		"✨ Os três equipamentos foram equipados automaticamente.\n\n",
	)

	builder.WriteString(
		"🎒 *!inventario*\n",
	)

	builder.WriteString(
		"🛡️ *!equipamentos*",
	)

	return builder.String()
}

// Caso raro de recuperação:
//
// Gold ainda não havia sido inicializado,
// porém o personagem RPG já existia.
func renderGoldRecovery(
	result *adventureOnboardingResult,
) string {
	return fmt.Sprintf(
		"╔════════════════════╗\n"+
			"💰 *CARTEIRA ATIVADA*\n"+
			"╚════════════════════╝\n\n"+
			"Você recebeu *%s Gold* iniciais.\n\n"+
			"💰 Saldo atual: *%s Gold*\n"+
			"🏆 Poder de Combate: *%s*\n\n"+
			"⚔️ Seu personagem RPG já estava ativo.",
		formatGoldAmount(gold.InitialGold),
		formatGoldAmount(result.Balance),
		formatRPGNumber(
			result.CombatPower,
		),
	)
}

func renderExistingAdventure(
	entryPoint string,
	result *adventureOnboardingResult,
) string {
	if entryPoint == "!gold" {
		return fmt.Sprintf(
			"╔════════════════════╗\n"+
				"💰 *CARTEIRA GOLD*\n"+
				"╚════════════════════╝\n\n"+
				"💰 Saldo atual: *%s Gold*\n"+
				"⚔️ Personagem RPG: *Ativo*\n"+
				"🏆 Poder de Combate: *%s*\n\n"+
				"🎒 *!inventario*\n"+
				"🛡️ *!equipamentos*",
			formatGoldAmount(result.Balance),
			formatRPGNumber(
				result.CombatPower,
			),
		)
	}

	return fmt.Sprintf(
		"╔════════════════════╗\n"+
			"⚔️ *JORNADA JÁ INICIADA*\n"+
			"╚════════════════════╝\n\n"+
			"🏆 Poder de Combate atual: *%s*\n"+
			"💰 Gold atual: *%s*\n\n"+
			"🎒 *!inventario*\n"+
			"🛡️ *!equipamentos*",
		formatRPGNumber(
			result.CombatPower,
		),
		formatGoldAmount(result.Balance),
	)
}

func renderStarterSet(
	builder *strings.Builder,
	result *rpg.StarterSetResult,
) {
	if builder == nil ||
		result == nil {

		return
	}

	fmt.Fprintf(
		builder,
		"⚔️ %s *%s*\n"+
			"└ PC %s • ATQ %s • DEF %s\n\n",
		rpgRarityIcon(
			result.Weapon.Rarity,
		),
		result.Weapon.Name,
		formatRPGNumber(
			result.Weapon.Power,
		),
		formatRPGNumber(
			result.Weapon.Attack,
		),
		formatRPGNumber(
			result.Weapon.Defense,
		),
	)

	fmt.Fprintf(
		builder,
		"🛡️ %s *%s*\n"+
			"└ PC %s • ATQ %s • DEF %s\n\n",
		rpgRarityIcon(
			result.Shield.Rarity,
		),
		result.Shield.Name,
		formatRPGNumber(
			result.Shield.Power,
		),
		formatRPGNumber(
			result.Shield.Attack,
		),
		formatRPGNumber(
			result.Shield.Defense,
		),
	)

	fmt.Fprintf(
		builder,
		"🥋 %s *%s*\n"+
			"└ PC %s • ATQ %s • DEF %s\n\n",
		rpgRarityIcon(
			result.Armor.Rarity,
		),
		result.Armor.Name,
		formatRPGNumber(
			result.Armor.Power,
		),
		formatRPGNumber(
			result.Armor.Attack,
		),
		formatRPGNumber(
			result.Armor.Defense,
		),
	)
}
