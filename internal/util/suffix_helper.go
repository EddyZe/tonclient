package util

import (
	"strconv"
	"strings"
)

func SuffixDay(num int) string {
	a := "дней"
	b := "дня"
	def := "день"
	return suffix(num, a, b, def)
}

func SuffixPol(num int) string {
	a := "пулов"
	b := "пула"
	def := "пул"

	return suffix(num, a, b, def)
}

func suffix(num int, var1, var2, var3 string) string {
	va := []string{"6", "8", "9", "0", "7", "11", "5", "12", "13", "14"}
	vb := []string{"2", "3", "4"}

	numStr := strconv.Itoa(num)
	for _, c := range va {
		if strings.HasSuffix(numStr, c) {
			return var1
		}
	}

	for _, c := range vb {
		if strings.HasSuffix(numStr, c) {
			return var2
		}
	}

	return var3
}
