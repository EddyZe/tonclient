package command

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
	appModels "tonclient/internal/models"
	"tonclient/internal/services"
	"tonclient/internal/tonbot/buttons"
	"tonclient/internal/util"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type OpenStakeInfo struct {
	b       *bot.Bot
	ss      *services.StakeService
	ps      *services.PoolService
	backBtn string
}

func NewOpenStakeInfoCommand(b *bot.Bot, ss *services.StakeService, ps *services.PoolService, backbtn string) *OpenStakeInfo {
	return &OpenStakeInfo{
		b:       b,
		ss:      ss,
		ps:      ps,
		backBtn: backbtn,
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
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			"❌ Стейк не найден. Возможно он был удален!",
		); err != nil {
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

	buttonId := fmt.Sprintf("%v:%v", c.backBtn, jettonName)
	backBtn := util.CreateDefaultButton(buttonId, buttons.BackStakesFromGroup)

	if stake.EndDate.After(time.Now()) && stake.IsActive {
		info += "\n\n<b>При досрочном закрытии</b>:\n- Нет компенсации падения цены\n- Процент за стейкинг не начисляется"
		idBtn := fmt.Sprintf("%v:%v", buttons.CloseStakeId, stake.Id.Int64)
		btn := util.CreateDefaultButton(idBtn, buttons.CloseStake)
		btns = append(btns, btn)
	}

	if !stake.IsActive {
		procientEditPrice := util.CalculateProcientEditPrice(stake.JettonPriceClosed, stake.DepositCreationPrice)
		log.Infoln(procientEditPrice)
		if procientEditPrice < float64(p.InsuranceCoating)*-1 && !stake.IsInsurancePaid && !stake.IsRewardPaid {
			idbtn := fmt.Sprintf("%v:%v", buttons.TakeInsuranceId, stake.Id.Int64)
			btnInsurance := util.CreateDefaultButton(idbtn, buttons.TakeInsurance)
			btns = append(btns, btnInsurance)
		} else if !stake.IsRewardPaid && !stake.IsInsurancePaid {
			idBtn := fmt.Sprintf("%v:%v", buttons.TakeProfitId, stake.Id.Int64)
			btn := util.CreateDefaultButton(idBtn, buttons.TakeProfit)
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
		log.Error(markup)
		log.Error(err)
		return
	}
}

func (c *OpenStakeInfo) generateInfo(stake *appModels.Stake, jettonName string, pool *appModels.Pool) string {
	currentPrice := util.GetCurrentPriceJettonAddr(pool.JettonMaster)
	status := "⌛ Активен"
	if !stake.IsActive {
		if !stake.IsCommissionPaid && !stake.IsRewardPaid {
			status = "✅ Выплачен"
		} else {
			status = "✅ Завершен"
		}
	}

	text := `
	<b> 📦 Стейк с токеном [%v]</b>
	<b>💰 Ставка:</b> +%v%% за %v %v
	<b>🛡 Компенсация:</b> При снижении цены %v более чем на %v%%

	<b>💰 Вложено:</b> %v
	<b>💵 Цена на входе:</b> %v $
	<b>📉 Текущая цена:</b> %v $ (%v%%)
	<b>📅Старт:</b> %v
	<b>📅Финиш:</b> %v
	<b>⏳ Статус:</b> %v
	<b>🎁 Доход</b> +%v %v
	`
	profit := stake.Balance - stake.Amount
	//leftDay := stake.StartDate.Add(time.Duration(pool.Period) * 24 * time.Hour).Sub(time.Now())
	procientPriceEdit := int(util.CalculateProcientEditPrice(currentPrice, stake.DepositCreationPrice))
	timeFormat := "02 January 2006 15:04:05"
	formatText := fmt.Sprintf(
		text,
		jettonName,
		util.RemoveZeroFloat(pool.Reward*float64(pool.Period)),
		pool.Period,
		util.SuffixDay(int(pool.Period)),
		jettonName,
		pool.InsuranceCoating,
		util.RemoveZeroFloat(stake.Amount),
		util.RemoveZeroFloat(stake.DepositCreationPrice),
		util.RemoveZeroFloat(currentPrice),
		procientPriceEdit,
		stake.StartDate.Format(timeFormat),
		stake.StartDate.Add(time.Duration(pool.Period)*24*time.Hour).Format(timeFormat),
		status,
		util.RemoveZeroFloat(profit),
		pool.JettonName,
	)

	if !stake.IsActive {
		formatText += fmt.Sprintf(
			"\n\n<b>📉 Цена на момент закрытия стейка</b>: %v$ (%v%%)",
			util.RemoveZeroFloat(stake.JettonPriceClosed),
			int(util.CalculateProcientEditPrice(stake.JettonPriceClosed, stake.DepositCreationPrice)),
		)
		if !stake.IsInsurancePaid && !stake.IsRewardPaid {
			paid := 0.
			precientEdit := util.CalculateProcientEditPrice(stake.JettonPriceClosed, stake.DepositCreationPrice)
			if precientEdit < float64(pool.InsuranceCoating)*-1 {
				insurance := util.CalculateInsurance(pool, stake)
				paid += insurance + stake.Balance
				formatText += fmt.Sprintf(
					"\n<b>💥 Компенсация за падение на %.0f%%</b>: %v %v",
					math.Ceil(precientEdit),
					util.RemoveZeroFloat(insurance),
					pool.JettonName,
				)
			} else {
				paid += stake.Balance
				formatText += fmt.Sprintf(
					"\n<b>💥 Выплата с процентами(%.0f%%)</b>: %v %v",
					math.Ceil(precientEdit),
					util.RemoveZeroFloat(paid),
					pool.JettonName,
				)
			}
			formatText += fmt.Sprintf(
				"\n<b>💎 К выплате</b>: %v %v", util.RemoveZeroFloat(paid), pool.JettonName)
		}
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
