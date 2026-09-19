package profile

const (
	XPDuelWin  = 40
	XPDuelLoss = 10
)

func RecordDuelResult(
	groupJID string,
	jid string,
	won bool,
) (*XPResult, error) {
	delta :=
		progressDelta{}

	if won {
		delta.DuelsWon = 1
		delta.XP = XPDuelWin
	} else {
		delta.DuelsLost = 1
		delta.XP = XPDuelLoss
	}

	return applyProgressDelta(
		groupJID,
		jid,
		delta,
	)
}
