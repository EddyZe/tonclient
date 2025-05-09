package models

const (
	//операции
	OP_STAKE             = "stake"             //застейкать токены
	OP_CLAIM             = "claim"             //забрать депозит + награду + компенсация
	OP_CLAIM_INSURANCE   = "insurance_claim"   //забрать компенсацию
	OP_ADMIN_CREATE_POOL = "admin_create_pool" //создать пул
	OP_ADMIN_ADD_RESERVE = "admin_add_reserve" //добавить резерв
	OP_ADMIN_CLOSE_POOL  = "admin_close_pool"  //закрыть пул
	OP_GET_USER_STAKES   = "get_user_stake"    // показать список активных стейков пользователя
)

type SubmitTransaction struct {
	Hash             string
	OperationType    string
	Reward           uint32 // награда
	UserId           uint64
	Amount           float64
	Currency         string
	JettonWalletAddr string
}
