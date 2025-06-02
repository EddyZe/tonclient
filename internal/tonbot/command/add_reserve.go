package command

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	appModels "tonclient/internal/models"
	"tonclient/internal/services"
	"tonclient/internal/tonbot/userstate"
	"tonclient/internal/util"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type CommandType interface {
	*models.Message | *models.CallbackQuery
}

var currentPoolId = make(map[int64]uint64)

type AddReserve[T CommandType] struct {
	b   *bot.Bot
	ps  *services.PoolService
	tcs *services.TonConnectService
	us  *services.UserService
	ws  *services.WalletTonService
}

func NewAddReserveCommand[T CommandType](b *bot.Bot, ps *services.PoolService,
	tcs *services.TonConnectService, us *services.UserService, ws *services.WalletTonService) *AddReserve[T] {
	return &AddReserve[T]{
		b:   b,
		ps:  ps,
		tcs: tcs,
		us:  us,
		ws:  ws,
	}
}

func (c *AddReserve[T]) Execute(ctx context.Context, args T) {
	if v, ok := any(args).(*models.Message); ok {
		c.executeMessage(ctx, v)
		return
	}

	if v, ok := any(args).(*models.CallbackQuery); ok {
		c.executeCallback(ctx, v)
		return
	}
}

func (c *AddReserve[T]) executeMessage(ctx context.Context, msg *models.Message) {
	chatId := msg.Chat.ID
	text := msg.Text

	poolId, ok := currentPoolId[chatId]
	if !ok || poolId == 0 {
		if _, err := util.SendTextMessage(c.b, uint64(chatId), "❌ Что-то пошло не так, начните операцию сначала!"); err == nil {
			log.Error(err)
		}
		return
	}

	pool, err := c.ps.GetId(poolId)
	if err != nil {
		log.Error(err)
		return
	}

	user, err := c.us.GetByTelegramChatId(uint64(chatId))
	if err != nil {
		log.Error(err)
		if _, err := util.SendTextMessage(c.b, uint64(chatId), "❌ Аккаунт не активирован! Введите команду /start"); err == nil {
			log.Error(err)
		}
		return
	}

	if uint64(user.Id.Int64) != pool.OwnerId {
		if _, err := util.SendTextMessage(c.b, uint64(chatId), "❌ Данный пул не принадлежит вам!"); err != nil {
			log.Error(err)
		}
		return
	}

	amount, err := strconv.ParseFloat(text, 64)
	if err != nil {
		log.Error(err)
		if _, err := util.SendTextMessage(c.b, uint64(chatId), "❌ Сумма должна быть числом! Например: 23"); err == nil {
			log.Error(err)
		}
		return
	}

	addReserve := appModels.AddReserve{
		PoolId: poolId,
		Amount: amount,
	}

	data, err := json.Marshal(addReserve)
	if err != nil {
		log.Error(err)
		return
	}

	payload := appModels.Payload{
		OperationType: appModels.OP_ADMIN_ADD_RESERVE,
		JettonMaster:  pool.JettonMaster,
		Amount:        amount,
		Payload:       string(data),
	}

	s, err := c.tcs.LoadSession(fmt.Sprint(chatId))
	if err != nil {
		log.Error(err)
		if _, err := util.SendTextMessage(c.b, uint64(chatId), "❌ Потеряно соединение с TonConnect. Перейдите в профиль и повторите подключение!, затем попробуйте еще раз!"); err != nil {
			log.Error(err)
		}
		return
	}

	w, err := c.ws.GetByUserId(uint64(user.Id.Int64))
	if err != nil {
		log.Error(err)
		if _, err := util.SendTextMessage(c.b, uint64(chatId), "❌ У вас не привязан кошелек! Это можно сделать в профиле!"); err != nil {
			log.Error(err)
		}
		return
	}
	adminAddr := os.Getenv("WALLET_ADDR")

	btns := util.GenerateButtonWallets(w, c.tcs)

	markup := util.CreateInlineMarup(1, btns...)
	if _, err := util.SendTextMessageMarkup(
		c.b,
		uint64(chatId),
		"✅ Подтвердите транзакцию на вашем кошельке!",
		markup,
	); err != nil {
		log.Error(err)
		return
	}

	if _, err := c.tcs.SendJettonTransaction(
		pool.JettonWallet,
		adminAddr,
		w.Addr,
		fmt.Sprint(amount),
		&payload,
		s,
	); err != nil {
		log.Error(err)
		return
	}

	currentPoolId[chatId] = 0
	userstate.CurrentState[chatId] = -1

}

func (c *AddReserve[T]) executeCallback(ctx context.Context, callback *models.CallbackQuery) {
	if err := util.CheckTypeMessage(c.b, callback); err != nil {
		return
	}
	msg := callback.Message.Message
	chatId := msg.Chat.ID
	splitData := strings.Split(callback.Data, ":")

	if len(splitData) < 3 {
		if _, err := util.SendTextMessage(c.b, uint64(chatId), "❌ Произошла ошибка! Повторите позже"); err != nil {
			log.Error(err)
		}
		return
	}

	num, err := strconv.ParseFloat(splitData[1], 64)
	if err != nil {
		if _, err := util.SendTextMessage(c.b, uint64(chatId), "❌ ID пула невалидный"); err != nil {
			log.Error(err)
		}
		return
	}

	if _, err := util.SendTextMessage(c.b, uint64(chatId), "Введите кол-во токенов, которое хотите добавить в резерв:"); err != nil {
		log.Error(err)
		return
	}

	currentPoolId[chatId] = uint64(num)
	userstate.CurrentState[chatId] = userstate.EnterAddReserveTokens
}
