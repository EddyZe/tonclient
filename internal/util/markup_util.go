package util

import (
	"math"

	"github.com/go-telegram/bot/models"
)

func CreateInlineMarup(numberButtonInRow int, buttons ...models.InlineKeyboardButton) *models.InlineKeyboardMarkup {
	markup := make([][]models.InlineKeyboardButton, 0, 5)

	numberRows := math.Ceil(float64(len(buttons)) / float64(numberButtonInRow))

	index := 0
	for i := 0; i < int(numberRows); i++ {
		row := make([]models.InlineKeyboardButton, 0, 2)
		for j := 0; j < numberButtonInRow; j++ {
			if index == len(buttons) {
				break
			}
			row = append(row, buttons[index])
			index++
		}
		markup = append(markup, row)
	}

	return &models.InlineKeyboardMarkup{
		InlineKeyboard: markup,
	}
}

func CreateDefaultButton(idButton, text string) models.InlineKeyboardButton {
	return models.InlineKeyboardButton{
		Text:         text,
		CallbackData: idButton,
	}
}

func CreateDefaultButtonsReplay(numberButtonInRow int, textButton ...string) *models.ReplyKeyboardMarkup {
	markup := make([][]models.KeyboardButton, 0, 5)

	numberRows := math.Ceil(float64(len(textButton)) / float64(numberButtonInRow))
	index := 0
	for i := 0; i < int(numberRows); i++ {
		row := make([]models.KeyboardButton, 0, 2)
		for j := 0; j < numberButtonInRow; j++ {
			if index == len(textButton) {
				break
			}
			row = append(row, models.KeyboardButton{
				Text: textButton[index],
			})
			index++
		}
		markup = append(markup, row)
	}

	return &models.ReplyKeyboardMarkup{
		Keyboard:       markup,
		ResizeKeyboard: true,
	}
}
