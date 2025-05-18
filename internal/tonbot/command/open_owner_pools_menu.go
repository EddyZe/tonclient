package command

import (
	"context"
	"tonclient/internal/tonbot/buttons"
	"tonclient/internal/util"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type OpenOwnerPoolsMenu struct {
	b *bot.Bot
}

func NewOpenOwnerPoolsMenu(b *bot.Bot) *OpenOwnerPoolsMenu {
	return &OpenOwnerPoolsMenu{b}
}

func (c *OpenOwnerPoolsMenu) Execute(ctx context.Context, callback *models.CallbackQuery) {
	if err := util.CheckTypeMessage(c.b, callback); err != nil {
		return
	}

	msg := callback.Message.Message
	chatId := msg.Chat.ID
	markup := util.CreateDefaultButtonsReplay(
		2,
		buttons.CreatePool,
		buttons.Profile,
		buttons.MyPools,
		buttons.LearnMore,
		buttons.Setting,
	)

	if _, err := util.SendTextMessageMarkup(
		c.b,
		uint64(chatId),
		"Роль была успешно поменена. Воспользуйтесь меню 👇",
		markup); err != nil {
		log.Error("Failed to send markup: ", err)
		return
	}
}
