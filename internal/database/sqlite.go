package database

import (
    "database/sql"
    "fmt"

    _ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func Init() error {
    var err error

    DB, err = sql.Open("sqlite3", "storage/gold.db")
    if err != nil {
        return fmt.Errorf("erro ao abrir banco: %w", err)
    }

    return createTables()
}

func createTables() error {
    query := `
    CREATE TABLE IF NOT EXISTS users (
        jid TEXT PRIMARY KEY,
        name TEXT,
        gold INTEGER NOT NULL DEFAULT 1000,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP
    );`

    _, err := DB.Exec(query)
    return err
}
