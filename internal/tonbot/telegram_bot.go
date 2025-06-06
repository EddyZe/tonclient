package tonbot

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"strconv"
	"strings"
	"tonclient/internal/config"
	appModels "tonclient/internal/models"
	"tonclient/internal/schedulers"
	"tonclient/internal/services"
	"tonclient/internal/tonbot/buttons"
	"tonclient/internal/tonbot/callbacksuf"
	"tonclient/internal/tonbot/command"
	"tonclient/internal/tonbot/userstate"
	"tonclient/internal/util"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/robfig/cron/v3"
	"github.com/xssnick/tonutils-go/address"
)

var log = config.InitLogger()
var sendJettonInsurance = make(chan func())
var sendJettonProfit = make(chan func())
var sendJettonClosingTimeStake = make(chan func())
var sendJettonClosePool = make(chan func())

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
	rs    *services.ReferalService
}

func NewTgBot(token string, us *services.UserService, ts *services.TelegramService,
	ps *services.PoolService, aws *services.AdminWalletService, ss *services.StakeService,
	ws *services.WalletTonService, tcs *services.TonConnectService,
	opS *services.OperationService, rs *services.ReferalService) *TgBot {
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
		rs:    rs,
	}
}

func (t *TgBot) StartBot(ch chan appModels.SubmitTransaction) error {
	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt)

	opts := []bot.Option{
		bot.WithDefaultHandler(t.handler),
	}

	tgbot, err := bot.New(t.token, opts...)
	if err != nil {
		log.Fatal("Failed to start bot: ", err)
		return err
	}

	go t.checkingOperation(tgbot, ch)
	go t.createCron(tgbot)
	go checkSendJettonOperation()

	tgbot.Start(ctx)

	return nil
}

func (t *TgBot) createCron(b *bot.Bot) {
	stakes := make(chan *appModels.NotificationStake)
	c := cron.New()
	//TODO изменить на каждый день!
	_, err := c.AddFunc("* * * * *", schedulers.AddStakeBonusActiveStakes(t.ss, t.ps, stakes))
	if err != nil {
		log.Fatal(err)
	}
	c.Start()

	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt)

	go t.checkMessageBonusStakes(ctx, b, stakes)

	go func() {
		for {
			select {
			case <-ctx.Done():
				c.Stop()
				break
			default:
				continue
			}
		}
	}()
}

func (t *TgBot) checkMessageBonusStakes(ctx context.Context, b *bot.Bot, ch chan *appModels.NotificationStake) {
	for {
		select {
		case <-ctx.Done():
		case notification, ok := <-ch:
			if !ok {
				continue
			}
			tg, err := t.ts.GetByUserId(notification.Stake.UserId)
			if err != nil {
				continue
			}
			if _, err := util.SendTextMessage(
				b,
				tg.TelegramId,
				notification.Msg,
			); err != nil {
				log.Error(err)
				continue
			}
		}
	}
}

func (t *TgBot) handler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update == nil {
		return
	}

	if update.Message != nil {
		msg := update.Message
		go t.handleMessage(ctx, b, msg)
		return
	}

	if update.CallbackQuery != nil {
		callback := update.CallbackQuery

		go t.handleCallback(ctx, b, callback)

		if _, err := b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: callback.ID,
		}); err != nil {
			log.Error("AnswerCallbackQuery: ", err)
		}
		return
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

		if text == buttons.LearnMore {
			cmd := command.NewInfoCommand(b)
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

		if text == buttons.MyStakes {
			userstate.ResetState(chatId)
			cmd := command.NewStakesUserList[*models.Message](b, t.us, t.ss)
			cmd.Execute(ctx, msg)
			return
		}

		if text == buttons.TakeAwards {
			userstate.ResetState(chatId)
			cmd := command.NewStakeProfitList[*models.Message](b, t.us, t.ss, t.ps)
			cmd.Execute(ctx, msg)
			return
		}

		if text == buttons.CheckInsurance {
			userstate.ResetState(chatId)
			cmd := command.NewStakeInsuranceList[*models.Message](b, t.us, t.ss, t.ps)
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

	if strings.HasPrefix(data, buttons.CloseStakeId) {
		sendJettonClosingTimeStake <- func() {
			command.NewCloseStakeCommand(b, t.aws, t.ws, t.ss, t.ps, t.opS).Execute(ctx, callback)
		}
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
		sendJettonClosePool <- func() {
			command.NewTakeTokensCommand(b, t.us, t.ps, t.ss, t.aws, t.ws, t.opS).Execute(ctx, callback)
		}
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

	if strings.HasPrefix(data, buttons.BackListGroupId) {
		cmd := command.NewStakesUserList[*models.CallbackQuery](b, t.us, t.ss)
		cmd.BackStakesGroup(callback)
		return
	}

	if strings.HasPrefix(data, buttons.OpenGroupId) {
		cmd := command.NewStakesUserList[*models.CallbackQuery](b, t.us, t.ss)
		cmd.Execute(ctx, callback)
		return
	}

	if data == buttons.NextListStakesGroupId {
		cmd := command.NewStakesUserList[*models.CallbackQuery](b, t.us, t.ss)
		cmd.NextGroupPage(callback)
		return
	}

	if strings.HasPrefix(data, buttons.BackListStakesGroupId) {
		cmd := command.NewStakesUserList[*models.CallbackQuery](b, t.us, t.ss)
		cmd.BackGroupPage(callback)
		return
	}

	if data == buttons.CloseListStakesGroupId {
		cmd := command.NewStakesUserList[*models.CallbackQuery](b, t.us, t.ss)
		cmd.CloseGroupList(callback)
		return
	}

	if strings.HasPrefix(data, buttons.NextPageStakesFromGroupJettonName) {
		cmd := command.NewStakesUserList[*models.CallbackQuery](b, t.us, t.ss)
		cmd.NextPageStakesFromGroup(callback)
		return
	}

	if strings.HasPrefix(data, buttons.OpenStakeInfo) {
		cmd := command.NewOpenStakeInfoCommand(b, t.ss, t.ps, buttons.BackListStakesGroupId)
		cmd.Execute(ctx, callback)
		return
	}

	if strings.HasPrefix(data, buttons.ProfitOpenStakeInfo) {
		command.NewOpenStakeInfoCommand(b, t.ss, t.ps, buttons.ProfitBackListGroup).Execute(ctx, callback)
		return
	}

	if strings.HasPrefix(data, buttons.ProfitOpenGroupId) {
		command.NewStakeProfitList[*models.CallbackQuery](b, t.us, t.ss, t.ps).Execute(ctx, callback)
		return
	}

	if strings.HasPrefix(data, buttons.ProfitNextPageJettonName) {
		command.NewStakeProfitList[*models.CallbackQuery](b, t.us, t.ss, t.ps).NextPageProfitStake(ctx, callback)
		return
	}

	if strings.HasPrefix(data, buttons.ProfitBackPageJettonName) {
		command.NewStakeProfitList[*models.CallbackQuery](b, t.us, t.ss, t.ps).BackPageProfitStake(ctx, callback)
		return
	}

	if strings.HasPrefix(data, buttons.ProfitBackListGroup) {
		command.NewStakeProfitList[*models.CallbackQuery](b, t.us, t.ss, t.ps).Execute(ctx, callback)
		return
	}

	if strings.HasPrefix(data, buttons.InsuranceOpenStakeInfo) {
		command.NewOpenStakeInfoCommand(b, t.ss, t.ps, buttons.InsuranceBackListGroup).Execute(ctx, callback)
		return
	}

	if strings.HasPrefix(data, buttons.InsuranceOpenGroupId) {
		command.NewStakeInsuranceList[*models.CallbackQuery](b, t.us, t.ss, t.ps).Execute(ctx, callback)
		return
	}

	if strings.HasPrefix(data, buttons.InsuranceNextPageJettonName) {
		command.NewStakeInsuranceList[*models.CallbackQuery](b, t.us, t.ss, t.ps).NextPageInsuranceStake(ctx, callback)
		return
	}

	if strings.HasPrefix(data, buttons.InsuranceBackPageJettonName) {
		command.NewStakeInsuranceList[*models.CallbackQuery](b, t.us, t.ss, t.ps).BackPageInsuranceStake(ctx, callback)
		return
	}

	if strings.HasPrefix(data, buttons.InsuranceBackListGroup) {
		command.NewStakeInsuranceList[*models.CallbackQuery](b, t.us, t.ss, t.ps).Execute(ctx, callback)
		return
	}

	if strings.HasPrefix(data, buttons.BackPageStakesFromGroupJettonName) {
		cmd := command.NewStakesUserList[*models.CallbackQuery](b, t.us, t.ss)
		cmd.BackStakesFromGroup(callback)
		return
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
		command.NewAddReserveCommand[*models.CallbackQuery](b, t.ps, t.tcs, t.us, t.ws, t.aws).Execute(ctx, callback)
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
		command.NewCreateStackeCommand[*models.CallbackQuery](
			b,
			t.ps,
			t.us,
			t.tcs,
			t.ss,
			t.ts,
			t.aws,
			t.ws,
		).Execute(ctx, callback)
		return
	}

	if strings.HasPrefix(data, buttons.PoolDataButton) {
		command.NewPoolInfo(b, t.ps, t.us, t.ss, t.aws).Execute(ctx, callback)
		return
	}

	if strings.HasPrefix(data, buttons.TakeInsuranceId) {
		sendJettonInsurance <- func() {
			command.NewTakeInsuranceFromStake(
				b,
				t.us,
				t.ss,
				t.ps,
				t.ts,
				t.opS,
				t.ws,
				t.aws,
			).Execute(ctx, callback)
		}
		return
	}

	if strings.HasPrefix(data, buttons.DeletePoolId) {
		command.NewDeletePool(
			b,
			t.ps,
			t.opS,
		).Execute(ctx, callback)
		return
	}

	if strings.HasPrefix(data, buttons.TakeProfitId) {
		sendJettonProfit <- func() {
			command.NewTakeProfitFromStake(b, t.us, t.ps, t.ws, t.aws, t.ss, t.opS, t.ts).Execute(ctx, callback)
		}
		return
	}

	//профит лист
	if strings.HasPrefix(data, buttons.ProfitNextPageGroup) {
		cmd := command.NewStakeProfitList[*models.CallbackQuery](b, t.us, t.ss, t.ps)
		cmd.NextPageGroup(ctx, callback)
		return
	}
	if strings.HasPrefix(data, buttons.ProfitBackPageGroup) {
		cmd := command.NewStakeProfitList[*models.CallbackQuery](b, t.us, t.ss, t.ps)
		cmd.BackPageGroup(ctx, callback)
		return
	}
	if strings.HasPrefix(data, buttons.ProfitCloseGroup) {
		cmd := command.NewStakeProfitList[*models.CallbackQuery](b, t.us, t.ss, t.ps)
		cmd.CloseList(ctx, callback)
		return
	}

	if strings.HasPrefix(data, buttons.InsuranceNextPageGroup) {
		cmd := command.NewStakeInsuranceList[*models.CallbackQuery](b, t.us, t.ss, t.ps)
		cmd.NextPageGroup(ctx, callback)
		return
	}
	if strings.HasPrefix(data, buttons.InsuranceBackPageGroup) {
		cmd := command.NewStakeInsuranceList[*models.CallbackQuery](b, t.us, t.ss, t.ps)
		cmd.BackPageGroup(ctx, callback)
		return
	}
	if strings.HasPrefix(data, buttons.InsuranceCloseGroup) {
		cmd := command.NewStakeInsuranceList[*models.CallbackQuery](b, t.us, t.ss, t.ps)
		cmd.CloseList(ctx, callback)
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
		command.NewAddReserveCommand[*models.Message](b, t.ps, t.tcs, t.us, t.ws, t.aws).Execute(ctx, msg)
		break
	case userstate.CreateStake:
		command.NewCreateStackeCommand[*models.Message](b, t.ps, t.us, t.tcs, t.ss, t.ts, t.aws, t.ws).Execute(ctx, msg)
		break
	default:
		log.Error(state)
		return
	}
}

func (t *TgBot) checkingOperation(b *bot.Bot, ch chan appModels.SubmitTransaction) {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("PANIC in checkingOperation: %v\n%s", r, debug.Stack())
			go t.checkingOperation(b, ch)
		} else {
			log.Infoln("Channel operation handler exited normally")
		}
	}()
	for {
		select {
		case v, ok := <-ch:
			if !ok {
				log.Infoln("Channel operation is closed")
				continue
			}
			log.Infoln(v)
			go t.processOperation(b, v)
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
	case appModels.OP_PAID_COMMISSION_STAKE:
		t.commissionStakePaid(&payload, b)
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

func (t *TgBot) commissionStakePaid(payload *appModels.Payload, b *bot.Bot) {
	var stake appModels.Stake
	if err := json.Unmarshal([]byte(payload.Payload), &stake); err != nil {
		log.Error("Failed to unmarshal stake data:", err)
		return
	}

	pool, err := t.ps.GetId(stake.PoolId)
	if err != nil {
		log.Error("Failed to get pool id:", err)
		err := t.returnTokens(stake.UserId, payload.JettonMaster, payload.Amount)
		if err != nil {
			return
		}
		return
	}

	tg, err := t.ts.GetByUserId(stake.UserId)
	if err != nil {
		err := t.returnTokens(stake.UserId, payload.JettonMaster, payload.Amount)
		if err != nil {
			return
		}
	}

	payload.OperationType = appModels.OP_STAKE
	payload.JettonMaster = pool.JettonMaster
	payload.Amount = stake.Amount

	w, err := t.ws.GetByUserId(stake.UserId)
	if err != nil {
		log.Error("Failed to get user wallet:", err)
		return
	}

	s, err := t.tcs.LoadSession(fmt.Sprint(tg.TelegramId))
	if err != nil {
		if _, err := util.SendTextMessage(
			b,
			tg.TelegramId,
			"❌ Привяжите свой кошелей заново! Потом повторите стейк. Комиссия уже учтена, он будет в списке ваших стейков",
		); err != nil {
			log.Error(err)
		}
		return
	}

	btns := util.GenerateButtonWallets(w, t.tcs, true)

	markup := util.CreateInlineMarup(1, btns...)
	if _, err := util.SendTextMessageMarkup(
		b,
		tg.TelegramId,
		fmt.Sprintf("✅ Комиссия принята. Подтвердите свой стейк в кошельке. %v %v", stake.Amount, pool.JettonName),
		markup,
	); err != nil {
		log.Error(err)
		return
	}

	jettonAddr, err := t.aws.TokenWalletAddress(pool.JettonMaster, address.MustParseAddr(w.Addr))
	if err != nil {
		log.Error(err)
		return
	}

	if _, err := t.tcs.SendJettonTransaction(
		fmt.Sprint(tg.TelegramId),
		jettonAddr.Address().String(),
		t.aws.GetAdminWalletAddr().String(),
		w.Addr,
		fmt.Sprintf("%f", stake.Amount),
		payload,
		s,
	); err != nil {
		log.Error(err)
		//if _, err := util.SendTextMessage(
		//	b,
		//	tg.TelegramId,
		//	fmt.Sprintf(
		//		"❌ Транзакция %f %v стейкинга не была подтверждена!",
		//		stake.Amount,
		//		pool.JettonName,
		//	),
		//); err != nil {
		//	log.Error(err)
		//}
		return
	}
	if _, err := t.opS.Create(
		stake.UserId,
		appModels.OP_PAID_COMMISSION_STAKE,
		fmt.Sprintf("Оплата комиссии за стейк %v", pool.JettonName),
	); err != nil {
		log.Error(err)
		return
	}
}

func (t *TgBot) stake(payload *appModels.Payload, b *bot.Bot) {
	var stake appModels.Stake
	if err := json.Unmarshal([]byte(payload.Payload), &stake); err != nil {
		log.Error("Failed to unmarshal stake data:", err)
		return
	}
	log.Infoln("начало создания стейка")
	stake.IsCommissionPaid = true

	log.Infoln("Поиск пула")
	pool, err := t.ps.GetId(stake.PoolId)
	if err != nil {
		log.Error("Failed to get pool id:", err)
		if err := t.returnTokens(stake.UserId, pool.JettonMaster, stake.Amount); err != nil {
			log.Error("Failed to return tokens:", err)
		}
		return
	}

	log.Infoln("Получение инфы о стейке")
	jettodData, err := t.aws.DataJetton(pool.JettonMaster)
	if err != nil {
		log.Error("Failed to get jettod data:", err)
		return
	}

	log.Infoln("поиск телеграмов")
	tgOwnerPool, err := t.ts.GetByUserId(pool.OwnerId)
	if err != nil {
		log.Error("Failed to get user wall:", err)
	}
	tgStaker, err := t.ts.GetByUserId(stake.UserId)
	if err != nil {
		log.Error("Failed to get user wall:", err)
	}

	log.Infoln("проверка кол-во стейкаов")
	stakesCountUser := t.ss.CountUser(stake.UserId)
	if stakesCountUser == 0 {
		u, err := t.us.GetById(stake.UserId)
		if err == nil {
			log.Infoln("отправка бонуса")
			if u.RefererId.Valid && u.RefererId.Int64 != 0 {
				go func() {
					if err := t.sendBonus(
						b,
						uint64(u.RefererId.Int64),
						&stake,
						tgStaker,
					); err != nil {
						log.Error("Failed to send bonus:", err)
						return
					}
				}()
			}
		}
	}
	log.Infoln("Сохранение стейка")
	_, err = t.ss.CreateStake(&stake)
	if err != nil {
		log.Error("Failed to create stake:", err)
		if err := t.returnTokens(stake.UserId, pool.JettonMaster, stake.Amount); err != nil {
			log.Error("Failed to return tokens:", err)
		}
		return
	}

	description := fmt.Sprintf("Стейк в jetton: %v. Кол-во: %v", jettodData.Name, stake.Amount)

	log.Infoln("Создание операции")
	_, err = t.opS.Create(stake.UserId, appModels.OP_STAKE, description)
	if err != nil {
		log.Error("Failed to create stake:", err)
		return
	}

	log.Infoln("Отправка сообщений в ТГ")

	if tgOwnerPool != nil {
		if _, err := util.SendTextMessage(b, tgOwnerPool.TelegramId, "✅ Новый стейк"); err != nil {
			log.Error("Failed to send message:", err)
			return
		}
	}
	if tgStaker != nil {
		if _, err := util.SendTextMessage(b, tgStaker.TelegramId, "✅ Стейк создан!"); err != nil {
			log.Error("Failed to send message:", err)
			return
		}
	}

	log.Infoln("Создание стейка завершено")
}

func (t *TgBot) sendBonus(b *bot.Bot, referalId uint64, stake *appModels.Stake, tgStaker *appModels.Telegram) error {
	u, err := t.us.GetByTelegramChatId(referalId)
	if err != nil {
		log.Error("Failed to get user :", err)
		return err
	}
	w, err := t.ws.GetByUserId(uint64(u.Id.Int64))
	if err != nil {
		log.Error("Failed to get user:", err)
		return err
	}
	jettonAdminAddr := os.Getenv("JETTON_CONTRACT_ADMIN_JETTON")
	if jettonAdminAddr == "" {
		return err
	}
	bonus := os.Getenv("REFERAL_BONUS")
	if bonus == "" {
		bonus = "2"
	}
	bonusNum, err := strconv.ParseFloat(bonus, 64)
	if err != nil {
		log.Error("Failed to parse bonus:", err)
		return err
	}
	decimal := os.Getenv("JETTON_DECIMAL")
	if decimal == "" {
		decimal = "9"
	}
	decimalNum, err := strconv.Atoi(decimal)
	if err != nil {
		log.Error("Failed to parse decimal:", err)
		return err
	}
	bonusAmount := stake.Amount * (bonusNum / 100)
	if _, err := t.aws.SendJetton(
		jettonAdminAddr,
		w.Addr,
		"",
		bonusAmount,
		decimalNum,
	); err != nil {
		log.Error("Failed to send bonus:", err)
		return err
	}
	tokenName := os.Getenv("JETTON_NAME_COIN")
	if tokenName == "" {
		tokenName = "NESTRAH"
	}

	if tgStaker != nil {
		if _, err := util.SendTextMessage(
			b,
			referalId,
			fmt.Sprintf("✅ Вы получили бонус %.2f %v, за пользователя %v. Токены были отправлены на привязанный кошелек", bonusAmount, tokenName, tgStaker.Username),
		); err != nil {
			log.Error("Failed to send bonus:", err)
			return err
		}
	}

	if err := t.rs.Save(&appModels.Referral{
		ReferrerUserId: u.Id,
		ReferralUserId: sql.NullInt64{
			Int64: int64(stake.UserId),
			Valid: true,
		},
		FirstStakeId: stake.Id,
		RewardGiven:  true,
		RewardAmount: bonusAmount,
	}); err != nil {
		log.Error("Failed to save referral:", err)
		return err
	}

	return nil
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
		if err := t.returnTokens(pool.OwnerId, pool.JettonMaster, pool.Reserve); err != nil {
			log.Error("Failed to return tokens:", err)
		}
		return
	}

	telegram, err := t.ts.GetByUserId(pool.OwnerId)
	if err != nil {
		log.Errorf("Failed to get telegram: %v", err)
		return
	}

	jettonData, err := t.aws.DataJetton(pool.JettonMaster)
	if err != nil {
		log.Errorf("Failed to get jetton: %v", err)
		return
	}

	text := fmt.Sprint(
		"✅ Пул был успешно создан! Оплатите комиссию, чтобы активировать его!\n\n",
		util.PoolInfo(&pool, t.ss, jettonData),
	)
	markup := util.GenerateOwnerPoolInlineKeyboard(
		pool.Id.Int64,
		buttons.BackMyPoolListId,
		pool.IsActive,
		pool.IsCommissionPaid,
		callbacksuf.My,
	)

	if _, err := util.SendTextMessageMarkup(b, telegram.TelegramId, text, markup); err != nil {
		log.Error("Failed to send telegram:", err)
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
		if err := t.returnTokens(pool.OwnerId, pool.JettonMaster, addReserve.Amount); err != nil {
			log.Error("Failed to return tokens:", err)
		}
		return
	}

	newReserve, err := t.ps.AddReserve(addReserve.PoolId, addReserve.Amount)
	if err != nil {
		log.Errorf("Failed to add reserve: %v", err)
		if err := t.returnTokens(pool.OwnerId, pool.JettonMaster, addReserve.Amount); err != nil {
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
		"Оплата комиссии jetton: %v. Комиссия: %v %v",
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

func (t *TgBot) returnTokens(userId uint64, jettonMaster string, amount float64) error {
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

	hash, err := t.aws.SendJetton(
		jettonMaster,
		userWall.Addr,
		"",
		amount,
		jetData.Decimals,
	)
	if err != nil {
		log.Error("Failed to send jetton data:", err)
		return err
	}

	_, err = t.opS.Create(userId, appModels.OP_RETURNING_TOKENS, fmt.Sprintf("Возврат. Hash операции: %v", base64.StdEncoding.EncodeToString(hash)))
	if err != nil {
		return err
	}

	return nil
}

func checkSendJettonOperation() {
	for {
		select {
		case f, ok := <-sendJettonInsurance:
			if !ok {
				continue
			}
			f()
		case f, ok := <-sendJettonClosePool:
			if !ok {
				continue
			}
			f()
		case f, ok := <-sendJettonProfit:
			if !ok {
				continue
			}
			f()
		case f, ok := <-sendJettonClosingTimeStake:
			if !ok {
				continue
			}
			f()
		}
	}
}
