package command

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
	"tonclient/internal/services"
	"tonclient/internal/util"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type CloseStake struct {
	b   *bot.Bot
	aws *services.AdminWalletService
	ws  *services.WalletTonService
	ss  *services.StakeService
	ps  *services.PoolService
}

func NewCloseStakeCommand(
	b *bot.Bot,
	aws *services.AdminWalletService,
	ws *services.WalletTonService,
	ss *services.StakeService,
	ps *services.PoolService,
) *CloseStake {
	return &CloseStake{
		b:   b,
		aws: aws,
		ws:  ws,
		ss:  ss,
		ps:  ps,
	}
}

func (c *CloseStake) Execute(ctx context.Context, callback *models.CallbackQuery) {
	if err := util.CheckTypeMessage(c.b, callback); err != nil {
		return
	}

	chatId := callback.From.ID
	splitData := strings.Split(callback.Data, ":")
	if len(splitData) != 2 {
		return
	}

	stakeId, err := strconv.ParseUint(splitData[1], 10, 64)
	if err != nil {
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			"❌ Не могу обработать данную кнопку",
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

	if stake.IsRewardPaid || stake.IsInsurancePaid {
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			"❌ Токены уже получены!",
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

	w, err := c.ws.GetByUserId(stake.UserId)
	if err != nil {
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			"❌ У вас не привязан кошелек!",
		); err != nil {
			log.Println(err)
		}
		return
	}

	p, err := c.ps.GetId(stake.PoolId)
	if err != nil {
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			"❌ Не смог найти нужный пул!",
		); err != nil {
			log.Println(err)
		}
		return
	}

	adminAmount := stake.Balance - stake.Amount
	p.Reserve -= adminAmount

	if err := c.ps.Update(p); err != nil {
		log.Println(err)
	}

	jettonData, err := c.aws.DataJetton(p.JettonMaster)
	if err != nil {
		log.Println(err)
		return
	}

	if _, err := c.aws.SendJetton(
		p.JettonMaster,
		w.Addr,
		"",
		stake.Amount,
		jettonData.Decimals,
	); err != nil {
		stake.IsRewardPaid = false
		if err := c.ss.Update(stake); err != nil {
			log.Error(err)
			return
		}
		log.Println(err)
		return
	}

	if _, err := util.SendTextMessage(
		c.b,
		uint64(chatId),
		fmt.Sprintf("💸 %f %v были отправлены на ваш привязанный кошелек: %v", stake.Amount, p.JettonName, w.Addr),
	); err != nil {
		log.Println(err)
		return
	}

	closePrice := util.GetCurrentPriceJettonAddr(p.JettonMaster)

	stake.IsActive = false
	stake.CloseDate = time.Now()
	stake.IsRewardPaid = true
	stake.EndDate = time.Now()
	stake.JettonPriceClosed = closePrice
	if err := c.ss.Update(stake); err != nil {
		log.Println("error update stake id ", stake.Id.Int64, "error: ", err)
		return
	}

	if _, err := c.aws.SendJetton(
		p.JettonMaster,
		c.aws.GetUserAdminAddr(),
		"",
		adminAmount,
		jettonData.Decimals,
	); err != nil {
		log.Println(err)
		return
	}

}
