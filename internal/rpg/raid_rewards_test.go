package rpg

import (
	"testing"
	"time"

	"whatsapp-sticker-bot/internal/database"
)

func setupRaidRewardWalletTables(
	t *testing.T,
) {
	t.Helper()

	_, err :=
		database.DB.Exec(
			`
			CREATE TABLE IF NOT EXISTS group_wallets (
				group_jid TEXT NOT NULL,
				jid TEXT NOT NULL,
				name TEXT NOT NULL DEFAULT '',
				gold INTEGER NOT NULL DEFAULT 0,
				created_at DATETIME NOT NULL
					DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME NOT NULL
					DEFAULT CURRENT_TIMESTAMP,

				PRIMARY KEY (
					group_jid,
					jid
				)
			);

			CREATE TABLE IF NOT EXISTS gold_transactions (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				group_jid TEXT NOT NULL,
				jid TEXT NOT NULL,
				related_jid TEXT,
				amount INTEGER NOT NULL,
				type TEXT NOT NULL,
				description TEXT NOT NULL,
				created_at DATETIME NOT NULL
					DEFAULT CURRENT_TIMESTAMP
			);
			`,
		)

	if err != nil {
		t.Fatal(err)
	}
}

func prepareWinningRaidForRewards(
	t *testing.T,
) int64 {
	t.Helper()

	setupRaidHistoryTestDB(t)

	setupRaidRewardWalletTables(t)

	catalog, err :=
		LoadRaidBossCatalog()

	if err != nil {
		t.Fatal(err)
	}

	boss :=
		mustRaidBoss(
			t,
			catalog,
			"black_dragon",
		)

	now :=
		time.Date(
			2026,
			9,
			30,
			12,
			0,
			0,
			0,
			time.UTC,
		)

	session, err :=
		CreateRaidSession(
			"group-a",
			boss,
			now,
		)

	if err != nil {
		t.Fatal(err)
	}

	for index := 0; index < 2; index++ {

		jid :=
			fmtTestRaidJID(
				index,
			)

		_, err :=
			database.DB.Exec(
				`
				INSERT INTO group_wallets (
					group_jid,
					jid,
					name,
					gold
				)
				VALUES (?, ?, ?, 0)
				`,
				"group-a",
				jid,
				jid,
			)

		if err != nil {
			t.Fatal(err)
		}

		err =
			JoinActiveRaidSession(
				"group-a",
				RaidParticipant{
					JID: jid,

					Name: jid,

					CombatPower: 12000,
				},
				now.Add(
					time.Second,
				),
			)

		if err != nil {
			t.Fatal(err)
		}
	}

	_, err =
		ResolveRaidSession(
			session.ID,
			session.StartsAt,
			1,
		)

	if err != nil {
		t.Fatal(err)
	}

	return session.ID
}

func TestRaidRewardConfigs(
	t *testing.T,
) {
	tests :=
		[]Rarity{
			RarityLegendary,
			RarityMythic,
			RaritySacred,
		}

	for _, rarity := range tests {

		config, exists :=
			RaidRewardConfigForRarity(
				rarity,
			)

		if !exists {
			t.Fatalf(
				"config inexistente para %s",
				rarity,
			)
		}

		if config.GoldMin <= 0 ||
			config.GoldMax <
				config.GoldMin {

			t.Fatalf(
				"Gold inválido para %s",
				rarity,
			)
		}

		if config.CrystalMin <= 0 ||
			config.CrystalMax <
				config.CrystalMin {

			t.Fatalf(
				"Cristais inválidos para %s",
				rarity,
			)
		}
	}
}

func TestApplyRaidRewards(
	t *testing.T,
) {
	raidID :=
		prepareWinningRaidForRewards(
			t,
		)

	catalog, err :=
		LoadCatalog()

	if err != nil {
		t.Fatal(err)
	}

	rewards, err :=
		ApplyRaidRewards(
			raidID,
			catalog,
		)

	if err != nil {
		t.Fatal(err)
	}

	if len(rewards) != 2 {
		t.Fatalf(
			"esperadas 2 recompensas, recebidas %d",
			len(rewards),
		)
	}

	config, _ :=
		RaidRewardConfigForRarity(
			RarityLegendary,
		)

	for _, reward := range rewards {

		if reward.Gold <
			config.GoldMin ||
			reward.Gold >
				config.GoldMax {

			t.Fatalf(
				"Gold fora do intervalo: %d",
				reward.Gold,
			)
		}

		if reward.MagicCrystals <
			config.CrystalMin ||
			reward.MagicCrystals >
				config.CrystalMax {

			t.Fatalf(
				"Cristais fora do intervalo: %d",
				reward.MagicCrystals,
			)
		}
	}

	stored, err :=
		RaidRewardsForRaid(
			raidID,
		)

	if err != nil {
		t.Fatal(err)
	}

	if len(stored) != 2 {
		t.Fatalf(
			"esperadas 2 recompensas persistidas, recebidas %d",
			len(stored),
		)
	}
}

func TestRaidRewardsAreIdempotent(
	t *testing.T,
) {
	raidID :=
		prepareWinningRaidForRewards(
			t,
		)

	catalog, err :=
		LoadCatalog()

	if err != nil {
		t.Fatal(err)
	}

	first, err :=
		ApplyRaidRewards(
			raidID,
			catalog,
		)

	if err != nil {
		t.Fatal(err)
	}

	second, err :=
		ApplyRaidRewards(
			raidID,
			catalog,
		)

	if err != nil {
		t.Fatal(err)
	}

	if len(first) != 2 {
		t.Fatalf(
			"primeira aplicação deveria criar 2 recompensas, criou %d",
			len(first),
		)
	}

	if len(second) != 0 {
		t.Fatalf(
			"segunda aplicação deveria criar 0 recompensas, criou %d",
			len(second),
		)
	}

	var transactionCount int

	err =
		database.DB.QueryRow(
			`
			SELECT COUNT(*)
			FROM gold_transactions
			WHERE type = 'RPG_RAID_REWARD'
			`,
		).Scan(
			&transactionCount,
		)

	if err != nil {
		t.Fatal(err)
	}

	if transactionCount != 2 {
		t.Fatalf(
			"esperadas 2 transações Gold, recebidas %d",
			transactionCount,
		)
	}
}

func TestPendingRaidRewardsDisappearAfterApply(
	t *testing.T,
) {
	raidID :=
		prepareWinningRaidForRewards(
			t,
		)

	pending, err :=
		PendingRaidRewardIDs(
			10,
		)

	if err != nil {
		t.Fatal(err)
	}

	if len(pending) != 1 ||
		pending[0] !=
			raidID {

		t.Fatalf(
			"Raid pendente inesperada: %v",
			pending,
		)
	}

	catalog, err :=
		LoadCatalog()

	if err != nil {
		t.Fatal(err)
	}

	if _, err :=
		ApplyRaidRewards(
			raidID,
			catalog,
		); err != nil {

		t.Fatal(err)
	}

	pending, err =
		PendingRaidRewardIDs(
			10,
		)

	if err != nil {
		t.Fatal(err)
	}

	if len(pending) != 0 {
		t.Fatalf(
			"não deveriam restar recompensas pendentes: %v",
			pending,
		)
	}
}
