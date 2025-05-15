package command

import (
	"context"
	"fmt"
	"os"
	"tonclient/internal/services"
	"tonclient/internal/util"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type InviteFriendCommand struct {
	b  *bot.Bot
	us *services.UserService
}

func NewInviteFriendCommand(b *bot.Bot, us *services.UserService) *InviteFriendCommand {
	return &InviteFriendCommand{
		b:  b,
		us: us,
	}
}

func (c *InviteFriendCommand) Execute(ctx context.Context, msg *models.Message) {
	chatId := msg.Chat.ID

	if _, err := c.us.GetByTelegramChatId(uint64(chatId)); err != nil {
		if _, er := util.SendTextMessage(
			c.b,
			uint64(chatId),
			"❌ Ваш профиль не найден. Введите команду: /start и повторите попытку",
		); er != nil {
			log.Error(err)
			return
		}
		return
	}

	referalCode := util.GenerateReferralTelegramCode(fmt.Sprint(chatId))

	url := fmt.Sprint("https://t.me/insuranceton_bot?start=", referalCode)

	if _, err := util.SendTextMessage(
		c.b,
		uint64(chatId),
		fmt.Sprint(generateMessage(), "Ваша реферальная ссылка: ", url),
	); err != nil {
		log.Error(err)
		return
	}
}

func generateMessage() string {
	coinName := os.Getenv("JETTON_NAME_COIN")
	if coinName == "" {
		coinName = "NESTRAH"
	}
	referalBonus := os.Getenv("REFERAL_BONUS")
	if referalBonus == "" {
		referalBonus = "2"
	}
	return fmt.Sprintf(
		"Пригласи друга и получи <b>%v%% %v</b> коинов за его первый стейк!\n\n",
		referalBonus,
		coinName,
	)
}
