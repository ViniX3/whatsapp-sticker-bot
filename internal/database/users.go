package database

import "database/sql"

type User struct {
    JID  string
    Name string
    Gold int
}

func GetUser(jid string) (*User, error) {
    row := DB.QueryRow(
        "SELECT jid, name, gold FROM users WHERE jid = ?",
        jid,
    )

    var u User
    err := row.Scan(&u.JID, &u.Name, &u.Gold)
    if err == sql.ErrNoRows {
        return nil, nil
    }
    if err != nil {
        return nil, err
    }

    return &u, nil
}

func CreateUser(jid, name string) error {
    _, err := DB.Exec(
        "INSERT INTO users (jid, name, gold) VALUES (?, ?, 1000)",
        jid, name,
    )
    return err
}

func UpdateGold(jid string, gold int) error {
    _, err := DB.Exec(
        "UPDATE users SET gold = ? WHERE jid = ?",
        gold, jid,
    )
    return err
}
