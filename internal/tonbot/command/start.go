package command

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
	"tonclient/internal/config"
	appModel "tonclient/internal/models"
	"tonclient/internal/services"
	"tonclient/internal/tonbot/buttons"
	"tonclient/internal/util"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

var log = config.InitLogger()

type StartCommand struct {
	bt *bot.Bot
	us *services.UserService
	ts *services.TelegramService
}

func NewStartCommand(b *bot.Bot, us *services.UserService, ts *services.TelegramService) *StartCommand {
	return &StartCommand{
		bt: b,
		us: us,
		ts: ts,
	}
}

func (c *StartCommand) Execute(ctx context.Context, msg *models.Message) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	chatId := msg.Chat.ID

	btn1 := util.CreateDefaultButton(buttons.RoleButtonUserId, buttons.RoleButtonUserText)
	btn2 := util.CreateDefaultButton(buttons.RoleButtonOwnerTokensId, buttons.RoleButtonOwnerTokensText)

	if _, err := util.SendTextMessageMarkup(
		c.bt,
		uint64(chatId),
		generateResponse(),
		util.CreateInlineMarup(2, btn1, btn2),
	); err != nil {
		log.Error(err)
		return
	}

	_, err := c.us.GetByTelegramChatId(uint64(chatId))
	if err != nil {
		if err.Error() == "user not found" {
			text := strings.Split(msg.Text, " ")
			newUser := &appModel.User{
				Username:  msg.Chat.Username,
				CreatedAt: time.Now(),
			}
			if len(text) > 1 {
				id, err := util.DecodeReferralTelegramCode(text[1])
				if err != nil {
					log.Debugln("Failed to decode referral telegram code: ", err)
					if _, err := util.SendTextMessage(
						c.bt,
						uint64(chatId),
						"❌ Реферальный код не был применен. Возможно он не действителен!"); err != nil {
						log.Error(err)
					}
					return
				}

				if chatId != id {
					newUser.RefererId = sql.NullInt64{
						Int64: id,
						Valid: true,
					}
				}
			}

			newUser, err := c.us.CreateUser(newUser)
			if err != nil {
				log.Error("Failed to create user: ", err)
				if _, err := util.SendTextMessage(c.bt, uint64(chatId), "Ошибка при создании профиля. Введите команду /start, чтобы попробовать снова!"); err != nil {
					log.Error(err)
					return
				}
			}

			err = c.createTelegram(newUser, chatId, msg)
			if err != nil {
				err := c.us.DeleteById(uint64(newUser.Id.Int64))
				if err != nil {
					log.Error(err)
					return
				}
				return
			}
			return
		}
	}
}

func (c *StartCommand) createTelegram(user *appModel.User, chatId int64, msg *models.Message) error {
	if !user.Id.Valid {
		log.Error("userId invalid")
		if _, err := util.SendTextMessage(
			c.bt,
			uint64(chatId),
			"❌ Ошибка при создании профиля, попробуйте повторить попытку введя команду: /start"); err != nil {
			log.Error(err)
			return err
		}

		return errors.New("UserId invalid")
	}

	userid := user.Id.Int64
	if _, err := c.ts.CreateTelegram(uint64(userid), msg.Chat.Username, uint64(chatId)); err != nil {
		log.Error(err)
		if _, err := util.SendTextMessage(
			c.bt,
			uint64(chatId),
			"❌ Ошибка при создании профиля, попробуйте повторить попытку введя команду: /start"); err != nil {
			log.Error(err)
			return err
		}
		return err
	}

	return nil
}

func generateResponse() string {
	return `
👋 <b>Добро пожаловать в NESTRAH — вашу защиту от рисков в мире криптовалют!</b>


🚀 <b>Что мы предлагаем:</b>

		• Стейкинг с доходностью — замораживайте токены и получайте ежедневные награды.

		• Страхование падения цены — компенсация до 30% при обвале курса.

		• Простота и безопасность — всё через Telegram и TON-кошелек.


💡 <b>Как начать:</b>

		• Выберите пул с лучшими условиями.

		• Подключите кошелек через TON Connect.

		• Застейкайте токены и спите спокойно — мы защитим ваши инвестиции!


🔒 <b>Почему NESTRAH?</b>

		• Прозрачность — все операции через смарт-контракты.

		• Гибкость — создавайте свои пулы или присоединяйтесь к существующим.

		• Рефералы — приглашайте друзей и получайте бонусы в токенах NESTRAH.

`
}
