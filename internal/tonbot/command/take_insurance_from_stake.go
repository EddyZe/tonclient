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

type TakeInsuranceFromStake struct {
	b   *bot.Bot
	us  *services.UserService
	ss  *services.StakeService
	ps  *services.PoolService
	ts  *services.TelegramService
	ops *services.OperationService
	ws  *services.WalletTonService
	aws *services.AdminWalletService
}

func NewTakeInsuranceFromStake(
	b *bot.Bot,
	us *services.UserService,
	ss *services.StakeService,
	ps *services.PoolService,
	ts *services.TelegramService,
	ops *services.OperationService,
	ws *services.WalletTonService,
	aws *services.AdminWalletService,
) *TakeInsuranceFromStake {
	return &TakeInsuranceFromStake{
		b:   b,
		us:  us,
		ss:  ss,
		ps:  ps,
		ts:  ts,
		ops: ops,
		ws:  ws,
		aws: aws,
	}
}

func (c *TakeInsuranceFromStake) Execute(ctx context.Context, callback *models.CallbackQuery) {
	if err := util.CheckTypeMessage(c.b, callback); err != nil {
		return
	}

	data := callback.Data
	chatId := callback.From.ID
	splitdata := strings.Split(data, ":")
	if len(splitdata) != 2 {
		return
	}

	stakeId, err := strconv.ParseInt(splitdata[1], 10, 64)
	if err != nil {
		log.Error(err)
		return
	}

	stake, err := c.ss.GetById(uint64(stakeId))
	if err != nil {
		log.Error(err)
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			"❌ Стейк не найден. Возможно он был удален.",
		); err != nil {
			log.Error(err)
		}
		return
	}

	if stake.IsActive {
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			"❌ Дождитесь закрытия стейка!",
		); err != nil {
			log.Error(err)
		}
		return
	}

	if stake.IsRewardPaid || stake.IsInsurancePaid {
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			"❌ Возмещение или вознаграждение уже были выплачены!",
		); err != nil {
			log.Error(err)
		}
		return
	}

	pool, err := c.ps.GetId(stake.PoolId)
	if err != nil {
		log.Error(err)
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			"❌ Пул не был найден.",
		); err != nil {
			log.Error(err)
		}
		return
	}

	u, err := c.us.GetById(stake.UserId)
	if err != nil {
		log.Error(err)
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			"❌ Ваш аккаунт не активирован! Введите команду /start",
		); err != nil {
			log.Error(err)
		}
		return
	}

	w, err := c.ws.GetByUserId(uint64(u.Id.Int64))
	if err != nil {
		log.Error(err)
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			"❌ У вас не привязан кошелек! Возмещение не возможно!",
		); err != nil {
			log.Error(err)
		}
		return
	}

	jettonData, err := c.aws.DataJetton(pool.JettonMaster)
	if err != nil {
		log.Error(err)
		return
	}

	insurance := util.CalculateInsurance(pool, stake)
	amount := stake.Balance + insurance
	profit := stake.Balance - stake.Amount

	if pool.Reserve < amount {
		util.SendMessageOwnerAndUserIfBadReserve(
			uint64(chatId),
			pool.OwnerId,
			uint64(pool.Id.Int64),
			jettonData.Name,
			c.b,
			c.ts,
		)
		return
	}

	boc, err := c.aws.SendJetton(
		pool.JettonMaster,
		w.Addr,
		"",
		amount,
		jettonData.Decimals,
	)
	if err != nil {
		log.Error(err)
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			"❌ Ошибка. Пока что вывод не доступен. Попробуйте позже",
		); err != nil {
			log.Error(err)

		}
		return
	}

	stake.IsInsurancePaid = true
	if err := c.ss.Update(stake); err != nil {
		log.Error(err)
		return
	}

	pool.Reserve -= profit + insurance
	stakes := c.ss.GetPoolStakes(stake.PoolId)
	pool.TempReserve = pool.Reserve - util.CalculateSumStakesFromPool(stakes, pool)
	if err := c.ps.Update(pool); err != nil {
		log.Error(err)
		return
	}

	hash := base64.StdEncoding.EncodeToString(boc)

	if _, err := c.ops.Create(
		uint64(u.Id.Int64),
		appModels.OP_CLAIM_INSURANCE,
		fmt.Sprintf("\n-Получение страховки.\n-Сумма: %v %v.\n-Hash: %v", amount, jettonData.Name, hash),
	); err != nil {
		log.Error(err)
		return
	}

	if _, err := util.SendTextMessage(
		c.b,
		uint64(chatId),
		fmt.Sprintf("✅ Вам отправлено %v %v\nHash операции: %v", amount, jettonData.Name, hash),
	); err != nil {
		log.Error(err)
	}

}
