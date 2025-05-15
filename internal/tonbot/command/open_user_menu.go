package command

import (
	"context"
	"tonclient/internal/tonbot/buttons"
	"tonclient/internal/util"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type OpenUserMenuCommand struct {
	bt *bot.Bot
}

func NewOpenUserMenuCommand(bt *bot.Bot) *OpenUserMenuCommand {
	return &OpenUserMenuCommand{bt: bt}
}

func (c *OpenUserMenuCommand) Execute(ctx context.Context, callback *models.CallbackQuery) {
	var message *models.Message

	switch callback.Message.Type {
	case models.MaybeInaccessibleMessageTypeMessage:
		message = callback.Message.Message
		break
	case models.MaybeInaccessibleMessageTypeInaccessibleMessage:
		_, err := util.SendTextMessage(c.bt, uint64(callback.From.ID), "❌ Я не могу обработать это сообщение")
		if err != nil {
			return
		}
		return
	}

	if message == nil {
		return
	}

	chatId := message.Chat.ID

	keys := util.CreateDefaultButtonsReplay(
		2,
		buttons.SelectPool,
		buttons.Profile,
		buttons.HistoryOperation,
		buttons.TakeAwards,
		buttons.CheckInsurance,
		buttons.InviteFriend,
		buttons.Setting,
	)

	if _, err := util.SendTextMessageMarkup(
		c.bt,
		uint64(chatId),
		"Вы открыли меню пользователя 👇. Сменить меню, вы можете в настройках",
		keys); err != nil {
		log.Error("Failed to send message open user menu: ", err)
		return
	}
}
