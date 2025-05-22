package tonfi

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
)

const (
	TonfiBaseUrl = "https://api.ston.fi/v1"
	TonfiAsset   = "/assets"
)

type AssetInfo struct {
	Asset Asset `json:"asset"`
}

type Asset struct {
	ContractAddress string   `json:"contract_address"`
	Symbol          string   `json:"symbol"`
	DisplayName     string   `json:"display_name"`
	Decimals        int      `json:"decimals"`
	ImageUrl        string   `json:"image_url"`
	Kind            string   `json:"kind"`
	Priority        int      `json:"priority"`
	Deprecated      bool     `json:"deprecated"`
	Community       bool     `json:"community"`
	Blacklisted     bool     `json:"blacklisted"`
	DefaultSymbol   bool     `json:"default_symbol"`
	Taxable         bool     `json:"taxable"`
	PopularityIndex float64  `json:"popularity_index"`
	Tags            []string `json:"tags"`
	DexUsdPrice     string   `json:"dex_usd_price"`
	DexPriceUsd     string   `json:"dex_price_usd"`
}

func GetAssetByAddr(addr string) (*Asset, error) {
	var res AssetInfo
	resp, err := http.Get(TonfiBaseUrl + TonfiAsset + "/" + addr)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			return
		}
	}(resp.Body)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	err = json.Unmarshal(body, &res)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return &res.Asset, nil
}
