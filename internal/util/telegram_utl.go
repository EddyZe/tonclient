package util

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"tonclient/internal/config"
	appModel "tonclient/internal/models"
	"tonclient/internal/services"
	"tonclient/internal/tonbot/buttons"
	"tonclient/internal/tonbot/callbacksuf"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

var log = config.InitLogger()

func SendTextMessage(bt *bot.Bot, chatId uint64, text string) (*models.Message, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	message, err := bt.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatId,
		Text:      text,
		ParseMode: "HTML",
	})
	if err != nil {
		log.Error("Failed to send message: ", err)
		return nil, err
	}

	return message, nil
}

func SendTextMessageMarkup(bt *bot.Bot, chatId uint64, text string, markup models.ReplyMarkup) (*models.Message, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	message, err := bt.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatId,
		Text:        text,
		ParseMode:   "HTML",
		ReplyMarkup: markup,
	})

	if err != nil {
		log.Error("Failed to send message: ", err)
		return nil, err
	}
	return message, nil
}

func CheckTypeMessage(b *bot.Bot, callback *models.CallbackQuery) error {
	msgType := callback.Message.Type
	if msgType == models.MaybeInaccessibleMessageTypeInaccessibleMessage {
		if _, err := SendTextMessage(
			b,
			uint64(callback.From.ID),
			"❌ Не могу обработать данное сообщение! Скорее всего оно мне не доступно!"); err != nil {
			log.Error(err)
		}
		return errors.New("message type inaccessible")
	}

	return nil
}

func DeleteMessage(ctx context.Context, b *bot.Bot, chatId uint64, messageId int) error {
	if _, err := b.DeleteMessage(ctx, &bot.DeleteMessageParams{
		ChatID:    chatId,
		MessageID: messageId,
	}); err != nil {
		log.Error("Failed delete message", err)
		return err
	}

	return nil
}

func EditMessageMarkup(ctx context.Context, b *bot.Bot, chatId uint64, messageId int, markup models.ReplyMarkup) error {
	if _, err := b.EditMessageReplyMarkup(
		ctx,
		&bot.EditMessageReplyMarkupParams{
			ChatID:      chatId,
			MessageID:   messageId,
			ReplyMarkup: markup,
		}); err != nil {
		log.Error("Failed edit message", err)
		return err
	}

	return nil
}

func EditTextMessage(ctx context.Context, b *bot.Bot, chatId uint64, messageId int, message string) error {
	if _, err := b.EditMessageText(
		ctx,
		&bot.EditMessageTextParams{
			Text:      message,
			ChatID:    chatId,
			MessageID: messageId,
			ParseMode: "HTML",
		},
	); err != nil {
		log.Error("Failed edit message", err)
		return err
	}
	return nil
}

func EditTextMessageMarkup(ctx context.Context, b *bot.Bot, chatId uint64, messageId int, message string, markup models.ReplyMarkup) error {
	if _, err := b.EditMessageText(
		ctx,
		&bot.EditMessageTextParams{
			Text:        message,
			ChatID:      chatId,
			MessageID:   messageId,
			ParseMode:   "HTML",
			ReplyMarkup: markup,
		},
	); err != nil {
		log.Error("Failed edit message", err)
		return err
	}
	return nil
}

func ConnectingTonConnect(b *bot.Bot, chatId uint64, tcs *services.TonConnectService) (*appModel.TonConnectResult, error) {
	sessionTonConnect, err := tcs.CreateSession()
	if err != nil {
		log.Error(err)
		if _, err := SendTextMessage(b, chatId, "❌ Что-то пошло не так. Попробуйте повторить попытку!"); err != nil {
			log.Error(err)
		}
		return nil, err
	}

	urls, err := tcs.GenerateConnectUrls(sessionTonConnect)
	if err != nil {
		log.Error(err)
		if _, err := SendTextMessage(b, chatId, "❌ Произошла ошибка генерации ссылок, для подключения кошелька. Повторите попытку!"); err != nil {
			log.Error(err)
		}
		return nil, err
	}

	btns := make([]models.InlineKeyboardButton, 0, 2)
	for k, v := range urls {
		btn2 := CreateUrlInlineButton(
			fmt.Sprintf("%v: %v", buttons.OpenWallet, k),
			v,
		)
		btns = append(btns, btn2)
	}

	markup := MenuWithBackButton(buttons.DefCloseId, buttons.DefCloseText, btns...)
	if _, err := SendTextMessageMarkup(b, chatId, "Выберите кошелек, который хотите подключить: ", markup); err != nil {
		log.Error(err)
		return nil, err
	}

	res, err := tcs.Connect(sessionTonConnect)
	if err != nil {
		log.Error(err)
		if _, err := SendTextMessage(b, chatId, "❌ Произошла ошибка подключения. Повторите попытку!"); err != nil {
			log.Error(err)
		}
		return nil, err
	}
	err = tcs.SaveSession(fmt.Sprint(chatId), sessionTonConnect)
	if err != nil {
		log.Error(err)
		if _, err := SendTextMessage(b, chatId, "❌ Произошла ошибка при подключении, повторите попытку!"); err != nil {
			log.Error(err)
		}
		return nil, err
	}

	return res, nil
}

func RequestRepeatTonConnect(b *bot.Bot, chatId int64, markup *models.InlineKeyboardMarkup, tcs *services.TonConnectService) error {
	if _, err := SendTextMessageMarkup(
		b,
		uint64(chatId),
		"❌ Возможно вы отключили TonConnect! Подтвердите подключение снова! А затем нажмите <b>Повторить попытку</b>",
		markup,
	); err != nil {
		log.Error(err)
		return err
	}
	if _, err := ConnectingTonConnect(b, uint64(chatId), tcs); err != nil {
		log.Error(err)
		return err
	}
	if _, err := SendTextMessageMarkup(
		b,
		uint64(chatId),
		"✅ Кошелек привязан. Нажмите 'повторить попытку' и подтвердите транзакцию по резерву в привязанном кошельке",
		markup); err != nil {
		log.Error(err)
		return err
	}
	return nil
}

func SendMessageOwnerAndUserIfBadReserve(
	chatId, ownerPoolId, poolId uint64,
	jettonName string,
	b *bot.Bot,
	ts *services.TelegramService,
) {
	if _, err := SendTextMessage(
		b,
		chatId,
		"❌ Не хватает резерва пула. Мы отправили владельцу пула уведомление. Попробуйте позже!",
	); err != nil {
		log.Println(err)
	}
	ownerPoolTelegram, er := ts.GetByUserId(ownerPoolId)
	if er != nil {
		return
	}
	idButton := fmt.Sprintf("%v:%v:%v", buttons.PoolDataButton, poolId, callbacksuf.My)
	btn := CreateDefaultButton(idButton, "Открыть пул")
	markup := CreateInlineMarup(1, btn)
	textMessage := fmt.Sprintf("В вашем пуле с токеном %v кончается резерв! Пополните его!", jettonName)
	if _, err := SendTextMessageMarkup(
		b,
		ownerPoolTelegram.TelegramId,
		textMessage,
		markup,
	); err != nil {
		log.Println(err)
	}
}

func GetJettonNameFromCallbackData(b *bot.Bot, chatId uint64, data string) (string, error) {
	splitDat := strings.Split(data, ":")

	if len(splitDat) != 2 {
		if _, err := SendTextMessage(
			b,
			chatId,
			"❌ Не могу обработать эту кнопку!",
		); err != nil {
			log.Error(err)
		}
		return "", errors.New("invalid callback data")
	}

	return splitDat[1], nil
}

func GetCurrentPage(chatId int64, pages map[int64]int) int {
	page, ok := pages[chatId]
	if !ok {
		page = 0
	}
	pages[chatId] = page
	return page
}

func GenerateGroupButtons(groups *[]appModel.GroupElements, idButton string) []models.InlineKeyboardButton {
	res := make([]models.InlineKeyboardButton, 0, 5)
	for _, g := range *groups {
		idButton := fmt.Sprintf("%v:%v", idButton, g.Name)
		text := fmt.Sprintf("%v. Стейков: %v", g.Name, g.Count)
		btn := CreateDefaultButton(idButton, text)
		res = append(res, btn)
	}

	return res
}

func GenerateStakeListByGroup(stakes []appModel.Stake, jettonName, idButton string) []models.InlineKeyboardButton {
	res := make([]models.InlineKeyboardButton, 0, 5)
	for _, s := range stakes {
		idbtn := fmt.Sprintf("%v:%v:%v", idButton, jettonName, s.Id.Int64)
		text := fmt.Sprintf("Стейк от %v", s.StartDate.Format("02.01.2006 15:04"))
		if !s.IsActive && s.IsRewardPaid || !s.IsActive && s.IsInsurancePaid {
			text += " ⚪️"
		} else if !s.IsActive && !s.IsRewardPaid || !s.IsInsurancePaid && !s.IsActive {
			text += " 🟡"
		} else {
			text += " 🟢"
		}
		btn := CreateDefaultButton(idbtn, text)
		res = append(res, btn)
	}

	return res
}

func SendAgreement(b *bot.Bot, chatId uint64) {
	message := `
📜<b> Пользовательское соглашение</b>

Продолжая использование данного бота, вы подтверждаете, что ознакомлены и согласны со следующими условиями:
• Вы используете сервис добровольно и на свой страх и риск.
Администраторы бота, разработчики и владельцы проекта не несут ответственности за любые потери средств, включая, но не ограничиваясь: техническими сбоями, ошибками в коде, изменениями рыночной цены токена или действиями третьих лиц.
• Никакие функции бота не являются финансовой рекомендацией, гарантией дохода или страхованием.
Все действия — включая стейкинг, вывод средств и использование компенсирующих механизмов — реализуются как есть («as is») и без каких-либо гарантий.
• Пользователь самостоятельно выбирает токены, с которыми работает в рамках сервиса.
Администрация бота не проводит проверку, аудит, отбор, оценку надёжности или перспективности токенов, принимаемых пользователями. Ответственность за выбор и последствия использования конкретных токенов лежит исключительно на пользователе.
• Пользователь самостоятельно оценивает все риски, связанные с участием в проекте.
В том числе, риски полной или частичной потери вложенных средств.
• Бот может быть изменён, приостановлен или отключён в любой момент без предварительного уведомления.
• Используя бота, вы подтверждаете, что обладаете достаточным пониманием принципов работы криптовалют, блокчейна и связанных с ними рисков.
✅ Нажимая «Согласен», вы подтверждаете принятие всех условий.
`
	btn := CreateDefaultButton(buttons.AcceptUserAgreementId, buttons.AcceptUserAgreement)
	markup := CreateInlineMarup(1, btn)
	if _, err := SendTextMessageMarkup(b, chatId, message, markup); err != nil {
		log.Error(err)
	}
}
