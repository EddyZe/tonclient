package tonbot

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"tonclient/internal/config"
	appModels "tonclient/internal/models"
	"tonclient/internal/services"
	"tonclient/internal/tonbot/buttons"
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
}

func NewTgBot(token string, us *services.UserService, ts *services.TelegramService,
	ps *services.PoolService, aws *services.AdminWalletService, ss *services.StakeService,
	ws *services.WalletTonService) *TgBot {
	return &TgBot{
		token: token,
		us:    us,
		ts:    ts,
		ps:    ps,
		aws:   aws,
		ss:    ss,
		ws:    ws,
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

		if state, ok := userstate.CurrentState[msg.Chat.ID]; ok {
			t.handleState(ctx, state, b, msg)
			return
		}

		if strings.HasPrefix(text, "/start") {
			cmd := command.NewStartCommand(b, t.us, t.ts)
			cmd.Execute(ctx, msg)
			return
		}

		if text == buttons.InviteFriend {
			cmd := command.NewInviteFriendCommand(b, t.us)
			cmd.Execute(ctx, msg)
			return
		}

		if text == buttons.SelectPool {
			command.NewListPoolCommand(b, t.ps, t.aws).Execute(ctx, msg)
			return
		}

		if text == buttons.Setting {
			command.NewOpenSetting(b).Execute(ctx, msg)
			return
		}

		if text == buttons.Profile {
			command.NewProfileCommand(b, t.us, t.ws, t.aws, t.ps, t.ss).Execute(ctx, msg)
			return
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
		cmd := command.NewSetWalletCommand[*models.CallbackQuery](b, t.ws, t.us, t.aws)
		cmd.Execute(ctx, callback)
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

		userstate.CurrentState[msg.Chat.ID] = -1
	}

	if strings.HasPrefix(data, buttons.NextPagePool) {
		command.NewListPoolCommand(b, t.ps, t.aws).NextPage(ctx, callback)
		return
	}

	if strings.HasPrefix(data, buttons.BackPagePool) {
		command.NewListPoolCommand(b, t.ps, t.aws).BackPage(ctx, callback)
		return
	}

	if data == buttons.CloseListPool {
		command.NewListPoolCommand(b, t.ps, t.aws).CloseList(ctx, callback)
		return
	}

	if data == buttons.BackPoolListId {
		if err := util.CheckTypeMessage(b, callback); err != nil {
			log.Error("CheckTypeMessage: ", err)
			return
		}
		command.NewListPoolCommand(b, t.ps, t.aws).Execute(ctx, callback.Message.Message)
	}

	if strings.HasPrefix(data, buttons.CreateStakeId) {
		//TODO Реализовать стейк токенов
	}

	if strings.HasPrefix(data, buttons.PoolDataButton) {
		log.Infoln(data)
		command.NewPoolInfo(b, t.ps, t.us, t.ss).Execute(ctx, callback)
		return
	}
}

func (t *TgBot) handleState(ctx context.Context, state int, b *bot.Bot, msg *models.Message) {
	switch state {
	case userstate.EnterWalletAddr:
		command.NewSetWalletCommand[*models.Message](b, t.ws, t.us, t.aws).Execute(ctx, msg)
		break
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
	switch tr.OperationType {
	case appModels.OP_STAKE:
		var stake appModels.Stake
		if err := json.Unmarshal(tr.Payload, &stake); err != nil {
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

		if _, err := util.SendTextMessage(b, tg.TelegramId, "✅ Стейк был успешно создан!"); err != nil {
			log.Error("Failed to send message:", err)
			return
		}

		break
	case appModels.OP_CLAIM:
		break
	case appModels.OP_CLAIM_INSURANCE:
		break
	case appModels.OP_ADMIN_CREATE_POOL:
		var pool appModels.Pool
		if err := json.Unmarshal(tr.Payload, &pool); err != nil {
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

		if _, err := util.SendTextMessage(b, telegram.TelegramId, "✅ Пул был успешно создан!"); err != nil {
			log.Error("Failed to send telegram:", err)
			return
		}

		break
	case appModels.OP_ADMIN_ADD_RESERVE:
		var addReserve appModels.AddReserve
		if err := json.Unmarshal(tr.Payload, &addReserve); err != nil {
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

		break
	case appModels.OP_ADMIN_CLOSE_POOL:
		break
	case appModels.OP_GET_USER_STAKES:
		break
	case appModels.OP_PAY_COMMISION:
		var pool appModels.Pool
		if err := json.Unmarshal(tr.Payload, &pool); err != nil {
			log.Errorf("Failed to unmarshal payload data: %v", err)
			return
		}

		if !pool.Id.Valid {
			log.Error("pool id is not valid")
			return
		}
		id := pool.Id.Int64

		if tr.Amount < config.COMMISSION_AMOUNT {
			log.Error("Invalid amount received:", tr.Amount)
			if err := t.returnTokens(
				pool.OwnerId,
				pool.JettonMaster,
				fmt.Sprintf("❌ Комиссия должна быть %v. Возврат", config.COMMISSION_AMOUNT),
				tr.Amount); err != nil {
				log.Error("Failed to return tokens:", err)
			}
			return
		}

		if err := t.ps.SetCommissionPaid(uint64(id), true); err != nil {
			log.Errorf("Failed to set commission paid: %v", err)
			if err := t.returnTokens(
				pool.OwnerId,
				pool.JettonMaster,
				fmt.Sprintf("❌ Комиссия должна быть %v. Возврат", config.COMMISSION_AMOUNT),
				tr.Amount); err != nil {
				log.Error("Failed to return tokens:", err)
			}
			return
		}

		tg, err := t.ts.GetByUserId(pool.OwnerId)
		if err != nil {
			log.Errorf("Failed to get telegram: %v", err)
			return
		}

		if _, err := util.SendTextMessage(
			b,
			tg.TelegramId,
			"✅ Комиссия принята. Теперь ваш пул активен",
		); err != nil {
			log.Error("Failed to send telegram:", err)
			return
		}
		break
	default:
		return
	}
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
