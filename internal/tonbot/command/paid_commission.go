package command

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
	"tonclient/internal/config"
	appModels "tonclient/internal/models"
	"tonclient/internal/services"
	"tonclient/internal/tonbot/buttons"
	"tonclient/internal/util"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/xssnick/tonutils-go/address"
)

type PaidCommission struct {
	b   *bot.Bot
	aws *services.AdminWalletService
	tcs *services.TonConnectService
	ps  *services.PoolService
	ws  *services.WalletTonService
	us  *services.UserService
}

func NewPaidCommissionCommand(b *bot.Bot, aws *services.AdminWalletService,
	tcs *services.TonConnectService, ps *services.PoolService,
	ws *services.WalletTonService, us *services.UserService) *PaidCommission {
	return &PaidCommission{
		b:   b,
		aws: aws,
		tcs: tcs,
		ps:  ps,
		us:  us,
		ws:  ws,
	}
}

func (c *PaidCommission) Execute(ctx context.Context, callback *models.CallbackQuery) {
	if err := util.CheckTypeMessage(c.b, callback); err != nil {
		log.Error(err)
		return
	}

	msg := callback.Message.Message
	chatId := msg.Chat.ID
	dataSplit := strings.Split(callback.Data, ":")
	if len(dataSplit) != 2 {
		return
	}

	poolId, err := strconv.ParseInt(dataSplit[1], 10, 64)
	if err != nil {
		if _, err := util.SendTextMessage(c.b, uint64(chatId), "❌ Что-то пошло не так! Повторите попытку!"); err != nil {
			log.Error(err)
		}
		return
	}

	user, err := c.us.GetByTelegramChatId(uint64(chatId))
	if err != nil {
		if _, err := util.SendTextMessage(c.b, uint64(chatId), "❌ Аккаунт не активирован. Введите команду /start"); err != nil {
			log.Error(err)
		}
		return
	}

	w, err := c.ws.GetByUserId(uint64(user.Id.Int64))
	if err != nil {
		log.Error(err)
		if _, err := util.SendTextMessage(c.b, uint64(chatId), "❌ У вас не привязан кошелек!"); err != nil {
			log.Error(err)
		}
		return
	}

	pool, err := c.ps.GetId(uint64(poolId))
	if err != nil {
		log.Error(err)
		if _, err := util.SendTextMessage(c.b, uint64(chatId), "❌ Пул не найден! Возможно он был удален!"); err != nil {
			log.Error(err)
		}
		return
	}

	if pool.IsCommissionPaid {
		if _, err := util.SendTextMessage(c.b, uint64(chatId), "❌ Комиссия за этот пул уже оплачена!"); err != nil {
			log.Error(err)
		}
		return
	}

	servCtx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
	defer cancel()

	jettonMasterAdmin := os.Getenv("JETTON_CONTRACT_ADMIN_JETTON")

	jettonAddr, err := c.aws.TokenWalletAddress(servCtx, jettonMasterAdmin, address.MustParseAddr(w.Addr))
	if err != nil {
		if _, err := util.SendTextMessage(c.b, uint64(chatId), "❌ Что-то пошло не так! Повторите попытку!"); err != nil {
			log.Error(err)
		}
		return
	}

	s, err := c.tcs.LoadSession(servCtx, fmt.Sprint(chatId))
	if err != nil {
		log.Error(err)
		repeatBtn := util.CreateDefaultButton(buttons.RepeatCreatePoolId, buttons.Repeat)
		markup := util.CreateInlineMarup(1, repeatBtn)
		if err := util.RequestRepeatTonConnect(c.b, chatId, markup, c.tcs); err != nil {
			log.Error(err)
		}
		return
	}

	jsonData, err := json.Marshal(pool)
	if err != nil {
		log.Error(err)
		if _, err := util.SendTextMessage(c.b, uint64(chatId), "❌ Что-то пошло не так! Повторите попытку!"); err != nil {
			log.Error(err)
		}
		return
	}

	payload := appModels.Payload{
		OperationType: appModels.OP_PAY_COMMISION,
		JettonMaster:  jettonMasterAdmin,
		Amount:        config.COMMISSION_AMOUNT,
		Payload:       string(jsonData),
	}

	if _, err := util.SendTextMessage(c.b, uint64(chatId), "✅ Подтвердите транзакцию на вашем кошельке!"); err != nil {
		log.Error(err)
		return
	}

	if _, err := c.tcs.SendJettonTransaction(
		servCtx,
		jettonAddr.Address().String(),
		c.aws.GetAdminWalletAddr().String(),
		w.Addr,
		fmt.Sprint(config.COMMISSION_AMOUNT),
		&payload,
		s,
	); err != nil {
		log.Error(err)
		if _, err := util.SendTextMessage(c.b, uint64(chatId), "❌ Что-то пошло не так! Повторите попытку!"); err != nil {
			log.Error(err)
		}
		return
	}

}
