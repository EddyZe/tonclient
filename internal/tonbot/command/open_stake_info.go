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

	info := c.generateInfo(stake, jettonName, p)
	btns := make([]models.InlineKeyboardButton, 0, 3)

	buttonId := fmt.Sprintf("%v:%v", buttons.OpenGroupId, jettonName)
	backBtn := util.CreateDefaultButton(buttonId, buttons.BackStakesFromGroup)

	//TODO реализовать кнопки для сбора страховки и получения наград
	if !stake.IsActive {
		procientEditPrice := util.CalculateProcientEditPrice(stake.JettonPriceClosed, stake.DepositCreationPrice)
		log.Infoln(procientEditPrice)
		if procientEditPrice <= -30 {
			btnInsurance := util.CreateDefaultButton("test", "test")
			btns = append(btns, btnInsurance)
		} else {
			btn := util.CreateDefaultButton("test2", "test2")
			btns = append(btns, btn)
		}
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

func (c *OpenStakeInfo) generateInfo(stake *appModels.Stake, jettonName string, pool *appModels.Pool) string {
	jettonData, err := tonfi.GetAssetByAddr(pool.JettonMaster)
	if err != nil {
		log.Error(err)
		return ""
	}

	currentPrice, err := strconv.ParseFloat(jettonData.DexPriceUsd, 64)
	if err != nil {
		currentPrice = 0.
	}

	text := `
	<b>Стейк с токеном %v</b>

	<b>Ставка:</b> +%v%% за %v %v
	<b>Гарантия:</b> Компенсация при снижении цены %v более чем на 30%%

	<b>Сумма стейка:</b> %v
	<b>Цена на момент стейка:</b> %v $
	<b>Текущая цена:</b> %f $ (%v%%)

	<b>Старт:</b> %v
	<b>Стоп:</b> %v

	<b>Осталось дней:</b> %v %v

	<b>Заработано:</b> %v %v
	<b>Итого баланс:</b> %v %v
	`
	profit := stake.Balance - stake.Amount
	leftDay := stake.StartDate.Add(time.Duration(pool.Period) * 24 * time.Hour).Sub(time.Now())
	procientPriceEdit := util.CalculateProcientEditPrice(currentPrice, stake.DepositCreationPrice)
	timeFormat := "02 January 2006 15:04:05"
	formatText := fmt.Sprintf(
		text,
		jettonName,
		pool.Reward*pool.Period,
		pool.Period,
		util.SuffixDay(int(pool.Period)),
		jettonName,
		stake.Amount,
		stake.DepositCreationPrice,
		currentPrice,
		procientPriceEdit,
		stake.StartDate.Format(timeFormat),
		stake.StartDate.Add(time.Duration(pool.Period)*24*time.Hour).Format(timeFormat),
		int(leftDay.Hours()/24),
		util.SuffixDay(int(leftDay.Hours()/24)),
		profit,
		jettonName,
		stake.Balance,
		jettonName,
	)

	if !stake.IsActive {
		formatText += fmt.Sprintf(
			"\n\n<b>Цена на момент закрытия стейка</b>: %f$ (%v%%)",
			stake.JettonPriceClosed,
			util.CalculateProcientEditPrice(stake.JettonPriceClosed, stake.DepositCreationPrice),
		)
	}

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
