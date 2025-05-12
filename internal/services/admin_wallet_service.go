package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
	"tonclient/internal/config"
	"tonclient/internal/models"
	"tonclient/internal/tonbot"
	"tonclient/internal/util"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/jetton"
	"github.com/xssnick/tonutils-go/ton/nft"
	"github.com/xssnick/tonutils-go/ton/wallet"
)

type AdminWalletService struct {
	poolServ        *PoolService
	tgServ          *TelegramService
	stakeServ       *StakeService
	tgbot           *tonbot.TgBot
	api             *ton.APIClient
	master          *ton.BlockIDExt
	wallet          *wallet.Wallet
	acc             *tlb.Account
	transaction     chan *tlb.Transaction
	treasuryAddress *address.Address
}

func NewAdminWalletService(config *config.TonClientConfig, ps *PoolService, ts *TelegramService, ss *StakeService, tgbot *tonbot.TgBot) (*AdminWalletService, error) {
	ctx := context.Background()
	api, err := initApi(ctx)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	master, err := api.CurrentMasterchainInfo(ctx) // we fetch block just to trigger chain proof check
	if err != nil {
		log.Fatalln("get masterchain info err: ", err.Error())
		return nil, err
	}

	wall, err := getWallet(api, config.Seed)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	wallAddr := wall.WalletAddress()
	treasuryAddress := address.MustParseAddr(wallAddr.String())

	acc, err := api.GetAccount(ctx, master, treasuryAddress)
	if err != nil {
		log.Fatalln("get masterchain info err: ", err.Error())
		return nil, err
	}

	lastProcessedLT := acc.LastTxLT
	transactions := make(chan *tlb.Transaction)

	go api.SubscribeOnTransactions(ctx, treasuryAddress, lastProcessedLT, transactions)
	return &AdminWalletService{
		poolServ:        ps,
		tgServ:          ts,
		stakeServ:       ss,
		tgbot:           tgbot,
		api:             api,
		master:          master,
		wallet:          wall,
		treasuryAddress: treasuryAddress,
		acc:             acc,
		transaction:     transactions,
	}, nil
}

func (s *AdminWalletService) StartSubscribeTransaction() {

	log.Infoln("waiting for transfers...")

	//netstrah := jetton.NewJettonMasterClient(api, address.MustParseAddr(s.cfg.JettonAdminContract))

	//treasuryJettonWallet, err := netstrah.GetJettonWalletAtBlock(context.Background(), treasuryAddress, master)
	//if err != nil {
	//	log.Fatalln("get jetton wallet address err: ", err.Error())
	//	return
	//}

	for tx := range s.transaction {
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
			if err := tlb.LoadFromCell(&transfer, ti.Body.BeginParse()); err == nil {

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
		var stake models.Stake
		if err := json.Unmarshal(data, &stake); err != nil {
			log.Error("Failed to unmarshal stake data:", err)
			return
		}

		_, err := s.stakeServ.CreateStake(&stake)
		if err != nil {
			log.Error("Failed to create stake:", err)
			return
		}

		tg, err := s.tgServ.GetByUserId(stake.UserId)
		if err != nil {
			log.Error("Failed to get tg:", err)
			return
		}

		util.SendMessage(tg.TelegramId, "Стейк создан")

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

		log.Infoln(pool)

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
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		s.tgbot.SendMessage(ctx, "Пул создан, чтобы активировать пул оплатите комиссию.", telegram.TelegramId)

		break
	case models.OP_ADMIN_ADD_RESERVE:
		var addReserve models.AddReserve
		if err := json.Unmarshal(data, &addReserve); err != nil {
			log.Errorf("Failed to unmarshal payload data: %v", err)
			return
		}

		newReserve, err := s.poolServ.AddReserve(addReserve.PoolId, addReserve.Amount)
		if err != nil {
			log.Errorf("Failed to add reserve: %v", err)
			return
		}

		pool, err := s.poolServ.GetId(addReserve.PoolId)
		if err != nil {
			log.Errorf("Failed to get pool id: %v", err)
			return
		}

		tg, err := s.tgServ.GetByUserId(pool.OwnerId)
		if err != nil {
			log.Errorf("Failed to get telegram: %v", err)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		s.tgbot.SendMessage(ctx, fmt.Sprintf("Резерв пополнен. Объем нового резерва: %v", newReserve), tg.TelegramId)

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

func (s *AdminWalletService) DataJetton(masterAddr string) (*models.JettonData, error) {
	tokenContract := address.MustParseAddr(masterAddr)
	master := jetton.NewJettonMasterClient(s.api, tokenContract)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	data, err := master.GetJettonData(ctx)
	if err != nil {
		return nil, err
	}

	return getContent(data), nil
}

func getContent(any *jetton.Data) *models.JettonData {
	decimals := 9
	totalSupply := any.TotalSupply.Uint64()
	mintable := any.Mintable
	adminAddr := any.AdminAddr
	name := ""
	description := ""
	symbol := ""
	content := any.Content
	switch content.(type) {
	case *nft.ContentOnchain:
		c := content.(*nft.ContentOnchain)
		name = c.GetAttribute("name")
		symbol = c.GetAttribute("symbol")
		if c.GetAttribute("decimals") != "" {
			d, err := strconv.Atoi(c.GetAttribute("decimals"))
			if err != nil {
				return nil
			}
			decimals = d
		}
		description = c.GetAttribute("description")
		break
	case *nft.ContentSemichain:
		c := content.(*nft.ContentSemichain)
		name = c.GetAttribute("name")
		symbol = c.GetAttribute("symbol")
		if c.GetAttribute("decimals") != "" {
			d, err := strconv.Atoi(c.GetAttribute("decimals"))
			if err != nil {
				return nil
			}
			decimals = d
		}
		description = c.GetAttribute("description")
		break
	}

	return &models.JettonData{
		TotalSupply: totalSupply,
		Mintable:    mintable,
		AdminAddr:   adminAddr.String(),
		Name:        name,
		Symbol:      symbol,
		Decimals:    decimals,
		Description: description,
	}
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
