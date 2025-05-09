package services

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"errors"
	"strconv"
	"time"
	"tonclient/internal/config"

	"github.com/cameo-engineering/tonconnect"
	"github.com/redis/go-redis/v9"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/ton/wallet"
	"github.com/xssnick/tonutils-go/tvm/cell"
)

var log = config.InitLogger()

const TON_MANIFEST_URL = "https://raw.githubusercontent.com/cameo-engineering/tonconnect/master/tonconnect-manifest.json"

type TonConnectService struct {
	redisCli *redis.Client
}

func NewTonConnectService(redis *redis.Client) *TonConnectService {
	return &TonConnectService{
		redisCli: redis,
	}
}

func (s *TonConnectService) LoadSession(key string) (*tonconnect.Session, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	result, err := s.redisCli.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}

	if err != nil {
		log.Error("Error loading session", err)
		return nil, err
	}

	var session tonconnect.Session
	err = session.UnmarshalJSON([]byte(result))
	if err != nil {
		log.Error("Error loading session", err)
		return nil, err
	}
	return &session, nil
}

func (s *TonConnectService) SaveSession(key string, session *tonconnect.Session) error {
	data, err := session.MarshalJSON()
	if err != nil {
		log.Error("Error marshaling session json", err)
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	return s.redisCli.Set(ctx, key, data, 0).Err()
}

func (s *TonConnectService) DeleteSession(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	return s.redisCli.Del(ctx, key).Err()
}

func (s *TonConnectService) GenerateConnectUrls(session *tonconnect.Session) (connectUrls map[string]string, error error) {
	result := make(map[string]string)
	data := make([]byte, 32)
	_, err := rand.Read(data)
	if err != nil {
		log.Error("Error generating connect url", err)
		return nil, err
	}
	connreq, err := tonconnect.NewConnectRequest(
		TON_MANIFEST_URL,
		tonconnect.WithProofRequest(base32.StdEncoding.EncodeToString(data)),
	)
	if err != nil {
		log.Error("Error generating connect urls", err)
		return nil, err
	}
	deeplink, err := session.GenerateDeeplink(*connreq, tonconnect.WithBackReturnStrategy())
	if err != nil {
		log.Error("Error generating deeplink", err)
		return nil, err
	}

	log.Debugln("Generated deeplink: ", deeplink)

	for _, w := range tonconnect.Wallets {
		if w.Name == "Tonkeeper" || w.Name == "Tonhub" {
			link, err := session.GenerateUniversalLink(w, *connreq)
			log.Debugln("Generated link: ", link)
			if err != nil {
				return nil, err
			}
			result[w.Name] = link
		}

	}

	return result, nil
}

func (s *TonConnectService) Connect(session *tonconnect.Session) error {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	res, err := session.Connect(ctx, tonconnect.Wallets["tonkeeper"], tonconnect.Wallets["tonhub"])
	if err != nil {
		log.Error("Error generating connect urls", err)
		return err
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
	log.Printf(
		"%s %s for %s is connected to %s with %s address\n\n",
		res.Device.AppName,
		res.Device.AppVersion,
		res.Device.Platform,
		network,
		addr,
	)

	return nil
}

func (s *TonConnectService) CreateSession() (*tonconnect.Session, error) {
	return tonconnect.NewSession()
}

func (s *TonConnectService) SendJettonTransaction(jettonAddr, receiverAddr, senderAddr, amount, comment string, session *tonconnect.Session) ([]byte, error) {

	commentCell, err := wallet.CreateCommentCell(comment)
	if err != nil {
		log.Error("Error creating commentCell", err)
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	parsed, err := strconv.ParseFloat(amount, 64)

	payload := cell.BeginCell().
		MustStoreUInt(0x0f8a7ea5, 32).                      // opcode
		MustStoreUInt(uint64(time.Now().Unix()), 64).       // query_id (UNIX timestamp)
		MustStoreCoins(uint64(parsed) * 1e9).               // amount (с учетом decimals!)
		MustStoreAddr(address.MustParseAddr(receiverAddr)). // destination
		MustStoreAddr(address.MustParseAddr(senderAddr)).   // response_destination
		MustStoreBoolBit(false).                            // custom_payload
		MustStoreCoins(0.01 * 1e9).                         // forward_ton_amount (0.05 TON)
		MustStoreMaybeRef(commentCell).                     // forward_payload
		EndCell()

	msg, err := tonconnect.NewMessage(
		jettonAddr,
		strconv.FormatUint(0.05*1e9, 10),
		tonconnect.WithPayload(payload.ToBOC()),
	)
	if err != nil {
		log.Error("Error creating transaction", err)
		return nil, err
	}
	tx, err := tonconnect.NewTransaction(
		tonconnect.WithTimeout(5*time.Minute),
		tonconnect.WithMessage(*msg),
	)
	if err != nil {
		log.Error("Error creating transaction", err)
		return nil, err
	}

	boc, err := session.SendTransaction(ctx, *tx)
	if err != nil {
		log.Error("Error sending transaction", err)
		return nil, err
	}
	return boc, nil
}

func (s *TonConnectService) SendTransaction(receiverAddr, amount, comment string, session *tonconnect.Session) ([]byte, error) {
	commentCell, err := wallet.CreateCommentCell(comment)
	if err != nil {
		log.Error("Error creating commentCell", err)
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	msg, err := tonconnect.NewMessage(
		receiverAddr,
		amount,
		tonconnect.WithPayload(commentCell.ToBOC()),
	)
	if err != nil {
		log.Error("Error creating transaction", err)
		return nil, err
	}
	tx, err := tonconnect.NewTransaction(
		tonconnect.WithTimeout(5*time.Minute),
		tonconnect.WithMessage(*msg),
	)
	if err != nil {
		log.Error("Error creating transaction", err)
		return nil, err
	}

	boc, err := session.SendTransaction(ctx, *tx)
	if err != nil {
		log.Error("Error sending transaction", err)
		return nil, err
	}
	return boc, nil
}
