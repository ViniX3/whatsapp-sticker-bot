package rpg

import "testing"

func TestPVEWinChanceEqualPower(
	t *testing.T,
) {
	got :=
		PVEWinChance(
			1000,
			1000,
			false,
		)

	if got != 50 {
		t.Fatalf(
			"chance esperada 50%%, recebida %d%%",
			got,
		)
	}
}

func TestPVEWinChanceVeryStrongEnemy(
	t *testing.T,
) {
	got :=
		PVEWinChance(
			500,
			1500,
			false,
		)

	if got > 5 {
		t.Fatalf(
			"monstro >= 2x PC deveria ter chance máxima de 5%%, recebida %d%%",
			got,
		)
	}
}

func TestPVEBossVeryStrong(
	t *testing.T,
) {
	got :=
		PVEWinChance(
			1000,
			3000,
			true,
		)

	if got !=
		PVEBossMinWinChancePercent {

		t.Fatalf(
			"boss muito superior deveria ter %d%%, recebido %d%%",
			PVEBossMinWinChancePercent,
			got,
		)
	}
}

func TestPVEBossHarderThanNormalEnemy(
	t *testing.T,
) {
	normal :=
		PVEWinChance(
			2000,
			3000,
			false,
		)

	boss :=
		PVEWinChance(
			2000,
			3000,
			true,
		)

	if boss >= normal {
		t.Fatalf(
			"boss deveria ser mais difícil: normal=%d boss=%d",
			normal,
			boss,
		)
	}
}

func TestPVEChanceMaximum(
	t *testing.T,
) {
	got :=
		PVEWinChance(
			100000,
			500,
			false,
		)

	if got !=
		PVEMaxWinChancePercent {

		t.Fatalf(
			"chance máxima esperada %d%%, recebida %d%%",
			PVEMaxWinChancePercent,
			got,
		)
	}
}
