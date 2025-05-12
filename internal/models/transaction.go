package models

const (
	//операции
	OP_STAKE             = 1 //застейкать токены
	OP_CLAIM             = 2 //забрать депозит + награду + компенсация
	OP_CLAIM_INSURANCE   = 3 //забрать компенсацию
	OP_ADMIN_CREATE_POOL = 4 //создать пул
	OP_ADMIN_ADD_RESERVE = 5 //добавить резерв
	OP_ADMIN_CLOSE_POOL  = 6 //закрыть пул
	OP_GET_USER_STAKES   = 7 // показать список активных стейков пользователя
	OP_PAY_COMMISION     = 8
)

type SubmitTransaction struct {
	OperationType    string
	Reward           uint32 // награда
	UserId           uint64
	Amount           float64
	Currency         string
	JettonWalletAddr string
}

type Payload struct {
	OperationType uint64 `json:"operation_type"`
	JettonMaster  string `json:"master_jetton"`
	Payload       string `json:"payload"`
}

type AddReserve struct {
	PoolId uint64  `json:"pool_id"`
	Amount float64 `json:"amount"`
}
