package main

import (
    "github.com/bot011max/medical-bot/internal/telegram"
    "github.com/bot011max/medical-bot/internal/security"
)

func main() {
    armor := security.NewAbsoluteArmor()
    bot := telegram.NewBot(armor)
    bot.Start()
}
