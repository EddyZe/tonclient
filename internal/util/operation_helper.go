package util

import (
	"fmt"
	appModel "tonclient/internal/models"
	"tonclient/internal/tonbot/buttons"

	"github.com/go-telegram/bot/models"
)

func GenerateOperationButtons(operation []appModel.Operation) []models.InlineKeyboardButton {
	res := make([]models.InlineKeyboardButton, 0, 5)
	for _, op := range operation {
		if !op.Id.Valid {
			continue
		}
		opId := op.Id.Int64
		text := fmt.Sprintf("%v %v", op.Name, op.CreatedAt.Format("02.01.2006 15:04:05"))
		idButton := fmt.Sprintf("%v:%v", buttons.OpenOperationHistory, opId)
		res = append(res, CreateDefaultButton(idButton, text))
	}

	return res
}
