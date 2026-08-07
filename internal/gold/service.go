package gold

import "whatsapp-sticker-bot/internal/database"

func GetOrCreateWallet(jid, name string) (*database.User, bool, error) {
    user, err := database.GetUser(jid)
    if err != nil {
        return nil, false, err
    }

    if user != nil {
        return user, false, nil
    }

    if err := database.CreateUser(jid, name); err != nil {
        return nil, false, err
    }

    user, err = database.GetUser(jid)
    return user, true, err
}

func GetBalance(jid string) (int, error) {
    user, err := database.GetUser(jid)
    if err != nil {
        return 0, err
    }
    if user == nil {
        return 0, nil
    }

    return user.Gold, nil
}
