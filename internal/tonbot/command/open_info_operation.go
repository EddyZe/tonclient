package command

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	appModels "tonclient/internal/models"
	"tonclient/internal/services"
	"tonclient/internal/tonbot/buttons"
	"tonclient/internal/util"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type OpenInfoOperation struct {
	b   *bot.Bot
	opS *services.OperationService
}

func NewOpenInfoOperation(b *bot.Bot, opS *services.OperationService) *OpenInfoOperation {
	return &OpenInfoOperation{
		b:   b,
		opS: opS,
	}
}

func (c *OpenInfoOperation) Execute(ctx context.Context, callback *models.CallbackQuery) {
	if err := util.CheckTypeMessage(c.b, callback); err != nil {
		return
	}

	msg := callback.Message.Message
	chatId := msg.Chat.ID
	splitData := strings.Split(callback.Data, ":")
	if len(splitData) != 2 {
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			"❌ Не могу обработать эту кнопку!",
		); err != nil {
			log.Error(err)
			return
		}
		return
	}

	opId, err := strconv.ParseInt(splitData[1], 10, 64)
	if err != nil {
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			"❌ Не могу открыть данную операцию!",
		); err != nil {
			log.Error(err)
		}
		return
	}

	op, err := c.opS.GetById(uint64(opId))
	if err != nil {
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			"❌ Операция не найдена! Возможно она была удалена!",
		); err != nil {
			log.Error(err)
		}
		return
	}

	info := c.generateTextOperation(op)
	backListHistory := util.CreateDefaultButton(buttons.BackHistoryListId, buttons.BackHistoryList)
	markup := util.CreateInlineMarup(1, backListHistory)
	if err := util.EditTextMessageMarkup(
		ctx,
		c.b,
		uint64(chatId),
		msg.ID,
		info,
		markup,
	); err != nil {
		log.Error(err)
		return
	}
}

func (c *OpenInfoOperation) generateTextOperation(op *appModels.Operation) string {
	text := `
	<b>%v</b>

	<b>Детали</b>: %v
	<b>Время выполнения операции</b>: %v
	`

	return fmt.Sprintf(text, op.Name, op.Description, op.CreatedAt.Format("02.01.2006 15:04:05"))
}
