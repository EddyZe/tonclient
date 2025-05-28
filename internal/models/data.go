package models

type JettonData struct {
	TotalSupply float64
	Mintable    bool
	AdminAddr   string
	Name        string
	Symbol      string
	Decimals    int
	Description string
}
