package command

import (
	"context"
	"fmt"
	"math"
	appModels "tonclient/internal/models"
	"tonclient/internal/services"
	"tonclient/internal/tonbot/buttons"
	"tonclient/internal/util"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

var currentPageGroupProfit = make(map[int64]int)
var currentPageStakeProfit = make(map[int64]int)

type StakeProfitList[T CommandType] struct {
	b  *bot.Bot
	us *services.UserService
	ss *services.StakeService
	ps *services.PoolService
}

func NewStakeProfitList[T CommandType](b *bot.Bot, us *services.UserService, ss *services.StakeService, ps *services.PoolService) *StakeProfitList[T] {
	return &StakeProfitList[T]{
		b:  b,
		us: us,
		ss: ss,
		ps: ps,
	}
}

func (c *StakeProfitList[T]) Execute(ctx context.Context, args T) {
	if v, ok := any(args).(*models.Message); ok {
		c.executeMessage(ctx, v)
		return
	}

	if v, ok := any(args).(*models.CallbackQuery); ok {
		c.executeCallback(ctx, v)
		return
	}
}

func (c *StakeProfitList[T]) executeMessage(ctx context.Context, msg *models.Message) {
	chatId := msg.Chat.ID
	c.exc(ctx, chatId, 0)
}

func (c *StakeProfitList[T]) executeCallback(ctx context.Context, callback *models.CallbackQuery) {
	if err := util.CheckTypeMessage(c.b, callback); err != nil {
		return
	}

	msg := callback.Message.Message
	chatId := callback.From.ID
	messageId := msg.ID

	if callback.Data == buttons.ProfitBackListGroup {
		c.exc(ctx, chatId, messageId)
		return
	}

	jettonName, err := util.GetJettonNameFromCallbackData(c.b, uint64(chatId), callback.Data)
	if err != nil {
		log.Error(err)
		return
	}

	u, err := c.us.GetByTelegramChatId(uint64(chatId))
	if err != nil {
		log.Error(err)
		return
	}

	page := util.GetCurrentPage(chatId, currentPageStakeProfit)
	totalPage := c.totalPageStakesFromGroup(uint64(u.Id.Int64), jettonName)
	offset := page * numberElementPage
	limit := numberElementPage

	stakes := c.ss.GetByJettonNameAndUserIdLimitIsProfitPaid(
		uint64(u.Id.Int64),
		jettonName,
		offset,
		limit,
		false,
		false,
	)

	markup := util.GenerateNextBackMenu(
		page,
		totalPage,
		fmt.Sprintf("%v:%v", buttons.ProfitNextPageJettonName, jettonName),
		fmt.Sprintf("%v:%v", buttons.ProfitBackPageJettonName, jettonName),
		buttons.CloseListStakesGroupId,
		util.GenerateStakeListByGroup(*stakes, jettonName, buttons.ProfitOpenStakeInfo)...,
	)

	btns := markup.InlineKeyboard
	btns[len(btns)-1][0] = util.CreateDefaultButton(buttons.ProfitBackListGroup, buttons.BackListGroup)
	markup.InlineKeyboard = btns

	if err := util.EditTextMessageMarkup(
		ctx,
		c.b,
		uint64(chatId),
		messageId,
		fmt.Sprintf("Список стейков по токену %v", jettonName),
		markup,
	); err != nil {
		log.Error(err)
	}
}

func (c *StakeProfitList[T]) exc(ctx context.Context, chatId int64, messageId int) {
	u, err := c.us.GetByTelegramChatId(uint64(chatId))
	if err != nil {
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			"❌ Ваш аккаунт не активировать. Введите команду /start",
		); err != nil {
			log.Error(err)
		}
		return
	}

	groups := c.getGroups(uint64(chatId), uint64(u.Id.Int64), false)
	makup := c.generateMarkup(chatId, u, groups)

	if messageId != 0 {
		if err := util.EditMessageMarkup(
			ctx,
			c.b,
			uint64(chatId),
			messageId,
			makup); err != nil {
			log.Error(err)
		}
		return
	}

	if _, err := util.SendTextMessageMarkup(
		c.b,
		uint64(chatId),
		"Список стейков в которых вы можете получить награду:",
		makup); err != nil {
		log.Error(err)
	}
}

func (c *StakeProfitList[T]) NextPageProfitStake(ctx context.Context, callback *models.CallbackQuery) {
	if err := util.CheckTypeMessage(c.b, callback); err != nil {
		return
	}
	chatId := callback.From.ID

	u, err := c.us.GetByTelegramChatId(uint64(chatId))
	if err != nil {
		return
	}

	jettonName, err := util.GetJettonNameFromCallbackData(c.b, uint64(chatId), callback.Data)
	if err != nil {
		log.Error(err)
		return
	}

	totalPage := c.totalPageStakesFromGroup(uint64(u.Id.Int64), jettonName)

	currentPageStakeProfit = util.NextPageV2(
		callback,
		currentPageStakeProfit,
		totalPage,
		c.b,
		func() {
			c.executeCallback(ctx, callback)
		},
	)
}

func (c *StakeProfitList[T]) BackPageProfitStake(ctx context.Context, callback *models.CallbackQuery) {
	if err := util.CheckTypeMessage(c.b, callback); err != nil {
		return
	}

	currentPageStakeProfit = util.BackPageV2(
		callback,
		currentPageStakeProfit,
		c.b,
		func() {
			c.executeCallback(ctx, callback)
		},
	)
}

func (c *StakeProfitList[T]) NextPageGroup(ctx context.Context, callback *models.CallbackQuery) {
	if err := util.CheckTypeMessage(c.b, callback); err != nil {
		return
	}
	chatId := callback.From.ID

	u, err := c.us.GetByTelegramChatId(uint64(chatId))
	if err != nil {
		return
	}

	totalPage := c.totalPageGroupsStakes(uint64(u.Id.Int64))

	currentPageGroupProfit = util.NextPageV2(
		callback,
		currentPageGroupProfit,
		totalPage,
		c.b,
		func() {
			c.exc(ctx, chatId, callback.Message.Message.ID)
		},
	)

}

func (c *StakeProfitList[T]) BackPageGroup(ctx context.Context, callback *models.CallbackQuery) {
	if err := util.CheckTypeMessage(c.b, callback); err != nil {
		return
	}

	chatId := callback.From.ID

	currentPageGroupProfit = util.BackPageV2(
		callback,
		currentPageGroupProfit,
		c.b,
		func() {
			c.exc(ctx, chatId, callback.Message.Message.ID)
		},
	)
}

func (c *StakeProfitList[T]) CloseList(ctx context.Context, callback *models.CallbackQuery) {
	if err := util.CheckTypeMessage(c.b, callback); err != nil {
		return
	}
	currentPageGroupProfit = util.CloseList(ctx, callback, currentPageGroupProfit, c.b)
}

func (c *StakeProfitList[T]) generateMarkup(chatId int64, u *appModels.User, groups *[]appModels.GroupElements) *models.InlineKeyboardMarkup {
	page := util.GetCurrentPage(chatId, currentPageGroupProfit)
	totalPage := c.totalPageGroupsStakes(uint64(u.Id.Int64))

	markup := util.GenerateNextBackMenu(
		page,
		totalPage,
		buttons.ProfitNextPageGroup,
		buttons.ProfitBackPageGroup,
		buttons.ProfitCloseGroup,
		util.GenerateGroupButtons(groups, buttons.ProfitOpenGroupId)...,
	)
	return markup
}

func (c *StakeProfitList[T]) getGroups(chatId, userId uint64, b bool) *[]appModels.GroupElements {
	page := util.GetCurrentPage(int64(chatId), currentPageGroupProfit)
	offset := page * numberElementPage
	limit := numberElementPage

	return c.ss.GroupFromPoolByUserIdLimitIsProfitPaid(userId, limit, offset, b, false)
}

func (c *StakeProfitList[T]) totalPageGroupsStakes(userId uint64) int {
	return int(math.Ceil(float64(c.ss.CountGroupsStakesUserIdProfitPaid(userId, false, false)) / float64(numberElementPage)))
}

func (c *StakeProfitList[T]) totalPageStakesFromGroup(userId uint64, jettonName string) int {
	return int(math.Ceil(float64(c.ss.CountGroupsStakesByUserIdAndJettonNameIsProfitPaid(userId, jettonName, false, false)) / float64(numberElementPage)))
}
