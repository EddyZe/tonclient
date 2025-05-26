package command

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
	appModels "tonclient/internal/models"
	"tonclient/internal/services"
	"tonclient/internal/tonbot/buttons"
	"tonclient/internal/tonfi"
	"tonclient/internal/util"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type OpenStakeInfo struct {
	b  *bot.Bot
	ss *services.StakeService
	ps *services.PoolService
}

func NewOpenStakeInfoCommand(b *bot.Bot, ss *services.StakeService, ps *services.PoolService) *OpenStakeInfo {
	return &OpenStakeInfo{
		b:  b,
		ss: ss,
		ps: ps,
	}
}

func (c *OpenStakeInfo) Execute(ctx context.Context, callback *models.CallbackQuery) {
	if err := util.CheckTypeMessage(c.b, callback); err != nil {
		return
	}

	chatId := callback.From.ID

	jettonName, stakeId, err := c.getDataFromCallback(callback.Data)
	if err != nil {
		log.Error(err)
		return
	}

	stake, err := c.ss.GetById(stakeId)
	if err != nil {
		log.Error(err)
		if _, err := util.SendTextMessage(c.b, uint64(chatId), "❌ Стейк не найден. Возможно он был удален!"); err != nil {
			log.Error(err)
		}
		return
	}

	p, err := c.ps.GetId(stake.PoolId)
	if err != nil {
		log.Error(err)
		return
	}

	info := c.generateInfo(stake, jettonName, p.Period)
	btns := make([]models.InlineKeyboardButton, 0, 3)

	buttonId := fmt.Sprintf("%v:%v", buttons.OpenGroupId, jettonName)
	backBtn := util.CreateDefaultButton(buttonId, buttons.BackStakesFromGroup)

	asset, err := tonfi.GetAssetByAddr(p.JettonMaster)
	if err != nil {
		log.Error(err)
		return
	}

	currentPrice, err := strconv.ParseFloat(asset.DexPriceUsd, 64)
	if err != nil {
		log.Error(err)
		currentPrice = 0.0
	}

	subPrice := util.SubProcientFromNumber(stake.DepositCreationPrice, 30)
	//TODO реализовать кнопки для сбора страховки и получения наград
	if currentPrice <= subPrice {
		btnInsurance := util.CreateDefaultButton("test", "test")
		btns = append(btns, btnInsurance)
	} else {
		btn := util.CreateDefaultButton("test2", "test2")
		btns = append(btns, btn)
	}

	btns = append(btns, backBtn)

	markup := util.CreateInlineMarup(1, btns...)

	if err := util.EditTextMessageMarkup(
		ctx,
		c.b,
		uint64(chatId),
		callback.Message.Message.ID,
		info,
		markup,
	); err != nil {
		log.Error(err)
		return
	}
}

func (c *OpenStakeInfo) generateInfo(stake *appModels.Stake, jettonName string, period uint) string {
	text := `
	<b>Стейк с токеном %v</b>

	<b>Сумма стейка</b>: %v
	<b>Цена на момент стейка:</b> %.9f $
	<b>Старт стейка</b>: %v
	<b>Осталось до закрытия</b>: %v %v
	<b>Общий баланс</b>: %v
	<b>Заработано на текущий момент</b>: %v
	`
	profit := stake.Balance - stake.Amount
	leftDay := stake.StartDate.Add(time.Duration(period) * 24 * time.Hour).Sub(time.Now())
	timeFormat := "02 January 2006 15:04:05"
	formatText := fmt.Sprintf(
		text,
		jettonName,
		stake.Amount,
		stake.DepositCreationPrice,
		stake.StartDate.Format(timeFormat),
		int(leftDay.Hours()/24),
		util.SuffixDay(int(leftDay.Hours()/24)),
		stake.Balance,
		profit)

	return formatText
}

func (c *OpenStakeInfo) getDataFromCallback(data string) (jettonName string, stakeId uint64, err error) {
	splitData := strings.Split(data, ":")
	jettonName = splitData[1]
	stakeId, err = strconv.ParseUint(splitData[2], 10, 64)
	if err != nil {
		return "", 0, err
	}
	return jettonName, stakeId, nil
}
