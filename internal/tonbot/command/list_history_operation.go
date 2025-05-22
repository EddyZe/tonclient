package command

import (
	"context"
	"math"
	appModel "tonclient/internal/models"
	"tonclient/internal/services"
	"tonclient/internal/tonbot/buttons"
	"tonclient/internal/util"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

var currentPageOperation = make(map[int64]int)

type ListHistoryOperation struct {
	b   *bot.Bot
	us  *services.UserService
	opS *services.OperationService
}

func NewListHistoryOperation(b *bot.Bot, us *services.UserService, ops *services.OperationService) *ListHistoryOperation {
	return &ListHistoryOperation{
		b:   b,
		us:  us,
		opS: ops,
	}
}

func (c *ListHistoryOperation) Execute(ctx context.Context, msg *models.Message) {
	c.executeMessage(ctx, msg)
}

func (c *ListHistoryOperation) executeMessage(ctx context.Context, msg *models.Message) {
	chatId := msg.Chat.ID

	u, err := c.us.GetByTelegramChatId(uint64(chatId))
	if err != nil {
		if _, err := util.SendTextMessage(c.b, uint64(chatId), "❌ Аккаунт не активирован! Введите команду /start"); err != nil {
			log.Error(err)
		}
		return
	}

	markup, err := c.generateOperationList(chatId, u)
	if err != nil {
		log.Error(err)
		return
	}

	if err := util.EditMessageMarkup(
		ctx,
		c.b,
		uint64(chatId),
		msg.ID,
		markup,
	); err != nil {
		if _, err := util.SendTextMessageMarkup(
			c.b,
			uint64(chatId),
			"<b>Список ваших операций</b>\n\n Выберите операцию из списка, чтобы узнать подробности.",
			markup); err != nil {
			log.Error(err)
			return
		}
	}
}

func (c *ListHistoryOperation) generateOperationList(chatId int64, u *appModel.User) (*models.InlineKeyboardMarkup, error) {
	page := c.getCurrentPage(chatId)

	totalPage := c.getTotalPage(uint64(u.Id.Int64))
	offset := page * numberElementPage
	limit := numberElementPage

	operations, err := c.opS.GetByUserIdLimit(uint64(u.Id.Int64), offset, limit)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	markup := util.GenerateNextBackMenu(
		page,
		totalPage,
		buttons.NextPageHistory,
		buttons.BackPageHistory,
		buttons.CloseListHistory,
		util.GenerateOperationButtons(operations)...,
	)

	return markup, nil
}

func (c *ListHistoryOperation) getCurrentPage(chatId int64) int {
	page, ok := currentPageOperation[chatId]
	if !ok {
		page = 0
		currentPageOperation[chatId] = page
	}

	return page
}

func (c *ListHistoryOperation) NextPage(ctx context.Context, callback *models.CallbackQuery) {
	if err := util.CheckTypeMessage(c.b, callback); err != nil {
		return
	}
	msg := callback.Message.Message
	chatId := msg.Chat.ID

	u, err := c.us.GetByTelegramChatId(uint64(chatId))
	if err != nil {
		if _, err := util.SendTextMessage(c.b, uint64(chatId), "❌ Аккаунт не активирован! Введите команду /start"); err != nil {
			log.Error(err)
		}
		return
	}

	totalPage := c.getTotalPage(uint64(u.Id.Int64))
	currentPageOperation = util.NextPage(ctx, callback, currentPageOperation, totalPage, c.b, c)
}

func (c *ListHistoryOperation) BackPage(ctx context.Context, callback *models.CallbackQuery) {
	currentPageOperation = util.BackPage(ctx, callback, currentPageOperation, c.b, c)
}

func (c *ListHistoryOperation) CloseListHistory(ctx context.Context, callback *models.CallbackQuery) {
	currentPageOperation = util.CloseList(ctx, callback, currentPageOperation, c.b)
}

func (c *ListHistoryOperation) getTotalPage(userId uint64) int {
	return int(math.Ceil(float64(c.opS.CountByUserId(userId)) / float64(numberElementPage)))
}
