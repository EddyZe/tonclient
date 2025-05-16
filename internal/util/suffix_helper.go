package util

import (
	"strconv"
	"strings"
)

func SuffixDay(num int) string {
	a := "дней"
	b := "дня"
	def := "день"

	va := []string{"6", "8", "9", "0", "7", "11", "5"}
	vb := []string{"2", "3", "4"}

	numStr := strconv.Itoa(num)
	for _, c := range va {
		if strings.HasSuffix(numStr, c) {
			return a
		}
	}

	for _, c := range vb {
		if strings.HasSuffix(numStr, c) {
			return b
		}
	}

	return def
}
