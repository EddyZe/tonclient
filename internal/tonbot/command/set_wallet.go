package command

import (
	"context"
	"fmt"
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
	tcs *services.TonConnectService
}

func NewSetWalletCommand[T SetWalletType](b *bot.Bot, ws *services.WalletTonService,
	us *services.UserService, aws *services.AdminWalletService,
	tcs *services.TonConnectService) *SetWalletCommand[T] {
	return &SetWalletCommand[T]{
		b:   b,
		ws:  ws,
		us:  us,
		aws: aws,
		tcs: tcs,
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

	resp := `
	<b>Привязка кошелька</b>

	Отправьте адрес кошелька. На привязанный адрес будут, отправляться средства и операции.
	`

	if _, err := util.SendTextMessageMarkup(
		s.b,
		uint64(chatId),
		resp,
		markup,
	); err != nil {
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
	case userstate.ConnectTonConnect:
		s.connectWallet(uint64(chatId), text)
		break
	default:
		log.Infoln(state)
		return
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
	closeButton := util.CreateDefaultButton(buttons.DefCloseId, buttons.DefCloseText)
	markup := util.CreateInlineMarup(1, closeButton)
	w, _ := s.ws.FindWalletByAddr(text)
	if w != nil {
		if _, err := util.SendTextMessageMarkup(
			s.b,
			chatId,
			"❌ Номер кошелька уже привязан к другому аккаунту! Повторите попытку",
			markup,
		); err != nil {
			log.Error(err)
		}
		return
	}

	s.connectWallet(chatId, text)
}

func (s *SetWalletCommand[T]) connectWallet(chatId uint64, addr string) {
	res, err := util.ConnectingTonConnect(s.b, chatId, s.tcs)
	if err != nil {
		log.Error(err)
		return
	}

	user, err := s.us.GetByTelegramChatId(chatId)
	if err != nil {
		log.Error(err)
		if _, err := util.SendTextMessage(s.b, chatId, "❌ Ваш аккаунт не найден! Напишите команду /start, чтобы активировать аккаунт, затем повторите попытку!"); err != nil {
			log.Error(err)
		}
		return
	}
	currentWall, err := s.ws.GetByUserId(uint64(user.Id.Int64))
	if err == nil {
		currentWall.Addr = addr
		currentWall.Name = res.WalletName
		err := s.ws.Update(currentWall)
		if err != nil {
			if _, er := util.SendTextMessage(s.b, chatId, "❌ Не удалось обновить кошелк. Повторите попытку!"); er != nil {
				log.Error(err)
			}
		}
		resp := fmt.Sprintf("✅ Кошелек %v, был успешно изменен. Имя кошелька: %v", addr, currentWall.Name)
		if _, err := util.SendTextMessage(s.b, chatId, resp); err != nil {
			log.Error(err)
		}
		return
	}
	wall, err := s.ws.CreateNewWallet(uint64(user.Id.Int64), addr, res.WalletName)
	if err != nil {
		log.Error(err)
		if err.Error() == "address already exists" {
			if _, err := util.SendTextMessage(s.b, chatId, "❌ Адрес кошелька привязан к другому аккаунту!"); err != nil {
				log.Error(err)
				return
			}
			return
		}
		if _, err := util.SendTextMessage(s.b, chatId, "❌ Что-то пошло не так. Попробуйте повторить попытку!"); err != nil {
			log.Error(err)
		}
		return
	}

	resp := fmt.Sprintf("✅ Кошелек %v, был успешно подключен. Имя кошелька: %v", addr, wall.Name)
	if _, err := util.SendTextMessage(s.b, chatId, resp); err != nil {
		log.Error(err)
	}
	delete(userstate.CurrentState, int64(chatId))
}
