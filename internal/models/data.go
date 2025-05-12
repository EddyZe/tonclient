package models

type JettonData struct {
	TotalSupply uint64
	Mintable    bool
	AdminAddr   string
	Name        string
	Symbol      string
	Decimals    int
	Description string
}
