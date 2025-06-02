package services

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math"
	"strconv"
	"strings"
	"time"
	"tonclient/internal/config"
	"tonclient/internal/models"

	"github.com/cameo-engineering/tonconnect"
	"github.com/redis/go-redis/v9"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/ton/wallet"
	"github.com/xssnick/tonutils-go/tvm/cell"
)

var log = config.InitLogger()

const TON_MANIFEST_URL = "https://raw.githubusercontent.com/EddyZe/tonclient/refs/heads/master/tonconnect-manifest.json"

type TonConnectService struct {
	redisCli        *redis.Client
	adminWalletServ *AdminWalletService
}

func NewTonConnectService(redis *redis.Client, adminWalletServ *AdminWalletService) *TonConnectService {
	return &TonConnectService{
		redisCli:        redis,
		adminWalletServ: adminWalletServ,
	}
}

func (s *TonConnectService) LoadSession(key string) (*tonconnect.Session, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
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
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	data, err := session.MarshalJSON()
	if err != nil {
		log.Error("Error marshaling session json", err)
		return err
	}
	return s.redisCli.Set(ctx, key, data, 0).Err()
}

func (s *TonConnectService) DeleteSession(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
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
		nameLower := strings.ToLower(w.Name)
		if nameLower == "tonkeeper" || nameLower == "tonhub" {
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

func (s *TonConnectService) GetTonConnector() (*tonconnect.ConnectRequest, error) {
	data := make([]byte, 32)
	connreq, err := tonconnect.NewConnectRequest(
		TON_MANIFEST_URL,
		tonconnect.WithProofRequest(base32.StdEncoding.EncodeToString(data)),
	)
	if err != nil {
		log.Error("Error generating connect urls", err)
	}

	return connreq, nil
}

func (s *TonConnectService) GetWalletUniversalLink(walletName string) string {
	w, err := s.GetWallet(walletName)
	if err != nil {
		return ""
	}

	return w.UniversalURL
}

func (s *TonConnectService) GetWallet(wal string) (*tonconnect.Wallet, error) {
	for _, w := range tonconnect.Wallets {
		nameLower := strings.ToLower(w.Name)
		if nameLower == wal {
			return &w, nil
		}
	}

	return nil, errors.New("wallet not found")
}

func (s *TonConnectService) GetTonkeeperUrl() string {
	return "https://wallet.tonkeeper.com/"
}

func (s *TonConnectService) GetTonkeeperAppUrl() string {
	return "https://app.tonkeeper.com/"
}

func (s *TonConnectService) GetTonhubUrl() string {
	return "https://tonhub.com/"
}

func (s *TonConnectService) Connect(session *tonconnect.Session) (*models.TonConnectResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	res, err := session.Connect(ctx, tonconnect.Wallets["tonkeeper"], tonconnect.Wallets["tonhub"])
	if err != nil {
		log.Error("Error generating connect urls", err)
		return nil, err
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

	return &models.TonConnectResult{
		WalletName: res.Device.AppName,
		Version:    res.Device.AppVersion,
		Addr:       addr,
		Platform:   res.Device.Platform,
	}, nil
}

func (s *TonConnectService) CreateSession() (*tonconnect.Session, error) {
	return tonconnect.NewSession()
}

func (s *TonConnectService) SendJettonTransaction(key, jettonAddr, receiverAddr, senderAddr, amount string, payload *models.Payload, session *tonconnect.Session) ([]byte, error) {
	defer func() {
		if err := s.SaveSession(key, session); err != nil {
			log.Error("Error saving session", err)
		}
	}()

	payloadJson, err := json.Marshal(payload)
	if err != nil {
		log.Error("Error marshaling payload", err)
		return nil, err
	}

	commentCell := cell.BeginCell().
		MustStoreUInt(payload.OperationType, 32).
		MustStoreStringSnake(base64.StdEncoding.EncodeToString(payloadJson)).
		EndCell()

	jettonData, err := s.adminWalletServ.DataJetton(payload.JettonMaster)
	if err != nil {
		log.Error("Error getting jetton data", err)
		return nil, err
	}

	log.Infoln(payload.Amount)

	parsed, err := strconv.ParseFloat(amount, 64)
	if err != nil {
		log.Error("Error parsing amount", err)
		return nil, err
	}
	decimals := math.Pow10(jettonData.Decimals)

	pld := cell.BeginCell().
		MustStoreUInt(0x0f8a7ea5, 32).                         // opcode
		MustStoreUInt(uint64(time.Now().Unix()), 64).          // query_id (UNIX timestamp)
		MustStoreCoins(uint64(math.Round(parsed * decimals))). // amount (с учетом decimals!)
		MustStoreAddr(address.MustParseAddr(receiverAddr)).    // destination
		MustStoreAddr(address.MustParseAddr(senderAddr)).      // response_destination
		MustStoreBoolBit(false).                               // custom_payload
		MustStoreCoins(0.01 * 1e9).                            // forward_ton_amount (0.05 TON)
		MustStoreMaybeRef(commentCell).                        // forward_payload
		EndCell()

	msg, err := tonconnect.NewMessage(
		jettonAddr,
		strconv.FormatUint(0.05*1e9, 10),
		tonconnect.WithPayload(pld.ToBOC()),
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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	boc, err := session.SendTransaction(ctx, *tx)
	if err != nil {
		log.Error("Error sending transaction", err)
		return nil, err
	}
	return boc, nil
}

func (s *TonConnectService) ConnectSession(ses *tonconnect.Session) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	_, err := ses.Connect(ctx, tonconnect.Wallets["tonkeeper"])
	if err != nil {
		return err
	}

	return nil
}

func (s *TonConnectService) SendTransaction(ctx context.Context, receiverAddr, amount, comment string, session *tonconnect.Session) ([]byte, error) {
	commentCell, err := wallet.CreateCommentCell(comment)
	if err != nil {
		log.Error("Error creating commentCell", err)
		return nil, err
	}

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

	log.Info(tx)

	boc, err := session.SendTransaction(ctx, *tx)
	if err != nil {
		log.Error("Error sending transaction", err)
		return nil, err
	}
	return boc, nil
}
