package rpg

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"whatsapp-sticker-bot/internal/database"
)

const (
	StarterRecruitWeaponID = "starter_recruit_weapon"
	StarterRecruitShieldID = "starter_recruit_shield"
	StarterRecruitArmorID  = "starter_recruit_armor"
)

var (
	ErrStarterAlreadyClaimed = errors.New(
		"kit inicial do RPG já foi reivindicado",
	)
)

type StarterClaim struct {
	GroupJID string

	JID string

	ClaimedAt time.Time
}

type StarterSetResult struct {
	Weapon Item

	Shield Item

	Armor Item

	Attack int

	Defense int

	CombatPower int

	ClaimedAt time.Time
}

// StarterEquipmentIDs retorna os IDs oficiais
// dos três equipamentos iniciais.
func StarterEquipmentIDs() []string {
	return []string{
		StarterRecruitWeaponID,
		StarterRecruitShieldID,
		StarterRecruitArmorID,
	}
}

// HasClaimedStarterSet informa se o jogador
// já iniciou sua jornada RPG naquele grupo.
func HasClaimedStarterSet(
	groupJID string,
	jid string,
) (bool, error) {
	if err := ensureStarterSchema(); err != nil {
		return false, err
	}

	var exists int

	err := database.DB.QueryRow(`
		SELECT 1
		FROM rpg_starter_claims
		WHERE group_jid = ?
		  AND jid = ?
		LIMIT 1
	`,
		groupJID,
		jid,
	).Scan(
		&exists,
	)

	if errors.Is(
		err,
		sql.ErrNoRows,
	) {
		return false, nil
	}

	if err != nil {
		return false,
			fmt.Errorf(
				"erro consultando claim do kit inicial: %w",
				err,
			)
	}

	return true, nil
}

// GetStarterClaim retorna os dados do claim.
//
// Se ainda não existir:
//
// nil, nil
func GetStarterClaim(
	groupJID string,
	jid string,
) (*StarterClaim, error) {
	if err := ensureStarterSchema(); err != nil {
		return nil, err
	}

	var claimedAt time.Time

	err := database.DB.QueryRow(`
		SELECT claimed_at
		FROM rpg_starter_claims
		WHERE group_jid = ?
		  AND jid = ?
	`,
		groupJID,
		jid,
	).Scan(
		&claimedAt,
	)

	if errors.Is(
		err,
		sql.ErrNoRows,
	) {
		return nil, nil
	}

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro consultando dados do kit inicial: %w",
				err,
			)
	}

	return &StarterClaim{
		GroupJID:  groupJID,
		JID:       jid,
		ClaimedAt: claimedAt,
	}, nil
}

// ClaimStarterSet inicia oficialmente o jogador
// no sistema RPG.
//
// Toda a operação ocorre dentro de UMA transação:
//
//  1. valida/cria jogador RPG;
//  2. verifica se o kit já foi reivindicado;
//  3. garante os três equipamentos no inventário;
//  4. equipa arma;
//  5. equipa escudo;
//  6. equipa armadura;
//  7. registra o claim;
//  8. commit.
//
// Se qualquer etapa falhar, tudo é revertido.
func ClaimStarterSet(
	groupJID string,
	jid string,
	catalog *Catalog,
) (*StarterSetResult, error) {
	if catalog == nil {
		return nil,
			errors.New(
				"catálogo RPG não inicializado",
			)
	}

	if err := ensureStarterSchema(); err != nil {
		return nil, err
	}

	weapon, shield, armor, err :=
		validateStarterEquipment(
			catalog,
		)

	if err != nil {
		return nil, err
	}

	tx, err :=
		database.DB.Begin()

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro iniciando transação do kit inicial: %w",
				err,
			)
	}

	defer tx.Rollback()

	// Garante que o jogador RPG exista.
	//
	// Essa função também preserva nossa regra:
	// sem carteira Gold no grupo, não existe
	// personagem RPG.
	if _, err :=
		ensurePlayerTx(
			tx,
			groupJID,
			jid,
		); err != nil {

		return nil, err
	}

	alreadyClaimed, err :=
		hasStarterClaimTx(
			tx,
			groupJID,
			jid,
		)

	if err != nil {
		return nil, err
	}

	if alreadyClaimed {
		return nil,
			ErrStarterAlreadyClaimed
	}

	// Garante uma unidade de cada equipamento
	// no inventário.
	//
	// Se por algum motivo o jogador já possuir
	// um desses itens, não cria uma duplicata.
	for _, itemID := range StarterEquipmentIDs() {

		if err :=
			ensureStarterInventoryItemTx(
				tx,
				groupJID,
				jid,
				itemID,
			); err != nil {

			return nil, err
		}
	}

	// Equipa automaticamente o conjunto completo.
	if err :=
		equipStarterItemTx(
			tx,
			groupJID,
			jid,
			SlotWeapon,
			weapon.ID,
		); err != nil {

		return nil, err
	}

	if err :=
		equipStarterItemTx(
			tx,
			groupJID,
			jid,
			SlotShield,
			shield.ID,
		); err != nil {

		return nil, err
	}

	if err :=
		equipStarterItemTx(
			tx,
			groupJID,
			jid,
			SlotArmor,
			armor.ID,
		); err != nil {

		return nil, err
	}

	claimedAt :=
		time.Now().
			UTC().
			Truncate(time.Second)

	if err :=
		createStarterClaimTx(
			tx,
			groupJID,
			jid,
			claimedAt,
		); err != nil {

		return nil, err
	}

	if err :=
		tx.Commit(); err != nil {

		return nil,
			fmt.Errorf(
				"erro confirmando kit inicial RPG: %w",
				err,
			)
	}

	attack :=
		weapon.Attack +
			shield.Attack +
			armor.Attack

	defense :=
		weapon.Defense +
			shield.Defense +
			armor.Defense

	combatPower :=
		weapon.Power +
			shield.Power +
			armor.Power

	return &StarterSetResult{
		Weapon: weapon,

		Shield: shield,

		Armor: armor,

		Attack: attack,

		Defense: defense,

		CombatPower: combatPower,

		ClaimedAt: claimedAt,
	}, nil
}

func validateStarterEquipment(
	catalog *Catalog,
) (
	Item,
	Item,
	Item,
	error,
) {
	weapon, exists :=
		catalog.ItemByID(
			StarterRecruitWeaponID,
		)

	if !exists {
		return Item{},
			Item{},
			Item{},
			fmt.Errorf(
				"equipamento inicial não encontrado: %s",
				StarterRecruitWeaponID,
			)
	}

	if weapon.Type !=
		ItemTypeWeapon {

		return Item{},
			Item{},
			Item{},
			fmt.Errorf(
				"%s não é uma arma",
				StarterRecruitWeaponID,
			)
	}

	shield, exists :=
		catalog.ItemByID(
			StarterRecruitShieldID,
		)

	if !exists {
		return Item{},
			Item{},
			Item{},
			fmt.Errorf(
				"equipamento inicial não encontrado: %s",
				StarterRecruitShieldID,
			)
	}

	if shield.Type !=
		ItemTypeShield {

		return Item{},
			Item{},
			Item{},
			fmt.Errorf(
				"%s não é um escudo",
				StarterRecruitShieldID,
			)
	}

	armor, exists :=
		catalog.ItemByID(
			StarterRecruitArmorID,
		)

	if !exists {
		return Item{},
			Item{},
			Item{},
			fmt.Errorf(
				"equipamento inicial não encontrado: %s",
				StarterRecruitArmorID,
			)
	}

	if armor.Type !=
		ItemTypeArmor {

		return Item{},
			Item{},
			Item{},
			fmt.Errorf(
				"%s não é uma armadura",
				StarterRecruitArmorID,
			)
	}

	if weapon.Rarity !=
		RarityWorn {

		return Item{},
			Item{},
			Item{},
			fmt.Errorf(
				"%s deveria possuir raridade WORN",
				StarterRecruitWeaponID,
			)
	}

	if shield.Rarity !=
		RarityWorn {

		return Item{},
			Item{},
			Item{},
			fmt.Errorf(
				"%s deveria possuir raridade WORN",
				StarterRecruitShieldID,
			)
	}

	if armor.Rarity !=
		RarityWorn {

		return Item{},
			Item{},
			Item{},
			fmt.Errorf(
				"%s deveria possuir raridade WORN",
				StarterRecruitArmorID,
			)
	}

	return weapon,
		shield,
		armor,
		nil
}

func hasStarterClaimTx(
	tx *sql.Tx,
	groupJID string,
	jid string,
) (bool, error) {
	var exists int

	err :=
		tx.QueryRow(`
			SELECT 1
			FROM rpg_starter_claims
			WHERE group_jid = ?
			  AND jid = ?
			LIMIT 1
		`,
			groupJID,
			jid,
		).Scan(
			&exists,
		)

	if errors.Is(
		err,
		sql.ErrNoRows,
	) {
		return false, nil
	}

	if err != nil {
		return false,
			fmt.Errorf(
				"erro verificando claim do kit inicial: %w",
				err,
			)
	}

	return true, nil
}

func ensureStarterInventoryItemTx(
	tx *sql.Tx,
	groupJID string,
	jid string,
	itemID string,
) error {
	_, err :=
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
					CASE
						WHEN rpg_inventory.quantity < 1
							THEN 1
						ELSE rpg_inventory.quantity
					END,

				updated_at =
					CURRENT_TIMESTAMP
		`,
			groupJID,
			jid,
			itemID,
		)

	if err != nil {
		return fmt.Errorf(
			"erro adicionando equipamento inicial %s: %w",
			itemID,
			err,
		)
	}

	return nil
}

func equipStarterItemTx(
	tx *sql.Tx,
	groupJID string,
	jid string,
	slot EquipmentSlot,
	itemID string,
) error {
	if !slot.Valid() {
		return fmt.Errorf(
			"%w: %s",
			ErrInvalidEquipmentSlot,
			slot,
		)
	}

	_, err :=
		tx.Exec(`
			INSERT INTO rpg_equipment (
				group_jid,
				jid,
				slot,
				item_id
			)
			VALUES (?, ?, ?, ?)

			ON CONFLICT (
				group_jid,
				jid,
				slot
			)
			DO UPDATE SET
				item_id =
					excluded.item_id,

				equipped_at =
					CURRENT_TIMESTAMP
		`,
			groupJID,
			jid,
			slot,
			itemID,
		)

	if err != nil {
		return fmt.Errorf(
			"erro equipando item inicial %s no slot %s: %w",
			itemID,
			slot,
			err,
		)
	}

	return nil
}

// createStarterClaimTx registra o claim dentro
// da transação existente.
//
// Deve ser executado somente depois que inventário
// e equipamentos tiverem sido preparados.
func createStarterClaimTx(
	tx *sql.Tx,
	groupJID string,
	jid string,
	claimedAt time.Time,
) error {
	if tx == nil {
		return errors.New(
			"transação do kit inicial não inicializada",
		)
	}

	result, err :=
		tx.Exec(`
			INSERT INTO rpg_starter_claims (
				group_jid,
				jid,
				claimed_at
			)
			VALUES (?, ?, ?)

			ON CONFLICT (
				group_jid,
				jid
			)
			DO NOTHING
		`,
			groupJID,
			jid,
			claimedAt.UTC(),
		)

	if err != nil {
		return fmt.Errorf(
			"erro registrando claim do kit inicial: %w",
			err,
		)
	}

	rowsAffected, err :=
		result.RowsAffected()

	if err != nil {
		return fmt.Errorf(
			"erro verificando claim do kit inicial: %w",
			err,
		)
	}

	if rowsAffected == 0 {
		return ErrStarterAlreadyClaimed
	}

	return nil
}

func ensureStarterSchema() error {
	// Precisamos primeiro das tabelas base RPG.
	if err := ensureSchema(); err != nil {
		return err
	}

	if database.DB == nil {
		return errors.New(
			"banco de dados não inicializado",
		)
	}

	_, err :=
		database.DB.Exec(`
			CREATE TABLE IF NOT EXISTS rpg_starter_claims (
				group_jid TEXT NOT NULL,
				jid TEXT NOT NULL,

				claimed_at DATETIME NOT NULL
					DEFAULT CURRENT_TIMESTAMP,

				PRIMARY KEY (
					group_jid,
					jid
				),

				FOREIGN KEY (
					group_jid,
					jid
				)
					REFERENCES rpg_players (
						group_jid,
						jid
					)
					ON DELETE CASCADE
			);
		`)

	if err != nil {
		return fmt.Errorf(
			"erro criando tabela rpg_starter_claims: %w",
			err,
		)
	}

	return nil
}
