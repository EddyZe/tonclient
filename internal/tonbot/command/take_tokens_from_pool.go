package command

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	appModels "tonclient/internal/models"
	"tonclient/internal/services"
	"tonclient/internal/util"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type TakeTokens struct {
	b   *bot.Bot
	us  *services.UserService
	ps  *services.PoolService
	ss  *services.StakeService
	aws *services.AdminWalletService
	ws  *services.WalletTonService
	opS *services.OperationService
}

func NewTakeTokensCommand(
	b *bot.Bot,
	us *services.UserService,
	ps *services.PoolService,
	ss *services.StakeService,
	aws *services.AdminWalletService,
	ws *services.WalletTonService,
	opS *services.OperationService,
) *TakeTokens {
	return &TakeTokens{
		b:   b,
		us:  us,
		ps:  ps,
		ss:  ss,
		aws: aws,
		ws:  ws,
		opS: opS,
	}
}

func (c *TakeTokens) Execute(ctx context.Context, callback *models.CallbackQuery) {
	if err := util.CheckTypeMessage(c.b, callback); err != nil {
		return
	}

	chatId := callback.From.ID
	splitData := strings.Split(callback.Data, ":")
	if len(splitData) != 3 {
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			"❌ Не могу обработать данную кнопку! Попробуйте позже!",
		); err != nil {
			log.Error(err)
		}
		return
	}

	poolId, err := strconv.ParseInt(splitData[1], 10, 64)
	if err != nil {
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			"❌ Не могу обработать данную кнопку! Попробуйте позже!",
		); err != nil {
			log.Error(err)
		}
		return
	}

	u, err := c.us.GetByTelegramChatId(uint64(chatId))
	if err != nil {
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			"❌ Аккаунт не активирован. Введите /start",
		); err != nil {
			log.Error(err)
		}
		return
	}

	p, err := c.ps.GetId(uint64(poolId))
	if err != nil {
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			"❌ Пул не найден! Возможно он был удален!",
		); err != nil {
			log.Error(err)
		}
		return
	}

	if p.Reserve <= 0 {
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			"❌ Резерв пуст",
		); err != nil {
			log.Error(err)
		}
		return
	}

	if p.OwnerId != uint64(u.Id.Int64) {
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			"❌ Вы не создатель пула",
		); err != nil {
			log.Error(err)
		}
		return
	}

	if !p.IsCommissionPaid {
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			"❌ Комиссия не оплачена!",
		); err != nil {
			log.Error(err)
		}
		return
	}

	if p.IsActive {
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			"❌ Сначала вы должны закрыть пул!",
		); err != nil {
			log.Error(err)
		}
		return
	}

	stakes := c.ss.CountStakesPoolIdAndStatus(uint64(poolId), true)
	if stakes > 0 {
		text := fmt.Sprintf(
			"❌ Нельзя вывести токены пока есть активные стейки. Вывод будет доступен, когда стейки будут закрыты! Активных стейков: %d",
			stakes,
		)
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			text,
		); err != nil {
			log.Error(err)
		}
		return
	}

	w, err := c.ws.GetByUserId(uint64(u.Id.Int64))
	if err != nil {
		if _, err := util.SendTextMessage(c.b, uint64(chatId), "❌ У вас не привязан кошелек!"); err != nil {
			log.Error(err)
		}
		return
	}

	jettonData, err := c.aws.DataJetton(p.JettonMaster)
	if err != nil {
		log.Error(err)
		if _, err := util.SendTextMessage(c.b, uint64(chatId), "❌ Что-то пошло не так. Повторите попытку"); err != nil {
			log.Error(err)
		}
		return
	}

	hash, err := c.aws.SendJetton(
		p.JettonMaster,
		w.Addr,
		"",
		p.Reserve,
		jettonData.Decimals,
	)
	if err != nil {
		log.Error(err)
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			"❌ Произошла ошибка при выводе средств, повторите попытку позже!",
		); err != nil {
			log.Error(err)
		}
		return
	}

	currentReserve := p.Reserve
	p.Reserve = 0
	if err := c.ps.Update(p); err != nil {
		log.Error(err)
		return
	}

	resp := fmt.Sprintf("✅ Снятие средст прошло успешно! Снято: %v %v", currentReserve, jettonData.Name)

	if _, err := util.SendTextMessage(c.b, uint64(chatId), resp); err != nil {
		log.Error(err)
		return
	}

	str := base64.StdEncoding.EncodeToString(hash)

	if _, err := c.opS.Create(
		uint64(u.Id.Int64),
		appModels.OP_CLAIM_RESERVE,
		fmt.Sprintf("Снятие резерва. Hash: %v", str),
	); err != nil {
		log.Error(err)
		return
	}
}
