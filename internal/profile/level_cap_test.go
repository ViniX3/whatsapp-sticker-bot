package profile

import "testing"

func TestMaxPlayerLevel(
	t *testing.T,
) {

	info :=
		LevelFromXP(
			100000000,
		)

	if info.Level != MaxPlayerLevel {

		t.Fatalf(
			"esperado nível máximo %d, recebido %d",
			MaxPlayerLevel,
			info.Level,
		)
	}

	if info.ProgressPercent != 100 {

		t.Fatalf(
			"nível máximo deveria estar em 100%%, recebido %d%%",
			info.ProgressPercent,
		)
	}
}
