package handler

import (
	"database/sql"
	"fmt"
	"strings"

	"whatsapp-sticker-bot/internal/database"
	"whatsapp-sticker-bot/internal/logger"
	"whatsapp-sticker-bot/internal/rpg"
	"whatsapp-sticker-bot/internal/whatsapp"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
)

const forgeNotificationMaxRecipes = 3

func notifyForgeReadyRecipes(
	client *whatsmeow.Client,
	msgEvent *events.Message,
	groupJID string,
	jid string,
	catalog *rpg.Catalog,
	materials *rpg.MaterialCatalog,
) {
	if client == nil ||
		msgEvent == nil ||
		catalog == nil ||
		materials == nil {

		return
	}

	if err :=
		ensureForgeNotificationSchema(); err != nil {

		logger.Warn(
			"Não foi possível inicializar notificações da Forja:",
			err,
		)

		return
	}

	recipes, err :=
		rpg.ListForgeRecipes(
			catalog,
			rpg.RarityEpic,
		)

	if err != nil {
		logger.Warn(
			"Não foi possível listar receitas Épicas para notificação:",
			err,
		)

		return
	}

	type readyRecipe struct {
		Code string
		Name string
	}

	ready :=
		make(
			[]readyRecipe,
			0,
			forgeNotificationMaxRecipes,
		)

	for _, recipe := range recipes {

		alreadyNotified, err :=
			forgeRecipeAlreadyNotified(
				groupJID,
				jid,
				recipe.Code,
			)

		if err != nil {
			logger.Warn(
				"Erro consultando histórico de notificação da Forja:",
				err,
			)

			continue
		}

		if alreadyNotified {
			continue
		}

		check, err :=
			rpg.CheckForgeRequirements(
				groupJID,
				jid,
				recipe.Code,
				catalog,
				materials,
			)

		if err != nil {
			continue
		}

		if !check.CanForge ||
			check.UniqueAlreadyOwned {

			continue
		}

		isUpgrade, err :=
			forgeRecipeIsUpgrade(
				groupJID,
				jid,
				recipe.Item,
				catalog,
			)

		if err != nil {
			logger.Warn(
				"Erro comparando receita da Forja com equipamento atual:",
				recipe.Code,
				err,
			)

			continue
		}

		if !isUpgrade {
			continue
		}

		ready =
			append(
				ready,
				readyRecipe{
					Code: recipe.Code,
					Name: recipe.Item.Name,
				},
			)

		if len(ready) >=
			forgeNotificationMaxRecipes {

			break
		}
	}

	if len(ready) == 0 {
		return
	}

	sender :=
		msgEvent.Info.Sender.
			ToNonAD()

	name :=
		strings.TrimSpace(
			msgEvent.Info.PushName,
		)

	if name == "" {
		name =
			strings.TrimSpace(
				sender.User,
			)
	}

	if name == "" {
		name =
			"aventureiro"
	}

	var builder strings.Builder

	fmt.Fprintf(
		&builder,
		"🔥 *@%s*, sua Forja está pronta!\n",
		name,
	)

	for _, recipe := range ready {

		fmt.Fprintf(
			&builder,
			"🟣 *%s* • %s\n",
			recipe.Code,
			recipe.Name,
		)
	}

	if len(ready) == 1 {
		fmt.Fprintf(
			&builder,
			"⚒️ *!forjar %s*",
			ready[0].Code,
		)
	} else {
		builder.WriteString(
			"⚒️ Use *!forja epico* para conferir.",
		)
	}

	err =
		whatsapp.SendMentionedText(
			client,
			msgEvent.Info.Chat,
			strings.TrimSpace(
				builder.String(),
			),
			[]types.JID{
				sender,
			},
		)

	if err != nil {
		logger.Warn(
			"Erro enviando notificação inteligente da Forja:",
			err,
		)

		return
	}

	for _, recipe := range ready {

		if err :=
			markForgeRecipeNotified(
				groupJID,
				jid,
				recipe.Code,
			); err != nil {

			logger.Warn(
				"Notificação da Forja enviada, mas não registrada:",
				recipe.Code,
				err,
			)

			continue
		}

		logger.Success(
			"Receita Épica notificada:",
			recipe.Code,
			"Jogador:",
			jid,
			"Grupo:",
			groupJID,
		)
	}
}

func ensureForgeNotificationSchema() error {
	if database.DB == nil {
		return fmt.Errorf(
			"banco de dados não inicializado",
		)
	}

	_, err :=
		database.DB.Exec(`
			CREATE TABLE IF NOT EXISTS rpg_forge_notifications (
				group_jid TEXT NOT NULL,
				jid TEXT NOT NULL,
				recipe_code TEXT NOT NULL,
				notified_at DATETIME NOT NULL
					DEFAULT CURRENT_TIMESTAMP,

				PRIMARY KEY (
					group_jid,
					jid,
					recipe_code
				)
			)
		`)

	if err != nil {
		return fmt.Errorf(
			"erro criando tabela de notificações da Forja: %w",
			err,
		)
	}

	return nil
}

func forgeRecipeAlreadyNotified(
	groupJID string,
	jid string,
	recipeCode string,
) (
	bool,
	error,
) {
	var marker int

	err :=
		database.DB.QueryRow(`
			SELECT 1
			FROM rpg_forge_notifications
			WHERE group_jid = ?
			  AND jid = ?
			  AND recipe_code = ?
			LIMIT 1
		`,
			groupJID,
			jid,
			recipeCode,
		).Scan(
			&marker,
		)

	if err == sql.ErrNoRows {
		return false,
			nil
	}

	if err != nil {
		return false,
			err
	}

	return true,
		nil
}

func markForgeRecipeNotified(
	groupJID string,
	jid string,
	recipeCode string,
) error {
	_, err :=
		database.DB.Exec(`
			INSERT OR IGNORE INTO rpg_forge_notifications (
				group_jid,
				jid,
				recipe_code
			)
			VALUES (?, ?, ?)
		`,
			groupJID,
			jid,
			recipeCode,
		)

	return err
}

func forgeRecipeIsUpgrade(
	groupJID string,
	jid string,
	item rpg.Item,
	catalog *rpg.Catalog,
) (
	bool,
	error,
) {
	summary, err :=
		rpg.GetEquipmentSummary(
			groupJID,
			jid,
			catalog,
		)

	if err != nil {
		return false,
			err
	}

	var current *rpg.Item

	switch item.Type {
	case rpg.ItemTypeWeapon:
		current =
			summary.Weapon

	case rpg.ItemTypeShield:
		current =
			summary.Shield

	case rpg.ItemTypeArmor:
		current =
			summary.Armor

	default:
		return false,
			nil
	}

	if current == nil {
		return true,
			nil
	}

	return item.Power >
			current.Power,
		nil
}
