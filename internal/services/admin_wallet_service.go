package services

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"math"
	"math/big"
	"os"
	"os/signal"
	"strconv"
	"time"
	"tonclient/internal/config"
	"tonclient/internal/models"

	"github.com/go-telegram/bot"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/jetton"
	"github.com/xssnick/tonutils-go/ton/nft"
	"github.com/xssnick/tonutils-go/ton/wallet"
	"github.com/xssnick/tonutils-go/tvm/cell"
)

type AdminWalletService struct {
	poolServ        *PoolService
	tgServ          *TelegramService
	stakeServ       *StakeService
	wallServ        *WalletTonService
	tgbot           *bot.Bot
	api             *ton.APIClient
	master          *ton.BlockIDExt
	wallet          *wallet.Wallet
	acc             *tlb.Account
	transaction     chan *tlb.Transaction
	treasuryAddress *address.Address
	adminWalletAddr string
}

func NewAdminWalletService(config *config.TonClientConfig, ps *PoolService, ts *TelegramService, ss *StakeService, ws *WalletTonService) (*AdminWalletService, error) {
	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt)
	api, err := initApi(ctx)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	adminAddr := os.Getenv("ADMIN_WALLET_ADDR")
	if adminAddr == "" {
		if _, err := address.ParseAddr(adminAddr); err != nil {
			log.Fatalln(adminAddr + " is not a valid address")
		}
		return nil, errors.New("ADMIN_WALLET_ADDR environment variable is not set")
	}

	master, err := api.CurrentMasterchainInfo(ctx)
	if err != nil {
		log.Fatalln("get masterchain info err: ", err.Error())
		return nil, err
	}

	wall, err := getWalletFromSeed(api, config.Seed)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	log.Infoln(wall.WalletAddress().String())
	log.Infoln(wall.WalletAddress().String() == os.Getenv("WALLET_ADDR"))

	wallAddr := wall.WalletAddress()
	treasuryAddress, err := address.ParseAddr(wallAddr.String())
	if err != nil {
		log.Error(err)
		return nil, err
	}

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
		wallServ:        ws,
		api:             api,
		master:          master,
		wallet:          wall,
		treasuryAddress: treasuryAddress,
		acc:             acc,
		transaction:     transactions,
		adminWalletAddr: adminAddr,
	}, nil
}

func (s *AdminWalletService) StartSubscribeTransaction(ch chan models.SubmitTransaction) {

	log.Infoln("waiting for transfers...")

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

				go s.processOperation(op, amount, transfer.Sender.String(), payloadDataBase64, ch)
			}

			if ti.Amount.Nano().Sign() > 0 {
				log.Println("received", ti.Amount.String(), "TON from", src.String())
			}
		}
	}
}

func (s *AdminWalletService) processOperation(op uint64, amount float64, senderAddr, payloadDataBase64 string, ch chan models.SubmitTransaction) {
	data, err := base64.StdEncoding.DecodeString(payloadDataBase64)
	if err != nil {
		log.Infoln("Failed to decode payload data:", err)
		return
	}
	tr := models.SubmitTransaction{
		OperationType: op,
		Amount:        amount,
		Payload:       data,
		SenderAddr:    senderAddr,
	}

	log.Infoln("запись в канал")
	ch <- tr
	log.Infoln("запись добавлена")
}

func (s *AdminWalletService) SendJetton(jettonMaster, receiverAddr, comment string, amount float64, decimal int) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	amountTok, err := tlb.FromDecimal(fmt.Sprint(amount), decimal)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	tokenWallet, err := s.TokenWalletAddress(jettonMaster, s.wallet.WalletAddress())
	if err != nil {
		log.Errorf("Failed to get jetton token: %v", err)
		return nil, err
	}

	balance, err := tokenWallet.GetBalance(ctx)
	if err != nil {
		log.Errorf("Failed to get balance: %v", err)
		return nil, err
	}
	pow := math.Pow10(decimal)

	log.Infoln(float64(balance.Int64()) / pow)
	if float64(balance.Int64())/pow < amount {
		return nil, errors.New("balance is insufficient")
	}

	block, err := s.api.CurrentMasterchainInfo(context.Background())
	if err != nil {
		log.Fatalln("get masterchain info err: ", err.Error())
		return nil, err
	}

	commission := tlb.MustFromTON("0.05")
	balanceTon, err := s.wallet.GetBalance(ctx, block)
	if err != nil {
		log.Fatalln("get balance err: ", err.Error())
		return nil, err
	}

	if balanceTon.Nano().Int64() < commission.Nano().Int64() {
		return nil, errors.New("balance is insufficient")
	}

	to, err := address.ParseAddr(receiverAddr)
	if err != nil {
		log.Errorf("Failed to parse receiver address: %v", err)
		return nil, err
	}

	var c *cell.Cell
	if comment != "" {
		c, err = wallet.CreateCommentCell(comment)
	}

	transferPayload, err := tokenWallet.BuildTransferPayloadV2(
		to,
		s.wallet.WalletAddress(),
		amountTok,
		tlb.ZeroCoins,
		c,
		nil,
	)
	if err != nil {
		log.Errorf("Failed to build transfer payload: %v", err)
		return nil, err
	}

	msg := wallet.SimpleMessage(tokenWallet.Address(), commission, transferPayload)

	log.Infoln("sending transaction...")
	tx, _, err := s.wallet.SendWaitTransaction(ctx, msg)
	if err != nil {
		log.Errorf("Failed to send transaction: %v", err)
		return nil, err
	}

	log.Infoln("transaction confirmed, hash:", base64.StdEncoding.EncodeToString(tx.Hash))
	return tx.Hash, nil
}

func (s *AdminWalletService) TokenWalletAddress(jettonMaster string, walletAddr *address.Address) (*jetton.WalletClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	tokenContract, err := address.ParseAddr(jettonMaster)
	if err != nil {
		log.Error("Failed to parse jetton token address:", err)
		return nil, err
	}
	token := jetton.NewJettonMasterClient(s.api, tokenContract)
	tokenWallet, err := token.GetJettonWallet(ctx, walletAddr)
	if err != nil {
		log.Errorf("Failed to get jetton token: %v", err)
		return nil, err
	}

	return tokenWallet, nil
}

func (s *AdminWalletService) DataJetton(masterAddr string) (*models.JettonData, error) {
	tokenContract, err := address.ParseAddr(masterAddr)
	if err != nil {
		log.Error("Failed to parse jetton token address:", err)
		return nil, err
	}
	master := jetton.NewJettonMasterClient(s.api, tokenContract)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	data, err := master.GetJettonData(ctx)
	if err != nil {
		return nil, err
	}

	return getContent(data), nil
}

func (s *AdminWalletService) CheckValidAddr(addr string) error {
	if _, err := address.ParseAddr(addr); err != nil {
		return err
	}

	return nil
}

func getContent(any *jetton.Data) *models.JettonData {
	decimals := 9
	totalSupply, _ := any.TotalSupply.Float64()
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

func getWalletFromSeed(api *ton.APIClient, seed []string) (*wallet.Wallet, error) {
	return wallet.FromSeed(api, seed, wallet.ConfigV5R1Final{
		NetworkGlobalID: -239,
		Workchain:       0,
	})
}

func (s *AdminWalletService) GetAdminWalletAddr() *address.Address {
	return s.wallet.WalletAddress()
}

func (s *AdminWalletService) GetJettonBalance(wallAddr, jettonMaster string) (*big.Int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	addr, err := address.ParseAddr(wallAddr)
	if err != nil {
		return nil, err
	}
	tokenWallet, err := s.TokenWalletAddress(jettonMaster, addr)
	if err != nil {
		log.Errorf("Failed to get jetton token: %v", err)
		return nil, err
	}

	balance, err := tokenWallet.GetBalance(ctx)
	if err != nil {
		log.Errorf("Failed to get balance: %v", err)
		return nil, err
	}

	return balance, nil
}

func (s *AdminWalletService) GetUserAdminAddr() string {
	return s.adminWalletAddr
}
