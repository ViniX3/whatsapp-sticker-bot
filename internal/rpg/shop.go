package rpg

import (
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"whatsapp-sticker-bot/internal/database"
)

const (
	// ShopEpicMaxPower define o teto de poder de um
	// equipamento Épico comprável diretamente com Gold.
	//
	// Épicos acima deste valor continuam existindo no
	// catálogo e podem ser obtidos por progressão RPG,
	// como Forja e PvE.
	ShopEpicMaxPower = 620

	ShopWornMultiplier   = 3
	ShopCommonMultiplier = 5
	ShopRareMultiplier   = 6
	ShopEpicMultiplier   = 10

	ShopWornMinimumGold   = 1500
	ShopCommonMinimumGold = 15000
)

var (
	ErrShopOfferNotFound = errors.New(
		"oferta da loja não encontrada",
	)

	ErrShopItemNotAllowed = errors.New(
		"item não pode ser vendido na loja comum",
	)

	ErrShopInsufficientGold = errors.New(
		"Gold insuficiente para comprar o item",
	)
)

type ShopGoldError struct {
	Required int
	Owned    int
}

func (err *ShopGoldError) Error() string {
	return fmt.Sprintf(
		"%s: necessário %d, disponível %d",
		ErrShopInsufficientGold,
		err.Required,
		err.Owned,
	)
}

func (err *ShopGoldError) Unwrap() error {
	return ErrShopInsufficientGold
}

type ShopOffer struct {
	Code string

	Item Item

	Price int
}

type ShopCheck struct {
	Offer ShopOffer

	GoldOwned int

	UniqueAlreadyOwned bool

	CanBuy bool
}

type ShopResult struct {
	Offer ShopOffer

	Quantity int

	GoldBalance int
}

func ensureShopSchema() error {
	if database.DB == nil {
		return errors.New(
			"banco de dados não inicializado",
		)
	}

	_, err :=
		database.DB.Exec(`
			CREATE TABLE IF NOT EXISTS
				rpg_shop_offers (
					item_id TEXT PRIMARY KEY,

					offer_code TEXT NOT NULL UNIQUE,

					rarity TEXT NOT NULL,

					price INTEGER NOT NULL
						CHECK (price > 0),

					enabled INTEGER NOT NULL
						DEFAULT 1
						CHECK (enabled IN (0, 1)),

					created_at DATETIME NOT NULL
						DEFAULT CURRENT_TIMESTAMP,

					updated_at DATETIME NOT NULL
						DEFAULT CURRENT_TIMESTAMP
				);

			CREATE INDEX IF NOT EXISTS
				idx_rpg_shop_offers_code
			ON rpg_shop_offers (
				offer_code
			);

			CREATE INDEX IF NOT EXISTS
				idx_rpg_shop_offers_rarity
			ON rpg_shop_offers (
				rarity,
				enabled
			);
		`)

	if err != nil {
		return fmt.Errorf(
			"erro criando estrutura da loja: %w",
			err,
		)
	}

	return nil
}

func shopEligibleBase(
	item Item,
) bool {
	// A Loja Comum nunca vende conteúdo avançado.
	switch item.Rarity {
	case RarityWorn,
		RarityCommon,
		RarityRare,
		RarityEpic:

	default:
		return false
	}

	// Atualmente os itens iniciais/intermediários
	// possuem CRAFT como origem principal.
	//
	// A oferta da Loja é um segundo canal de aquisição
	// e não altera Source no catálogo.
	if item.Source != SourceCraft &&
		item.Source != SourceShop {

		return false
	}

	if item.Currency != CurrencyGold {
		return false
	}

	return true
}

// shopEligible aplica as regras gerais da Loja e também
// a barreira de progressão do endgame.
//
// Épicos com PC acima de ShopEpicMaxPower não podem ser
// comprados diretamente com Gold.
func shopEligible(
	item Item,
) bool {
	if !shopEligibleBase(
		item,
	) {
		return false
	}

	if item.Rarity ==
		RarityEpic &&
		item.Power >
			ShopEpicMaxPower {

		return false
	}

	return true
}

func ShopPrice(
	item Item,
) (
	int,
	error,
) {
	if !shopEligible(
		item,
	) {
		return 0,
			ErrShopItemNotAllowed
	}

	switch item.Rarity {
	case RarityWorn:
		price :=
			item.Value *
				ShopWornMultiplier

		if price <
			ShopWornMinimumGold {

			price =
				ShopWornMinimumGold
		}

		return price, nil

	case RarityCommon:
		price :=
			item.Value *
				ShopCommonMultiplier

		if price <
			ShopCommonMinimumGold {

			price =
				ShopCommonMinimumGold
		}

		return price, nil

	case RarityRare:
		forgePrice, err :=
			ForgeGoldCost(
				item,
			)

		if err != nil {
			return 0, err
		}

		return forgePrice *
				ShopRareMultiplier,
			nil

	case RarityEpic:
		forgePrice, err :=
			ForgeGoldCost(
				item,
			)

		if err != nil {
			return 0, err
		}

		return forgePrice *
				ShopEpicMultiplier,
			nil
	}

	return 0,
		ErrShopItemNotAllowed
}

func shopCandidateItems(
	catalog *Catalog,
) (
	[]Item,
	error,
) {
	if catalog == nil {
		return nil,
			errors.New(
				"catálogo RPG não inicializado",
			)
	}

	items :=
		make(
			[]Item,
			0,
		)

	for _, item := range catalog.Items() {

		if !shopEligible(
			item,
		) {
			continue
		}

		items =
			append(
				items,
				item,
			)
	}

	sort.Slice(
		items,
		func(
			i int,
			j int,
		) bool {
			if items[i].Rarity !=
				items[j].Rarity {

				return items[i].
					Rarity.
					Rank() <
					items[j].
						Rarity.
						Rank()
			}

			if items[i].Power !=
				items[j].Power {

				return items[i].Power <
					items[j].Power
			}

			return items[i].ID <
				items[j].ID
		},
	)

	return items, nil
}

// InitShop garante a infraestrutura da Loja
// e sincroniza ofertas existentes.
//
// Os códigos L001, L002... são persistentes.
// Adicionar itens novos futuramente não renumera
// ofertas já existentes.
func initShopBase(
	catalog *Catalog,
) error {
	if err :=
		ensureShopSchema(); err != nil {

		return err
	}

	return syncShopOffers(
		catalog,
	)
}

// InitShop inicializa a Loja e aplica as regras atuais de
// progressão aos códigos persistidos.
//
// Códigos antigos de Épicos acima do limite não são
// apagados. Eles apenas ficam desabilitados.
func InitShop(
	catalog *Catalog,
) error {
	if err :=
		initShopBase(
			catalog,
		); err != nil {

		return err
	}

	return syncShopProgressionRestrictions(
		catalog,
	)
}

// syncShopProgressionRestrictions garante que ofertas
// persistidas anteriormente respeitem as regras atuais.
//
// Isso impede que um código Lxxx antigo continue sendo
// comprado depois que o item saiu da Loja comum.
func syncShopProgressionRestrictions(
	catalog *Catalog,
) error {
	if catalog == nil {
		return ErrItemNotFound
	}

	for _, item := range catalog.Items() {

		if item.Rarity !=
			RarityEpic {

			continue
		}

		enabled := 1

		if item.Power >
			ShopEpicMaxPower {

			enabled = 0
		}

		_, err :=
			database.DB.Exec(`
				UPDATE rpg_shop_offers
				SET
					enabled = ?,
					updated_at =
						CURRENT_TIMESTAMP
				WHERE item_id = ?
			`,
				enabled,
				item.ID,
			)

		if err != nil {
			return fmt.Errorf(
				"erro aplicando progressão da loja ao item %s: %w",
				item.ID,
				err,
			)
		}
	}

	return nil
}

func syncShopOffers(
	catalog *Catalog,
) error {
	if err :=
		ensureShopSchema(); err != nil {

		return err
	}

	items, err :=
		shopCandidateItems(
			catalog,
		)

	if err != nil {
		return err
	}

	tx, err :=
		database.DB.Begin()

	if err != nil {
		return fmt.Errorf(
			"erro iniciando sincronização da loja: %w",
			err,
		)
	}

	defer tx.Rollback()

	rows, err :=
		tx.Query(`
			SELECT
				item_id,
				offer_code
			FROM rpg_shop_offers
		`)

	if err != nil {
		return fmt.Errorf(
			"erro consultando ofertas existentes: %w",
			err,
		)
	}

	existing :=
		make(
			map[string]string,
		)

	maxCode := 0

	for rows.Next() {
		var itemID string
		var code string

		if err :=
			rows.Scan(
				&itemID,
				&code,
			); err != nil {

			rows.Close()

			return fmt.Errorf(
				"erro lendo oferta da loja: %w",
				err,
			)
		}

		existing[itemID] =
			code

		upper :=
			strings.ToUpper(
				strings.TrimSpace(
					code,
				),
			)

		if !strings.HasPrefix(
			upper,
			"L",
		) {
			continue
		}

		number, parseErr :=
			strconv.Atoi(
				strings.TrimPrefix(
					upper,
					"L",
				),
			)

		if parseErr != nil {
			continue
		}

		if number > maxCode {
			maxCode =
				number
		}
	}

	if err :=
		rows.Err(); err != nil {

		rows.Close()

		return fmt.Errorf(
			"erro iterando ofertas existentes: %w",
			err,
		)
	}

	if err :=
		rows.Close(); err != nil {

		return fmt.Errorf(
			"erro fechando consulta da loja: %w",
			err,
		)
	}

	for _, item := range items {

		price, err :=
			ShopPrice(
				item,
			)

		if err != nil {
			return err
		}

		if code, exists :=
			existing[item.ID]; exists {

			_, err =
				tx.Exec(`
					UPDATE rpg_shop_offers
					SET
						rarity = ?,
						price = ?,
						enabled = 1,
						updated_at =
							CURRENT_TIMESTAMP
					WHERE item_id = ?
				`,
					string(
						item.Rarity,
					),
					price,
					item.ID,
				)

			if err != nil {
				return fmt.Errorf(
					"erro atualizando oferta %s: %w",
					code,
					err,
				)
			}

			continue
		}

		maxCode++

		code :=
			fmt.Sprintf(
				"L%03d",
				maxCode,
			)

		_, err =
			tx.Exec(`
				INSERT INTO
					rpg_shop_offers (
						item_id,
						offer_code,
						rarity,
						price,
						enabled
					)
				VALUES (?, ?, ?, ?, 1)
			`,
				item.ID,
				code,
				string(
					item.Rarity,
				),
				price,
			)

		if err != nil {
			return fmt.Errorf(
				"erro registrando oferta %s: %w",
				code,
				err,
			)
		}

		existing[item.ID] =
			code
	}

	if err :=
		tx.Commit(); err != nil {

		return fmt.Errorf(
			"erro confirmando ofertas da loja: %w",
			err,
		)
	}

	return nil
}

func ListShopOffers(
	catalog *Catalog,
	rarity Rarity,
) (
	[]ShopOffer,
	error,
) {
	if rarity != "" {
		switch rarity {
		case RarityWorn,
			RarityCommon,
			RarityRare,
			RarityEpic:

		default:
			return nil,
				ErrShopItemNotAllowed
		}
	}

	if err :=
		syncShopOffers(
			catalog,
		); err != nil {

		return nil, err
	}

	rows, err :=
		database.DB.Query(`
			SELECT
				item_id,
				offer_code,
				price
			FROM rpg_shop_offers
			WHERE enabled = 1
			ORDER BY offer_code ASC
		`)

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro listando ofertas da loja: %w",
				err,
			)
	}

	defer rows.Close()

	offers :=
		make(
			[]ShopOffer,
			0,
		)

	for rows.Next() {
		var itemID string
		var code string
		var price int

		if err :=
			rows.Scan(
				&itemID,
				&code,
				&price,
			); err != nil {

			return nil,
				fmt.Errorf(
					"erro lendo oferta da loja: %w",
					err,
				)
		}

		item, exists :=
			catalog.ItemByID(
				itemID,
			)

		if !exists ||
			!shopEligible(
				item,
			) {

			continue
		}

		if rarity != "" &&
			item.Rarity != rarity {

			continue
		}

		offers =
			append(
				offers,
				ShopOffer{
					Code: code,

					Item: item,

					Price: price,
				},
			)
	}

	if err :=
		rows.Err(); err != nil {

		return nil,
			fmt.Errorf(
				"erro iterando ofertas da loja: %w",
				err,
			)
	}

	sort.Slice(
		offers,
		func(
			i int,
			j int,
		) bool {
			if offers[i].
				Item.
				Rarity !=
				offers[j].
					Item.
					Rarity {

				return offers[i].
					Item.
					Rarity.
					Rank() <
					offers[j].
						Item.
						Rarity.
						Rank()
			}

			if offers[i].
				Item.
				Power !=
				offers[j].
					Item.
					Power {

				return offers[i].
					Item.
					Power <
					offers[j].
						Item.
						Power
			}

			return offers[i].Code <
				offers[j].Code
		},
	)

	return offers, nil
}

func FindShopOffer(
	catalog *Catalog,
	reference string,
) (
	ShopOffer,
	error,
) {
	reference =
		strings.TrimSpace(
			reference,
		)

	if reference == "" {
		return ShopOffer{},
			ErrShopOfferNotFound
	}

	offers, err :=
		ListShopOffers(
			catalog,
			"",
		)

	if err != nil {
		return ShopOffer{},
			err
	}

	for _, offer := range offers {

		if strings.EqualFold(
			offer.Code,
			reference,
		) ||
			strings.EqualFold(
				offer.Item.ID,
				reference,
			) {

			return offer, nil
		}
	}

	return ShopOffer{},
		ErrShopOfferNotFound
}

func CheckShopPurchase(
	groupJID string,
	jid string,
	reference string,
	catalog *Catalog,
) (
	*ShopCheck,
	error,
) {
	offer, err :=
		FindShopOffer(
			catalog,
			reference,
		)

	if err != nil {
		return nil, err
	}

	if err :=
		ensureSchema(); err != nil {

		return nil, err
	}

	var gold int

	err =
		database.DB.QueryRow(`
			SELECT gold
			FROM group_wallets
			WHERE group_jid = ?
			  AND jid = ?
		`,
			groupJID,
			jid,
		).Scan(
			&gold,
		)

	if errors.Is(
		err,
		sql.ErrNoRows,
	) {
		return nil,
			ErrWalletNotFound
	}

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro consultando Gold da loja: %w",
				err,
			)
	}

	check :=
		&ShopCheck{
			Offer: offer,

			GoldOwned: gold,

			CanBuy: gold >=
				offer.Price,
		}

	if offer.Item.Unique {
		owned, err :=
			GetInventoryQuantity(
				groupJID,
				jid,
				offer.Item.ID,
			)

		if err != nil {
			return nil, err
		}

		if owned > 0 {
			check.UniqueAlreadyOwned =
				true

			check.CanBuy =
				false
		}
	}

	return check, nil
}

func BuyShopItem(
	groupJID string,
	jid string,
	reference string,
	catalog *Catalog,
) (
	*ShopResult,
	error,
) {
	offer, err :=
		FindShopOffer(
			catalog,
			reference,
		)

	if err != nil {
		return nil, err
	}

	if err :=
		ensureSchema(); err != nil {

		return nil, err
	}

	tx, err :=
		database.DB.Begin()

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro iniciando compra da loja: %w",
				err,
			)
	}

	defer tx.Rollback()

	if _, err :=
		ensurePlayerTx(
			tx,
			groupJID,
			jid,
		); err != nil {

		return nil, err
	}

	if offer.Item.Unique {
		var owned int

		err :=
			tx.QueryRow(`
				SELECT quantity
				FROM rpg_inventory
				WHERE group_jid = ?
				  AND jid = ?
				  AND item_id = ?
			`,
				groupJID,
				jid,
				offer.Item.ID,
			).Scan(
				&owned,
			)

		if err == nil &&
			owned > 0 {

			return nil,
				ErrUniqueItemOwned
		}

		if err != nil &&
			!errors.Is(
				err,
				sql.ErrNoRows,
			) {

			return nil,
				fmt.Errorf(
					"erro consultando item único da loja: %w",
					err,
				)
		}
	}

	var gold int

	err =
		tx.QueryRow(`
			SELECT gold
			FROM group_wallets
			WHERE group_jid = ?
			  AND jid = ?
		`,
			groupJID,
			jid,
		).Scan(
			&gold,
		)

	if errors.Is(
		err,
		sql.ErrNoRows,
	) {
		return nil,
			ErrWalletNotFound
	}

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro consultando Gold para compra: %w",
				err,
			)
	}

	if gold <
		offer.Price {

		return nil,
			&ShopGoldError{
				Required: offer.Price,

				Owned: gold,
			}
	}

	result, err :=
		tx.Exec(`
			UPDATE group_wallets
			SET
				gold = gold - ?,
				updated_at =
					CURRENT_TIMESTAMP
			WHERE group_jid = ?
			  AND jid = ?
			  AND gold >= ?
		`,
			offer.Price,
			groupJID,
			jid,
			offer.Price,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro debitando Gold da loja: %w",
				err,
			)
	}

	rowsAffected, err :=
		result.RowsAffected()

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro validando pagamento da loja: %w",
				err,
			)
	}

	if rowsAffected != 1 {
		return nil,
			&ShopGoldError{
				Required: offer.Price,

				Owned: gold,
			}
	}

	description :=
		fmt.Sprintf(
			"Compra na Loja: %s (%s)",
			offer.Item.Name,
			offer.Code,
		)

	_, err =
		tx.Exec(`
			INSERT INTO gold_transactions (
				group_jid,
				jid,
				related_jid,
				amount,
				type,
				description
			)
			VALUES (?, ?, NULL, ?, ?, ?)
		`,
			groupJID,
			jid,
			-offer.Price,
			"RPG_SHOP_PURCHASE",
			description,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro registrando compra da loja: %w",
				err,
			)
	}

	_, err =
		tx.Exec(`
			INSERT INTO rpg_inventory (
				group_jid,
				jid,
				item_id,
				quantity
			)
			VALUES (?, ?, ?, 1)

			ON CONFLICT (
				group_jid,
				jid,
				item_id
			)
			DO UPDATE SET
				quantity =
					rpg_inventory.quantity + 1,

				updated_at =
					CURRENT_TIMESTAMP
		`,
			groupJID,
			jid,
			offer.Item.ID,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro adicionando item comprado ao inventário: %w",
				err,
			)
	}

	newBalance :=
		gold -
			offer.Price

	if err :=
		tx.Commit(); err != nil {

		return nil,
			fmt.Errorf(
				"erro confirmando compra da loja: %w",
				err,
			)
	}

	return &ShopResult{
		Offer: offer,

		Quantity: 1,

		GoldBalance: newBalance,
	}, nil
}
