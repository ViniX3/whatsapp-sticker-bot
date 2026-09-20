package lottery

import (
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"math/big"

	"whatsapp-sticker-bot/internal/database"
)

const (
	TicketPrice         = 500
	MaxTickets          = 100
	MaxTicketsPerPlayer = 20
	JackpotPercent      = 95

	RoundStatusOpen     = "OPEN"
	RoundStatusFinished = "FINISHED"
)

var (
	ErrInvalidQuantity = errors.New(
		"quantidade de bilhetes inválida",
	)

	ErrWalletNotFound = errors.New(
		"carteira não encontrada",
	)

	ErrInsufficientGold = errors.New(
		"gold insuficiente",
	)

	ErrPlayerTicketLimit = errors.New(
		"limite de bilhetes por jogador atingido",
	)

	ErrNotEnoughTickets = errors.New(
		"não há bilhetes suficientes disponíveis",
	)

	ErrRoundNotFound = errors.New(
		"rodada da loteria não encontrada",
	)
)

type Round struct {
	GroupJID string

	RoundNumber int
	TicketsSold int
	Jackpot     int

	Status string

	WinnerJID     string
	WinningTicket int
}

type Status struct {
	Round Round

	UserTickets int

	RemainingTickets int
	UserCanBuy       int

	ChancePercent float64
}

type PurchaseResult struct {
	RoundNumber int

	Quantity int
	Cost     int

	TicketFrom int
	TicketTo   int

	UserTickets int

	TicketsSold      int
	RemainingTickets int

	Jackpot int
	Balance int

	ChancePercent float64

	Drawn bool

	WinningTicket int
	WinnerJID     string

	Prize         int
	WinnerBalance int

	NextRoundNumber int
}

func Init() error {
	return ensureSchema()
}

func GetStatus(
	groupJID string,
	jid string,
) (*Status, error) {
	if err := ensureSchema(); err != nil {
		return nil, err
	}

	tx, err :=
		database.DB.Begin()

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro iniciando consulta da loteria: %w",
				err,
			)
	}

	defer tx.Rollback()

	round, err :=
		getOrCreateOpenRoundTx(
			tx,
			groupJID,
		)

	if err != nil {
		return nil, err
	}

	userTickets, err :=
		countPlayerTicketsTx(
			tx,
			groupJID,
			round.RoundNumber,
			jid,
		)

	if err != nil {
		return nil, err
	}

	if err :=
		tx.Commit(); err != nil {

		return nil,
			fmt.Errorf(
				"erro confirmando consulta da loteria: %w",
				err,
			)
	}

	remaining :=
		MaxTickets -
			round.TicketsSold

	userCanBuy :=
		MaxTicketsPerPlayer -
			userTickets

	if userCanBuy < 0 {
		userCanBuy = 0
	}

	if userCanBuy > remaining {
		userCanBuy =
			remaining
	}

	return &Status{
		Round: *round,

		UserTickets: userTickets,

		RemainingTickets: remaining,

		UserCanBuy: userCanBuy,

		ChancePercent: float64(userTickets),
	}, nil
}

func BuyTickets(
	groupJID string,
	jid string,
	quantity int,
) (*PurchaseResult, error) {
	if quantity <= 0 ||
		quantity > MaxTicketsPerPlayer {

		return nil,
			ErrInvalidQuantity
	}

	if err := ensureSchema(); err != nil {
		return nil, err
	}

	tx, err :=
		database.DB.Begin()

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro iniciando compra de bilhetes: %w",
				err,
			)
	}

	defer tx.Rollback()

	round, err :=
		getOrCreateOpenRoundTx(
			tx,
			groupJID,
		)

	if err != nil {
		return nil, err
	}

	currentPlayerTickets, err :=
		countPlayerTicketsTx(
			tx,
			groupJID,
			round.RoundNumber,
			jid,
		)

	if err != nil {
		return nil, err
	}

	if currentPlayerTickets+quantity >
		MaxTicketsPerPlayer {

		return nil,
			ErrPlayerTicketLimit
	}

	remainingTickets :=
		MaxTickets -
			round.TicketsSold

	if quantity > remainingTickets {
		return nil,
			ErrNotEnoughTickets
	}

	cost :=
		quantity *
			TicketPrice

	var currentBalance int

	err =
		tx.QueryRow(`
			SELECT gold
			FROM group_wallets
			WHERE group_jid = ?
			  AND jid = ?
		`,
			groupJID,
			jid,
		).Scan(
			&currentBalance,
		)

	if errors.Is(
		err,
		sql.ErrNoRows,
	) {
		return nil,
			ErrWalletNotFound
	}

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro consultando carteira para loteria: %w",
				err,
			)
	}

	if currentBalance < cost {
		return nil,
			ErrInsufficientGold
	}

	result, err :=
		tx.Exec(`
			UPDATE group_wallets
			SET
				gold = gold - ?,
				updated_at = CURRENT_TIMESTAMP
			WHERE group_jid = ?
			  AND jid = ?
			  AND gold >= ?
		`,
			cost,
			groupJID,
			jid,
			cost,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro debitando compra da loteria: %w",
				err,
			)
	}

	rowsAffected, err :=
		result.RowsAffected()

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro validando débito da loteria: %w",
				err,
			)
	}

	if rowsAffected != 1 {
		return nil,
			ErrInsufficientGold
	}

	description :=
		fmt.Sprintf(
			"Compra de %d bilhete(s) da Loteria - rodada %d",
			quantity,
			round.RoundNumber,
		)

	_, err =
		tx.Exec(`
			INSERT INTO gold_transactions (
				group_jid,
				jid,
				related_jid,
				amount,
				type,
				description
			)
			VALUES (?, ?, NULL, ?, ?, ?)
		`,
			groupJID,
			jid,
			-cost,
			"LOTTERY_TICKET_BUY",
			description,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro registrando compra da loteria: %w",
				err,
			)
	}

	firstTicket :=
		round.TicketsSold + 1

	lastTicket :=
		round.TicketsSold +
			quantity

	for ticketNumber :=
		firstTicket; ticketNumber <= lastTicket; ticketNumber++ {

		_, err =
			tx.Exec(`
				INSERT INTO lottery_tickets (
					group_jid,
					round_number,
					ticket_number,
					jid
				)
				VALUES (?, ?, ?, ?)
			`,
				groupJID,
				round.RoundNumber,
				ticketNumber,
				jid,
			)

		if err != nil {
			return nil,
				fmt.Errorf(
					"erro criando bilhete #%d: %w",
					ticketNumber,
					err,
				)
		}
	}

	jackpotContribution :=
		quantity *
			TicketPrice *
			JackpotPercent /
			100

	newTicketsSold :=
		round.TicketsSold +
			quantity

	newJackpot :=
		round.Jackpot +
			jackpotContribution

	_, err =
		tx.Exec(`
			UPDATE lottery_rounds
			SET
				tickets_sold = ?,
				jackpot = ?
			WHERE group_jid = ?
			  AND round_number = ?
			  AND status = ?
		`,
			newTicketsSold,
			newJackpot,
			groupJID,
			round.RoundNumber,
			RoundStatusOpen,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro atualizando rodada da loteria: %w",
				err,
			)
	}

	newBalance :=
		currentBalance -
			cost

	newPlayerTickets :=
		currentPlayerTickets +
			quantity

	purchase :=
		&PurchaseResult{
			RoundNumber: round.RoundNumber,

			Quantity: quantity,

			Cost: cost,

			TicketFrom: firstTicket,

			TicketTo: lastTicket,

			UserTickets: newPlayerTickets,

			TicketsSold: newTicketsSold,

			RemainingTickets: MaxTickets -
				newTicketsSold,

			Jackpot: newJackpot,

			Balance: newBalance,

			ChancePercent: float64(
				newPlayerTickets,
			),
		}

	if newTicketsSold == MaxTickets {
		drawResult, err :=
			drawTx(
				tx,
				groupJID,
				round.RoundNumber,
				newJackpot,
			)

		if err != nil {
			return nil, err
		}

		purchase.Drawn =
			true

		purchase.WinningTicket =
			drawResult.WinningTicket

		purchase.WinnerJID =
			drawResult.WinnerJID

		purchase.Prize =
			drawResult.Prize

		purchase.WinnerBalance =
			drawResult.WinnerBalance

		purchase.NextRoundNumber =
			drawResult.NextRoundNumber

		if drawResult.WinnerJID == jid {
			purchase.Balance =
				drawResult.WinnerBalance
		}
	}

	if err :=
		tx.Commit(); err != nil {

		return nil,
			fmt.Errorf(
				"erro confirmando compra da loteria: %w",
				err,
			)
	}

	return purchase, nil
}

type drawResult struct {
	WinningTicket int
	WinnerJID     string

	Prize         int
	WinnerBalance int

	NextRoundNumber int
}

func drawTx(
	tx *sql.Tx,
	groupJID string,
	roundNumber int,
	jackpot int,
) (*drawResult, error) {
	winningTicket, err :=
		randomTicket()

	if err != nil {
		return nil, err
	}

	var winnerJID string

	err =
		tx.QueryRow(`
			SELECT jid
			FROM lottery_tickets
			WHERE group_jid = ?
			  AND round_number = ?
			  AND ticket_number = ?
		`,
			groupJID,
			roundNumber,
			winningTicket,
		).Scan(
			&winnerJID,
		)

	if errors.Is(
		err,
		sql.ErrNoRows,
	) {
		return nil,
			fmt.Errorf(
				"%w: bilhete vencedor #%d",
				ErrRoundNotFound,
				winningTicket,
			)
	}

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro consultando vencedor da loteria: %w",
				err,
			)
	}

	result, err :=
		tx.Exec(`
			UPDATE group_wallets
			SET
				gold = gold + ?,
				updated_at = CURRENT_TIMESTAMP
			WHERE group_jid = ?
			  AND jid = ?
		`,
			jackpot,
			groupJID,
			winnerJID,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro creditando jackpot da loteria: %w",
				err,
			)
	}

	rowsAffected, err :=
		result.RowsAffected()

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro validando prêmio da loteria: %w",
				err,
			)
	}

	if rowsAffected != 1 {
		return nil,
			fmt.Errorf(
				"carteira do vencedor da loteria não encontrada",
			)
	}

	description :=
		fmt.Sprintf(
			"Jackpot da Loteria - rodada %d - bilhete #%d",
			roundNumber,
			winningTicket,
		)

	_, err =
		tx.Exec(`
			INSERT INTO gold_transactions (
				group_jid,
				jid,
				related_jid,
				amount,
				type,
				description
			)
			VALUES (?, ?, NULL, ?, ?, ?)
		`,
			groupJID,
			winnerJID,
			jackpot,
			"LOTTERY_WIN",
			description,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro registrando prêmio da loteria: %w",
				err,
			)
	}

	_, err =
		tx.Exec(`
			UPDATE lottery_rounds
			SET
				status = ?,
				winner_jid = ?,
				winning_ticket = ?,
				finished_at = CURRENT_TIMESTAMP
			WHERE group_jid = ?
			  AND round_number = ?
			  AND status = ?
		`,
			RoundStatusFinished,
			winnerJID,
			winningTicket,
			groupJID,
			roundNumber,
			RoundStatusOpen,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro encerrando rodada da loteria: %w",
				err,
			)
	}

	var winnerBalance int

	err =
		tx.QueryRow(`
			SELECT gold
			FROM group_wallets
			WHERE group_jid = ?
			  AND jid = ?
		`,
			groupJID,
			winnerJID,
		).Scan(
			&winnerBalance,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro consultando saldo do vencedor da loteria: %w",
				err,
			)
	}

	nextRound, err :=
		createOpenRoundTx(
			tx,
			groupJID,
		)

	if err != nil {
		return nil, err
	}

	return &drawResult{
		WinningTicket: winningTicket,

		WinnerJID: winnerJID,

		Prize: jackpot,

		WinnerBalance: winnerBalance,

		NextRoundNumber: nextRound.RoundNumber,
	}, nil
}

func getOrCreateOpenRoundTx(
	tx *sql.Tx,
	groupJID string,
) (*Round, error) {
	round := &Round{}

	err :=
		tx.QueryRow(`
			SELECT
				group_jid,
				round_number,
				tickets_sold,
				jackpot,
				status,
				COALESCE(winner_jid, ''),
				COALESCE(winning_ticket, 0)
			FROM lottery_rounds
			WHERE group_jid = ?
			  AND status = ?
			ORDER BY round_number DESC
			LIMIT 1
		`,
			groupJID,
			RoundStatusOpen,
		).Scan(
			&round.GroupJID,
			&round.RoundNumber,
			&round.TicketsSold,
			&round.Jackpot,
			&round.Status,
			&round.WinnerJID,
			&round.WinningTicket,
		)

	if err == nil {
		return round, nil
	}

	if !errors.Is(
		err,
		sql.ErrNoRows,
	) {
		return nil,
			fmt.Errorf(
				"erro consultando rodada aberta da loteria: %w",
				err,
			)
	}

	return createOpenRoundTx(
		tx,
		groupJID,
	)
}

func createOpenRoundTx(
	tx *sql.Tx,
	groupJID string,
) (*Round, error) {
	var nextRoundNumber int

	err :=
		tx.QueryRow(`
			SELECT
				COALESCE(
					MAX(round_number),
					0
				) + 1
			FROM lottery_rounds
			WHERE group_jid = ?
		`,
			groupJID,
		).Scan(
			&nextRoundNumber,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro calculando próxima rodada da loteria: %w",
				err,
			)
	}

	_, err =
		tx.Exec(`
			INSERT INTO lottery_rounds (
				group_jid,
				round_number,
				tickets_sold,
				jackpot,
				status
			)
			VALUES (?, ?, 0, 0, ?)
		`,
			groupJID,
			nextRoundNumber,
			RoundStatusOpen,
		)

	if err != nil {
		return nil,
			fmt.Errorf(
				"erro criando rodada da loteria: %w",
				err,
			)
	}

	return &Round{
		GroupJID: groupJID,

		RoundNumber: nextRoundNumber,

		TicketsSold: 0,

		Jackpot: 0,

		Status: RoundStatusOpen,
	}, nil
}

func countPlayerTicketsTx(
	tx *sql.Tx,
	groupJID string,
	roundNumber int,
	jid string,
) (int, error) {
	var count int

	err :=
		tx.QueryRow(`
			SELECT COUNT(*)
			FROM lottery_tickets
			WHERE group_jid = ?
			  AND round_number = ?
			  AND jid = ?
		`,
			groupJID,
			roundNumber,
			jid,
		).Scan(
			&count,
		)

	if err != nil {
		return 0,
			fmt.Errorf(
				"erro contando bilhetes do jogador: %w",
				err,
			)
	}

	return count, nil
}

func randomTicket() (int, error) {
	value, err :=
		rand.Int(
			rand.Reader,
			big.NewInt(
				MaxTickets,
			),
		)

	if err != nil {
		return 0,
			fmt.Errorf(
				"erro sorteando bilhete da loteria: %w",
				err,
			)
	}

	return int(
		value.Int64(),
	) + 1, nil
}

func ensureSchema() error {
	if database.DB == nil {
		return errors.New(
			"banco de dados não inicializado",
		)
	}

	statements := []string{
		`
		CREATE TABLE IF NOT EXISTS lottery_rounds (
			group_jid TEXT NOT NULL,

			round_number INTEGER NOT NULL
				CHECK (round_number > 0),

			tickets_sold INTEGER NOT NULL DEFAULT 0
				CHECK (
					tickets_sold >= 0
					AND tickets_sold <= 100
				),

			jackpot INTEGER NOT NULL DEFAULT 0
				CHECK (jackpot >= 0),

			status TEXT NOT NULL DEFAULT 'OPEN'
				CHECK (
					status IN (
						'OPEN',
						'FINISHED'
					)
				),

			winner_jid TEXT,

			winning_ticket INTEGER
				CHECK (
					winning_ticket IS NULL
					OR (
						winning_ticket >= 1
						AND winning_ticket <= 100
					)
				),

			created_at DATETIME NOT NULL
				DEFAULT CURRENT_TIMESTAMP,

			finished_at DATETIME,

			PRIMARY KEY (
				group_jid,
				round_number
			)
		);
		`,

		`
		CREATE TABLE IF NOT EXISTS lottery_tickets (
			group_jid TEXT NOT NULL,

			round_number INTEGER NOT NULL,

			ticket_number INTEGER NOT NULL
				CHECK (
					ticket_number >= 1
					AND ticket_number <= 100
				),

			jid TEXT NOT NULL,

			purchased_at DATETIME NOT NULL
				DEFAULT CURRENT_TIMESTAMP,

			PRIMARY KEY (
				group_jid,
				round_number,
				ticket_number
			),

			FOREIGN KEY (
				group_jid,
				round_number
			)
				REFERENCES lottery_rounds (
					group_jid,
					round_number
				)
				ON DELETE CASCADE
		);
		`,

		`
		CREATE UNIQUE INDEX IF NOT EXISTS
			idx_lottery_one_open_round
		ON lottery_rounds (
			group_jid
		)
		WHERE status = 'OPEN';
		`,

		`
		CREATE INDEX IF NOT EXISTS
			idx_lottery_tickets_owner
		ON lottery_tickets (
			group_jid,
			round_number,
			jid
		);
		`,

		`
		CREATE INDEX IF NOT EXISTS
			idx_lottery_round_history
		ON lottery_rounds (
			group_jid,
			round_number DESC
		);
		`,
	}

	for _, statement := range statements {

		_, err :=
			database.DB.Exec(
				statement,
			)

		if err != nil {
			return fmt.Errorf(
				"erro inicializando schema da loteria: %w",
				err,
			)
		}
	}

	return nil
}
