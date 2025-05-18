package userstate

var CurrentState = make(map[int64]int)

const (
	EnterWalletAddr int = iota
)
