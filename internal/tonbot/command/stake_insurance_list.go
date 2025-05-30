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

var currentPageGroupInsurance = make(map[int64]int)
var currentPageStakeInsurance = make(map[int64]int)

type StakeInsuranceList[T CommandType] struct {
	b  *bot.Bot
	us *services.UserService
	ss *services.StakeService
}

func NewStakeInsuranceList[T CommandType](b *bot.Bot, us *services.UserService, ss *services.StakeService) *StakeInsuranceList[T] {
	return &StakeInsuranceList[T]{
		b:  b,
		us: us,
		ss: ss,
	}
}

func (c *StakeInsuranceList[T]) Execute(ctx context.Context, args T) {
	if v, ok := any(args).(*models.Message); ok {
		c.executeMessage(ctx, v)
		return
	}

	if v, ok := any(args).(*models.CallbackQuery); ok {
		c.executeCallback(ctx, v)
		return
	}
}

func (c *StakeInsuranceList[T]) executeMessage(ctx context.Context, msg *models.Message) {
	chatId := msg.Chat.ID
	c.exc(ctx, chatId, 0)
}

func (c *StakeInsuranceList[T]) executeCallback(ctx context.Context, callback *models.CallbackQuery) {
	if err := util.CheckTypeMessage(c.b, callback); err != nil {
		return
	}

	msg := callback.Message.Message
	chatId := callback.From.ID
	messageId := msg.ID

	if callback.Data == buttons.InsuranceBackListGroup {
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

	page := util.GetCurrentPage(chatId, currentPageStakeInsurance)
	totalPage := c.totalPageStakesFromGroup(uint64(u.Id.Int64), jettonName, -30)
	offset := page * numberElementPage
	limit := numberElementPage

	stakes := c.ss.GetByJettonNameAndUserIdLimitIsInsurancePaid(
		uint64(u.Id.Int64),
		jettonName,
		offset,
		limit,
		false,
		false,
	)

	stakes = util.FilterProcientStakes(*stakes, -30, false)

	markup := util.GenerateNextBackMenu(
		page,
		totalPage,
		fmt.Sprintf("%v:%v", buttons.InsuranceNextPageJettonName, jettonName),
		fmt.Sprintf("%v:%v", buttons.InsuranceBackPageJettonName, jettonName),
		buttons.InsuranceCloseGroup,
		util.GenerateStakeListByGroup(*stakes, jettonName, buttons.InsuranceOpenStakeInfo)...,
	)

	btns := markup.InlineKeyboard
	btns[len(btns)-1][0] = util.CreateDefaultButton(buttons.InsuranceBackListGroup, buttons.BackListGroup)
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

func (c *StakeInsuranceList[T]) exc(ctx context.Context, chatId int64, messageId int) {
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

	groups := c.getGroups(uint64(chatId), uint64(u.Id.Int64), false, -30)
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
		"Список стейков в которых вы можете получить компенсацию:",
		makup); err != nil {
		log.Error(err)
	}
}

func (c *StakeInsuranceList[T]) NextPageInsuranceStake(ctx context.Context, callback *models.CallbackQuery) {
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

	totalPage := c.totalPageStakesFromGroup(uint64(u.Id.Int64), jettonName, -30)

	currentPageStakeInsurance = util.NextPageV2(
		callback,
		currentPageStakeInsurance,
		totalPage,
		c.b,
		func() {
			c.executeCallback(ctx, callback)
		},
	)
}

func (c *StakeInsuranceList[T]) BackPageInsuranceStake(ctx context.Context, callback *models.CallbackQuery) {
	if err := util.CheckTypeMessage(c.b, callback); err != nil {
		return
	}

	currentPageStakeInsurance = util.BackPageV2(
		callback,
		currentPageStakeInsurance,
		c.b,
		func() {
			c.executeCallback(ctx, callback)
		},
	)
}

func (c *StakeInsuranceList[T]) NextPageGroup(ctx context.Context, callback *models.CallbackQuery) {
	if err := util.CheckTypeMessage(c.b, callback); err != nil {
		return
	}
	chatId := callback.From.ID

	u, err := c.us.GetByTelegramChatId(uint64(chatId))
	if err != nil {
		return
	}

	totalPage := c.totalPageGroupsStakes(uint64(u.Id.Int64), -30)

	currentPageGroupInsurance = util.NextPageV2(
		callback,
		currentPageGroupInsurance,
		totalPage,
		c.b,
		func() {
			c.exc(ctx, chatId, callback.Message.Message.ID)
		},
	)
}

func (c *StakeInsuranceList[T]) BackPageGroup(ctx context.Context, callback *models.CallbackQuery) {
	if err := util.CheckTypeMessage(c.b, callback); err != nil {
		return
	}

	chatId := callback.From.ID

	currentPageGroupInsurance = util.BackPageV2(
		callback,
		currentPageGroupInsurance,
		c.b,
		func() {
			c.exc(ctx, chatId, callback.Message.Message.ID)
		},
	)
}

func (c *StakeInsuranceList[T]) CloseList(ctx context.Context, callback *models.CallbackQuery) {
	if err := util.CheckTypeMessage(c.b, callback); err != nil {
		return
	}
	currentPageGroupInsurance = util.CloseList(ctx, callback, currentPageGroupInsurance, c.b)
}

func (c *StakeInsuranceList[T]) generateMarkup(chatId int64, u *appModels.User, groups *[]appModels.GroupElements) *models.InlineKeyboardMarkup {
	page := util.GetCurrentPage(chatId, currentPageGroupInsurance)
	totalPage := c.totalPageGroupsStakes(uint64(u.Id.Int64), -30)

	markup := util.GenerateNextBackMenu(
		page,
		totalPage,
		buttons.InsuranceNextPageGroup,
		buttons.InsuranceBackPageGroup,
		buttons.InsuranceCloseGroup,
		util.GenerateGroupButtons(groups, buttons.InsuranceOpenGroupId)...,
	)
	return markup
}

func (c *StakeInsuranceList[T]) getGroups(chatId, userId uint64, b bool, procient float64) *[]appModels.GroupElements {
	page := util.GetCurrentPage(int64(chatId), currentPageGroupInsurance)
	offset := page * numberElementPage
	limit := numberElementPage

	return c.ss.GroupFromPoolByUserIdLimitIsInsurancePaid(userId, limit, offset, b, false, procient)
}

func (c *StakeInsuranceList[T]) totalPageGroupsStakes(userId uint64, procient float64) int {
	return int(math.Ceil(float64(c.ss.CountGroupsStakesUserIdIsInsurancePaid(userId, false, false, procient)) / float64(numberElementPage)))
}

func (c *StakeInsuranceList[T]) totalPageStakesFromGroup(userId uint64, jettonName string, procient float64) int {
	return int(math.Ceil(float64(c.ss.CountGroupsStakesByUserIdAndJettonNameIsInsurancePaid(userId, jettonName, false, false, procient)) / float64(numberElementPage)))
}
