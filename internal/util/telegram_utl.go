package util

import (
	"context"
	"errors"
	"time"
	"tonclient/internal/config"

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
