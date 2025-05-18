package command

import (
	"context"
	"tonclient/internal/services"
	"tonclient/internal/tonbot/buttons"
	"tonclient/internal/tonbot/userstate"
	"tonclient/internal/util"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type SetWalletType interface {
	*models.Message | *models.CallbackQuery
}

type SetWalletCommand[T SetWalletType] struct {
	b   *bot.Bot
	ws  *services.WalletTonService
	us  *services.UserService
	aws *services.AdminWalletService
}

func NewSetWalletCommand[T SetWalletType](b *bot.Bot, ws *services.WalletTonService,
	us *services.UserService, aws *services.AdminWalletService) *SetWalletCommand[T] {
	return &SetWalletCommand[T]{
		b:   b,
		ws:  ws,
		us:  us,
		aws: aws,
	}
}

func (s *SetWalletCommand[T]) Execute(ctx context.Context, msg T) {
	if callback, ok := any(msg).(*models.CallbackQuery); ok {
		s.executeCallback(ctx, callback)
		return
	}

	if message, ok := any(msg).(*models.Message); ok {
		s.executeMessage(ctx, message)
		return
	}
}

func (s *SetWalletCommand[T]) executeCallback(ctx context.Context, callback *models.CallbackQuery) {
	if err := util.CheckTypeMessage(s.b, callback); err != nil {
		log.Error(err)
		return
	}

	msg := callback.Message.Message
	chatId := msg.Chat.ID
	btnClose := util.CreateDefaultButton(buttons.DefCloseId, buttons.DefCloseText)
	markup := util.CreateInlineMarup(1, btnClose)

	if _, err := util.SendTextMessageMarkup(s.b, uint64(chatId), "Отправьте адрес кошелька: ", markup); err != nil {
		log.Error(err)
		return
	}

	userstate.CurrentState[chatId] = userstate.EnterWalletAddr
}

func (s *SetWalletCommand[T]) executeMessage(ctx context.Context, msg *models.Message) {
	chatId := msg.Chat.ID
	text := msg.Text
	state, ok := userstate.CurrentState[chatId]
	if !ok {
		if _, err := util.SendTextMessage(s.b, uint64(chatId), "❌ Что-то пошло не так. Выберите повторно операцию."); err != nil {
			log.Error(err)
		}
		return
	}

	switch state {
	case userstate.EnterWalletAddr:
		s.enterAddrWallet(uint64(chatId), text)
		break
	}

}

func (s *SetWalletCommand[T]) enterAddrWallet(chatId uint64, text string) {
	if err := s.aws.CheckValidAddr(text); err != nil {
		btnClose := util.CreateDefaultButton(buttons.DefCloseId, buttons.DefCloseText)
		markup := util.CreateInlineMarup(1, btnClose)
		if _, err := util.SendTextMessageMarkup(s.b, chatId, "❌ Невалидный адрес кошелька! Повторите попытку!", markup); err != nil {
			log.Error(err)
			return
		}
		return
	}
}
