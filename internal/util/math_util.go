package util

// отнимает процент от числа
func SubProcientFromNumber(num1, procient float64) float64 {
	temp := num1 * (procient / 100)
	return num1 - temp
}
