package command

import (
	"context"
	"strconv"
	"strings"
	appModels "tonclient/internal/models"
	"tonclient/internal/services"
	"tonclient/internal/tonbot/buttons"
	"tonclient/internal/util"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type CloseOrOpenPool struct {
	b  *bot.Bot
	ps *services.PoolService
	us *services.UserService
	ss *services.StakeService
}

func NewCloseOrOpenPoolCommand(b *bot.Bot, ps *services.PoolService, us *services.UserService, ss *services.StakeService) *CloseOrOpenPool {
	return &CloseOrOpenPool{
		b:  b,
		ps: ps,
		us: us,
		ss: ss,
	}
}

func (c *CloseOrOpenPool) Execute(ctx context.Context, callback *models.CallbackQuery) {
	if err := util.CheckTypeMessage(c.b, callback); err != nil {
		log.Error("CheckTypeMessage: ", err)
		return
	}
	msg := callback.Message.Message
	chatId := msg.Chat.ID
	messageId := msg.ID
	splitText := strings.Split(callback.Data, ":")
	if len(splitText) < 2 {
		if _, err := util.SendTextMessage(c.b, uint64(chatId), "❌ Что-то пошло не так, повторите попытку!"); err != nil {
			log.Error(err)
		}
		return
	}

	poolId, err := strconv.ParseInt(splitText[1], 10, 64)
	if err != nil {
		if _, err := util.SendTextMessage(c.b, uint64(chatId), "❌ Не верный ID пула!"); err != nil {
			log.Error(err)
		}
		return
	}

	pool, err := c.ps.GetId(uint64(poolId))
	if err != nil {
		log.Error("GetId: ", err)
		if _, err := util.SendTextMessage(c.b, uint64(chatId), "❌ Пул не найден. Возможно он был удален!"); err != nil {
			log.Error(err)
		}
		return
	}

	if !pool.IsCommissionPaid {
		if _, err := util.SendTextMessage(c.b, uint64(chatId), "❌ В текущем пуле не оплачена комиссия! Сначала оплатите комиссию!"); err != nil {
			log.Error(err)
		}
		return
	}

	u, err := c.us.GetByTelegramChatId(uint64(chatId))
	if err != nil {
		log.Error("GetByTelegramChatId: ", err)
		if _, err := util.SendTextMessage(c.b, uint64(chatId), "❌ Аккаунт не активирован. Введите команду /start"); err != nil {
			log.Error(err)
		}
		return
	}

	if uint64(u.Id.Int64) != pool.OwnerId {
		if _, err := util.SendTextMessage(c.b, uint64(chatId), "❌ Вы не владелец этого пула!"); err != nil {
			log.Error(err)
		}
		return
	}

	if pool.IsActive {
		c.editStatus(ctx, uint64(poolId), uint64(chatId), messageId, pool, false)
		return
	} else {
		c.editStatus(ctx, uint64(poolId), uint64(chatId), messageId, pool, true)
		return
	}
}

func (c *CloseOrOpenPool) editStatus(ctx context.Context, poolId, chatId uint64, messageId int, pool *appModels.Pool, isActive bool) {
	if err := c.ps.SetActive(poolId, isActive); err != nil {
		if _, err := util.SendTextMessage(c.b, chatId, "❌ Статус не был изменен. Повторите попытку позже!"); err != nil {
			log.Error(err)
		}
	}

	if err := util.EditTextMessageMarkup(
		ctx,
		c.b,
		chatId,
		messageId,
		util.PoolInfo(pool, c.ss),
		util.GenerateOwnerPoolInlineKeyboard(int64(poolId), buttons.BackMyPoolListId, pool.IsActive),
	); err != nil {
		log.Error(err)
	}
}
