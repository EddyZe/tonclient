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

type ListPoolCommand struct {
	b   *bot.Bot
	ps  *services.PoolService
	aws *services.AdminWalletService
}

var numberElementPage = 5

var currentPage = make(map[int64]int)

func NewListPoolCommand(b *bot.Bot, ps *services.PoolService, aws *services.AdminWalletService) *ListPoolCommand {
	return &ListPoolCommand{
		b:   b,
		ps:  ps,
		aws: aws,
	}
}

func (c *ListPoolCommand) Execute(ctx context.Context, msg *models.Message) {
	chatId := msg.Chat.ID
	page, ok := currentPage[chatId]
	if !ok {
		page = 0
		currentPage[chatId] = page
	}

	totalPage := int(math.Ceil(float64(c.ps.CountAll()) / float64(numberElementPage)))
	offset := page * numberElementPage
	limit := numberElementPage

	pools := c.ps.AllLimit(offset, limit)
	markup := util.GenerateNextBackMenu(
		page,
		totalPage,
		buttons.NextPagePool,
		buttons.BackPagePool,
		buttons.CloseListPool,
		c.generatePoolButtons(pools)...,
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

	if err := util.CheckTypeMessage(c.b, callback); err != nil {
		log.Error(err)
		return
	}

	msg := callback.Message.Message
	chatId := msg.Chat.ID
	page, ok := currentPage[chatId]
	if !ok {
		page = 0
	}
	if page < c.ps.CountAll()-1 {
		page++
		currentPage[chatId] = page
	}
	c.Execute(ctx, msg)
}

func (c *ListPoolCommand) BackPage(ctx context.Context, callback *models.CallbackQuery) {
	if err := util.CheckTypeMessage(c.b, callback); err != nil {
		log.Error(err)
		return
	}

	msg := callback.Message.Message
	chatId := msg.Chat.ID
	page, ok := currentPage[chatId]
	if !ok {
		page = 0
		currentPage[chatId] = page
	}
	if page > 0 {
		page--
		currentPage[chatId] = page
	}
	c.Execute(ctx, msg)
}

func (c *ListPoolCommand) CloseList(ctx context.Context, callback *models.CallbackQuery) {
	if err := util.CheckTypeMessage(c.b, callback); err != nil {
		log.Error(err)
		return
	}
	msg := callback.Message.Message
	chatId := msg.Chat.ID
	currentPage[chatId] = 0

	if err := util.DeleteMessage(ctx, c.b, uint64(chatId), msg.ID); err != nil {
		log.Error(err)
		return
	}
}

func (c *ListPoolCommand) generateNamePool(pool *appModels.Pool) string {
	jettonData, err := c.aws.DataJetton(pool.JettonMaster)
	if err != nil {
		return "Без названия"
	}
	return fmt.Sprintf("%v (%d %v / %d%% / резерв %v)", jettonData.Name, pool.Period, util.SuffixDay(int(pool.Period)), pool.Reward, pool.Reserve)
}

func (c *ListPoolCommand) generatePoolButtons(pool *[]appModels.Pool) []models.InlineKeyboardButton {
	res := make([]models.InlineKeyboardButton, 0, len(*pool))
	for _, p := range *pool {
		if !p.Id.Valid {
			continue
		}
		poolId := p.Id.Int64
		res = append(
			res,
			util.CreateDefaultButton(
				fmt.Sprintf("%v:%d", buttons.PoolDataButton, poolId),
				c.generateNamePool(&p),
			),
		)
	}
	return res
}
