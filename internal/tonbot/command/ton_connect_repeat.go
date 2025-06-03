package command

import (
	"context"
	"tonclient/internal/services"
	"tonclient/internal/util"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type TonConnectRepeat struct {
	b   *bot.Bot
	us  *services.UserService
	ws  *services.WalletTonService
	tcs *services.TonConnectService
}

func NewTonConnectRepeat(b *bot.Bot, us *services.UserService, ws *services.WalletTonService, tcs *services.TonConnectService) *TonConnectRepeat {
	return &TonConnectRepeat{
		b:   b,
		us:  us,
		ws:  ws,
		tcs: tcs,
	}
}

func (c *TonConnectRepeat) Execute(ctx context.Context, callback *models.CallbackQuery) {
	if err := util.CheckTypeMessage(c.b, callback); err != nil {
		return
	}

	msg := callback.Message.Message
	chatId := msg.Chat.ID
	if _, err := util.SendTextMessage(
		c.b,
		uint64(chatId),
		"‼️ Подключайте уже привязанный кошелек! Так как все выводы будут идти на него!",
	); err != nil {
		log.Error(err)
		return
	}

	res, err := util.ConnectingTonConnect(c.b, uint64(chatId), c.tcs)
	if err != nil {
		log.Error(err)
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			"❌ Повторите попытку! Что-то пошло не так!",
		); err != nil {
			log.Error(err)
			return
		}
		return
	}

	u, err := c.us.GetByTelegramChatId(uint64(chatId))
	if err != nil {
		log.Error(err)
		return
	}

	w, err := c.ws.GetByUserId(uint64(u.Id.Int64))
	if err != nil {
		log.Error(err)
		return
	}

	w.Name = res.WalletName
	if err := c.ws.Update(w); err != nil {
		log.Error(err)
		return
	}

	if _, err := util.SendTextMessage(
		c.b,
		uint64(chatId),
		"✅ Кошелек был успешно привязан!",
	); err != nil {
		log.Error(err)
		return
	}
}
