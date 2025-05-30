package command

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
	appModels "tonclient/internal/models"
	"tonclient/internal/services"
	"tonclient/internal/tonbot/buttons"
	"tonclient/internal/tonbot/callbacksuf"
	"tonclient/internal/tonbot/userstate"
	"tonclient/internal/tonfi"
	"tonclient/internal/util"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/xssnick/tonutils-go/address"
)

var currentStakePoolId = make(map[int64]uint64)

type CreateStakeCommand[T CommandType] struct {
	b   *bot.Bot
	ps  *services.PoolService
	us  *services.UserService
	tcs *services.TonConnectService
	ss  *services.StakeService
	ts  *services.TelegramService
	aws *services.AdminWalletService
	ws  *services.WalletTonService
}

func NewCreateStackeCommand[T CommandType](
	b *bot.Bot,
	ps *services.PoolService,
	us *services.UserService,
	tcs *services.TonConnectService,
	ss *services.StakeService,
	ts *services.TelegramService,
	aws *services.AdminWalletService,
	ws *services.WalletTonService,
) *CreateStakeCommand[T] {
	return &CreateStakeCommand[T]{
		b:   b,
		ps:  ps,
		us:  us,
		tcs: tcs,
		ss:  ss,
		ts:  ts,
		aws: aws,
		ws:  ws,
	}
}

func (c *CreateStakeCommand[T]) Execute(ctx context.Context, args T) {

	if v, ok := any(args).(*models.CallbackQuery); ok {
		c.executeCallback(v)
		return
	}

	if v, ok := any(args).(*models.Message); ok {
		c.executeMessage(v)
		return
	}
}

func (c *CreateStakeCommand[T]) executeMessage(msg *models.Message) {
	chatId := msg.Chat.ID
	pooldId, ok := currentStakePoolId[chatId]
	if !ok {
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			"❌ Что-то пошло не так. Повторите операцию сначала!",
		); err != nil {
			log.Error(err)
			return
		}
		return
	}

	tokens, err := strconv.ParseFloat(msg.Text, 64)
	if err != nil {
		log.Error(err)
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			"❌ Вводите только цифры! Например: 1.0",
		); err != nil {
			log.Error(err)
			return
		}
		return
	}

	p, err := c.ps.GetId(pooldId)
	if err != nil {
		log.Error(err)
		return
	}

	stakes := c.ss.GetPoolStakes(pooldId)
	if stakes != nil {
		sumStakes := c.calculateSumStakesFromPool(stakes)
		newSum := sumStakes + tokens
		if err := c.checkSumStakes(newSum, p, uint64(chatId)); err != nil {
			log.Error(err)
			return
		}
	}

	if err := c.checkSumStakes(tokens, p, uint64(chatId)); err != nil {
		log.Error(err)
		return
	}

	u, err := c.us.GetByTelegramChatId(uint64(chatId))
	if err != nil {
		log.Error(err)
		return
	}

	jettonData, err := tonfi.GetAssetByAddr(p.JettonMaster)
	if err != nil {
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			"❌ Что-то пошло не так. Повторите операцию сначала!",
		); err != nil {
			log.Error(err)
			return
		}
	}

	if jettonData.DexPriceUsd == "" {
		jettonData.DexPriceUsd = "0."
	}

	price, err := strconv.ParseFloat(jettonData.DexPriceUsd, 64)
	if err != nil {
		log.Error(err)
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			"❌ Что-то пошло не так. Повторите операцию сначала!",
		); err != nil {
			log.Error(err)
			return
		}
		return
	}
	newStake := &appModels.Stake{
		UserId:               uint64(u.Id.Int64),
		PoolId:               pooldId,
		Amount:               tokens,
		Balance:              tokens,
		IsCommissionPaid:     false,
		StartDate:            time.Now(),
		IsActive:             true,
		DepositCreationPrice: price,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	w, err := c.ws.GetByUserId(uint64(u.Id.Int64))
	if err != nil {
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			"❌ Привяжите кошелек!",
		); err != nil {
			log.Error(err)
		}
		return
	}

	s, err := c.tcs.LoadSession(ctx, fmt.Sprint(chatId))
	if err != nil {
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			"❌ Привяжите свой кошелей заново!",
		); err != nil {
			log.Error(err)
		}
		return
	}

	adminJettonMaster := os.Getenv("JETTON_CONTRACT_ADMIN_JETTON")

	jettonAddr, err := c.aws.TokenWalletAddress(ctx, adminJettonMaster, address.MustParseAddr(w.Addr))
	if err != nil {
		log.Error(err)
		return
	}

	jsonData, err := json.Marshal(newStake)
	if err != nil {
		log.Error(err)
		return
	}

	commission, err := strconv.ParseFloat(os.Getenv("COMMISSION_STAKE_AMOUNT"), 64)
	if err != nil {
		commission = 1.
	}

	payload := appModels.Payload{
		OperationType: appModels.OP_PAID_COMMISSION_STAKE,
		JettonMaster:  adminJettonMaster,
		Amount:        commission,
		Payload:       string(jsonData),
	}

	var urlWallet string
	if w.Name == "tonkeeper" {
		urlWallet = "https://wallet.tonkeeper.com/"
	} else {
		urlWallet = "https://tonhub.com/"
	}

	btn := util.CreateUrlInlineButton(buttons.OpenBrowser, urlWallet)
	btn2 := util.CreateWebAppButton(buttons.OpenWallet, urlWallet)
	markup := util.CreateInlineMarup(1, btn, btn2)
	if _, err := util.SendTextMessageMarkup(
		c.b,
		uint64(chatId),
		fmt.Sprintf("✅ Оплатите комиссию. %v %v", commission, os.Getenv("JETTON_NAME_COIN")),
		markup,
	); err != nil {
		log.Error(err)
		return
	}

	if _, err := c.tcs.SendJettonTransaction(
		ctx,
		jettonAddr.Address().String(),
		c.aws.GetAdminWalletAddr().String(),
		w.Addr,
		fmt.Sprint(commission),
		&payload,
		s,
	); err != nil {
		log.Error(err)
		return
	}

	delete(userstate.CurrentState, chatId)
}

func (c *CreateStakeCommand[T]) executeCallback(callback *models.CallbackQuery) {
	if err := util.CheckTypeMessage(c.b, callback); err != nil {
		return
	}

	msg := callback.Message.Message
	chatId := msg.Chat.ID
	splitData := strings.Split(callback.Data, ":")

	if len(splitData) != 2 {
		if _, err := util.SendTextMessage(c.b, uint64(chatId), "❌ Не могу выполнить эту команду!"); err != nil {
			log.Error(err)
		}
		return
	}
	poolId, err := strconv.ParseInt(splitData[1], 10, 64)
	if err != nil {
		log.Error(err)
		return
	}

	pool, err := c.ps.GetId(uint64(poolId))
	if err != nil {
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			"❌ Не могу найти данный пул! возможно он был удален!",
		); err != nil {
			log.Error(err)
		}
		return
	}

	if !pool.IsActive {
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			"❌ Нельзя стейкнуть в закрытый пул!",
		); err != nil {
			log.Error(err)
			return
		}
		return
	}

	currentStakesFromPool := c.ss.GetPoolStakes(uint64(poolId))
	if currentStakesFromPool != nil {
		currentSumStakes := c.calculateSumStakesFromPool(currentStakesFromPool)
		if err := c.checkSumStakes(currentSumStakes, pool, uint64(chatId)); err != nil {
			log.Error(err)
			return
		}
	}

	if pool.Reserve == 0 {
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			"❌ Нельзя сделать стейк, так как резерв пуст!",
		); err != nil {
			log.Error(err)
		}
		return
	}

	if _, err := util.SendTextMessage(
		c.b,
		uint64(chatId),
		"Введите кол-во токенов, которое хотите стейкнуть:",
	); err != nil {
		log.Error(err)
		return
	}

	currentStakePoolId[chatId] = uint64(poolId)
	userstate.CurrentState[chatId] = userstate.CreateStake
}

func (c *CreateStakeCommand[T]) calculateSumStakesFromPool(stakes *[]appModels.Stake) float64 {
	res := 0.
	for _, stake := range *stakes {
		res += stake.Amount
	}

	return res
}

func (c *CreateStakeCommand[T]) checkSumStakes(
	currentSumStakes float64,
	pool *appModels.Pool,
	chatId uint64,
) error {
	tenProcientFromSum := pool.Reserve * 0.1
	if tenProcientFromSum <= currentSumStakes {
		if _, err := util.SendTextMessage(
			c.b,
			chatId,
			"❌ Нельзя сделать стейк на текущий момент! Недостаточно резерва",
		); err != nil {
			log.Error(err)
		}

		tgOwner, err := c.ts.GetByUserId(pool.OwnerId)
		if err != nil {
			log.Error(err)
			return err
		}

		jettonaData, err := c.aws.DataJetton(pool.JettonMaster)
		if err != nil {
			log.Error(err)
			return err
		}
		idButton := fmt.Sprintf("%v:%v:%v", buttons.PoolDataButton, pool.Id.Int64, callbacksuf.My)
		btn := util.CreateDefaultButton(idButton, "Открыть пул")
		markup := util.CreateInlineMarup(1, btn)
		pool.IsActive = false

		if err := c.ps.Update(pool); err != nil {
			log.Error(err)
		}

		if _, err := util.SendTextMessageMarkup(
			c.b,
			tgOwner.TelegramId,
			fmt.Sprintf("❌ В пуле %v недостаточно резерва. Пользователи не могут делать стейки на текущий момент. Пополните резерв!\n\nТекущий резерв: %v\n\nПул был закрыт автоматически. Вы сможете его сново открыть, после пополнения резерва!", jettonaData.Name, pool.Reserve),
			markup,
		); err != nil {
			log.Error(err)
		}
		return errors.New("недостаточная сумма для стейка")
	}
	return nil
}
