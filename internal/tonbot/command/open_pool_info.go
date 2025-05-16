package command

import (
	"tonclient/internal/services"

	"github.com/go-telegram/bot"
)

type OpenPoolInfoCommand struct {
	b  *bot.Bot
	ps *services.PoolService
	us *services.UserService
}
