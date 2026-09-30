package rpg

import (
	"testing"
	"time"
)

func TestPVEBatchPenaltyAddsSeconds(t *testing.T) {
	got := addPVEBatchPenalty(10*time.Second, 3)
	want := 13 * time.Second

	if got != want {
		t.Fatalf("penalidade esperada %s, recebida %s", want, got)
	}
}

func TestPVEBatchPenaltyCap(t *testing.T) {
	got := PVEBatchRecoveryCap - time.Second
	got = addPVEBatchPenalty(got, 3)

	if got != PVEBatchRecoveryCap {
		t.Fatalf("cap esperado %s, recebido %s", PVEBatchRecoveryCap, got)
	}
}
