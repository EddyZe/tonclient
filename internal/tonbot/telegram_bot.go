package tonbot

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"tonclient/internal/config"
	"tonclient/internal/services"
	"tonclient/internal/tonbot/buttons"
	"tonclient/internal/tonbot/command"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

var log = config.InitLogger()

type TgBot struct {
	token string
	us    *services.UserService
	ts    *services.TelegramService
	ps    *services.PoolService
	aws   *services.AdminWalletService
}

func NewTgBot(token string, us *services.UserService, ts *services.TelegramService,
	ps *services.PoolService, aws *services.AdminWalletService) *TgBot {
	return &TgBot{
		token: token,
		us:    us,
		ts:    ts,
		ps:    ps,
		aws:   aws,
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

	tgbot.Start(ctx)

	return nil
}

func (t *TgBot) handler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update == nil {
		return
	}

	if update.Message != nil {
		msg := update.Message
		t.handleMessage(ctx, b, msg)
	}

	if update.CallbackQuery != nil {
		callback := update.CallbackQuery

		t.handleCallback(ctx, b, callback)

		if _, err := b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: callback.ID,
		}); err != nil {
			log.Error("AnswerCallbackQuery: ", err)
		}
	}
}

func (t *TgBot) handleMessage(ctx context.Context, b *bot.Bot, msg *models.Message) {
	if msg.Chat.Type == models.ChatTypePrivate {
		text := msg.Text

		if strings.HasPrefix(text, "/start") {
			cmd := command.NewStartCommand(b, t.us, t.ts)
			cmd.Execute(ctx, msg)
			return
		}

		if text == buttons.InviteFriend {
			cmd := command.NewInviteFriendCommand(b, t.us)
			cmd.Execute(ctx, msg)
			return
		}

		if text == buttons.SelectPool {
			command.NewListPoolCommand(b, t.ps, t.aws).Execute(ctx, msg)
		}
	}
}

func (t *TgBot) handleCallback(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery) {
	data := callback.Data

	if data == buttons.RoleButtonUserId {
		command.NewOpenUserMenuCommand(b).Execute(ctx, callback)
		return
	}

	if strings.HasPrefix(data, buttons.NextPagePool) {
		command.NewListPoolCommand(b, t.ps, t.aws).NextPage(ctx, callback)
		return
	}

	if strings.HasPrefix(data, buttons.BackPagePool) {
		command.NewListPoolCommand(b, t.ps, t.aws).BackPage(ctx, callback)
		return
	}

	if data == buttons.CloseListPool {
		command.NewListPoolCommand(b, t.ps, t.aws).CloseList(ctx, callback)
		return
	}
}
