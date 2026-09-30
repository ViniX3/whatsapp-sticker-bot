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

const shopOffersPerPage = 6

func handleRPGShop(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	parts []string,
) {

	// Permite formas naturais:
	//
	// !loja comprar L001
	// !loja adquirir L001
	if len(parts) == 3 {

		action,
			ok :=
			canonicalAction(
				parts[1],
			)

		if ok &&
			action == actionBuy {

			handleRPGShopPurchase(
				client,
				msgEvent,
				[]string{
					"!comprar",
					parts[2],
				},
			)

			return
		}
	}

	catalog,
		_,
		err :=
		getRPGCatalogs()

	if err != nil {
		sendRPGInternalError(
			client,
			msgEvent,
			"carregando catálogo da loja",
			err,
		)

		return
	}

	if len(parts) == 1 {
		renderShopList(
			client,
			msgEvent,
			catalog,
			"",
			1,
		)

		return
	}

	if len(parts) == 2 {
		if itemType, ok :=
			parseShopItemType(
				parts[1],
			); ok {

			renderShopTypeList(
				client,
				msgEvent,
				catalog,
				itemType,
				1,
			)

			return
		}

		if rarity, ok :=
			parseShopRarity(
				parts[1],
			); ok {

			renderShopList(
				client,
				msgEvent,
				catalog,
				rarity,
				1,
			)

			return
		}

		if page, pageErr :=
			strconv.Atoi(
				parts[1],
			); pageErr == nil {

			if page < 1 {
				sendShopUsage(
					client,
					msgEvent,
				)

				return
			}

			renderShopList(
				client,
				msgEvent,
				catalog,
				"",
				page,
			)

			return
		}

		renderShopOfferDetails(
			client,
			msgEvent,
			parts[1],
			catalog,
		)

		return
	}

	if len(parts) == 3 {
		if itemType,
			rarity,
			ok :=
			parseShopCombinedFilters(
				parts[1],
				parts[2],
			); ok {

			renderShopCombinedList(
				client,
				msgEvent,
				catalog,
				itemType,
				rarity,
				1,
			)

			return
		}

		if itemType, ok :=
			parseShopItemType(
				parts[1],
			); ok {

			page, pageErr :=
				strconv.Atoi(
					parts[2],
				)

			if pageErr != nil ||
				page < 1 {

				sendShopUsage(
					client,
					msgEvent,
				)

				return
			}

			renderShopTypeList(
				client,
				msgEvent,
				catalog,
				itemType,
				page,
			)

			return
		}

		if rarity, ok :=
			parseShopRarity(
				parts[1],
			); ok {

			page, pageErr :=
				strconv.Atoi(
					parts[2],
				)

			if pageErr != nil ||
				page < 1 {

				sendShopUsage(
					client,
					msgEvent,
				)

				return
			}

			renderShopList(
				client,
				msgEvent,
				catalog,
				rarity,
				page,
			)

			return
		}

		sendShopUsage(
			client,
			msgEvent,
		)

		return
	}

	if len(parts) == 4 {
		itemType,
			rarity,
			ok :=
			parseShopCombinedFilters(
				parts[1],
				parts[2],
			)

		if !ok {
			sendShopUsage(
				client,
				msgEvent,
			)

			return
		}

		page, pageErr :=
			strconv.Atoi(
				parts[3],
			)

		if pageErr != nil ||
			page < 1 {

			sendShopUsage(
				client,
				msgEvent,
			)

			return
		}

		renderShopCombinedList(
			client,
			msgEvent,
			catalog,
			itemType,
			rarity,
			page,
		)

		return
	}

	sendShopUsage(
		client,
		msgEvent,
	)
}

func handleRPGShopPurchase(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	parts []string,
) {
	if len(parts) != 2 {
		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			"🛒 *COMPRAR EQUIPAMENTO*\n\n"+
				"Use:\n"+
				"*!comprar <código>*\n"+
				"*!adquirir <código>*\n\n"+
				"Ex.: *!comprar L001*",
		)

		return
	}

	catalog,
		_,
		err :=
		getRPGCatalogs()

	if err != nil {
		sendRPGInternalError(
			client,
			msgEvent,
			"carregando catálogo para compra",
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
		rpg.BuyShopItem(
			groupJID,
			jid,
			parts[1],
			catalog,
		)

	if err != nil {
		handleShopPurchaseError(
			client,
			msgEvent,
			parts[1],
			catalog,
			err,
		)

		return
	}

	autoEquip,
		autoEquipErr :=
		rpg.AutoEquipIfBetter(
			groupJID,
			jid,
			result.Offer.Item.ID,
			catalog,
		)

	var builder strings.Builder

	fmt.Fprintf(
		&builder,
		"✅ *Comprado*\n"+
			"%s %s *%s* • PC %s\n"+
			"💰 -%s • Saldo *%s*",
		forgeRarityIcon(
			result.Offer.Item.Rarity,
		),
		forgeItemTypeIcon(
			result.Offer.Item.Type,
		),
		result.Offer.Item.Name,
		forgeFormatNumber(
			result.Offer.Item.Power,
		),
		formatGold(
			result.Offer.Price,
		),
		formatGold(
			result.GoldBalance,
		),
	)

	if autoEquipErr == nil &&
		autoEquip != nil {

		switch {
		case autoEquip.Equipped &&
			autoEquip.HadPrevious:

			fmt.Fprintf(
				&builder,
				"\n⚡ Equipado • substituiu *%s*",
				autoEquip.PreviousItem.Name,
			)

		case autoEquip.Equipped:
			builder.WriteString(
				"\n⚡ Equipado automaticamente",
			)

		case autoEquip.HadPrevious:
			fmt.Fprintf(
				&builder,
				"\n🛡️ Mantido: *%s*",
				autoEquip.PreviousItem.Name,
			)
		}
	}

	_ = whatsapp.SendText(
		client,
		msgEvent.Info.Chat,
		builder.String(),
	)
}

func sendCompactShopPage(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	offers []rpg.ShopOffer,
	title string,
	page int,
	pageCommand string,
) {
	if len(offers) == 0 {
		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			"🛒 Nenhuma oferta disponível.",
		)

		return
	}

	totalPages :=
		(len(offers) +
			shopOffersPerPage - 1) /
			shopOffersPerPage

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
			shopOffersPerPage

	end :=
		start +
			shopOffersPerPage

	if end > len(offers) {
		end = len(offers)
	}

	var builder strings.Builder

	fmt.Fprintf(
		&builder,
		"🛒 *%s* • %d/%d\n\n",
		title,
		page,
		totalPages,
	)

	for _, offer := range offers[start:end] {

		item := offer.Item

		fmt.Fprintf(
			&builder,
			"%s %s %s *%s* • PC %s • 💰 %s\n",
			offer.Code,
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
				offer.Price,
			),
		)
	}

	builder.WriteString(
		"\n🔎 *!loja <código>* • 🛒 *!comprar <código>*",
	)

	if totalPages > 1 &&
		pageCommand != "" {

		fmt.Fprintf(
			&builder,
			"\n📖 *%s*",
			pageCommand,
		)
	}

	_ = whatsapp.SendText(
		client,
		msgEvent.Info.Chat,
		builder.String(),
	)
}

func renderShopList(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	catalog *rpg.Catalog,
	rarity rpg.Rarity,
	page int,
) {
	offers, err :=
		rpg.ListShopOffers(
			catalog,
			rarity,
		)

	if err != nil {
		sendRPGInternalError(
			client,
			msgEvent,
			"listando ofertas da loja",
			err,
		)

		return
	}

	title := "LOJA"

	pageCommand :=
		"!loja <página>"

	if rarity != "" {
		title =
			"LOJA • " +
				strings.ToUpper(
					forgeRarityName(
						rarity,
					),
				)

		pageCommand =
			fmt.Sprintf(
				"!loja %s <página>",
				shopRarityCommand(
					rarity,
				),
			)
	}

	sendCompactShopPage(
		client,
		msgEvent,
		offers,
		title,
		page,
		pageCommand,
	)
}

func renderShopTypeList(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	catalog *rpg.Catalog,
	itemType rpg.ItemType,
	page int,
) {
	offers, err :=
		rpg.ListShopOffers(
			catalog,
			"",
		)

	if err != nil {
		sendRPGInternalError(
			client,
			msgEvent,
			"listando equipamentos da loja",
			err,
		)

		return
	}

	filtered :=
		make(
			[]rpg.ShopOffer,
			0,
		)

	for _, offer := range offers {

		if offer.Item.Type !=
			itemType {

			continue
		}

		filtered =
			append(
				filtered,
				offer,
			)
	}

	sendCompactShopPage(
		client,
		msgEvent,
		filtered,
		"LOJA • "+
			strings.ToUpper(
				forgeItemTypeName(
					itemType,
				),
			),
		page,
		fmt.Sprintf(
			"!loja %s <página>",
			shopItemTypeCommand(
				itemType,
			),
		),
	)
}

func renderShopCombinedList(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	catalog *rpg.Catalog,
	itemType rpg.ItemType,
	rarity rpg.Rarity,
	page int,
) {
	offers, err :=
		rpg.ListShopOffers(
			catalog,
			rarity,
		)

	if err != nil {
		sendRPGInternalError(
			client,
			msgEvent,
			"listando ofertas filtradas da loja",
			err,
		)

		return
	}

	filtered :=
		make(
			[]rpg.ShopOffer,
			0,
		)

	for _, offer := range offers {

		if offer.Item.Type !=
			itemType {

			continue
		}

		filtered =
			append(
				filtered,
				offer,
			)
	}

	sendCompactShopPage(
		client,
		msgEvent,
		filtered,
		fmt.Sprintf(
			"LOJA • %s • %s",
			strings.ToUpper(
				forgeItemTypeName(
					itemType,
				),
			),
			strings.ToUpper(
				forgeRarityName(
					rarity,
				),
			),
		),
		page,
		fmt.Sprintf(
			"!loja %s %s <página>",
			shopItemTypeCommand(
				itemType,
			),
			shopRarityCommand(
				rarity,
			),
		),
	)
}

func renderShopOfferDetails(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	reference string,
	catalog *rpg.Catalog,
) {
	groupJID :=
		msgEvent.Info.Chat.String()

	jid :=
		canonicalSenderJID(
			msgEvent,
		)

	check, err :=
		rpg.CheckShopPurchase(
			groupJID,
			jid,
			reference,
			catalog,
		)

	if err != nil {
		switch {
		case errors.Is(
			err,
			rpg.ErrShopOfferNotFound,
		):
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"❌ Oferta não encontrada. Use *!loja*.",
			)

		case errors.Is(
			err,
			rpg.ErrWalletNotFound,
		):
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"❌ Use *!gold* ou *!rpg* primeiro.",
			)

		default:
			sendRPGInternalError(
				client,
				msgEvent,
				"consultando oferta da loja",
				err,
			)
		}

		return
	}

	item :=
		check.Offer.Item

	var builder strings.Builder

	fmt.Fprintf(
		&builder,
		"🛒 *%s* • %s %s *%s*\n"+
			"⚡ PC %s • ⚔️ %s • 🛡️ %s\n"+
			"💰 %s • Seu saldo: *%s*",
		check.Offer.Code,
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
		formatGold(
			check.Offer.Price,
		),
		formatGold(
			check.GoldOwned,
		),
	)

	if item.Rarity ==
		rpg.RarityRare ||
		item.Rarity ==
			rpg.RarityEpic {

		forgeCost, costErr :=
			rpg.ForgeGoldCost(
				item,
			)

		if costErr == nil {
			if recipe, recipeErr :=
				rpg.FindForgeRecipe(
					catalog,
					item.ID,
				); recipeErr == nil {

				fmt.Fprintf(
					&builder,
					"\n⚒️ Forja: %s • *!forja %s*",
					formatGold(
						forgeCost,
					),
					recipe.Code,
				)
			}
		}
	}

	switch {
	case check.UniqueAlreadyOwned:
		builder.WriteString(
			"\n🚫 Item único já possuído.",
		)

	case check.CanBuy:
		fmt.Fprintf(
			&builder,
			"\n✅ *!comprar %s*",
			check.Offer.Code,
		)

	default:
		builder.WriteString(
			"\n❌ Gold insuficiente.",
		)
	}

	_ = whatsapp.SendText(
		client,
		msgEvent.Info.Chat,
		builder.String(),
	)
}

func handleShopPurchaseError(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	reference string,
	catalog *rpg.Catalog,
	err error,
) {
	switch {
	case errors.Is(
		err,
		rpg.ErrShopOfferNotFound,
	):
		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			"❌ Oferta não encontrada.\n\n"+
				"Use *!loja* para consultar as ofertas.",
		)

	case errors.Is(
		err,
		rpg.ErrShopInsufficientGold,
	):
		renderShopOfferDetails(
			client,
			msgEvent,
			reference,
			catalog,
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
		sendRPGInternalError(
			client,
			msgEvent,
			"comprando equipamento na loja",
			err,
		)
	}
}

func sendShopUsage(
	client *whatsmeow.Client,
	msgEvent *events.Message,
) {
	_ = whatsapp.SendText(
		client,
		msgEvent.Info.Chat,
		"🛒 *LOJA*\n"+
			"*!loja* • *!loja raro* • *!loja epico*\n"+
			"*!loja arma* • *!loja escudo* • *!loja armadura*\n"+
			"*!loja arma raro* • *!loja <código>*\n\n"+
			"🛒 *Comprar*\n"+
			"*!comprar <código>*\n"+
			"*!adquirir <código>*\n"+
			"*!loja comprar <código>*\n"+
			"*!loja adquirir <código>*",
	)
}

func parseShopCombinedFilters(
	first string,
	second string,
) (
	rpg.ItemType,
	rpg.Rarity,
	bool,
) {
	if itemType, typeOK :=
		parseShopItemType(
			first,
		); typeOK {

		if rarity, rarityOK :=
			parseShopRarity(
				second,
			); rarityOK {

			return itemType,
				rarity,
				true
		}
	}

	if rarity, rarityOK :=
		parseShopRarity(
			first,
		); rarityOK {

		if itemType, typeOK :=
			parseShopItemType(
				second,
			); typeOK {

			return itemType,
				rarity,
				true
		}
	}

	return "",
		"",
		false
}

func parseShopItemType(
	value string,
) (
	rpg.ItemType,
	bool,
) {
	switch normalizeCommandToken(
		value,
	) {
	case "arma",
		"armas",
		"weapon",
		"weapons":

		return rpg.ItemTypeWeapon,
			true

	case "escudo",
		"escudos",
		"shield",
		"shields":

		return rpg.ItemTypeShield,
			true

	case "armadura",
		"armaduras",
		"armor",
		"armors":

		return rpg.ItemTypeArmor,
			true
	}

	return "",
		false
}

func shopItemTypeCommand(
	itemType rpg.ItemType,
) string {
	switch itemType {
	case rpg.ItemTypeWeapon:
		return "arma"

	case rpg.ItemTypeShield:
		return "escudo"

	case rpg.ItemTypeArmor:
		return "armadura"
	}

	return ""
}

func parseShopRarity(
	value string,
) (
	rpg.Rarity,
	bool,
) {
	switch normalizeCommandToken(
		value,
	) {
	case "desgastado",
		"worn":

		return rpg.RarityWorn,
			true

	case "comum",
		"common":

		return rpg.RarityCommon,
			true

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

func shopRarityCommand(
	rarity rpg.Rarity,
) string {
	switch rarity {
	case rpg.RarityWorn:
		return "desgastado"

	case rpg.RarityCommon:
		return "comum"

	case rpg.RarityRare:
		return "raro"

	case rpg.RarityEpic:
		return "epico"
	}

	return ""
}
