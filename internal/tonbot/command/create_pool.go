package command

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"
	appModels "tonclient/internal/models"
	"tonclient/internal/services"
	"tonclient/internal/tonbot/buttons"
	"tonclient/internal/tonbot/userstate"
	"tonclient/internal/util"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/xssnick/tonutils-go/address"
)

type CreatePoolCommandTypes interface {
	*models.Message | *models.CallbackQuery
}

var currentCreatingPool = make(map[int64]appModels.Pool)

type CreatePool[T CreatePoolCommandTypes] struct {
	b   *bot.Bot
	ps  *services.PoolService
	us  *services.UserService
	tcs *services.TonConnectService
	aws *services.AdminWalletService
	ws  *services.WalletTonService
}

func NewCreatePoolCommand[T CreatePoolCommandTypes](b *bot.Bot, ps *services.PoolService,
	us *services.UserService, tcs *services.TonConnectService, aws *services.AdminWalletService,
	ws *services.WalletTonService) *CreatePool[T] {
	return &CreatePool[T]{
		b:   b,
		ps:  ps,
		us:  us,
		tcs: tcs,
		aws: aws,
		ws:  ws,
	}
}

func (c *CreatePool[T]) Execute(ctx context.Context, args T) {

	if callback, ok := any(args).(*models.CallbackQuery); ok {
		c.executeCallback(ctx, callback)
		return
	}

	if message, ok := any(args).(*models.Message); ok {
		c.executeMessage(ctx, message)
		return
	}
}

func (c *CreatePool[T]) executeMessage(ctx context.Context, msg *models.Message) {
	chatId := msg.Chat.ID
	state, ok := userstate.CurrentState[chatId]
	if !ok || state == -1 {
		userstate.CurrentState[chatId] = userstate.EnterJettonMasterAddress
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			"Отлично! Давайте создадим новый пул\n\n1. Введите <b>адрес вашего токена</b> <b>(Jetton Master Address)</b>:\n",
		); err != nil {
			log.Error("Failed to send message: ", err)
			return
		}
		return
	}

	user, err := c.us.GetByTelegramChatId(uint64(chatId))
	if err != nil {
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			"❌ Аккаунт не активирован. Чтобы активировать аккаунт введите команду /start"); err != nil {
			log.Error(err)
		}
		userstate.ResetState(chatId)
		return
	}
	w, err := c.ws.GetByUserId(uint64(user.Id.Int64))
	if err != nil {
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			"❌ Привяжите кошелек ваш кошелек! Для этого откройте: <b>Профиль</b>",
		); err != nil {
			log.Error(err)
		}
		return
	}

	switch state {
	case userstate.EnterJettonMasterAddress:
		c.enterJettonMaster(msg, chatId, user)
		break
	case userstate.EnterCustomPeriodHold:
		c.enterCustomPeriodHold(msg)
		break
	case userstate.EnterProfitOnPercent:
		c.enterProfit(msg)
		break
	case userstate.EnterInsuranceCoating:
		c.enterInsuranceCoating(msg)
		break
	case userstate.EnterAmountTokens:
		c.enterAmountToken(msg)
		break
	case userstate.EnterJettonWallet:
		c.enterJettonWallet(msg, w)
		break
	default:
		if _, err := util.SendTextMessage(c.b, uint64(chatId), "❌ Что-то пошло не так! Повторите команду!"); err != nil {
			log.Error(err)
		}
	}
}

func (c *CreatePool[T]) enterJettonWallet(msg *models.Message, w *appModels.WalletTon) {
	chatId := msg.Chat.ID
	text := msg.Text
	if err := c.aws.CheckValidAddr(text); err != nil {
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			"❌ Адрес невалиден! Повторите попытка",
		); err != nil {
			log.Error(err)
		}
		return
	}

	pool, ok := currentCreatingPool[chatId]
	if !ok {
		if _, err := util.SendTextMessage(c.b, uint64(chatId), "❌ Что-то пошло не так! Повторите операцию!"); err != nil {
			log.Error(err)
		}
		return
	}

	pool.JettonWallet = text
	pool.IsCommissionPaid = false
	pool.CreatedAt = time.Now()
	pool.IsActive = false
	currentCreatingPool[chatId] = pool

	if _, err := util.SendTextMessage(c.b, uint64(chatId), "Подтвердите операцию в течении 5 минут в вашем привязанном кошельке, чтобы заморозить резерв."); err != nil {
		log.Error(err)
		return
	}

	if err := c.sendTransactionCreatingPool(&pool, chatId, w); err != nil {
		log.Error(err)
		return
	}
}

func (c *CreatePool[T]) sendTransactionCreatingPool(pool *appModels.Pool, chatId int64, w *appModels.WalletTon) error {
	repeatBtn := util.CreateDefaultButton(buttons.RepeatCreatePoolId, buttons.Repeat)
	markup := util.CreateInlineMarup(1, repeatBtn)
	poolJson, err := json.Marshal(pool)
	if err != nil {
		if _, err := util.SendTextMessageMarkup(
			c.b,
			uint64(chatId),
			"❌ Что-то пошло не так. Повторите попытку",
			markup); err != nil {
			log.Error(err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
	defer cancel()
	adminWal := os.Getenv("WALLET_ADDR")
	payload := appModels.Payload{
		OperationType: appModels.OP_ADMIN_CREATE_POOL,
		JettonMaster:  pool.JettonMaster,
		Payload:       string(poolJson),
	}

	s, err := c.tcs.LoadSession(ctx, fmt.Sprint(chatId))
	if err != nil {
		if _, err := util.SendTextMessageMarkup(
			c.b,
			uint64(chatId),
			"❌ Возможно вы отключили TonConnect! Подтвердите подключение снова! А затем нажмите <b>Повторить попытку</b>",
			markup,
		); err != nil {
			log.Error(err)
			return err
		}
		if _, err := util.ConnectingTonConnect(c.b, uint64(chatId), c.tcs); err != nil {
			log.Error(err)
			return err
		}
		if _, err := util.SendTextMessageMarkup(
			c.b,
			uint64(chatId),
			"✅ Кошелек привязан. Нажмите 'повторить попытку' и подтвердите транзакцию по резерву в привязанном кошельке",
			markup); err != nil {
			log.Error(err)
			return err
		}
		return err
	}

	boc, err := c.tcs.SendJettonTransaction(
		ctx,
		pool.JettonWallet,
		adminWal,
		w.Addr,
		fmt.Sprint(pool.Reserve),
		&payload,
		s,
	)
	if err != nil {
		log.Error(err)
		if _, err := util.SendTextMessageMarkup(c.b, uint64(chatId), "❌ Перевод резерва не был подтвержден", markup); err != nil {
			log.Error(err)
		}
		return err
	}

	log.Infoln(string(boc))

	currentCreatingPool[chatId] = appModels.Pool{}
	if _, err := util.SendTextMessage(
		c.b,
		uint64(chatId),
		"🔁 Пул создается! Пожалуйста подождите...",
	); err != nil {
		log.Error(err)
		return err
	}
	return nil
}

func (c *CreatePool[T]) enterAmountToken(msg *models.Message) {
	chatId := msg.Chat.ID
	text := msg.Text
	num, err := strconv.ParseFloat(text, 64)
	if err != nil {
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			"❌ Сумма должна быть числом! Например: 1",
		); err != nil {
			log.Error(err)
		}
		return
	}

	if num < 1 {
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			"❌ Сумма не может быть меньше чем 1!",
		); err != nil {
			log.Error(err)
		}
		return
	}

	pool, ok := currentCreatingPool[chatId]
	if !ok {
		if _, err := util.SendTextMessage(c.b, uint64(chatId), "❌ Что-то пошло не так! Повторите операцию!"); err != nil {
			log.Error(err)
		}
		return
	}

	if _, err := util.SendTextMessage(
		c.b,
		uint64(chatId),
		"✅ Отлично! Пул почти создан! Отправьте адрес вашего jetton кошелька(Jetton wallet)",
	); err != nil {
		log.Error(err)
		return
	}

	pool.Reserve = num
	currentCreatingPool[chatId] = pool
	userstate.CurrentState[chatId] = userstate.EnterJettonWallet
}

func (c *CreatePool[T]) enterInsuranceCoating(msg *models.Message) {
	chatId := msg.Chat.ID
	text := msg.Text
	num, err := strconv.Atoi(text)
	if err != nil {
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			"❌ Укажите страховое покрытие в цифрах! Например: 1",
		); err != nil {
			log.Error(err)
		}
		return
	}

	if num < 1 {
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			"❌ Страховое покрытие не может быть меньше чем 1.",
		); err != nil {
			log.Error(err)
		}
		return
	}

	pool, ok := currentCreatingPool[chatId]
	if !ok {
		if _, err := util.SendTextMessage(c.b, uint64(chatId), "❌ Что-то пошло не так! Повторите операцию!"); err != nil {
			log.Error(err)
		}
		return
	}

	resp := fmt.Sprintf("✅ Отлично! Вы указали %v%% за страховое покрытие.\nУкажите кол-во токенов, которое будет заморожены для резерва:", num)
	if _, err := util.SendTextMessage(
		c.b,
		uint64(chatId),
		resp,
	); err != nil {
		log.Error(err)
		return
	}

	pool.InsuranceCoating = uint(num)
	currentCreatingPool[chatId] = pool
	userstate.CurrentState[chatId] = userstate.EnterAmountTokens
}

func (c *CreatePool[T]) enterProfit(msg *models.Message) {
	chatId := msg.Chat.ID
	text := msg.Text
	num, err := strconv.Atoi(text)
	if err != nil {
		if _, err := util.SendTextMessage(c.b, uint64(chatId), "❌ Укажите число! Например: 1"); err != nil {
			log.Error(err)
		}
		return
	}

	if num < 1 {
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			"❌ Доходность не может быть меньше чем 1!",
		); err != nil {
			log.Error(err)
		}
		return
	}

	pool, ok := currentCreatingPool[chatId]
	if !ok {
		if _, err := util.SendTextMessage(c.b, uint64(chatId), "❌ Что-то пошло не так! Повторите операцию!"); err != nil {
			log.Error(err)
		}
		return
	}

	resp := fmt.Sprintf("✅ Отлично! Доходность <b>%v%%</b> указана!\n\nУкажите страховое покрытие в процентах: \nСработает, если цена упадет на указанное кол-во процентов.", num)
	if _, err := util.SendTextMessage(c.b, uint64(chatId), resp); err != nil {
		log.Error(err)
		return
	}
	pool.Reward = uint(num)
	currentCreatingPool[chatId] = pool
	userstate.CurrentState[chatId] = userstate.EnterInsuranceCoating
}

func (c *CreatePool[T]) enterCustomPeriodHold(msg *models.Message) {
	chatId := msg.Chat.ID
	text := msg.Text
	numPeriod, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		if _, err := util.SendTextMessage(c.b, uint64(chatId), "❌ Укажите срок холда в цифрах! Например: 1"); err != nil {
			log.Error(err)
		}
		return
	}

	if numPeriod < 1 {
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			"❌ Период не может быть меньше чем 1!",
		); err != nil {
			log.Error(err)
		}
		return
	}

	c.installPeriodPool(chatId, numPeriod)
}

func (c *CreatePool[T]) installPeriodPool(chatId, period int64) {
	pool, ok := currentCreatingPool[chatId]
	if !ok {
		if _, err := util.SendTextMessage(c.b, uint64(chatId), "❌ Что-то пошло не так. Повторите операцию сначала!"); err != nil {
			log.Error(err)
		}
		return
	}

	pool.Period = uint(period)
	currentCreatingPool[chatId] = pool
	text := fmt.Sprintf(
		"✅ Отлично. Вы выбрали <b>%v %v</b>. Укажите <b>доходность для участников</b> (%% в день). Например: 1.\n",
		period,
		util.SuffixDay(int(period)),
	)
	if _, err := util.SendTextMessage(
		c.b,
		uint64(chatId),
		text,
	); err != nil {
		log.Error(err)
		return
	}
	userstate.CurrentState[chatId] = userstate.EnterProfitOnPercent
}

func (c *CreatePool[T]) enterJettonMaster(msg *models.Message, chatId int64, user *appModels.User) {
	var newPool appModels.Pool
	jettonAddr := msg.Text
	if _, err := address.ParseAddr(jettonAddr); err != nil {
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			"❌ Невалидный адрес! Повторите попытку!",
		); err != nil {
			log.Error(err)
			userstate.ResetState(chatId)
			return
		}
		return
	}
	newPool.JettonMaster = jettonAddr
	newPool.OwnerId = uint64(user.Id.Int64)
	currentCreatingPool[chatId] = newPool
	jettonData, err := c.aws.DataJetton(jettonAddr)
	if err != nil {
		if _, err := util.SendTextMessage(c.b, uint64(chatId), "❌ Что-то пошло не так. Повторите попытку!"); err != nil {
			log.Error(err)
			return
		}
		return
	}

	text := fmt.Sprintf("✅ Отлично! Выбранный токен <b>%v</b>.\n\nВыберите срок холда:", jettonData.Name)

	if _, err := util.SendTextMessageMarkup(
		c.b,
		uint64(chatId),
		text,
		c.generateSelectPeriodHoldMarkup()); err != nil {
		log.Error(err)
		userstate.ResetState(chatId)
	}
	userstate.CurrentState[chatId] = userstate.SelectPeriodHold
}

func (c *CreatePool[T]) executeCallback(ctx context.Context, callback *models.CallbackQuery) {
	if err := util.CheckTypeMessage(c.b, callback); err != nil {
		return
	}
	msg := callback.Message.Message
	chatId := msg.Chat.ID
	state, ok := userstate.CurrentState[chatId]
	if !ok || state == -1 {
		if _, err := util.SendTextMessage(c.b, uint64(chatId), "❌ Что-то пошло не так. Повторите операцию сначала!"); err != nil {
			log.Error(err)
		}
		return
	}

	if state == userstate.SelectPeriodHold {
		holdPeriod := c.getHoldPeriod(callback.Data, uint64(chatId))
		if holdPeriod == 0 {
			return
		}

		c.installPeriodPool(chatId, int64(holdPeriod))
	}

	if callback.Data == buttons.RepeatCreatePoolId {
		user, err := c.us.GetByTelegramChatId(uint64(chatId))
		if err != nil {
			log.Error(err)
			if _, err := util.SendTextMessage(
				c.b,
				uint64(chatId),
				"❌ Ваш аккаунт не активирован! Введите команду /start",
			); err != nil {
				log.Error(err)
			}
			return
		}
		w, err := c.ws.GetByUserId(uint64(user.Id.Int64))
		if err != nil {
			log.Error(err)
			if _, err := util.SendTextMessage(c.b, uint64(chatId), "❌ Кошелек не привязан. Перейдите в профиль и привяжите его"); err != nil {
				log.Error(err)
			}
			return
		}
		pool, ok := currentCreatingPool[chatId]
		if !ok {
			if _, err := util.SendTextMessage(c.b, uint64(chatId), "❌ Что-то пошло не так. Повторите операцию сначала!"); err != nil {
				log.Error(err)
			}
			return
		}

		if err := c.sendTransactionCreatingPool(&pool, chatId, w); err != nil {
			log.Error(err)
			if _, err := util.SendTextMessage(c.b, uint64(chatId), "❌ Что-то пошло не так. Повторите операцию сначала!"); err != nil {
				log.Error(err)
			}
			return
		}
	}
}

func (c *CreatePool[T]) getHoldPeriod(data string, chatId uint64) int {
	switch data {
	case buttons.SevenDaysId:
		return 7
	case buttons.ThirtyDaysId:
		return 30
	case buttons.SixtyDaysId:
		return 60
	case buttons.EnterCustomPeriodId:
		if _, err := util.SendTextMessage(c.b, chatId, "Введите свое срок холда в днях: "); err != nil {
			log.Error(err)
			return 0
		}
		userstate.CurrentState[int64(chatId)] = userstate.EnterCustomPeriodHold
		break
	default:
		if _, err := util.SendTextMessage(c.b, chatId, "❌ Неизвестная мне команда!"); err != nil {
			log.Error(err)
		}
		break
	}

	return 0
}

func (c *CreatePool[T]) generateSelectPeriodHoldMarkup() *models.InlineKeyboardMarkup {
	seven := util.CreateDefaultButton(buttons.SevenDaysId, buttons.SevenDays)
	thirty := util.CreateDefaultButton(buttons.ThirtyDaysId, buttons.ThirtyDays)
	sixty := util.CreateDefaultButton(buttons.SixtyDaysId, buttons.SixtyDays)
	custom := util.CreateDefaultButton(buttons.EnterCustomPeriodId, buttons.EnterCustomPeriod)

	markup := util.CreateInlineMarup(1, seven, thirty, sixty, custom)
	return markup
}
