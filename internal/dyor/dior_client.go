package dyor

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const url = "https://api.dyor.io/v1/jettons/%v/price?currency=usd"

type PriceData struct {
	Value    string `json:"value"`
	Decimals int    `json:"decimals"`
}

type CurrencyInfo struct {
	Price     PriceData `json:"price"`
	ChangedAt time.Time `json:"changedAt"`
}

type CurrencyRates struct {
	TON      CurrencyInfo `json:"ton"`
	USD      CurrencyInfo `json:"usd"`
	Currency CurrencyInfo `json:"currency"`
}

func GetPrices(addr string) (*CurrencyRates, error) {
	u := fmt.Sprintf(url, addr)
	resp, err := http.Get(u)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
	}

	log.Println(string(body))

	var res CurrencyRates
	err = json.Unmarshal(body, &res)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return &res, nil
}
