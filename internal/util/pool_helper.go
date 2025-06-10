package util

import (
	"fmt"
	"math"
	"strconv"
	"tonclient/internal/dyor"
	appModels "tonclient/internal/models"
	"tonclient/internal/services"
	"tonclient/internal/tonbot/buttons"
	"tonclient/internal/tonfi"

	"github.com/go-telegram/bot/models"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

func generateNamePool(pool *appModels.Pool, aws *services.AdminWalletService, subSum float64) string {
	jettonData, err := aws.DataJetton(pool.JettonMaster)
	currentReserve := pool.Reserve - subSum
	if currentReserve < 0 {
		currentReserve = 0
	}
	if err != nil {
		return "Без названия"
	}
	return fmt.Sprintf(
		"%v (%d %v / %v%% / резерв %v)",
		jettonData.Name,
		pool.Period,
		SuffixDay(int(pool.Period)),
		pool.Reward,
		RemoveZeroFloat(currentReserve),
	)
}

func GeneratePoolButtons(pool *[]appModels.Pool, aws *services.AdminWalletService, suf string, ss *services.StakeService) []models.InlineKeyboardButton {
	res := make([]models.InlineKeyboardButton, 0, len(*pool))
	for _, p := range *pool {
		if !p.Id.Valid {
			continue
		}
		poolId := p.Id.Int64
		subSubStake := 0.
		stakes := ss.GetPoolStakes(uint64(poolId))
		subSubStake = CalculateSumStakesFromPool(stakes, &p)
		res = append(
			res,
			CreateDefaultButton(
				fmt.Sprintf("%v:%d:%v", buttons.PoolDataButton, poolId, suf),
				generateNamePool(&p, aws, subSubStake),
			),
		)
	}
	return res
}

func PoolInfo(p *appModels.Pool, ss *services.StakeService, jettonData *appModels.JettonData) string {
	allStakesPool := ss.GetPoolStakes(uint64(p.Id.Int64))
	var sumAmount float64
	subReserve := 0.

	if allStakesPool != nil {
		for _, stake := range *allStakesPool {
			if stake.IsActive {
				sumAmount += stake.Amount
			}
		}
		subReserve = CalculateSumStakesFromPool(allStakesPool, p)
	}

	tenProcientReserve := (p.Reserve - subReserve) * 0.1
	if tenProcientReserve < 0 {
		tenProcientReserve = 0
	}

	currentReserve := p.Reserve - subReserve
	if currentReserve < 0 {
		currentReserve = 0
	}

	foramter := message.NewPrinter(language.English)
	ut := foramter.Sprintf("%v", RemoveZeroFloat(sumAmount))
	reserve := foramter.Sprintf("%v", RemoveZeroFloat(tenProcientReserve))
	fullReserve := foramter.Sprintf("%v", RemoveZeroFloat(currentReserve))

	var status string
	if p.IsActive {
		status = "✅ Активен"
	} else {
		status = "⏳ Пул не активен"
	}

	jettonInfo, err := tonfi.GetAssetByAddr(p.JettonMaster)
	if err != nil {
		log.Error(err)
		return "-"
	}
	price := 0.
	price, err = strconv.ParseFloat(jettonInfo.DexPriceUsd, 64)
	if err != nil {
		log.Error(err)
		price = 0
	}
	if price == 0 {
		resp, err := dyor.GetPrices(p.JettonMaster)
		if err != nil {
			price = 0.
		} else {
			price, err = strconv.ParseFloat(resp.Currency.Price.Value, 64)
			if err != nil {
				log.Infoln(err.Error())
				price = 0.
			}

			price = price / math.Pow10(resp.Currency.Price.Decimals)
			log.Infoln(price)
		}
	}

	reliability := (p.Reserve / (jettonData.TotalSupply / (10e+8))) / 0.72 * 100
	reliability = math.Min(reliability, 100)

	var emoj string
	var level string

	if reliability < 5 {
		emoj = "🟥"
		level = "низкий"
	} else if reliability < 20 {
		emoj = "🟨"
		level = "средний"
	} else {
		emoj = "🟩"
		level = "высокий"
	}

	res := fmt.Sprintf(
		`
<b> 📦 Описание пула %v: </b>

<b>Статус</b>: %v
<b>Текущая цена токена:</b> %v$

<b>📈 Доходность: </b>
%v%% в день начисляется на застейканую сумму.

<b>⏳Срок холда:</b>
%v %v с возможностью досрочного вывода (но тогда без награды за стейкинг).

<b>💵 Минимальный размер стейка </b>
%v %v

<b>🛡️ Страховка:</b>
Если цена токена упадет более чем на %v%% к моменту окончания стейкинга, вам будет выплачена компенсация

<b>💸 Максимальная компенсация:</b>
До 50%% от вашей стейкнутой суммы.

🔒 Резерв пула:
 •	Заблокировано участниками: %v токенов
 •	Доступно для новых стейков: %v токенов
 •  Общий резерв: %v

🔐 <b>Надежность пула</b>: %v %v%% из 100%%
Уровень: %v, резерв составляет %v из %v токенов`,
		jettonInfo.DisplayName,
		status,
		RemoveZeroFloat(price),
		RemoveZeroFloat(p.Reward),
		p.Period,
		SuffixDay(int(p.Period)),
		RemoveZeroFloat(p.MinStakeAmount),
		p.JettonName,
		p.InsuranceCoating,
		ut,
		reserve,
		fullReserve,
		emoj,
		RemoveZeroFloat(reliability),
		level,
		RemoveZeroFloat(currentReserve),
		RemoveZeroFloat(jettonData.TotalSupply/(10e+8)),
	)
	return res
}

func GenerateOwnerPoolInlineKeyboard(poolId int64, backPoolListButtonId string, isActive, commissionPaid bool, sufData string) *models.InlineKeyboardMarkup {
	paidCommision := CreateDefaultButton(fmt.Sprintf("%v:%v", buttons.PaidCommissionId, poolId), buttons.PaidCommission)
	addReserve := CreateDefaultButton(fmt.Sprintf("%v:%v:%v", buttons.AddReserveId, poolId, sufData), buttons.AddReserve)
	var closePoolText string
	if isActive {
		closePoolText = buttons.ClosePool
	} else {
		closePoolText = buttons.OpePool
	}
	takeTokens := CreateDefaultButton(fmt.Sprintf("%v:%v:%v", buttons.TakeTokensId, poolId, sufData), buttons.TakeTokens)
	closePool := CreateDefaultButton(fmt.Sprintf("%v:%v:%v", buttons.ClosePoolId, poolId, sufData), closePoolText)
	backListPools := CreateDefaultButton(backPoolListButtonId, buttons.BackPoolList)
	deletePool := CreateDefaultButton(fmt.Sprintf("%v:%v", buttons.DeletePoolId, poolId), buttons.DeletePool)
	btns := make([]models.InlineKeyboardButton, 0, 5)
	if !commissionPaid {
		btns = append(btns, paidCommision)
	}

	btns = append(btns, addReserve)
	btns = append(btns, closePool)
	btns = append(btns, takeTokens)
	btns = append(btns, deletePool)
	btns = append(btns, backListPools)

	return CreateInlineMarup(1, btns...)
}
