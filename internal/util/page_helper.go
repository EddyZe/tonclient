package util

import (
	"context"
	"tonclient/internal/core/interfaces"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func NextPage(ctx context.Context, callback *models.CallbackQuery, pages map[int64]int, totalPages int, b *bot.Bot, c interfaces.Command[*models.Message]) map[int64]int {

	if err := CheckTypeMessage(b, callback); err != nil {
		log.Error(err)
		return pages
	}

	msg := callback.Message.Message
	chatId := msg.Chat.ID
	page, ok := pages[chatId]
	if !ok {
		page = 0
	}
	if page < totalPages {
		page++
		pages[chatId] = page
	}
	c.Execute(
		ctx,
		msg,
	)

	return pages
}

func BackPage(ctx context.Context, callback *models.CallbackQuery, pages map[int64]int, b *bot.Bot, c interfaces.Command[*models.Message]) map[int64]int {
	if err := CheckTypeMessage(b, callback); err != nil {
		log.Error(err)
		return pages
	}

	msg := callback.Message.Message
	chatId := msg.Chat.ID
	page, ok := pages[chatId]
	if !ok {
		page = 0
		pages[chatId] = page
	}
	if page > 0 {
		page--
		pages[chatId] = page
	}
	c.Execute(ctx, msg)
	return pages
}

func CloseList(ctx context.Context, callback *models.CallbackQuery, pages map[int64]int, b *bot.Bot) map[int64]int {
	if err := CheckTypeMessage(b, callback); err != nil {
		log.Error(err)
		return pages
	}
	msg := callback.Message.Message
	chatId := msg.Chat.ID
	pages[chatId] = 0

	if err := DeleteMessage(ctx, b, uint64(chatId), msg.ID); err != nil {
		log.Error(err)
		return pages
	}

	return pages
}
