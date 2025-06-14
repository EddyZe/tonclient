package util

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	appModels "tonclient/internal/models"
)

// отнимает процент от числа
func SubProcientFromNumber(num1, procient float64) float64 {
	temp := num1 * (procient / 100)
	return num1 - temp
}

func CalculateProcientEditPrice(currentPrice, oldPrice float64) float64 {
	if oldPrice == 0 {
		if currentPrice == 0 {
			return 0
		} else {
			return 10000
		}
	}
	subCurrentPriceAndOld := currentPrice - oldPrice
	return (subCurrentPriceAndOld / oldPrice) * 100
}

func CalculateInsurance(pool *appModels.Pool, stake *appModels.Stake) float64 {
	internalValue := stake.Amount * stake.DepositCreationPrice
	currentValue := stake.Amount * stake.JettonPriceClosed
	loss := internalValue - currentValue
	insurance := loss / stake.JettonPriceClosed
	return math.Min(stake.StartPoolDeposit*0.9, insurance)
}

func RemoveZeroFloat(number float64) string {
	num, _ := strconv.ParseFloat(fmt.Sprintf("%.9f", number), 64)
	str := strconv.FormatFloat(num, 'f', -1, 64)
	return str
}

func CalculateSumStakesFromPool(stakes *[]appModels.Stake, p *appModels.Pool) float64 {
	res := 0.
	for _, stake := range *stakes {
		if !stake.IsActive && !stake.IsRewardPaid && !stake.IsInsurancePaid {
			if CalculateProcientEditPrice(stake.JettonPriceClosed, stake.DepositCreationPrice) < float64(p.InsuranceCoating)*-1 {
				am := CalculateInsurance(p, &stake)
				profit := stake.Balance - stake.Amount
				res += am + profit
			} else {
				res += stake.Balance - stake.Amount
			}
		}

		if stake.IsActive {
			res += stake.Amount * 20
		}
	}

	if res < 0 {
		return 0
	}

	return res
}

func ReplaceThreeZerosToK(num int64) string {
	kCount := 0
	n := float64(num)

	for n >= 1000 {
		n /= 1000
		kCount++
	}

	// Формат: до одного знака после запятой, без лишних нулей
	nStr := strconv.FormatFloat(n, 'f', 1, 64)
	nStr = strings.TrimSuffix(nStr, ".0")

	return nStr + strings.Repeat("k", kCount)
}
