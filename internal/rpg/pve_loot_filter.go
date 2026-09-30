package rpg

func pveEquipmentEligibleForEnemy(
	enemy Enemy,
	item Item,
) bool {
	if item.Rarity !=
		enemy.Rarity {

		return false
	}

	// Equipamentos do Mercador nunca são loot PvE.
	if item.Source ==
		SourceOtherworldMerchant {

		return false
	}

	if enemy.Boss {
		// Boss PvE só pode entregar um BOSS_DROP
		// criado especificamente para ele.
		return item.Source ==
			SourceBossDrop &&
			item.BossID ==
				enemy.ID
	}

	// BOSS_DROP pertence exclusivamente a Bosses.
	// Nunca deve cair de inimigos PvE comuns.
	if item.Source ==
		SourceBossDrop {

		return false
	}

	return true
}
