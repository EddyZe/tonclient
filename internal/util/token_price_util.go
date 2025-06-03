package util

import (
	"math"
	"strconv"
	"tonclient/internal/dyor"
	"tonclient/internal/tonfi"
)

func GetCurrentPriceJettonAddr(addr string) float64 {
	jettonData, err := tonfi.GetAssetByAddr(addr)
	if err != nil {
		jettonData = &tonfi.Asset{}
	}

	currentPrice, err := strconv.ParseFloat(jettonData.DexPriceUsd, 64)
	if err != nil {
		currentPrice = 0.
	}

	if currentPrice == 0 {
		resp, err := dyor.GetPrices(addr)
		if err != nil {
			currentPrice = 0.
		} else {
			currentPrice, err = strconv.ParseFloat(resp.Currency.Price.Value, 64)
			if err != nil {
				log.Infoln(err.Error())
				currentPrice = 0.
			}

			currentPrice = currentPrice / math.Pow10(resp.Currency.Price.Decimals)
			log.Infoln(currentPrice)
		}
	}

	return currentPrice
}
