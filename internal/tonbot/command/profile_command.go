package command

import (
	"tonclient/internal/services"

	"github.com/go-telegram/bot"
)

type ProfileCommand struct {
	b  *bot.Bot
	us *services.UserService
	ts *services.TelegramService
	ws *services.WalletTonService
}
