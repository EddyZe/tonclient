package tonbot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"tonclient/internal/config"
	appModels "tonclient/internal/models"
	"tonclient/internal/services"
	"tonclient/internal/tonbot/buttons"
	"tonclient/internal/tonbot/callbacksuf"
	"tonclient/internal/tonbot/command"
	"tonclient/internal/tonbot/userstate"
	"tonclient/internal/util"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

var log = config.InitLogger()

type TgBot struct {
	token string
	us    *services.UserService
	ts    *services.TelegramService
	ps    *services.PoolService
	ss    *services.StakeService
	ws    *services.WalletTonService
	aws   *services.AdminWalletService
	tcs   *services.TonConnectService
	opS   *services.OperationService
}

func NewTgBot(token string, us *services.UserService, ts *services.TelegramService,
	ps *services.PoolService, aws *services.AdminWalletService, ss *services.StakeService,
	ws *services.WalletTonService, tcs *services.TonConnectService,
	opS *services.OperationService) *TgBot {
	return &TgBot{
		token: token,
		us:    us,
		ts:    ts,
		ps:    ps,
		aws:   aws,
		ss:    ss,
		ws:    ws,
		tcs:   tcs,
		opS:   opS,
	}
}

func (t *TgBot) StartBot(ch chan appModels.SubmitTransaction) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	opts := []bot.Option{
		bot.WithDefaultHandler(t.handler),
	}

	tgbot, err := bot.New(t.token, opts...)
	if err != nil {
		log.Fatal("Failed to start bot: ", err)
		return err
	}

	go t.checkingOperation(ctx, tgbot, ch)

	tgbot.Start(ctx)

	return nil
}

func (t *TgBot) handler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update == nil {
		return
	}

	if update.Message != nil {
		msg := update.Message
		t.handleMessage(ctx, b, msg)
	}

	if update.CallbackQuery != nil {
		callback := update.CallbackQuery

		t.handleCallback(ctx, b, callback)

		if _, err := b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: callback.ID,
		}); err != nil {
			log.Error("AnswerCallbackQuery: ", err)
		}
	}
}

func (t *TgBot) handleMessage(ctx context.Context, b *bot.Bot, msg *models.Message) {
	if msg.Chat.Type == models.ChatTypePrivate {
		text := msg.Text
		chatId := msg.Chat.ID

		if strings.HasPrefix(text, "/start") {
			userstate.ResetState(chatId)
			cmd := command.NewStartCommand(b, t.us, t.ts)
			cmd.Execute(ctx, msg)
			return
		}

		if text == buttons.InviteFriend {
			userstate.ResetState(chatId)
			cmd := command.NewInviteFriendCommand(b, t.us)
			cmd.Execute(ctx, msg)
			return
		}

		if text == buttons.MyPools {
			userstate.ResetState(chatId)
			command.NewMyPoolsCommand(b, t.us, t.ps, t.aws).Execute(ctx, msg)
			return
		}

		if text == buttons.SelectPool {
			userstate.ResetState(chatId)
			command.NewListPoolCommand(b, t.ps, t.aws).Execute(ctx, msg)
			return
		}

		if text == buttons.Setting {
			userstate.ResetState(chatId)
			command.NewOpenSetting(b).Execute(ctx, msg)
			return
		}

		if text == buttons.HistoryOperation {
			command.NewListHistoryOperation(b, t.us, t.opS).Execute(ctx, msg)
			return
		}

		if text == buttons.Profile {
			userstate.ResetState(chatId)
			command.NewProfileCommand(b, t.us, t.ws, t.aws, t.ps, t.ss).Execute(ctx, msg)
			return
		}

		if text == buttons.CreatePool {
			userstate.ResetState(chatId)
			cmd := command.NewCreatePoolCommand[*models.Message](b, t.ps, t.us, t.tcs, t.aws, t.ws)
			cmd.Execute(ctx, msg)
			return
		}

		if state, ok := userstate.CurrentState[msg.Chat.ID]; ok {
			if state != -1 {
				t.handleState(ctx, state, b, msg)
				return
			}
		}
	}
}

func (t *TgBot) handleCallback(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery) {
	data := callback.Data

	if data == buttons.RoleButtonUserId {
		command.NewOpenUserMenuCommand(b).Execute(ctx, callback)
		return
	}

	if data == buttons.RoleButtonOwnerTokensId {
		command.NewOpenOwnerPoolsMenu(b).Execute(ctx, callback)
		return
	}

	if data == buttons.SetNumberWalletId {
		cmd := command.NewSetWalletCommand[*models.CallbackQuery](b, t.ws, t.us, t.aws, t.tcs)
		cmd.Execute(ctx, callback)
		return
	}

	if strings.HasPrefix(data, buttons.OpenOperationHistory) {
		command.NewOpenInfoOperation(b, t.opS).Execute(ctx, callback)
		return
	}

	if data == buttons.BackHistoryListId {
		if err := util.CheckTypeMessage(b, callback); err != nil {
			return
		}

		command.NewListHistoryOperation(b, t.us, t.opS).Execute(ctx, callback.Message.Message)
		return
	}

	if strings.HasPrefix(data, buttons.TakeTokensId) {
		command.NewTakeTokensCommand(b, t.us, t.ps, t.ss, t.aws, t.ws).Execute(ctx, callback)
		return
	}

	if data == buttons.DefCloseId {
		if err := util.CheckTypeMessage(b, callback); err != nil {
			log.Error("CheckTypeMessage: ", err)
			return
		}
		msg := callback.Message.Message

		if err := util.DeleteMessage(ctx, b, uint64(msg.Chat.ID), msg.ID); err != nil {
			log.Error("DeleteMessage: ", err)
			return
		}

		userstate.ResetState(msg.Chat.ID)
	}

	if strings.HasPrefix(data, buttons.NextPagePool) {
		command.NewListPoolCommand(b, t.ps, t.aws).NextPage(ctx, callback)
		return
	}

	if strings.HasPrefix(data, buttons.BackPagePool) {
		command.NewListPoolCommand(b, t.ps, t.aws).BackPage(ctx, callback)
		return
	}

	if strings.HasPrefix(data, buttons.NextPageMyPool) {
		command.NewMyPoolsCommand(b, t.us, t.ps, t.aws).NextPage(ctx, callback)
		return
	}

	if strings.HasPrefix(data, buttons.BackPageMyPool) {
		command.NewMyPoolsCommand(b, t.us, t.ps, t.aws).BackPage(ctx, callback)
		return
	}

	if data == buttons.CloseListPool {
		command.NewListPoolCommand(b, t.ps, t.aws).CloseList(ctx, callback)
		return
	}

	if data == buttons.NextPageHistory {
		command.NewListHistoryOperation(b, t.us, t.opS).NextPage(ctx, callback)
		return
	}

	if data == buttons.BackPageHistory {
		command.NewListHistoryOperation(b, t.us, t.opS).BackPage(ctx, callback)
		return
	}

	if data == buttons.CloseListHistory {
		command.NewListHistoryOperation(b, t.us, t.opS).CloseListHistory(ctx, callback)
		return
	}

	if data == buttons.BackPoolListId {
		if err := util.CheckTypeMessage(b, callback); err != nil {
			log.Error("CheckTypeMessage: ", err)
			return
		}
		command.NewListPoolCommand(b, t.ps, t.aws).Execute(ctx, callback.Message.Message)
	}

	if data == buttons.SevenDaysId || data == buttons.ThirtyDaysId || data == buttons.SixtyDaysId || data == buttons.EnterCustomPeriodId || data == buttons.RepeatCreatePoolId {
		command.NewCreatePoolCommand[*models.CallbackQuery](b, t.ps, t.us, t.tcs, t.aws, t.ws).Execute(ctx, callback)
		return
	}

	if data == buttons.LinkTonConnectId {
		command.NewTonConnectRepeat(b, t.us, t.ws, t.tcs).Execute(ctx, callback)
		return
	}

	if strings.HasPrefix(data, buttons.PaidCommissionId) {
		command.NewPaidCommissionCommand(b, t.aws, t.tcs, t.ps, t.ws, t.us).Execute(ctx, callback)
		return
	}

	if strings.HasPrefix(data, buttons.ClosePoolId) {
		command.NewCloseOrOpenPoolCommand(b, t.ps, t.us, t.ss, t.opS, t.aws).Execute(ctx, callback)
		return
	}

	if strings.HasPrefix(data, buttons.AddReserveId) {
		command.NewAddReserveCommand[*models.CallbackQuery](b, t.ps, t.tcs, t.us, t.ws).Execute(ctx, callback)
		return
	}

	if data == buttons.BackMyPoolListId {
		if err := util.CheckTypeMessage(b, callback); err != nil {
			log.Error("CheckTypeMessage: ", err)
			return
		}

		command.NewMyPoolsCommand(b, t.us, t.ps, t.aws).Execute(ctx, callback.Message.Message)
		return
	}

	if strings.HasPrefix(data, buttons.CreateStakeId) {
		//TODO Реализовать стейк токенов
	}

	if strings.HasPrefix(data, buttons.PoolDataButton) {
		command.NewPoolInfo(b, t.ps, t.us, t.ss).Execute(ctx, callback)
		return
	}
}

func (t *TgBot) handleState(ctx context.Context, state int, b *bot.Bot, msg *models.Message) {
	switch state {
	case userstate.EnterWalletAddr:
		command.NewSetWalletCommand[*models.Message](b, t.ws, t.us, t.aws, t.tcs).Execute(ctx, msg)
		break
	case userstate.EnterCustomPeriodHold, userstate.EnterProfitOnPercent, userstate.EnterJettonMasterAddress, userstate.EnterInsuranceCoating, userstate.EnterAmountTokens:
		command.NewCreatePoolCommand[*models.Message](b, t.ps, t.us, t.tcs, t.aws, t.ws).Execute(ctx, msg)
		break
	case userstate.EnterAddReserveTokens:
		command.NewAddReserveCommand[*models.Message](b, t.ps, t.tcs, t.us, t.ws).Execute(ctx, msg)
		break
	default:
		log.Error(state)
		return
	}
}

func (t *TgBot) checkingOperation(ctx context.Context, b *bot.Bot, ch chan appModels.SubmitTransaction) {
	for {
		select {
		case <-ctx.Done():
			return
		case v, ok := <-ch:
			if !ok {
				log.Infoln("Channel operation is closed")
				return
			}
			t.processOperation(b, v)
		}
	}
}

func (t *TgBot) processOperation(b *bot.Bot, tr appModels.SubmitTransaction) {
	var payload appModels.Payload
	if err := json.Unmarshal(tr.Payload, &payload); err != nil {
		log.Error("Unmarshal: ", err)
		return
	}
	switch tr.OperationType {
	case appModels.OP_STAKE:
		t.stake(&payload, b)
		break
	case appModels.OP_CLAIM:
		break
	case appModels.OP_CLAIM_INSURANCE:
		break
	case appModels.OP_ADMIN_CREATE_POOL:
		t.createPool(&payload, b)
		break
	case appModels.OP_ADMIN_ADD_RESERVE:
		t.addReserve(&payload, b)
		break
	case appModels.OP_ADMIN_CLOSE_POOL:
		break
	case appModels.OP_GET_USER_STAKES:
		break
	case appModels.OP_PAY_COMMISION:
		if err := t.payCommission(&payload, b); err != nil {
			log.Error("Failed to payCommission:", err)
			return
		}
		break
	default:
		return
	}
}

func (t *TgBot) stake(payload *appModels.Payload, b *bot.Bot) {
	var stake appModels.Stake
	if err := json.Unmarshal([]byte(payload.Payload), &stake); err != nil {
		log.Error("Failed to unmarshal stake data:", err)
		return
	}

	pool, err := t.ps.GetId(stake.PoolId)
	if err != nil {
		log.Error("Failed to get pool id:", err)
		if err := t.returnTokens(stake.UserId, pool.JettonMaster, "Ошибка создания стейка. Возврат.", stake.Amount); err != nil {
			log.Error("Failed to return tokens:", err)
		}
		return
	}

	_, err = t.ss.CreateStake(&stake)
	if err != nil {
		log.Error("Failed to create stake:", err)
		if err := t.returnTokens(stake.UserId, pool.JettonMaster, "Ошибка создания стейка. Возврат", stake.Amount); err != nil {
			log.Error("Failed to return tokens:", err)
		}
		return
	}

	tg, err := t.ts.GetByUserId(pool.OwnerId)
	if err != nil {
		log.Error("Failed to get user wall:", err)
		return
	}

	if _, err := util.SendTextMessage(b, tg.TelegramId, "✅ Стейк создан"); err != nil {
		log.Error("Failed to send message:", err)
		return
	}
	jettodData, err := t.aws.DataJetton(pool.JettonMaster)
	if err != nil {
		log.Error("Failed to get jettod data:", err)
		return
	}

	description := fmt.Sprintf("Стейк в jetton: %v. Кол-во: %v", jettodData.Name, stake.Amount)

	_, err = t.opS.Create(stake.UserId, appModels.OP_STAKE, description)
	if err != nil {
		log.Error("Failed to create stake:", err)
		return
	}
}

func (t *TgBot) createPool(payload *appModels.Payload, b *bot.Bot) {
	var pool appModels.Pool
	if err := json.Unmarshal(
		[]byte(payload.Payload),
		&pool,
	); err != nil {
		log.Errorf("Failed to unmarshal payload data: %v", err)
		return
	}

	log.Infoln(pool)
	_, err := t.ps.CreatePool(&pool)
	if err != nil {
		log.Errorf("Failed to create pool: %v", err)
		if err := t.returnTokens(pool.OwnerId, pool.JettonMaster, "Ошибка создания пула. Возврат", pool.Reserve); err != nil {
			log.Error("Failed to return tokens:", err)
		}
		return
	}

	telegram, err := t.ts.GetByUserId(pool.OwnerId)
	if err != nil {
		log.Errorf("Failed to get telegram: %v", err)
		return
	}

	text := fmt.Sprint("✅ Пул был успешно создан! Оплатите оплатите комиссию, чтобы активировать его!\n\n", util.PoolInfo(&pool, t.ss))
	markup := util.GenerateOwnerPoolInlineKeyboard(pool.Id.Int64, buttons.BackMyPoolListId, pool.IsActive, callbacksuf.My)

	if _, err := util.SendTextMessageMarkup(b, telegram.TelegramId, text, markup); err != nil {
		log.Error("Failed to send telegram:", err)
		return
	}

	jettonData, err := t.aws.DataJetton(pool.JettonMaster)
	if err != nil {
		log.Error("Failed to get jettod data:", err)
		return
	}

	desc := fmt.Sprintf("Создание пула для jetton: %v", jettonData.Name)
	_, err = t.opS.Create(pool.OwnerId, appModels.OP_ADMIN_CREATE_POOL, desc)
	if err != nil {
		log.Error("Failed to create operation creating pool:", err)
		return
	}
}

func (t *TgBot) addReserve(payload *appModels.Payload, b *bot.Bot) {
	var addReserve appModels.AddReserve
	if err := json.Unmarshal([]byte(payload.Payload), &addReserve); err != nil {
		log.Errorf("Failed to unmarshal payload data: %v", err)
		return
	}

	pool, err := t.ps.GetId(addReserve.PoolId)
	if err != nil {
		log.Errorf("Failed to get pool id: %v", err)
		if err := t.returnTokens(pool.OwnerId, pool.JettonMaster, "Не удалось пополнить резерв. Возврат.", addReserve.Amount); err != nil {
			log.Error("Failed to return tokens:", err)
		}
		return
	}

	newReserve, err := t.ps.AddReserve(addReserve.PoolId, addReserve.Amount)
	if err != nil {
		log.Errorf("Failed to add reserve: %v", err)
		if err := t.returnTokens(pool.OwnerId, pool.JettonMaster, "Не удалось пополнить резерв. Возврат.", addReserve.Amount); err != nil {
			log.Error("Failed to return tokens:", err)
		}
		return
	}

	tg, err := t.ts.GetByUserId(pool.OwnerId)
	if err != nil {
		log.Error("Failed to get user wall:", err)
		return
	}

	if _, err := util.SendTextMessage(
		b,
		tg.TelegramId,
		fmt.Sprint("✅ Резерв пополнен. Новый баланс резерва: ", newReserve)); err != nil {
		log.Error("Failed to send telegram:", err)
		return
	}

	jettonData, err := t.aws.DataJetton(pool.JettonMaster)
	if err != nil {
		log.Error("Failed to get jettod data:", err)
		return
	}

	desc := fmt.Sprintf("Пополнение в пул с jetton: %v на сумму: %v", jettonData.Name, addReserve.Amount)
	_, err = t.opS.Create(pool.OwnerId, appModels.OP_PAY_COMMISION, desc)
	if err != nil {
		log.Error("Failed to create operation creating pool:", err)
		return
	}
}

func (t *TgBot) payCommission(payload *appModels.Payload, b *bot.Bot) error {
	var pool appModels.Pool
	if err := json.Unmarshal([]byte(payload.Payload), &pool); err != nil {
		log.Errorf("Failed to unmarshal payload data: %v", err)
		return err
	}

	if !pool.Id.Valid {
		log.Error("pool id is not valid")
		return errors.New("pool id is not valid")
	}

	tg, err := t.ts.GetByUserId(pool.OwnerId)
	if err != nil {
		log.Errorf("Failed to get telegram: %v", err)
		return err
	}

	if payload.Amount < config.COMMISSION_AMOUNT {
		log.Error("Invalid amount received:", payload.Payload)
		if err := t.returnTokens(
			pool.OwnerId,
			payload.JettonMaster,
			fmt.Sprintf("❌ Комиссия должна быть %v. Возврат", config.COMMISSION_AMOUNT),
			payload.Amount); err != nil {
			log.Error("Failed to return tokens:", err)
			if _, err := util.SendTextMessage(b, tg.TelegramId, "❌ Комиссия должна быть %v."); err != nil {
				log.Error("Failed to send telegram:", err)
			}
		}
		return err
	}

	pool.IsActive = true
	pool.IsCommissionPaid = true

	if err := t.ps.Update(&pool); err != nil {
		log.Errorf("Failed to set commission paid: %v", err)
		if err := t.returnTokens(
			pool.OwnerId,
			pool.JettonMaster,
			"Возврат",
			payload.Amount); err != nil {
			log.Error("Failed to return tokens:", err)
		}
		return err
	}

	if _, err := util.SendTextMessage(
		b,
		tg.TelegramId,
		"✅ Комиссия принята. Теперь ваш пул активен! Активность вы так же можете менять в настройках пула!",
	); err != nil {
		log.Error("Failed to send telegram:", err)
		return err
	}

	jettonData, err := t.aws.DataJetton(pool.JettonMaster)
	if err != nil {
		log.Error("Failed to get jettod data:", err)
		return err
	}
	adminJettonData, err := t.aws.DataJetton(payload.JettonMaster)
	if err != nil {
		log.Error("Failed to get jettod data:", err)
		return err
	}

	desc := fmt.Sprintf(
		"Оплата комиссии jetton: %v. Коммисия: %v %v",
		jettonData.Name,
		payload.Amount,
		adminJettonData.Name,
	)

	if _, err := t.opS.Create(pool.OwnerId, appModels.OP_PAY_COMMISION, desc); err != nil {
		log.Error("Failed to create operation creating pool:", err)
		return err
	}

	return nil
}

func (t *TgBot) returnTokens(userId uint64, jettonMaster, comment string, amount float64) error {
	userWall, err := t.ws.GetByUserId(userId)
	if err != nil {
		log.Error("Failed to get user wall:", err)
		return err
	}

	jetData, err := t.aws.DataJetton(jettonMaster)
	if err != nil {
		log.Error("Failed to get jetton data:", err)
		return err
	}

	if err := t.aws.SendJetton(
		jettonMaster,
		userWall.Addr,
		comment,
		amount,
		jetData.Decimals,
	); err != nil {
		log.Error("Failed to send jetton data:", err)
		return err
	}

	return nil
}
