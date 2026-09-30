package rpg

// raidProgressionLegendaryBossIDs contém os Raid Bosses
// Legendary pertencentes à progressão ativa de Raid.
//
// Os demais Legendary continuam no catálogo porque seus sets
// pertencem a outras fontes de progressão:
//   - Dungeon
//   - Mercado Arcano
//
// Isso evita que o Raid invalide essas etapas anteriores.
var raidProgressionLegendaryBossIDs = map[string]struct{}{
	"crimson_chimera": {},
	"ash_cerberus":    {},
}

// RaidBossEnabledForProgression informa se o Boss pertence
// ao pool normal da progressão atual de Raid.
//
// Mythic e Sacred pertencem integralmente ao Raid.
// Legendary é restrito ao conjunto específico de endgame.
func RaidBossEnabledForProgression(
	boss RaidBoss,
) bool {
	switch boss.Rarity {
	case RarityLegendary:
		_, exists :=
			raidProgressionLegendaryBossIDs[boss.ID]

		return exists

	case RarityMythic,
		RaritySacred:
		return true

	default:
		return false
	}
}
