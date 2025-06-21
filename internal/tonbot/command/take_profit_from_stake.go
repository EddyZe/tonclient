package command

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	appModels "tonclient/internal/models"
	"tonclient/internal/services"
	"tonclient/internal/util"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type TakeProfitFromStake struct {
	b   *bot.Bot
	us  *services.UserService
	ss  *services.StakeService
	ps  *services.PoolService
	ws  *services.WalletTonService
	aws *services.AdminWalletService
	ops *services.OperationService
	ts  *services.TelegramService
}

func NewTakeProfitFromStake(
	b *bot.Bot,
	us *services.UserService,
	ps *services.PoolService,
	ws *services.WalletTonService,
	aws *services.AdminWalletService,
	ss *services.StakeService,
	ops *services.OperationService,
	ts *services.TelegramService,
) *TakeProfitFromStake {
	return &TakeProfitFromStake{
		b:   b,
		us:  us,
		ps:  ps,
		ws:  ws,
		aws: aws,
		ss:  ss,
		ops: ops,
		ts:  ts,
	}
}

func (c *TakeProfitFromStake) Execute(ctx context.Context, callback *models.CallbackQuery) {
	if err := util.CheckTypeMessage(c.b, callback); err != nil {
		return
	}
	data := callback.Data
	chatId := callback.From.ID
	stakeId, err := c.getStakeIdFromCallbackData(data)
	if err != nil {
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			"❌ Что-то пошло не так. Повторите попытку позже!",
		); err != nil {
			log.Println(err)
		}
		return
	}
	stake, err := c.ss.GetById(stakeId)
	if err != nil {
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			"❌ Стейк не найден! Возможно он был удален!",
		); err != nil {
			log.Println(err)
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
			"❌ Награда или возмещение уже были получены!",
		); err != nil {
			log.Error(err)
		}
		return
	}

	pool, err := c.ps.GetId(stake.PoolId)
	if err != nil {
		log.Error("get pool err: ", err.Error())
		return
	}

	jettonMaster := pool.JettonMaster
	u, err := c.us.GetByTelegramChatId(uint64(chatId))
	if err != nil {
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			"❌ Аккаунт не активирован. Введите команду /start!",
		); err != nil {
			log.Println(err)
		}
		return
	}

	w, err := c.ws.GetByUserId(uint64(u.Id.Int64))
	if err != nil {
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			"❌ Не возможно получить награды, так как у вас не привязан кошелек!",
		); err != nil {
			log.Println(err)
		}
		return
	}

	jettonaData, err := c.aws.DataJetton(jettonMaster)
	if err != nil {
		log.Error("get jettona err: ", err.Error())
		return
	}

	if stake.Balance > pool.Reserve {
		util.SendMessageOwnerAndUserIfBadReserve(
			uint64(chatId),
			pool.OwnerId,
			uint64(pool.Id.Int64),
			jettonaData.Name,
			c.b,
			c.ts,
		)
		stake.IsRewardPaid = false
		if err := c.ss.Update(stake); err != nil {
			log.Error(err)
			return
		}
		return
	}

	boc, err := c.aws.SendJetton(jettonMaster, w.Addr, "", util.RemoveZeroFloat(stake.Balance), jettonaData.Decimals)
	if err != nil {
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			"❌ На данный момент вывод не возможен. Повторите попытку позже!",
		); err != nil {
			log.Println(err)
		}
		return
	}

	stake.IsRewardPaid = true
	if err := c.ss.Update(stake); err != nil {
		log.Error(err)
		return
	}

	pool.Reserve -= stake.Balance - stake.Amount
	stakes := c.ss.GetPoolStakes(stake.PoolId)
	pool.TempReserve = pool.Reserve - util.CalculateSumStakesFromPool(&stakes, pool)
	if err := c.ps.Update(pool); err != nil {
		log.Error("update pool err: ", err.Error())
		return
	}

	hash := base64.StdEncoding.EncodeToString(boc)

	if _, err := util.SendTextMessage(
		c.b,
		uint64(chatId),
		"💸 Токены были отправлены. Hash: "+hash,
	); err != nil {
		log.Println(err)
	}

	if _, err := c.ops.Create(
		uint64(u.Id.Int64),
		appModels.OP_CLAIM,
		fmt.Sprintf("Снятие токенов. %f %v. Hash: %v", stake.Balance, jettonaData.Name, hash),
	); err != nil {
		log.Error("create op err: ", err.Error())
	}
}

func (c *TakeProfitFromStake) getStakeIdFromCallbackData(data string) (uint64, error) {
	splitdata := strings.Split(data, ":")
	if len(splitdata) != 2 {
		return 0, errors.New("invalid data")
	}

	id, err := strconv.ParseUint(splitdata[1], 10, 64)
	if err != nil {
		return 0, errors.New("invalid id")
	}

	return id, nil
}
