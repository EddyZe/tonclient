package command

import (
	"context"
	"math"
	"tonclient/internal/services"
	"tonclient/internal/tonbot/buttons"
	"tonclient/internal/tonbot/callbacksuf"
	"tonclient/internal/util"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

var currentPageMyPools = make(map[int64]int)

type MyPools struct {
	b   *bot.Bot
	us  *services.UserService
	ps  *services.PoolService
	aws *services.AdminWalletService
	ss  *services.StakeService
}

func NewMyPoolsCommand(b *bot.Bot, us *services.UserService, ps *services.PoolService,
	aws *services.AdminWalletService, ss *services.StakeService) *MyPools {
	return &MyPools{
		b:   b,
		us:  us,
		ps:  ps,
		aws: aws,
		ss:  ss,
	}
}

func (c *MyPools) Execute(ctx context.Context, msg *models.Message) {
	chatId := msg.Chat.ID
	page, ok := currentPageMyPools[chatId]
	if !ok {
		page = 0
		currentPageMyPools[chatId] = page
	}

	user, err := c.us.GetByTelegramChatId(uint64(chatId))
	if err != nil {
		if _, err := util.SendTextMessage(c.b, uint64(chatId), "❌ Ваш аккаунт не активирован. Чтобы активировать аккаунт введите /start"); err != nil {
			log.Error(err)
		}
		return
	}
	totalPage := int(math.Ceil(float64(c.ps.CountUserPool(uint64(user.Id.Int64))) / float64(numberElementPage)))
	offset := page * numberElementPage
	limit := numberElementPage

	pools := c.ps.GetPoolsByUserIdLimit(uint64(user.Id.Int64), offset, limit)
	markup := util.GenerateNextBackMenu(
		page,
		totalPage,
		buttons.NextPageMyPool,
		buttons.BackPageMyPool,
		buttons.CloseListPool,
		util.GeneratePoolButtons(pools, c.aws, callbacksuf.My, c.ss)...,
	)

	if err := util.EditMessageMarkup(ctx, c.b, uint64(chatId), msg.ID, markup); err != nil {
		if _, err := util.SendTextMessageMarkup(
			c.b,
			uint64(chatId),
			"Выберите пул из списка, чтобы узнать подробную информацию о нем или управлять ими.\n\nВаши пулы: ",
			markup); err != nil {
			log.Error(err)
		}
	}
}

func (c *MyPools) NextPage(ctx context.Context, callback *models.CallbackQuery) {
	chatId := callback.From.ID
	user, err := c.us.GetByTelegramChatId(uint64(chatId))
	if err != nil {
		if _, err := util.SendTextMessage(c.b, uint64(chatId), "❌ Ваш аккаунт не активирован. Чтобы активировать аккаунт введите /start"); err != nil {
			log.Error(err)
		}
		return
	}
	totalPage := int(math.Ceil(float64(c.ps.CountUserPool(uint64(user.Id.Int64))) / float64(numberElementPage)))
	currentPageMyPools = util.NextPage(ctx, callback, currentPageMyPools, totalPage, c.b, c)
}

func (c *MyPools) BackPage(ctx context.Context, callback *models.CallbackQuery) {
	currentPageMyPools = util.BackPage(ctx, callback, currentPageMyPools, c.b, c)
}

func (c *MyPools) CloseList(ctx context.Context, callback *models.CallbackQuery) {
	currentPageMyPools = util.CloseList(ctx, callback, currentPageMyPools, c.b)
}
