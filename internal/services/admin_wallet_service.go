package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strconv"
	"tonclient/internal/config"
	"tonclient/internal/models"
	"tonclient/internal/util"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/jetton"
	"github.com/xssnick/tonutils-go/ton/wallet"
)

type AdminWalletService struct {
	cfg      *config.TonClientConfig
	poolServ *PoolService
	tgServ   *TelegramService
}

func NewAdminWalletService(config *config.TonClientConfig, ps *PoolService, ts *TelegramService) *AdminWalletService {
	return &AdminWalletService{
		cfg:      config,
		poolServ: ps,
		tgServ:   ts,
	}
}

func (s *AdminWalletService) StartSubscribeTransaction() {
	ctx := context.Background()
	api, err := initApi(ctx)
	if err != nil {
		log.Error(err)
		return
	}

	master, err := api.CurrentMasterchainInfo(ctx) // we fetch block just to trigger chain proof check
	if err != nil {
		log.Fatalln("get masterchain info err: ", err.Error())
		return
	}

	wall, err := getWallet(api, s.cfg.Seed)
	if err != nil {
		log.Error(err)
		return
	}

	wallAddr := wall.WalletAddress()
	treasuryAddress := address.MustParseAddr(wallAddr.String())

	acc, err := api.GetAccount(ctx, master, treasuryAddress)
	if err != nil {
		log.Fatalln("get masterchain info err: ", err.Error())
		return
	}

	lastProcessedLT := acc.LastTxLT
	transactions := make(chan *tlb.Transaction)

	go api.SubscribeOnTransactions(ctx, treasuryAddress, lastProcessedLT, transactions)

	log.Infoln("waiting for transfers...")

	//netstrah := jetton.NewJettonMasterClient(api, address.MustParseAddr(s.cfg.JettonAdminContract))

	//treasuryJettonWallet, err := netstrah.GetJettonWalletAtBlock(context.Background(), treasuryAddress, master)
	//if err != nil {
	//	log.Fatalln("get jetton wallet address err: ", err.Error())
	//	return
	//}

	for tx := range transactions {
		if tx.IO.In != nil && tx.IO.In.MsgType == tlb.MsgTypeInternal {
			ti := tx.IO.In.AsInternal()
			src := ti.SrcAddr

			if dsc, ok := tx.Description.(tlb.TransactionDescriptionOrdinary); ok && dsc.BouncePhase != nil {
				if _, ok = dsc.BouncePhase.Phase.(tlb.BouncePhaseOk); ok {
					continue
				}
			}

			if !ti.ExtraCurrencies.IsEmpty() {
				kv, err := ti.ExtraCurrencies.LoadAll()
				if err != nil {
					log.Fatalln("load extra currencies err: ", err.Error())
					return
				}

				for _, dictKV := range kv {
					currencyId := dictKV.Key.MustLoadUInt(32)
					amount := dictKV.Value.MustLoadVarUInt(32)

					log.Infoln("received", amount.String(), "ExtraCurrency with id", currencyId, "from", src.String())
				}
			}
			var transfer jetton.TransferNotification
			if err = tlb.LoadFromCell(&transfer, ti.Body.BeginParse()); err == nil {

				src = transfer.Sender
				payload := transfer.ForwardPayload.BeginParse()
				op := payload.MustLoadUInt(32)
				payloadDataBase64, err := payload.LoadStringSnake()
				if err != nil {
					log.Fatalln("load payload err: ", err.Error())
					continue
				}

				amount, err := strconv.ParseFloat(ti.Amount.String(), 64)
				if err != nil {
					log.Fatalln("parse amount err: ", err.Error())
					continue
				}

				s.processOperation(op, amount, payloadDataBase64)
			}

			if ti.Amount.Nano().Sign() > 0 {
				log.Println("received", ti.Amount.String(), "TON from", src.String())
			}
		}

		lastProcessedLT = tx.LT
	}

}

func getLastMaster(ctx context.Context, api *ton.APIClient) (*ton.BlockIDExt, error) {
	lastMaster, err := api.CurrentMasterchainInfo(ctx)
	if err != nil {
		log.Error("Failed to get master info:", err)
		return nil, err
	}

	return lastMaster, nil
}

func initApi(ctx context.Context) (*ton.APIClient, error) {
	client := liteclient.NewConnectionPool()
	cfg, err := liteclient.GetConfigFromUrl(ctx, config.CONFIG_TON_MAINNET_URL)
	if err != nil {
		log.Fatalln("get config err: ", err.Error())
		return nil, err
	}
	if err := client.AddConnectionsFromConfig(ctx, cfg); err != nil {
		log.Error("Failed to add connections to config server:", err)
		return nil, err
	}
	api := ton.NewAPIClient(client)
	api.SetTrustedBlockFromConfig(cfg)
	return api, nil
}

func getWallet(api *ton.APIClient, seed []string) (*wallet.Wallet, error) {
	return wallet.FromSeed(api, seed, wallet.HighloadV2Verified)
}

func (s *AdminWalletService) processOperation(op uint64, amount float64, payloadDataBase64 string) {
	data, err := base64.StdEncoding.DecodeString(payloadDataBase64)
	if err != nil {
		log.Infoln("Failed to decode payload data:", err)
		return
	}

	log.Infoln(op)
	log.Infoln(string(data))

	switch op {
	case models.OP_STAKE:

		break
	case models.OP_CLAIM:
		break
	case models.OP_CLAIM_INSURANCE:
		break
	case models.OP_ADMIN_CREATE_POOL:
		var pool models.Pool
		if err := json.Unmarshal(data, &pool); err != nil {
			log.Errorf("Failed to unmarshal payload data: %v", err)
			return
		}

		_, err := s.poolServ.CreatePool(&pool)
		if err != nil {
			log.Errorf("Failed to create pool: %v", err)
			return
		}

		telegram, err := s.tgServ.GetByUserId(pool.OwnerId)
		if err != nil {
			log.Errorf("Failed to get telegram: %v", err)
			return
		}

		util.SendMessage(telegram.TelegramId, "Пул создан")

		break
	case models.OP_ADMIN_ADD_RESERVE:
		break
	case models.OP_ADMIN_CLOSE_POOL:
		break
	case models.OP_GET_USER_STAKES:
		break
	case models.OP_PAY_COMMISION:
		var pool models.Pool
		if amount < config.COMMISSION_AMOUNT {
			log.Error("Invalid amount received:", amount)
			return
		}
		if err := json.Unmarshal(data, &pool); err != nil {
			log.Errorf("Failed to unmarshal payload data: %v", err)
			return
		}

		if !pool.Id.Valid {
			log.Error("pool id is not valid")
			return
		}

		id := pool.Id.Int64

		if err := s.poolServ.SetCommissionPaid(uint64(id), true); err != nil {
			log.Errorf("Failed to set commission paid: %v", err)
			return
		}

		tg, err := s.tgServ.GetByUserId(pool.OwnerId)
		if err != nil {
			log.Errorf("Failed to get telegram: %v", err)
			return
		}

		util.SendMessage(tg.TelegramId, "Комиссия оплачена")

		break
	default:
		return
	}
}
