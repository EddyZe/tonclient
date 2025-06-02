package userstate

var CurrentState = make(map[int64]int)

const (
	EnterWalletAddr int = iota
	ConnectTonConnect

	//Create pool
	EnterJettonMasterAddress
	SelectPeriodHold
	EnterCustomPeriodHold
	EnterProfitOnPercent
	EnterInsuranceCoating
	EnterAmountTokens

	//addreserve
	EnterAddReserveTokens

	//stakes
	CreateStake
)

func ResetState(chatId int64) {
	delete(CurrentState, chatId)
}
