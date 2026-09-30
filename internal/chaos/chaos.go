package chaos

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"sync"
	"time"
)

const (
	// Cada Presságio permanece ativo entre 5 e 10 minutos.
	MinDuration = 5 * time.Minute
	MaxDuration = 10 * time.Minute

	// O intervalo foi calibrado para aproximadamente
	// 10 Presságios a cada 24 horas.
	//
	// Considerando também os 5-10 minutos do evento,
	// o ciclo médio fica próximo de 142 minutos.
	MinInterval = 105 * time.Minute
	MaxInterval = 165 * time.Minute

	MinBonusPercent = 25
	MaxBonusPercent = 100
)

type Event struct {
	ID uint64

	BonusPercent int

	StartedAt time.Time

	EndsAt time.Time
}

type State struct {
	Active bool

	Event Event

	NextStartsAt time.Time
}

var manager = struct {
	sync.RWMutex

	active bool

	event Event

	nextStartsAt time.Time

	nextID uint64
}{}

// Snapshot retorna uma cópia segura do estado global.
func Snapshot() State {
	manager.RLock()
	defer manager.RUnlock()

	return State{
		Active: manager.active,

		Event: manager.event,

		NextStartsAt: manager.nextStartsAt,
	}
}

func IsActive() bool {
	manager.RLock()
	defer manager.RUnlock()

	return manager.active
}

// ApplyGoldReward aplica o bônus do Presságio a uma
// recompensa POSITIVA.
//
// IMPORTANTE:
//
// Para jogos de aposta, o chamador deve fornecer apenas
// o lucro/recompensa positiva e não o valor originalmente
// apostado.
//
// Exemplo:
//
//	aposta: 1.000
//	retorno: 1.500
//	lucro base: 500
//
// O Presságio deve ser aplicado sobre os 500.
//
// Retorna:
//
//	total
//	bonus
//	bonusPercent
//	active
func ApplyGoldReward(
	baseReward int,
) (
	int,
	int,
	int,
	bool,
) {
	if baseReward <= 0 {
		return baseReward,
			0,
			0,
			false
	}

	state := Snapshot()

	if !state.Active {
		return baseReward,
			0,
			0,
			false
	}

	bonus := int((int64(baseReward) * int64(state.Event.BonusPercent)) / 100)

	if bonus < 1 {
		bonus = 1
	}

	return baseReward + bonus,
		bonus,
		state.Event.BonusPercent,
		true
}

// RunScheduler executa continuamente o sistema.
//
// onStart e onEnd são chamados fora do lock interno,
// permitindo anúncios no WhatsApp sem bloquear consultas.
func RunScheduler(
	onStart func(Event),
	onEnd func(Event),
) error {
	for {
		wait, err :=
			randomDuration(
				MinInterval,
				MaxInterval,
			)

		if err != nil {
			return fmt.Errorf(
				"erro sorteando próximo Presságio: %w",
				err,
			)
		}

		next :=
			time.Now().Add(wait)

		manager.Lock()

		manager.nextStartsAt =
			next

		manager.Unlock()

		timer :=
			time.NewTimer(wait)

		<-timer.C

		event, err :=
			startEvent()

		if err != nil {
			return err
		}

		if onStart != nil {
			onStart(event)
		}

		duration :=
			time.Until(
				event.EndsAt,
			)

		if duration < 0 {
			duration = 0
		}

		timer =
			time.NewTimer(
				duration,
			)

		<-timer.C

		finished :=
			finishEvent()

		if onEnd != nil {
			onEnd(finished)
		}
	}
}

func startEvent() (
	Event,
	error,
) {
	duration, err :=
		randomDuration(
			MinDuration,
			MaxDuration,
		)

	if err != nil {
		return Event{},
			fmt.Errorf(
				"erro sorteando duração do Presságio: %w",
				err,
			)
	}

	bonus, err :=
		randomIntInclusive(
			MinBonusPercent,
			MaxBonusPercent,
		)

	if err != nil {
		return Event{},
			fmt.Errorf(
				"erro sorteando bônus do Presságio: %w",
				err,
			)
	}

	now := time.Now()

	manager.Lock()
	defer manager.Unlock()

	manager.nextID++

	event :=
		Event{
			ID: manager.nextID,

			BonusPercent: bonus,

			StartedAt: now,

			EndsAt: now.Add(duration),
		}

	manager.active = true

	manager.event =
		event

	manager.nextStartsAt =
		time.Time{}

	return event, nil
}

func finishEvent() Event {
	manager.Lock()
	defer manager.Unlock()

	event :=
		manager.event

	manager.active = false

	manager.event =
		Event{}

	return event
}

func randomDuration(
	minimum time.Duration,
	maximum time.Duration,
) (
	time.Duration,
	error,
) {
	if minimum <= 0 ||
		maximum < minimum {

		return 0,
			fmt.Errorf(
				"intervalo inválido: %s - %s",
				minimum,
				maximum,
			)
	}

	minMinutes :=
		int(
			minimum /
				time.Minute,
		)

	maxMinutes :=
		int(
			maximum /
				time.Minute,
		)

	value, err :=
		randomIntInclusive(
			minMinutes,
			maxMinutes,
		)

	if err != nil {
		return 0, err
	}

	return time.Duration(value) *
			time.Minute,
		nil
}

func randomIntInclusive(
	minimum int,
	maximum int,
) (
	int,
	error,
) {
	if maximum < minimum {
		return 0,
			fmt.Errorf(
				"intervalo aleatório inválido: %d-%d",
				minimum,
				maximum,
			)
	}

	size :=
		maximum -
			minimum +
			1

	value, err :=
		rand.Int(
			rand.Reader,
			big.NewInt(
				int64(size),
			),
		)

	if err != nil {
		return 0, err
	}

	return minimum +
			int(
				value.Int64(),
			),
		nil
}
