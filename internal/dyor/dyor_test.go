package dyor

import (
	"log"
	"testing"
)

func TestGetPrices(t *testing.T) {
	res, err := GetPrices("EQDP42F8wAbBIqccOstlCpQCV73RfK0lClfOFBw2Nf8xm2DF")
	if err != nil {
		log.Println(err)
	}

	log.Println(res)
}
