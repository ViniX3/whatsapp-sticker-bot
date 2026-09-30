package handler

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	"whatsapp-sticker-bot/internal/logger"
	"whatsapp-sticker-bot/internal/rpg"
	"whatsapp-sticker-bot/internal/whatsapp"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

const rpgInventoryPageSize = 12

var (
	rpgCatalogOnce sync.Once

	cachedRPGCatalog *rpg.Catalog

	cachedRPGMaterials *rpg.MaterialCatalog

	cachedRPGCatalogErr error
)

type rpgInventoryView struct {
	ID string

	Code string

	Name string

	Quantity int

	Rarity rpg.Rarity

	Kind string

	Power int

	Attack int

	Defense int

	IsEquipment bool

	Equipped bool
}

func handleRPGCommand(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	text string,
) bool {
	parts := strings.Fields(text)

	if len(parts) == 0 {
		return false
	}

	command := canonicalCommand(parts[0])

	switch command {
	case "!rpg",
		"!inventario",
		"!equipamentos",
		"!equipar",
		"!desequipar",
		"!coletar",
		"!forja",
		"!forjar",
		"!loja",
		"!comprar",
		"!pve",
		"!dungeons",
		"!cristais":

	default:
		return false
	}

	if !requireGoldGroup(
		client,
		msgEvent,
		command,
	) {
		return true
	}

	switch command {
	case "!rpg":
		handleRPGStarterCommand(
			client,
			msgEvent,
		)

	case "!inventario":
		handleRPGInventory(
			client,
			msgEvent,
			parts,
		)

	case "!equipamentos":
		handleRPGEquipment(
			client,
			msgEvent,
		)

	case "!equipar":
		handleRPGEquip(
			client,
			msgEvent,
			parts,
		)

	case "!desequipar":
		handleRPGUnequip(
			client,
			msgEvent,
			parts,
		)

	case "!coletar":
		handleRPGGathering(
			client,
			msgEvent,
			parts,
		)

	case "!forja":
		handleRPGForge(
			client,
			msgEvent,
			parts,
		)

	case "!forjar":
		handleRPGForgeCraft(
			client,
			msgEvent,
			parts,
		)

	case "!loja":
		handleRPGShop(
			client,
			msgEvent,
			parts,
		)

	case "!comprar":
		handleRPGShopPurchase(
			client,
			msgEvent,
			parts,
		)

	case "!pve":
		handleRPGPVE(
			client,
			msgEvent,
			parts,
		)

	case "!dungeons":
		handleRPGDungeons(
			client,
			msgEvent,
			text,
		)

	case "!cristais":
		handleRPGCrystals(
			client,
			msgEvent,
			parts,
		)

	}

	return true
}

func getRPGCatalogs() (
	*rpg.Catalog,
	*rpg.MaterialCatalog,
	error,
) {
	rpgCatalogOnce.Do(
		func() {
			cachedRPGCatalog,
				cachedRPGCatalogErr =
				rpg.LoadCatalog()

			if cachedRPGCatalogErr != nil {
				return
			}

			cachedRPGMaterials,
				cachedRPGCatalogErr =
				rpg.LoadMaterialCatalog()

			if cachedRPGCatalogErr != nil {
				return
			}

			cachedRPGCatalogErr =
				rpg.ValidateCraftRecipes(
					cachedRPGCatalog,
					cachedRPGMaterials,
				)
		},
	)

	return cachedRPGCatalog,
		cachedRPGMaterials,
		cachedRPGCatalogErr
}

func resolveRPGEquipmentCode(
	itemID string,
	catalog *rpg.Catalog,
) string {
	itemID =
		strings.TrimSpace(
			itemID,
		)

	if itemID == "" ||
		catalog == nil {

		return itemID
	}

	// Loja tem prioridade.
	offers, err :=
		rpg.ListShopOffers(
			catalog,
			"",
		)

	if err == nil {
		for _, offer := range offers {

			if strings.EqualFold(
				offer.Item.ID,
				itemID,
			) {
				code :=
					strings.TrimSpace(
						offer.Code,
					)

				if code != "" {
					return code
				}
			}
		}
	}

	// Forja como segundo caminho.
	recipes, err :=
		rpg.ListForgeRecipes(
			catalog,
			"",
		)

	if err == nil {
		for _, recipe := range recipes {

			if strings.EqualFold(
				recipe.Item.ID,
				itemID,
			) {
				code :=
					strings.TrimSpace(
						recipe.Code,
					)

				if code != "" {
					return code
				}
			}
		}
	}

	// Último fallback: ID real.
	return itemID
}

func handleRPGInventory(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	parts []string,
) {
	filter,
		page,
		parseErr :=
		parseInventoryArguments(
			parts[1:],
		)

	if parseErr != nil {
		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			"🎒 *INVENTÁRIO*\n\n"+
				"Use:\n"+
				"*!inventario*\n"+
				"*!inventario <página>*\n"+
				"*!inventario <filtro>*\n"+
				"*!inventario <filtro> <página>*\n\n"+
				"💡 Aliases:\n"+
				"*!inv • !mochila • !bolsa*\n\n"+
				"📦 Filtros:\n"+
				"*equipamentos • materiais • equipado*\n"+
				"*arma • escudo • armadura*\n\n"+
				"✨ Raridades:\n"+
				"*desgastado • comum • raro • épico*\n"+
				"*lendário • mítico • sagrado*\n\n"+
				"Exemplos:\n"+
				"*!mochila raro*\n"+
				"*!inventario materiais*\n"+
				"*!inventario épico 2*",
		)

		return
	}

	catalog,
		materialCatalog,
		err :=
		getRPGCatalogs()

	if err != nil {
		sendRPGInternalError(
			client,
			msgEvent,
			"carregando catálogos RPG",
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

	_, err =
		rpg.GetOrCreatePlayer(
			groupJID,
			jid,
		)

	if err != nil {
		sendRPGPlayerError(
			client,
			msgEvent,
			err,
		)

		return
	}

	inventory, err :=
		rpg.GetInventory(
			groupJID,
			jid,
		)

	if err != nil {
		sendRPGInternalError(
			client,
			msgEvent,
			"consultando inventário",
			err,
		)

		return
	}

	equippedItems, err :=
		rpg.GetEquippedItems(
			groupJID,
			jid,
			catalog,
		)

	if err != nil {
		sendRPGInternalError(
			client,
			msgEvent,
			"consultando equipamentos",
			err,
		)

		return
	}

	equipped :=
		make(
			map[string]bool,
		)

	for _, detail := range equippedItems {

		equipped[detail.Item.ID] =
			true
	}

	views :=
		make(
			[]rpgInventoryView,
			0,
			len(inventory),
		)

	totalQuantity := 0

	for _, entry := range inventory {

		totalQuantity +=
			entry.Quantity

		if item, exists :=
			catalog.ItemByID(
				entry.ItemID,
			); exists {

			views =
				append(
					views,
					rpgInventoryView{
						ID: item.ID,

						Code: resolveRPGEquipmentCode(
							item.ID,
							catalog,
						),

						Name: item.Name,

						Quantity: entry.Quantity,

						Rarity: item.Rarity,

						Kind: rpgItemTypeName(
							item.Type,
						),

						Power: item.Power,

						Attack: item.Attack,

						Defense: item.Defense,

						IsEquipment: true,

						Equipped: equipped[item.ID],
					},
				)

			continue
		}

		if material, exists :=
			materialCatalog.MaterialByID(
				entry.ItemID,
			); exists {

			views =
				append(
					views,
					rpgInventoryView{
						ID: material.ID,

						Name: material.Name,

						Quantity: entry.Quantity,

						Rarity: material.Rarity,

						Kind: rpgMaterialKindName(
							material.Kind,
						),
					},
				)

			continue
		}

		views =
			append(
				views,
				rpgInventoryView{
					ID: entry.ItemID,

					Name: "Item desconhecido",

					Quantity: entry.Quantity,

					Kind: "Desconhecido",
				},
			)
	}

	views =
		applyInventoryFilter(
			views,
			filter,
		)

	sort.SliceStable(
		views,
		func(
			i int,
			j int,
		) bool {
			left :=
				views[i]

			right :=
				views[j]

			if left.IsEquipment !=
				right.IsEquipment {

				return left.IsEquipment
			}

			if left.Rarity.Rank() !=
				right.Rarity.Rank() {

				return left.Rarity.Rank() >
					right.Rarity.Rank()
			}

			return left.Name <
				right.Name
		},
	)

	totalPages := 1

	if len(views) > 0 {
		totalPages = (len(views) + rpgInventoryPageSize - 1) / rpgInventoryPageSize
	}

	if page > totalPages {
		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			fmt.Sprintf(
				"❌ Página inexistente.\n\n"+
					"📖 O inventário possui *%d página(s)*.",
				totalPages,
			),
		)

		return
	}

	start :=
		(page - 1) *
			rpgInventoryPageSize

	end :=
		start +
			rpgInventoryPageSize

	if end > len(views) {
		end =
			len(views)
	}

	var builder strings.Builder

	title := "INVENTÁRIO"

	if filter.Label != "" {
		title +=
			" • " +
				strings.ToUpper(
					filter.Label,
				)
	}

	fmt.Fprintf(
		&builder,
		"🎒 *%s* • %d/%d\n\n",
		title,
		page,
		totalPages,
	)

	for _, view := range views[start:end] {
		if view.IsEquipment {
			equipped := ""

			if view.Equipped {
				equipped = " ⚡"
			}

			fmt.Fprintf(
				&builder,
				"%s *%s* • PC %s • `%s`%s\n",
				rpgRarityIcon(
					view.Rarity,
				),
				view.Name,
				formatRPGNumber(
					view.Power,
				),
				view.Code,
				equipped,
			)

			continue
		}

		fmt.Fprintf(
			&builder,
			"📦 %s *%s* ×%d\n",
			rpgRarityIcon(
				view.Rarity,
			),
			view.Name,
			view.Quantity,
		)
	}

	if totalPages > 1 {
		nextPage :=
			page + 1

		if page == totalPages {
			nextPage = 1
		}

		if filter.Token != "" {
			fmt.Fprintf(
				&builder,
				"\n📖 *!inventario %s %d*",
				filter.Token,
				nextPage,
			)
		} else {
			fmt.Fprintf(
				&builder,
				"\n📖 *!inventario %d*",
				nextPage,
			)
		}
	}

	builder.WriteString(
		"\n⚔️ *!equipar <código>*",
	)
	_ = whatsapp.SendText(
		client,
		msgEvent.Info.Chat,
		builder.String(),
	)
}

func handleRPGEquipment(
	client *whatsmeow.Client,
	msgEvent *events.Message,
) {
	catalog,
		_,
		err :=
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

	_, err =
		rpg.GetOrCreatePlayer(
			groupJID,
			jid,
		)

	if err != nil {
		sendRPGPlayerError(
			client,
			msgEvent,
			err,
		)

		return
	}

	summary, err :=
		rpg.GetEquipmentSummary(
			groupJID,
			jid,
			catalog,
		)

	if err != nil {
		sendRPGInternalError(
			client,
			msgEvent,
			"consultando equipamentos",
			err,
		)

		return
	}

	var builder strings.Builder

	builder.WriteString(
		"🛡️ *EQUIPAMENTOS*\n\n",
	)

	fmt.Fprintf(
		&builder,
		"⚔️ %s\n",
		renderRPGEquipmentSlot(
			summary.Weapon,
		),
	)

	fmt.Fprintf(
		&builder,
		"🛡️ %s\n",
		renderRPGEquipmentSlot(
			summary.Shield,
		),
	)

	fmt.Fprintf(
		&builder,
		"🥋 %s\n",
		renderRPGEquipmentSlot(
			summary.Armor,
		),
	)

	fmt.Fprintf(
		&builder,
		"\n⚡ *%s PC*",
		formatRPGNumber(
			summary.CombatPower,
		),
	)
	_ = whatsapp.SendText(
		client,
		msgEvent.Info.Chat,
		builder.String(),
	)
}

func handleRPGEquip(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	parts []string,
) {
	if len(parts) != 2 {
		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			"⚔️ *EQUIPAR ITEM*\n\n"+
				"Use:\n"+
				"*!equipar <código>*\n"+
				"*!vestir <código>*\n\n"+
				"Exemplos:\n"+
				"*!equipar L190*\n"+
				"*!vestir R012*",
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
			"carregando catálogo RPG",
			err,
		)

		return
	}

	reference :=
		strings.TrimSpace(
			parts[1],
		)

	item, resolveErr :=
		rpg.ResolveEquipmentReference(
			catalog,
			reference,
		)

	if resolveErr != nil {
		if errors.Is(
			resolveErr,
			rpg.ErrItemNotFound,
		) {
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"❌ Equipamento não encontrado.\n\n"+
					"Você pode usar o código da Loja, da Forja ou o ID interno.\n\n"+
					"Exemplos:\n"+
					"*!equipar L190*\n"+
					"*!equipar R012*",
			)

			return
		}

		sendRPGInternalError(
			client,
			msgEvent,
			"resolvendo código do equipamento",
			resolveErr,
		)

		return
	}

	itemID :=
		item.ID

	groupJID :=
		msgEvent.Info.Chat.String()

	jid :=
		canonicalSenderJID(
			msgEvent,
		)

	if _, err :=
		rpg.GetOrCreatePlayer(
			groupJID,
			jid,
		); err != nil {

		sendRPGPlayerError(
			client,
			msgEvent,
			err,
		)

		return
	}

	summary, err :=
		rpg.EquipItem(
			groupJID,
			jid,
			itemID,
			catalog,
		)

	if err != nil {
		switch {
		case errors.Is(
			err,
			rpg.ErrItemNotOwned,
		):
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"❌ Você não possui este equipamento.\n\n"+
					"Consulte sua mochila com *!inventario*.",
			)

		case errors.Is(
			err,
			rpg.ErrItemNotEquipment,
		):
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"❌ Este item não pode ser equipado.",
			)

		case errors.Is(
			err,
			rpg.ErrWalletNotFound,
		):
			sendRPGPlayerError(
				client,
				msgEvent,
				err,
			)

		default:
			sendRPGInternalError(
				client,
				msgEvent,
				"equipando item",
				err,
			)
		}

		return
	}

	response :=
		fmt.Sprintf(
			"⚔️ *Equipado*\n"+
				"%s %s *%s* • PC %s\n"+
				"⚡ Total: *%s PC*",
			rpgItemTypeIcon(
				item.Type,
			),
			rpgRarityIcon(
				item.Rarity,
			),
			item.Name,
			formatRPGNumber(
				item.Power,
			),
			formatRPGNumber(
				summary.CombatPower,
			),
		)

	_ = whatsapp.SendText(
		client,
		msgEvent.Info.Chat,
		response,
	)
}

func handleRPGUnequip(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	parts []string,
) {
	if len(parts) != 2 {
		sendRPGUnequipUsage(
			client,
			msgEvent,
		)

		return
	}

	slot, valid :=
		parseRPGEquipmentSlot(
			parts[1],
		)

	if !valid {
		sendRPGUnequipUsage(
			client,
			msgEvent,
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

	if _, err :=
		rpg.GetOrCreatePlayer(
			groupJID,
			jid,
		); err != nil {

		sendRPGPlayerError(
			client,
			msgEvent,
			err,
		)

		return
	}

	summary, err :=
		rpg.UnequipSlot(
			groupJID,
			jid,
			slot,
			catalog,
		)

	if err != nil {
		if errors.Is(
			err,
			rpg.ErrNothingEquipped,
		) {
			_ = whatsapp.SendText(
				client,
				msgEvent.Info.Chat,
				"📦 Não existe nenhum equipamento neste slot.",
			)

			return
		}

		sendRPGInternalError(
			client,
			msgEvent,
			"desequipando item",
			err,
		)

		return
	}

	response :=
		fmt.Sprintf(
			"📦 *Removido*\n%s %s\n⚡ Total: *%s PC*",
			rpgSlotIcon(
				slot,
			),
			rpgSlotName(
				slot,
			),
			formatRPGNumber(
				summary.CombatPower,
			),
		)

	_ = whatsapp.SendText(
		client,
		msgEvent.Info.Chat,
		response,
	)
}

func sendRPGUnequipUsage(
	client *whatsmeow.Client,
	msgEvent *events.Message,
) {
	_ = whatsapp.SendText(
		client,
		msgEvent.Info.Chat,
		"📦 *DESEQUIPAR*\n\n"+
			"Use:\n"+
			"*!desequipar arma*\n"+
			"*!desequipar escudo*\n"+
			"*!desequipar armadura*\n\n"+
			"Aliases:\n"+
			"*!tirar <slot>*\n"+
			"*!desvestir <slot>*",
	)
}

func sendRPGPlayerError(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	err error,
) {
	if errors.Is(
		err,
		rpg.ErrWalletNotFound,
	) {
		_ = whatsapp.SendText(
			client,
			msgEvent.Info.Chat,
			"📜 Antes de iniciar sua jornada RPG,\n"+
				"você precisa possuir uma carteira neste reino.\n\n"+
				"Use *!gold* primeiro.",
		)

		return
	}

	sendRPGInternalError(
		client,
		msgEvent,
		"acessando jogador RPG",
		err,
	)
}

func sendRPGInternalError(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	context string,
	err error,
) {
	logger.Error(
		"Erro RPG:",
		context,
		err,
	)

	_ = whatsapp.SendText(
		client,
		msgEvent.Info.Chat,
		"❌ Os escribas do reino encontraram um problema.\n\n"+
			"Tente novamente em alguns instantes.",
	)
}

func renderRPGEquipmentSlot(
	item *rpg.Item,
) string {
	if item == nil {
		return "📭 Vazio"
	}

	return fmt.Sprintf(
		"%s *%s* • PC %s",
		rpgRarityIcon(
			item.Rarity,
		),
		item.Name,
		formatRPGNumber(
			item.Power,
		),
	)
}

func parseRPGEquipmentSlot(
	value string,
) (rpg.EquipmentSlot, bool) {
	switch normalizeCommandToken(
		value,
	) {
	case "arma",
		"armas",
		"weapon",
		"weapons":
		return rpg.SlotWeapon,
			true

	case "escudo",
		"escudos",
		"shield",
		"shields":
		return rpg.SlotShield,
			true

	case "armadura",
		"armaduras",
		"armor",
		"armors":
		return rpg.SlotArmor,
			true
	}

	return "",
		false
}

func rpgRarityName(
	rarity rpg.Rarity,
) string {
	switch rarity {
	case rpg.RarityWorn:
		return "Desgastado"

	case rpg.RarityCommon:
		return "Comum"

	case rpg.RarityRare:
		return "Raro"

	case rpg.RarityEpic:
		return "Épico"

	case rpg.RarityLegendary:
		return "Lendário"

	case rpg.RarityMythic:
		return "Mítico"

	case rpg.RaritySacred:
		return "Sagrado"
	}

	return "Desconhecido"
}

func rpgRarityIcon(
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

	return "⚫"
}

func rpgItemTypeIcon(
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

func rpgItemTypeName(
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

func rpgTypeIconFromKind(
	kind string,
) string {
	switch kind {
	case "Arma":
		return "⚔️"

	case "Escudo":
		return "🛡️"

	case "Armadura":
		return "🥋"
	}

	return "📦"
}

func rpgMaterialKindName(
	kind rpg.MaterialKind,
) string {
	switch kind {
	case rpg.MaterialKindWood:
		return "Madeira"

	case rpg.MaterialKindMineral:
		return "Mineral"

	case rpg.MaterialKindMetal:
		return "Metal"

	case rpg.MaterialKindGem:
		return "Gema"

	case rpg.MaterialKindHide:
		return "Couro"

	case rpg.MaterialKindTextile:
		return "Tecido"

	case rpg.MaterialKindMonsterPart:
		return "Parte de Monstro"

	case rpg.MaterialKindBossPart:
		return "Troféu de Boss"

	case rpg.MaterialKindOtherworld:
		return "Material de Outro Mundo"
	}

	return "Material"
}

func rpgSlotName(
	slot rpg.EquipmentSlot,
) string {
	switch slot {
	case rpg.SlotWeapon:
		return "Arma"

	case rpg.SlotShield:
		return "Escudo"

	case rpg.SlotArmor:
		return "Armadura"
	}

	return "Equipamento"
}

func rpgSlotIcon(
	slot rpg.EquipmentSlot,
) string {
	switch slot {
	case rpg.SlotWeapon:
		return "⚔️"

	case rpg.SlotShield:
		return "🛡️"

	case rpg.SlotArmor:
		return "🥋"
	}

	return "📦"
}

func formatRPGNumber(
	value int,
) string {
	text :=
		strconv.Itoa(
			value,
		)

	if len(text) <= 3 {
		return text
	}

	var builder strings.Builder

	first :=
		len(text) % 3

	if first == 0 {
		first = 3
	}

	builder.WriteString(
		text[:first],
	)

	for index :=
		first; index < len(text); index += 3 {

		builder.WriteString(
			".",
		)

		builder.WriteString(
			text[index : index+3],
		)
	}

	return builder.String()
}
