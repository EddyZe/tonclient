package util

import (
	"fmt"
	"strconv"
	appModels "tonclient/internal/models"
	"tonclient/internal/services"
	"tonclient/internal/tonbot/buttons"
	"tonclient/internal/tonfi"

	"github.com/go-telegram/bot/models"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

func generateNamePool(pool *appModels.Pool, aws *services.AdminWalletService) string {
	jettonData, err := aws.DataJetton(pool.JettonMaster)
	if err != nil {
		return "Без названия"
	}
	return fmt.Sprintf("%v (%d %v / %d%% / резерв %v)", jettonData.Name, pool.Period, SuffixDay(int(pool.Period)), pool.Reward, pool.Reserve)
}

func GeneratePoolButtons(pool *[]appModels.Pool, aws *services.AdminWalletService, suf string) []models.InlineKeyboardButton {
	res := make([]models.InlineKeyboardButton, 0, len(*pool))
	for _, p := range *pool {
		if !p.Id.Valid {
			continue
		}
		poolId := p.Id.Int64
		res = append(
			res,
			CreateDefaultButton(
				fmt.Sprintf("%v:%d:%v", buttons.PoolDataButton, poolId, suf),
				generateNamePool(&p, aws),
			),
		)
	}
	return res
}

func PoolInfo(p *appModels.Pool, ss *services.StakeService) string {
	allStakesPool := ss.GetPoolStakes(uint64(p.Id.Int64))
	var sumAmount float64

	if allStakesPool != nil {
		for _, stake := range *allStakesPool {
			sumAmount += stake.Amount
		}
	}

	foramter := message.NewPrinter(language.English)
	ut := foramter.Sprintf("%.2f", sumAmount)
	reserve := foramter.Sprintf("%.2f", p.Reserve)

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
	if jettonInfo.DexPriceUsd != "" {
		price, err = strconv.ParseFloat(jettonInfo.DexPriceUsd, 64)
		if err != nil {
			log.Error(err)
			price = 0
		}
	}

	i := `
<b> Описание пула %v: </b>

<b>Статус</b>: %v
<b>Текущая цена токена:</b> %.6f$

<b>📈 Доходность: </b>
%v%% в день начисляется на ваш застейканый баланс.

<b>⏳Срок холда:</b>
%v %v без возможности досрочного вывода

<b>🛡️ Страховка:</b>
Если цена токена упадёт более чем на %v%% за время холда — вам будет выплачена компенсация.

<b>💸 Максимальная компенсация:</b>
До 30%% от вашей стейкнутой суммы.

🔒 Резерв пула:
 •	Заблокировано участниками: %v токенов
 •	Доступно для новых стейков: %v токенов
`

	res := fmt.Sprintf(i, jettonInfo.DisplayName, status, price, p.Reward, p.Period, SuffixDay(int(p.Period)), p.InsuranceCoating, ut, reserve)
	return res
}

func GenerateOwnerPoolInlineKeyboard(poolId int64, backPoolListButtonId string, isActive bool, sufData string) *models.InlineKeyboardMarkup {
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

	return CreateInlineMarup(1, paidCommision, addReserve, closePool, takeTokens, backListPools)
}
