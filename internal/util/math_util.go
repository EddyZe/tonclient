package util

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
