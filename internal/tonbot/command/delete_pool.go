package command

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	appModels "tonclient/internal/models"
	"tonclient/internal/services"
	"tonclient/internal/util"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type DeletePool struct {
	b   *bot.Bot
	ps  *services.PoolService
	ops *services.OperationService
	ss  *services.StakeService
}

func NewDeletePool(b *bot.Bot, ps *services.PoolService, ops *services.OperationService, ss *services.StakeService) *DeletePool {
	return &DeletePool{
		b:   b,
		ps:  ps,
		ops: ops,
		ss:  ss,
	}
}

func (c *DeletePool) Execute(ctx context.Context, callback *models.CallbackQuery) {
	if err := util.CheckTypeMessage(c.b, callback); err != nil {
		return
	}

	msg := callback.Message.Message
	chatId := msg.Chat.ID
	messageId := msg.ID

	splitData := strings.Split(callback.Data, ":")
	if len(splitData) != 2 {
		return
	}

	id, err := strconv.ParseUint(splitData[1], 10, 64)
	if err != nil {
		return
	}

	p, err := c.ps.GetId(id)
	if err != nil {
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			"❌ Пул не найден! возможно он был удален!",
		); err != nil {
			log.Error(err)
		}
		return
	}

	if p.IsActive || p.Reserve > 0 {
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			"❌ Невозможно удалить пул. Резерв должен быть пуст и пул закрыт!",
		); err != nil {
			log.Error(err)
		}
		return
	}

	count := 0
	stakeNoPayment := c.ss.GetPoolStakes(uint64(p.Id.Int64))
	for _, s := range stakeNoPayment {
		if !s.IsRewardPaid && s.IsInsurancePaid {
			count++
		}
	}

	if count > 0 {
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			fmt.Sprintf("❌ Дождитесь пока все заберут свои награды. Пользователей осталось: %d", count),
		); err != nil {
			log.Error(err)
		}
		return
	}

	err = c.ps.Delete(id)
	if err != nil {
		log.Error(err)
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			"❌ Ошибка при удалении пула. Повторите попытку!",
		); err != nil {
			log.Error(err)
		}
		return
	}

	if _, err := util.SendTextMessage(
		c.b,
		uint64(chatId),
		"✅ Пул успешно удален!",
	); err != nil {
		log.Error(err)
	}

	err = util.DeleteMessage(ctx, c.b, uint64(chatId), messageId)
	if err != nil {
		log.Error(err)
		return
	}

	if _, err := c.ops.Create(
		p.OwnerId,
		appModels.OP_DELETE_POOL,
		fmt.Sprintf("Удаление пула %v. ID: %v", p.JettonName, p.Id.Int64),
	); err != nil {
		log.Error(err)
	}
	return
}
