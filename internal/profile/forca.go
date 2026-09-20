package profile

const XPForcaWin = 30

func RecordForcaWin(
	groupJID string,
	jid string,
) (*XPResult, error) {
	return applyProgressDelta(
		groupJID,
		jid,
		progressDelta{
			XP: XPForcaWin,
		},
	)
}
