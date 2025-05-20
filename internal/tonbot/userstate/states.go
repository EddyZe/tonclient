package userstate

var CurrentState = make(map[int64]int)

const (
	EnterWalletAddr int = iota
	ConnectTonConnect

	//Create pool
	EnterJettonMasterAddress
	SelectPeriodHold
	EnterCustomPeriodHold
	EnterJettonWallet
	EnterProfitOnPercent
	EnterInsuranceCoating
	EnterAmountTokens
)

func ResetState(chatId int64) {
	CurrentState[chatId] = -1
}
