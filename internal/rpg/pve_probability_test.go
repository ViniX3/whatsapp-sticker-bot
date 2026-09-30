package rpg

import "testing"

func TestPVEWinChanceProgression(t *testing.T) {
	tests := []struct {
		name        string
		playerPower int
		enemyPower  int
		want        int
	}{
		{
			name:        "metade do poder",
			playerPower: 250,
			enemyPower:  500,
			want:        5,
		},
		{
			name:        "75 por cento do poder",
			playerPower: 375,
			enemyPower:  500,
			want:        30,
		},
		{
			name:        "poder igual",
			playerPower: 500,
			enemyPower:  500,
			want:        50,
		},
		{
			name:        "25 por cento mais forte",
			playerPower: 625,
			enemyPower:  500,
			want:        65,
		},
		{
			name:        "50 por cento mais forte",
			playerPower: 750,
			enemyPower:  500,
			want:        80,
		},
		{
			name:        "dobro do poder",
			playerPower: 1000,
			enemyPower:  500,
			want:        95,
		},
		{
			name:        "triplo do poder",
			playerPower: 1500,
			enemyPower:  500,
			want:        98,
		},
	}

	for _, test := range tests {
		t.Run(
			test.name,
			func(t *testing.T) {
				got :=
					PVEWinChance(
						test.playerPower,
						test.enemyPower,
						false,
					)

				if got != test.want {
					t.Fatalf(
						"esperado %d%%, encontrado %d%%",
						test.want,
						got,
					)
				}
			},
		)
	}
}

func TestPVEBossWinChance(t *testing.T) {
	tests := []struct {
		name        string
		playerPower int
		enemyPower  int
		want        int
	}{
		{
			name:        "boss com poder igual",
			playerPower: 500,
			enemyPower:  500,
			want:        35,
		},
		{
			name:        "jogador 50 por cento superior",
			playerPower: 750,
			enemyPower:  500,
			want:        65,
		},
		{
			name:        "jogador com dobro do poder",
			playerPower: 1000,
			enemyPower:  500,
			want:        80,
		},
		{
			name:        "jogador extremamente superior",
			playerPower: 1500,
			enemyPower:  500,
			want:        80,
		},
	}

	for _, test := range tests {
		t.Run(
			test.name,
			func(t *testing.T) {
				got :=
					PVEWinChance(
						test.playerPower,
						test.enemyPower,
						true,
					)

				if got != test.want {
					t.Fatalf(
						"esperado %d%%, encontrado %d%%",
						test.want,
						got,
					)
				}
			},
		)
	}
}
