package command

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"tonclient/internal/services"
	"tonclient/internal/tonbot/buttons"
	"tonclient/internal/tonbot/callbacksuf"
	"tonclient/internal/util"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type OpenPoolInfoCommand struct {
	b  *bot.Bot
	ps *services.PoolService
	us *services.UserService
	ss *services.StakeService
}

func NewPoolInfo(b *bot.Bot, ps *services.PoolService, us *services.UserService,
	ss *services.StakeService) *OpenPoolInfoCommand {

	return &OpenPoolInfoCommand{
		b:  b,
		ps: ps,
		us: us,
		ss: ss,
	}
}

func (c *OpenPoolInfoCommand) Execute(ctx context.Context, callback *models.CallbackQuery) {
	if err := util.CheckTypeMessage(c.b, callback); err != nil {
		log.Error("[OpenPoolInfoCommand.Execute]", err)
		return
	}

	data := callback.Data
	msg := callback.Message.Message
	chatId := msg.Chat.ID

	splitData := strings.Split(data, ":")
	if len(splitData) < 3 {
		if _, err := util.SendTextMessage(c.b, uint64(chatId), "❌ Что-то пошло не так! Повторите попытку!"); err != nil {
			log.Error(err)
		}
		return
	}
	poolIdStr := splitData[1]

	user, err := c.us.GetByTelegramChatId(uint64(chatId))
	if err != nil {
		log.Error("[OpenPoolInfoCommand.Execute]", err)
		if _, err := util.SendTextMessage(c.b, uint64(chatId), "❌ Ваш аккаунт не кативирован, чтобы активировать аккаунт введите команду /start"); err != nil {
			log.Error("[OpenPoolInfoCommand.Execute]", err)
		}
		return
	}

	poolId, err := strconv.ParseInt(poolIdStr, 10, 64)
	if err != nil {
		log.Error("[OpenPoolInfoCommand.Execute]", err)
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			"❌ Что-то пошло не так, попробуйте снова",
		); err != nil {
			log.Error("[OpenPoolInfoCommand.Execute]", err)
		}
		return
	}

	pool, err := c.ps.GetId(uint64(poolId))
	if err != nil {
		log.Error("[OpenPoolInfoCommand.Execute]", err)
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			"❌ Не смог найти выбранный пул. Возможно он был удален. Выберите другой",
		); err != nil {
			log.Error("[OpenPoolInfoCommand.Execute]", err)
		}
		return
	}

	poolInfo := util.PoolInfo(pool, c.ss)
	dataBtn := fmt.Sprintf("%v:%v", buttons.CreateStakeId, poolId)
	btn := util.CreateDefaultButton(dataBtn, buttons.StakePoolTokensText)
	var markup *models.InlineKeyboardMarkup

	if pool.OwnerId == uint64(user.Id.Int64) {
		var buttonId string
		if splitData[2] == callbacksuf.My {
			buttonId = buttons.BackMyPoolListId
		} else {
			buttonId = buttons.BackPoolListId
		}
		markup = util.GenerateOwnerPoolInlineKeyboard(poolId, buttonId, pool.IsActive)
	} else {
		markup = util.MenuWithBackButton(buttons.BackPoolListId, buttons.BackPoolList, btn)
	}
	if err := util.EditTextMessageMarkup(
		ctx,
		c.b,
		uint64(chatId),
		msg.ID,
		poolInfo,
		markup,
	); err != nil {
		log.Error("[OpenPoolInfoCommand.Execute]", err)
		return
	}

}
