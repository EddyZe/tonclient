package main

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
	"tonclient/internal/config"
	"tonclient/internal/database"
	"tonclient/internal/tonbot"

	"github.com/cameo-engineering/tonconnect"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/jetton"
	"github.com/xssnick/tonutils-go/ton/wallet"
	"github.com/xssnick/tonutils-go/tvm/cell"
)

const TON_MANIFEST_URL = "https://raw.githubusercontent.com/cameo-engineering/tonconnect/master/tonconnect-manifest.json"

func main() {
	run3()
}

func run3() {
	logger := config.InitLogger()
	if err := config.InitConfig(); err != nil {
		logger.Fatalf("Failed to init config: %v", err)
	}

	logger.Infoln("Config initialized")

	connectPostgres()
	logger.Infoln("Database initialized")

	tokenBot := os.Getenv("TELEGRAM_BOT_TOKEN")

	logger.Infoln("Telegram bot starting:", tokenBot)
	err := tonbot.StartBot(tokenBot)
	if err != nil {
		log.Fatal("Failed to start bot: ", err)
	}

}

func connectPostgres() *database.Postgres {
	psqlConfig := config.LoadPostgresConfig()
	psql, err := database.NewPostgres(psqlConfig)
	if err != nil {
		log.Fatal("Failed to connect to database")
	}
	defer func(db *database.Postgres) {
		err := db.Close()
		if err != nil {
			log.Fatal("Failed to close database")
		}
	}(psql)

	if err := psql.Ping(); err != nil {
		log.Fatal("Failed to ping database")
	}

	return psql
}

func run2() error {
	log := config.InitLogger()

	log.Infoln("Creating new session")
	s, err := tonconnect.NewSession()
	if err != nil {
		log.Error("Error creating new session", err)
		return err
	}

	data := make([]byte, 32)
	_, err = rand.Read(data)
	if err != nil {
		log.Error("Error generating random data", err)
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	connreq, err := tonconnect.NewConnectRequest(
		TON_MANIFEST_URL,
		tonconnect.WithProofRequest(base32.StdEncoding.EncodeToString(data)),
	)
	if err != nil {
		log.Error("Error creating connect request", err)
		return err
	}

	deeplink, err := s.GenerateDeeplink(*connreq, tonconnect.WithBackReturnStrategy())
	if err != nil {
		log.Error("Error generating deeplink", err)
		return err
	}
	fmt.Printf("Deeplink: %s\n\n", deeplink)

	wrapped := tonconnect.WrapDeeplink(deeplink)
	fmt.Printf("Wrapped deeplink: %s\n\n", wrapped)

	for _, w := range tonconnect.Wallets {
		link, err := s.GenerateUniversalLink(w, *connreq)
		fmt.Printf("%s: %s\n\n", w.Name, link)
		if err != nil {
			log.Error("Error generating universal link", err)
			return err
		}
	}

	var wts []tonconnect.Wallet
	for _, w := range tonconnect.Wallets {
		link, err := s.GenerateUniversalLink(w, *connreq)
		wts = append(wts, w)
		fmt.Printf("%s: %s\n\n", w.Name, link)
		if err != nil {
			log.Fatal(err)
		}
	}

	res, err := s.Connect(ctx, wts...)
	if err != nil {
		log.Fatal(err)
	}

	var addr string
	network := "mainnet"
	for _, item := range res.Items {
		if item.Name == "ton_addr" {
			addr = item.Address
			if item.Network == -3 {
				network = "testnet"
			}
		}
	}
	fmt.Printf(
		"%s %s for %s is connected to %s with %s address\n\n",
		res.Device.AppName,
		res.Device.AppVersion,
		res.Device.Platform,
		network,
		addr,
	)

	msg, err := tonconnect.NewMessage(
		"UQCrciOc9HE341fFtBs-WFuttXeciFDIvFwafCO4QQhAinLG",
		"100000000",
	)
	if err != nil {
		log.Fatal(err)
	}

	tx, err := tonconnect.NewTransaction(
		tonconnect.WithTimeout(10*time.Minute),
		tonconnect.WithMessage(*msg),
	)
	if err != nil {
		log.Fatal(err)
	}
	boc, err := s.SendTransaction(ctx, *tx)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Printf("Bag of Cells: %x", boc)
	}
	return nil
}

func run() error {
	log := config.InitLogger()
	if err := config.InitConfig(); err != nil {
		log.Error(err)
		return err
	}

	client := liteclient.NewConnectionPool()
	if err := client.AddConnectionsFromConfigUrl(context.Background(), config.CONFIG_TON_MAINNET_URL); err != nil {
		log.Error("Failed to add connections to config server:", err)
		return err
	}

	api := ton.NewAPIClient(client)

	seed := config.WALLET_SEED

	wall, err := wallet.FromSeed(api, seed, wallet.HighloadV2Verified)
	if err != nil {
		log.Error("Failed to get seed:", err)
		return err
	}

	log.Infoln(wall.WalletAddress())

	lastMaster, err := api.CurrentMasterchainInfo(context.Background())
	if err != nil {
		log.Error("Failed to get master info:", err)
		return err
	}

	balance, err := wall.GetBalance(context.Background(), lastMaster)
	if err != nil {
		log.Error("Failed to get balance:", err)
		return err
	}

	log.Infoln(balance)

	contract := address.MustParseAddr("EQBynBO23ywHy_CgarY9NK9FTz0yDsG82PtcbSTQgGoXwiuA")
	master := jetton.NewJettonMasterClient(api, contract)

	// get information about jetton
	data, err := master.GetJettonData(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	s := data.TotalSupply.String()
	zeros := len(s) - len(strings.TrimRight(s, "0"))
	fmt.Println(zeros)

	log.Println("total supply:", data.TotalSupply.String())

	// get jetton wallet for account
	ownerAddr := address.MustParseAddr(wall.WalletAddress().String())
	jettonWallet, err := master.GetJettonWallet(context.Background(), ownerAddr)
	if err != nil {
		log.Fatal(err)
	}

	jettonBalance, err := jettonWallet.GetBalance(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	log.Println("balance jetton:", jettonBalance.String())

	if err := wall.Send(
		context.Background(),
		wallet.SimpleMessage(
			contract,
			tlb.MustFromTON("0.05"),
			cell.BeginCell().
				MustStoreAddr(address.MustParseAddr("0QDYdanKBMZxz-McvhlrCq4etuUnc2lk1XDySB9RgXCS-MST")).
				EndCell(),
		),
		true,
	); err != nil {
		log.Error(err)
	}

	//cell, err := method.Cell(3)
	//if err != nil {
	//	log.Error("Failed to get cell:", err)
	//	return err
	//}

	//accountInfo, err := api.GetAccount(context.Background(), lastMaster, wall.WalletAddress())
	//if err != nil {
	//	log.Error("Failed to get account:", err)
	//	return err
	//}
	//
	//transactions := make(chan *tlb.Transaction)
	//
	//uid, err := uuid.NewRandom()
	//if err != nil {
	//	log.Error("Failed to get random uuid:", err)
	//	uid = uuid.New()
	//	return err
	//}
	//
	//log.Infoln(uid.String())
	//
	//go api.SubscribeOnTransactions(context.Background(), wall.WalletAddress(), accountInfo.LastTxLT, transactions)
	//
	//for {
	//	select {
	//	case tx := <-transactions:
	//		if tx.IO.In.MsgType != tlb.MsgTypeInternal {
	//			continue
	//		}
	//
	//		internal := tx.IO.In.AsInternal()
	//		log.Infoln("sender: ", internal.SrcAddr.String())
	//		log.Infoln("receiver: ", internal.DstAddr.String())
	//		log.Infoln("amount: ", internal.Amount.String())
	//
	//		if internal.Body == nil {
	//			continue
	//		}
	//
	//		if internal.Bounced {
	//			continue
	//		}
	//
	//		body := internal.Body.BeginParse()
	//		opcode, err := body.LoadUInt(32)
	//		if err != nil {
	//			log.Error("Failed to load big int:", err)
	//			continue
	//		}
	//
	//		log.Infoln(opcode)
	//		if opcode != 0 {
	//			continue
	//		}
	//
	//		com, err := body.LoadStringSnake()
	//		if err != nil {
	//			log.Error("Failed to load string snake:", err)
	//			continue
	//		}
	//
	//		log.Infoln(com)
	//		if com == uid.String() {
	//			log.Infoln("deposit confirmed, uuid equals")
	//		}
	//		break
	//	}
	//}

	return nil
}
