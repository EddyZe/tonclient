package util

import (
	"fmt"
	"math"
	"strings"
	appModels "tonclient/internal/models"
	"tonclient/internal/services"
	"tonclient/internal/tonbot/buttons"

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

func GenerateNextBackMenu(currentPage, totalPage int, nextButtonId, backButtonId, closeButtonId string, buttons ...models.InlineKeyboardButton) *models.InlineKeyboardMarkup {
	elements := CreateInlineMarup(1, buttons...)
	next := models.InlineKeyboardButton{
		CallbackData: nextButtonId,
		Text:         "Далее ⏩",
	}

	back := models.InlineKeyboardButton{
		CallbackData: backButtonId,
		Text:         "Назад ⏪",
	}

	closeButton := models.InlineKeyboardButton{
		CallbackData: closeButtonId,
		Text:         "Закрыть ❌",
	}

	nextAndBackRow := make([]models.InlineKeyboardButton, 0, 2)
	if currentPage > 0 {
		nextAndBackRow = append(nextAndBackRow, back)
	}

	if currentPage < totalPage-1 {
		nextAndBackRow = append(nextAndBackRow, next)
	}

	closeRow := []models.InlineKeyboardButton{
		closeButton,
	}

	markup := elements.InlineKeyboard
	markup = append(markup, nextAndBackRow)
	markup = append(markup, closeRow)
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: markup,
	}
}

func CreateUrlInlineButton(text, url string) models.InlineKeyboardButton {
	return models.InlineKeyboardButton{
		Text: text,
		URL:  url,
	}
}

func CreateWebAppButton(text, url string) models.InlineKeyboardButton {
	return models.InlineKeyboardButton{
		Text: text,
		WebApp: &models.WebAppInfo{
			URL: url,
		},
	}
}

func MenuWithBackButton(buttonBackId, buttonBackText string, buttons ...models.InlineKeyboardButton) *models.InlineKeyboardMarkup {
	mainMarkup := CreateInlineMarup(1, buttons...)
	elements := mainMarkup.InlineKeyboard
	backBtn := CreateDefaultButton(buttonBackId, buttonBackText)
	elements = append(elements, []models.InlineKeyboardButton{
		backBtn,
	})

	return &models.InlineKeyboardMarkup{
		InlineKeyboard: elements,
	}
}

func GenerateButtonWallets(w *appModels.WalletTon, tcs *services.TonConnectService) []models.InlineKeyboardButton {
	lowwerWalletNma := strings.ToLower(w.Name)
	var btns []models.InlineKeyboardButton

	if lowwerWalletNma == "tonkeeper" {
		txt := fmt.Sprintf("%v %v", buttons.OpenWallet, w.Name)
		btn := CreateUrlInlineButton(txt, tcs.GetWalletUniversalLink(lowwerWalletNma))
		btns = append(btns, btn)
	} else {
		btn := CreateUrlInlineButton(buttons.OpenWallet, tcs.GetWalletUniversalLink(lowwerWalletNma))
		btns = append(btns, btn)
	}

	return btns
}
