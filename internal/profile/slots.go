package profile

const (
	XPSlots2x   = 5
	XPSlots3x   = 8
	XPSlots5x   = 15
	XPSlots10x  = 25
	XPSlots25x  = 50
	XPSlots50x  = 80
	XPSlots100x = 150
)

func RecordSlotResult(
	groupJID string,
	jid string,
	multiplier int,
	prize int,
) (*XPResult, error) {
	delta := progressDelta{
		SlotsPlayed: 1,
	}

	switch multiplier {
	case 2:
		delta.SlotsWon = 1
		delta.SlotsBiggestPrize = prize
		delta.XP = XPSlots2x

	case 3:
		delta.SlotsWon = 1
		delta.SlotsBiggestPrize = prize
		delta.XP = XPSlots3x

	case 5:
		delta.SlotsWon = 1
		delta.SlotsBiggestPrize = prize
		delta.XP = XPSlots5x

	case 10:
		delta.SlotsWon = 1
		delta.SlotsBiggestPrize = prize
		delta.XP = XPSlots10x

	case 25:
		delta.SlotsWon = 1
		delta.SlotsBiggestPrize = prize
		delta.XP = XPSlots25x

	case 50:
		delta.SlotsWon = 1
		delta.SlotsBiggestPrize = prize
		delta.XP = XPSlots50x

	case 100:
		delta.SlotsWon = 1
		delta.SlotsBiggestPrize = prize
		delta.XP = XPSlots100x
	}

	return applyProgressDelta(
		groupJID,
		jid,
		delta,
	)
}
