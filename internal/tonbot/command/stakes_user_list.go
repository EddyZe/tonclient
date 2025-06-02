package command

import (
	"context"
	"fmt"
	"math"
	"time"
	"tonclient/internal/services"
	"tonclient/internal/tonbot/buttons"
	"tonclient/internal/util"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

var currentPageGroupStakes = make(map[int64]int)
var currentPageGroupStakesJettonName = make(map[int64]int)

type StakesUserList[T CommandType] struct {
	b  *bot.Bot
	us *services.UserService
	ss *services.StakeService
}

func NewStakesUserList[T CommandType](b *bot.Bot, us *services.UserService, ss *services.StakeService) *StakesUserList[T] {
	return &StakesUserList[T]{
		b:  b,
		us: us,
		ss: ss,
	}
}

func (c *StakesUserList[T]) Execute(ctx context.Context, args T) {
	if v, ok := any(args).(*models.Message); ok {
		c.executeMessage(v)
		return
	}

	if v, ok := any(args).(*models.CallbackQuery); ok {
		c.executeCallback(v)
		return
	}
}

func (c *StakesUserList[T]) executeCallback(callback *models.CallbackQuery) {
	if err := util.CheckTypeMessage(c.b, callback); err != nil {
		return
	}
	msg := callback.Message.Message
	chatId := msg.Chat.ID
	callbackData := callback.Data

	if callbackData == buttons.NextListStakesGroupId || callbackData == buttons.BackListStakesGroupId {
		c.backOrNextPageGroupStakes(chatId, msg.ID)
		return
	}

	jettonName, err := util.GetJettonNameFromCallbackData(c.b, uint64(chatId), callbackData)
	if err != nil {
		log.Error(err)
		return
	}

	u, err := c.us.GetByTelegramChatId(uint64(chatId))
	if err != nil {
		log.Error(err)
		return
	}

	page := c.getPageGroupStakesJetton(chatId)

	totalPage := c.totalPageStakesFromGroup(uint64(u.Id.Int64), jettonName)
	offset := page * numberElementPage
	limit := numberElementPage

	stakes := c.ss.GetByJettonNameAndUserIdLimit(
		uint64(u.Id.Int64),
		jettonName,
		offset,
		limit,
	)

	markup := util.GenerateNextBackMenu(
		page,
		totalPage,
		fmt.Sprintf("%v:%v", buttons.NextPageStakesFromGroupJettonName, jettonName),
		fmt.Sprintf("%v:%v", buttons.BackPageStakesFromGroupJettonName, jettonName),
		buttons.CloseListStakesGroupId,
		util.GenerateStakeListByGroup(*stakes, jettonName, buttons.OpenStakeInfo)...,
	)

	btns := markup.InlineKeyboard
	btns[len(btns)-1][0] = util.CreateDefaultButton(buttons.BackListGroupId, buttons.BackListGroup)
	markup.InlineKeyboard = btns
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := util.EditMessageMarkup(
		ctx,
		c.b,
		uint64(chatId),
		msg.ID,
		markup,
	); err != nil {
		log.Error(err)
		return
	}
}

func (c *StakesUserList[T]) executeMessage(msg *models.Message) {
	chatId := msg.Chat.ID

	markup, err := c.getGroupList(chatId)
	if err != nil {
		log.Error(err)
		return
	}

	if _, err := util.SendTextMessageMarkup(
		c.b,
		uint64(chatId),
		"Список ваших стейков. Выберите стейк из списка, чтобы посмотреть информацию: ",
		markup); err != nil {
		log.Error(err)
		return
	}
}

func (c *StakesUserList[T]) getGroupList(chatId int64) (*models.InlineKeyboardMarkup, error) {
	u, err := c.us.GetByTelegramChatId(uint64(chatId))
	if err != nil {
		log.Error(err)
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			"❌ Аккаунт не активирован. Введите команду /start",
		); err != nil {
			log.Error(err)
		}
		return nil, err
	}

	page := c.getPageGroupStakes(chatId)

	totalPage := c.totalPageGroupsStakes(uint64(u.Id.Int64))
	offset := page * numberElementPage
	limit := numberElementPage

	groups := c.ss.GroupFromPoolByUserIdLimit(uint64(u.Id.Int64), limit, offset)
	log.Infoln(groups)
	markup := util.GenerateNextBackMenu(
		page,
		totalPage,
		buttons.NextListStakesGroupId,
		buttons.BackListStakesGroupId,
		buttons.CloseListStakesGroupId,
		util.GenerateGroupButtons(groups, buttons.OpenGroupId)...,
	)

	return markup, nil
}

func (c *StakesUserList[T]) getPageGroupStakes(chatId int64) int {
	page, ok := currentPageGroupStakes[chatId]
	if !ok {
		page = 0
	}
	return page
}

func (c *StakesUserList[T]) getPageGroupStakesJetton(chatId int64) int {
	pageJetton, ok := currentPageGroupStakesJettonName[chatId]
	if !ok {
		pageJetton = 0
	}
	return pageJetton
}

func (c *StakesUserList[T]) BackStakesGroup(callback *models.CallbackQuery) {
	delete(currentPageGroupStakesJettonName, callback.From.ID)
	if err := util.CheckTypeMessage(c.b, callback); err != nil {
		return
	}
	msg := callback.Message.Message
	chatId := msg.Chat.ID
	markup, err := c.getGroupList(chatId)
	if err != nil {
		log.Error(err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := util.EditMessageMarkup(
		ctx,
		c.b,
		uint64(chatId),
		msg.ID,
		markup,
	); err != nil {
		log.Error(err)
		return
	}
}

func (c *StakesUserList[T]) CloseGroupList(callback *models.CallbackQuery) {
	if err := util.CheckTypeMessage(c.b, callback); err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	currentPageGroupStakes = util.CloseList(ctx, callback, currentPageGroupStakes, c.b)
}

// NextGroupPage открывает следующую страницу сгруперованных стейков
func (c *StakesUserList[T]) NextGroupPage(callback *models.CallbackQuery) {
	if err := util.CheckTypeMessage(c.b, callback); err != nil {
		return
	}

	chatId := callback.From.ID

	u, err := c.us.GetByTelegramChatId(uint64(chatId))
	if err != nil {
		log.Error(err)
		return
	}

	totalPage := c.totalPageGroupsStakes(uint64(u.Id.Int64))

	currentPageGroupStakes = util.NextPageV2(
		callback,
		currentPageGroupStakes,
		totalPage,
		c.b,
		func() {
			c.executeCallback(callback)
		},
	)
}

// BackGroupPage Открывает предыдущую страницу сгруперованных стейков
func (c *StakesUserList[T]) BackGroupPage(callback *models.CallbackQuery) {
	if err := util.CheckTypeMessage(c.b, callback); err != nil {
		return
	}

	currentPageGroupStakes = util.BackPageV2(
		callback,
		currentPageGroupStakes,
		c.b,
		func() {
			c.executeCallback(callback)
		},
	)
}

func (c *StakesUserList[T]) backOrNextPageGroupStakes(chatId int64, messageId int) {
	markup, err := c.getGroupList(chatId)
	if err != nil {
		log.Error(err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := util.EditMessageMarkup(
		ctx,
		c.b,
		uint64(chatId),
		messageId,
		markup,
	); err != nil {
		log.Error(err)
		return
	}
}

// NextPageStakesFromGroup Перевлючает следующую страницу стейков открытой группы
func (c *StakesUserList[T]) NextPageStakesFromGroup(callback *models.CallbackQuery) {
	if err := util.CheckTypeMessage(c.b, callback); err != nil {
		return
	}
	chatId := callback.From.ID

	u, err := c.us.GetByTelegramChatId(uint64(chatId))
	if err != nil {
		log.Error(err)
		return
	}

	jettonName, err := util.GetJettonNameFromCallbackData(c.b, uint64(chatId), callback.Data)

	totalPage := c.totalPageStakesFromGroup(uint64(u.Id.Int64), jettonName)
	currentPageGroupStakesJettonName = util.NextPageV2(
		callback,
		currentPageGroupStakesJettonName,
		totalPage,
		c.b,
		func() {
			c.executeCallback(callback)
		})
}

// BackStakesFromGroup Перевлючает предыдущую страницу стейков открытой группы
func (c *StakesUserList[T]) BackStakesFromGroup(callback *models.CallbackQuery) {
	if err := util.CheckTypeMessage(c.b, callback); err != nil {
		return
	}
	currentPageGroupStakesJettonName = util.BackPageV2(
		callback,
		currentPageGroupStakesJettonName,
		c.b,
		func() {
			c.executeCallback(callback)
		})
}

func (c *StakesUserList[T]) totalPageGroupsStakes(userId uint64) int {
	return int(math.Ceil(float64(c.ss.CountGroupsStakesUserId(userId)) / float64(numberElementPage)))
}

func (c *StakesUserList[T]) totalPageStakesFromGroup(userId uint64, jettonName string) int {
	return int(math.Ceil(float64(c.ss.CountGroupsStakesByUserIdAndJettonName(userId, jettonName)) / float64(numberElementPage)))
}
