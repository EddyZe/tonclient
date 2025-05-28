package util

import (
	"context"
	"errors"
	"fmt"
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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	urls, err := tcs.GenerateConnectUrls(ctx, sessionTonConnect)
	if err != nil {
		log.Error(err)
		if _, err := SendTextMessage(b, chatId, "❌ Произошла ошибка генерации ссылок, для подключения кошелька. Повторите попытку!"); err != nil {
			log.Error(err)
		}
		return nil, err
	}

	btns := make([]models.InlineKeyboardButton, 0, 2)
	for k, v := range urls {
		btn := CreateUrlInlineButton(k, v)
		btns = append(btns, btn)
	}

	markup := MenuWithBackButton(buttons.DefCloseId, buttons.DefCloseText, btns...)
	if _, err := SendTextMessageMarkup(b, chatId, "Выберите кошелек, который хотите подключить: ", markup); err != nil {
		log.Error(err)
		return nil, err
	}

	res, err := tcs.Connect(ctx, sessionTonConnect)
	if err != nil {
		log.Error(err)
		if _, err := SendTextMessage(b, chatId, "❌ Произошла ошибка подключения. Повторите попытку!"); err != nil {
			log.Error(err)
		}
		return nil, err
	}
	err = tcs.SaveSession(ctx, fmt.Sprint(chatId), sessionTonConnect)
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
