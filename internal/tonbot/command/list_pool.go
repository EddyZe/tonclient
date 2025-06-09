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

type ListPoolCommand struct {
	b   *bot.Bot
	ps  *services.PoolService
	aws *services.AdminWalletService
	ss  *services.StakeService
}

var numberElementPage = 5

var currentPageAllPools = make(map[int64]int)

func NewListPoolCommand(b *bot.Bot, ps *services.PoolService, aws *services.AdminWalletService, ss *services.StakeService) *ListPoolCommand {
	return &ListPoolCommand{
		b:   b,
		ps:  ps,
		aws: aws,
		ss:  ss,
	}
}

func (c *ListPoolCommand) Execute(ctx context.Context, msg *models.Message) {
	chatId := msg.Chat.ID
	page, ok := currentPageAllPools[chatId]
	if !ok {
		page = 0
		currentPageAllPools[chatId] = page
	}

	totalPage := int(math.Ceil(float64(c.ps.CountAllByStatus(true)) / float64(numberElementPage)))
	offset := page * numberElementPage
	limit := numberElementPage

	pools := c.ps.AllLimitByStatus(true, offset, limit)
	markup := util.GenerateNextBackMenu(
		page,
		totalPage,
		buttons.NextPagePool,
		buttons.BackPagePool,
		buttons.CloseListPool,
		util.GeneratePoolButtons(pools, c.aws, callbacksuf.All, c.ss)...,
	)

	if err := util.EditMessageMarkup(ctx, c.b, uint64(chatId), msg.ID, markup); err != nil {
		if _, err := util.SendTextMessageMarkup(
			c.b,
			uint64(chatId),
			"Выберите пул из списка, чтобы узнать подробную информацию о нем или сделать стейк.\n\nСозданные пулы: ",
			markup); err != nil {
			log.Error(err)
		}
	}

}

func (c *ListPoolCommand) NextPage(ctx context.Context, callback *models.CallbackQuery) {
	totalPage := int(math.Ceil(float64(c.ps.CountAll()) / float64(numberElementPage)))
	currentPageAllPools = util.NextPage(ctx, callback, currentPageAllPools, totalPage, c.b, c)
}

func (c *ListPoolCommand) BackPage(ctx context.Context, callback *models.CallbackQuery) {
	currentPageAllPools = util.BackPage(ctx, callback, currentPageAllPools, c.b, c)
}

func (c *ListPoolCommand) CloseList(ctx context.Context, callback *models.CallbackQuery) {
	currentPageAllPools = util.CloseList(ctx, callback, currentPageAllPools, c.b)
}
