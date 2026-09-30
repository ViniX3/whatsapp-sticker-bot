package handler

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"whatsapp-sticker-bot/internal/rpg"
	"whatsapp-sticker-bot/internal/whatsapp"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

const forgeRecipesPerPage = 6

func handleRPGForge(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	parts []string,
) {

	// Permite formas naturais:
	//
	// !forja forjar R001
	// !forja fabricar R001
	// !forja criar R001
	if len(parts) == 3 {

		action,
			ok :=
			canonicalAction(
				parts[1],
			)

		if ok &&
			action == actionCraft {

			handleRPGForgeCraft(
				client,
				msgEvent,
				[]string{
					"!forjar",
					parts[2],
				},
			)

			return
		}
	}

	catalog,
		materials,
		err :=
		getRPGCatalogs()

	if err != nil {
		sendRPGInternalError(
			client,
			msgEvent,
			"carregando catálogo da forja",
			err,
		)

		return
	}

	if len(parts) == 1 {
		renderForgeList(
			client,
			msgEvent,
			catalog,
			"",
			1,
		)

		return
	}

	if len(parts) == 2 {
		// !forja raro
		// !forja epico
		if rarity, ok :=
			parseForgeRarity(
				parts[1],
			); ok {

			renderForgeList(
				client,
				msgEvent,
				catalog,
				rarity,
				1,
			)

			return
		}

		// !forja 2
		if page, err :=
			strconv.Atoi(
				parts[1],
			); err == nil {

			if page < 1 {
				sendForgeUsage(
					client,
					msgEvent,
				)

				return
			}

			renderForgeList(
				client,
				msgEvent,
				catalog,
				"",
				page,
			)

			return
		}

		// !forja R001
		renderForgeRecipeDetails(
			client,
			msgEvent,
			parts[1],
			catalog,
			materials,
		)

		return
	}

	if len(parts) == 3 {
		// !forja raro 2
		// !forja epico 2
		rarity, ok :=
			parseForgeRarity(
				parts[1],
			)

		if !ok {
			sendForgeUsage(
				client,
				msgEvent,
			)

			return
		}

		page, err :=
			strconv.Atoi(
				parts[2],
			)

		if err != nil ||
			page < 1 {

			sendForgeUsage(
				client,
				msgEvent,
			)

			return
		}

		renderForgeList(
			client,
			msgEvent,
			catalog,
			rarity,
			page,
		)

		return
	}

	sendForgeUsage(
		client,
		msgEvent,
	)
}

func handleRPGForgeCraft(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	parts []string,
) {
	if len(parts) != 2 {
		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			"⚒️ *FORJAR EQUIPAMENTO*\n\n"+
				"Use:\n"+
				"*!forjar <código>*\n"+
				"*!fabricar <código>*\n"+
				"*!criar <código>*\n\n"+
				"Ex.: *!forjar R001*",
		)

		return
	}

	catalog,
		materials,
		err :=
		getRPGCatalogs()

	if err != nil {
		sendRPGInternalError(
			client,
			msgEvent,
			"carregando catálogo para fabricação",
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

	result, err :=
		rpg.ForgeItem(
			groupJID,
			jid,
			parts[1],
			catalog,
			materials,
		)

	if err != nil {
		handleForgeError(
			client,
			msgEvent,
			parts[1],
			catalog,
			materials,
			err,
		)

		return
	}

	autoEquip,
		autoEquipErr :=
		rpg.AutoEquipIfBetter(
			groupJID,
			jid,
			result.Recipe.Item.ID,
			catalog,
		)

	var builder strings.Builder

	fmt.Fprintf(
		&builder,
		"🔥 *Forjado*\n"+
			"%s %s *%s* • PC %s\n"+
			"💰 -%s • Saldo *%s*",
		forgeRarityIcon(
			result.Recipe.Item.Rarity,
		),
		forgeItemTypeIcon(
			result.Recipe.Item.Type,
		),
		result.Recipe.Item.Name,
		forgeFormatNumber(
			result.Recipe.Item.Power,
		),
		formatGold(
			result.Recipe.GoldCost,
		),
		formatGold(
			result.GoldBalance,
		),
	)

	if len(
		result.Recipe.Item.CraftMaterials,
	) > 0 {

		builder.WriteString(
			"\n📦 ",
		)

		for index, requirement := range result.Recipe.Item.CraftMaterials {

			if index > 0 {
				builder.WriteString(
					" • ",
				)
			}

			material, exists :=
				materials.MaterialByID(
					requirement.MaterialID,
				)

			name :=
				requirement.MaterialID

			if exists {
				name =
					material.Name
			}

			fmt.Fprintf(
				&builder,
				"%s×%d",
				name,
				requirement.Quantity,
			)
		}
	}

	if autoEquipErr == nil &&
		autoEquip != nil {

		switch {
		case autoEquip.Equipped &&
			autoEquip.HadPrevious:

			fmt.Fprintf(
				&builder,
				"\n⚡ *AUTO-EQUIP:* %s substituiu %s.\n"+
					"🏆 %s PC → %s PC",
				autoEquip.Item.Name,
				autoEquip.PreviousItem.Name,
				forgeFormatNumber(
					autoEquip.PreviousItem.Power,
				),
				forgeFormatNumber(
					autoEquip.Item.Power,
				),
			)

		case autoEquip.Equipped:
			builder.WriteString(
				"\n⚡ *AUTO-EQUIP:* equipado automaticamente.",
			)

		case autoEquip.HadPrevious:
			fmt.Fprintf(
				&builder,
				"\n🛡️ Mantido: *%s* (%s PC).",
				autoEquip.PreviousItem.Name,
				forgeFormatNumber(
					autoEquip.PreviousItem.Power,
				),
			)
		}
	}

	_ = whatsapp.SendText(
		client,
		msgEvent.Info.Chat,
		builder.String(),
	)
}

func renderForgeList(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	catalog *rpg.Catalog,
	rarity rpg.Rarity,
	page int,
) {
	recipes, err :=
		rpg.ListForgeRecipes(
			catalog,
			rarity,
		)

	if err != nil {
		sendRPGInternalError(
			client,
			msgEvent,
			"listando receitas da forja",
			err,
		)

		return
	}

	if len(recipes) == 0 {
		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			"⚒️ Nenhuma receita disponível.",
		)

		return
	}

	totalPages :=
		(len(recipes) +
			forgeRecipesPerPage - 1) /
			forgeRecipesPerPage

	if page < 1 ||
		page > totalPages {

		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			fmt.Sprintf(
				"❌ Página inválida. Disponíveis: *1-%d*.",
				totalPages,
			),
		)

		return
	}

	start :=
		(page - 1) *
			forgeRecipesPerPage

	end :=
		start +
			forgeRecipesPerPage

	if end > len(recipes) {
		end = len(recipes)
	}

	title := "FORJA"

	if rarity != "" {
		title =
			"FORJA • " +
				strings.ToUpper(
					forgeRarityName(
						rarity,
					),
				)
	}

	var builder strings.Builder

	fmt.Fprintf(
		&builder,
		"⚒️ *%s* • %d/%d\n\n",
		title,
		page,
		totalPages,
	)

	for _, recipe := range recipes[start:end] {

		item :=
			recipe.Item

		fmt.Fprintf(
			&builder,
			"%s %s %s *%s* • PC %s • 💰 %s\n",
			recipe.Code,
			forgeRarityIcon(
				item.Rarity,
			),
			forgeItemTypeIcon(
				item.Type,
			),
			item.Name,
			forgeFormatNumber(
				item.Power,
			),
			formatGold(
				recipe.GoldCost,
			),
		)
	}

	builder.WriteString(
		"\n🔎 *!forja <código>* • 🔥 *!forjar <código>*",
	)

	if totalPages > 1 {
		if rarity == "" {
			builder.WriteString(
				"\n📖 *!forja <página>*",
			)
		} else {
			fmt.Fprintf(
				&builder,
				"\n📖 *!forja %s <página>*",
				forgeRarityCommand(
					rarity,
				),
			)
		}
	}

	_ = whatsapp.SendText(
		client,
		msgEvent.Info.Chat,
		builder.String(),
	)
}

func renderForgeRecipeDetails(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	reference string,
	catalog *rpg.Catalog,
	materials *rpg.MaterialCatalog,
) {
	groupJID :=
		msgEvent.Info.Chat.String()

	jid :=
		canonicalSenderJID(
			msgEvent,
		)

	check, err :=
		rpg.CheckForgeRequirements(
			groupJID,
			jid,
			reference,
			catalog,
			materials,
		)

	if err != nil {
		if errors.Is(
			err,
			rpg.ErrForgeRecipeNotFound,
		) {
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"❌ Receita não encontrada. Use *!forja*.",
			)

			return
		}

		if errors.Is(
			err,
			rpg.ErrWalletNotFound,
		) {
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"❌ Use *!gold* ou *!rpg* primeiro.",
			)

			return
		}

		sendRPGInternalError(
			client,
			msgEvent,
			"consultando requisitos da forja",
			err,
		)

		return
	}

	item :=
		check.Recipe.Item

	var builder strings.Builder

	fmt.Fprintf(
		&builder,
		"⚒️ *%s* • %s %s *%s*\n"+
			"⚡ PC %s • ⚔️ %s • 🛡️ %s\n",
		check.Recipe.Code,
		forgeRarityIcon(
			item.Rarity,
		),
		forgeItemTypeIcon(
			item.Type,
		),
		item.Name,
		forgeFormatNumber(
			item.Power,
		),
		forgeFormatNumber(
			item.Attack,
		),
		forgeFormatNumber(
			item.Defense,
		),
	)

	goldIcon := "✅"

	if check.GoldOwned <
		check.Recipe.GoldCost {

		goldIcon = "❌"
	}

	fmt.Fprintf(
		&builder,
		"%s 💰 %s / %s\n",
		goldIcon,
		formatGold(
			check.GoldOwned,
		),
		formatGold(
			check.Recipe.GoldCost,
		),
	)

	for _, material := range check.Materials {

		status := "✅"

		if material.Owned <
			material.Required {

			status = "❌"
		}

		fmt.Fprintf(
			&builder,
			"%s %s %s %d/%d\n",
			status,
			forgeRarityIcon(
				material.Material.Rarity,
			),
			material.Material.Name,
			material.Owned,
			material.Required,
		)
	}

	switch {
	case check.UniqueAlreadyOwned:
		builder.WriteString(
			"🚫 Item único já possuído.",
		)

	case check.CanForge:
		fmt.Fprintf(
			&builder,
			"🔥 *!forjar %s*",
			check.Recipe.Code,
		)

	default:
		builder.WriteString(
			"❌ Faltam recursos.",
		)
	}

	_ = whatsapp.SendText(
		client,
		msgEvent.Info.Chat,
		builder.String(),
	)
}

func handleForgeError(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	reference string,
	catalog *rpg.Catalog,
	materials *rpg.MaterialCatalog,
	err error,
) {
	switch {
	case errors.Is(
		err,
		rpg.ErrForgeRecipeNotFound,
	):
		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			"❌ Receita não encontrada.\n\n"+
				"Use *!forja* para consultar as receitas disponíveis.",
		)

	case errors.Is(
		err,
		rpg.ErrForgeInsufficientGold,
	):
		renderForgeRecipeDetails(
			client,
			msgEvent,
			reference,
			catalog,
			materials,
		)

	case errors.Is(
		err,
		rpg.ErrUniqueItemOwned,
	):
		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			"🚫 Este equipamento é único e você já possui uma unidade.",
		)

	case errors.Is(
		err,
		rpg.ErrWalletNotFound,
	):
		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			"❌ Você ainda não possui uma carteira neste grupo.\n\n"+
				"Use *!gold* ou *!rpg* primeiro.",
		)

	default:
		var missingErr *rpg.MissingMaterialsError

		if errors.As(
			err,
			&missingErr,
		) {
			renderForgeRecipeDetails(
				client,
				msgEvent,
				reference,
				catalog,
				materials,
			)

			return
		}

		sendRPGInternalError(
			client,
			msgEvent,
			"fabricando equipamento na forja",
			err,
		)
	}
}

func sendForgeUsage(
	client *whatsmeow.Client,
	msgEvent *events.Message,
) {
	_ = whatsapp.SendText(
		client,
		msgEvent.Info.Chat,
		"⚒️ *FORJA*\n"+
			"*!forja* • *!forja raro* • *!forja epico*\n"+
			"*!forja <código>* • *!forja <página>*\n\n"+
			"🔥 *Fabricar*\n"+
			"*!forjar <código>*\n"+
			"*!fabricar <código>*\n"+
			"*!criar <código>*\n"+
			"*!forja fabricar <código>*\n"+
			"*!forja criar <código>*",
	)
}

func parseForgeRarity(
	value string,
) (
	rpg.Rarity,
	bool,
) {
	switch normalizeCommandToken(
		value,
	) {
	case "raro",
		"rare":

		return rpg.RarityRare,
			true

	case "epico",
		"épico",
		"epic":

		return rpg.RarityEpic,
			true
	}

	return "",
		false
}

func forgeRarityName(
	rarity rpg.Rarity,
) string {
	switch rarity {
	case rpg.RarityRare:
		return "Raro"

	case rpg.RarityEpic:
		return "Épico"

	case rpg.RarityWorn:
		return "Desgastado"

	case rpg.RarityCommon:
		return "Comum"

	case rpg.RarityLegendary:
		return "Lendário"

	case rpg.RarityMythic:
		return "Mítico"

	case rpg.RaritySacred:
		return "Sagrado"
	}

	return "Desconhecido"
}

func forgeRarityIcon(
	rarity rpg.Rarity,
) string {
	switch rarity {
	case rpg.RarityWorn:
		return "⚪"

	case rpg.RarityCommon:
		return "🟢"

	case rpg.RarityRare:
		return "🔵"

	case rpg.RarityEpic:
		return "🟣"

	case rpg.RarityLegendary:
		return "🔴"

	case rpg.RarityMythic:
		return "🟠"

	case rpg.RaritySacred:
		return "🌟"
	}

	return "📦"
}

func forgeItemTypeName(
	itemType rpg.ItemType,
) string {
	switch itemType {
	case rpg.ItemTypeWeapon:
		return "Arma"

	case rpg.ItemTypeShield:
		return "Escudo"

	case rpg.ItemTypeArmor:
		return "Armadura"
	}

	return "Equipamento"
}

func forgeItemTypeIcon(
	itemType rpg.ItemType,
) string {
	switch itemType {
	case rpg.ItemTypeWeapon:
		return "⚔️"

	case rpg.ItemTypeShield:
		return "🛡️"

	case rpg.ItemTypeArmor:
		return "🥋"
	}

	return "📦"
}

func forgeRarityCommand(
	rarity rpg.Rarity,
) string {
	switch rarity {
	case rpg.RarityRare:
		return "raro"

	case rpg.RarityEpic:
		return "epico"
	}

	return ""
}

func forgeFormatNumber(
	value int,
) string {
	negative :=
		value < 0

	if negative {
		value =
			-value
	}

	raw :=
		strconv.Itoa(
			value,
		)

	if len(raw) <= 3 {
		if negative {
			return "-" + raw
		}

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

		builder.WriteString(
			".",
		)

		builder.WriteString(
			raw[index : index+3],
		)
	}

	result :=
		builder.String()

	if negative {
		result =
			"-" + result
	}

	return result
}
