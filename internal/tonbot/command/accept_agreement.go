package command

import (
	"context"
	"tonclient/internal/services"
	"tonclient/internal/util"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type AcceptAgreementCommand struct {
	b  *bot.Bot
	us *services.UserService
}

func NewAcceptAgreementCommand(b *bot.Bot, us *services.UserService) *AcceptAgreementCommand {
	return &AcceptAgreementCommand{
		b:  b,
		us: us,
	}
}

func (c *AcceptAgreementCommand) Execute(ctx context.Context, callback *models.CallbackQuery) {
	if err := util.CheckTypeMessage(c.b, callback); err != nil {
		return
	}

	chatId := callback.From.ID

	u, err := c.us.GetByTelegramChatId(uint64(chatId))
	if err != nil {
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			"❌ Аккаунт не активирован! Введите команду /start",
		); err != nil {
			log.Println(err)
		}
		return
	}

	u.IsAcceptAgreement = true
	if err := c.us.Update(u); err != nil {
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			"❌ Что-то пошло не так. Повторите попытку.",
		); err != nil {
			log.Println(err)
		}
		return
	}

	if _, err := util.SendTextMessage(
		c.b,
		uint64(chatId),
		"✅ Вы приняли пользовательское соглашение!",
	); err != nil {
		log.Println(err)
		return
	}
}
