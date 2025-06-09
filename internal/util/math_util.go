package util

import (
	"math"
	"strconv"
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
	tenProcientFromReserve := pool.Reserve * 0.1
	ninetyProcientFromReserve := pool.Reserve * 0.9

	share := stake.Amount / tenProcientFromReserve

	insurance := ninetyProcientFromReserve * share
	insurance = math.Ceil(insurance)
	return insurance
}

func RemoveZeroFloat(number float64) string {
	str := strconv.FormatFloat(number, 'f', -1, 64)
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
	}

	if res < 0 {
		return 0
	}

	return res
}
