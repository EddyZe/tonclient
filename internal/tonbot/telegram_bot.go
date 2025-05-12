package tonbot

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type TgBot struct {
	token string
	*bot.Bot
}

func NewTgBot(token string) *TgBot {
	return &TgBot{
		token: token,
	}
}

func (t *TgBot) StartBot() error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	opts := []bot.Option{
		bot.WithDefaultHandler(t.handler),
	}

	tgbot, err := bot.New(t.token, opts...)
	if err != nil {
		log.Fatal("Failed to start bot: ", err)
		return err
	}

	t.Bot = tgbot

	t.Start(ctx)

	return nil
}

func (t *TgBot) handler(ctx context.Context, b *bot.Bot, update *models.Update) {
	t.Bot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   update.Message.Text,
	})
}

func (t *TgBot) SendMessage(ctx context.Context, text string, chatID uint64) {
	if _, err := t.Bot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      text,
		ParseMode: "HTML",
	}); err != nil {
		log.Println("Failed to send message: ", err)
	}
}
