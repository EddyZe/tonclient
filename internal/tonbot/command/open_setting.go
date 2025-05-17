package command

import (
	"context"
	"fmt"
	"tonclient/internal/tonbot/buttons"
	"tonclient/internal/util"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type OpenSetting struct {
	b *bot.Bot
}

func NewOpenSetting(b *bot.Bot) *OpenSetting {
	return &OpenSetting{b}
}

func (c *OpenSetting) Execute(ctx context.Context, msg *models.Message) {
	chatId := msg.Chat.ID
	btn1 := util.CreateDefaultButton(buttons.RoleButtonUserId, buttons.RoleButtonUserText)
	btn2 := util.CreateDefaultButton(buttons.RoleButtonOwnerTokensId, buttons.RoleButtonOwnerTokensText)

	markup := util.CreateInlineMarup(2, btn1, btn2)

	if _, err := util.SendTextMessageMarkup(c.b, uint64(chatId), c.generateMessageResponse(), markup); err != nil {
		log.Error(err)
		return
	}
}

func (c *OpenSetting) generateMessageResponse() string {
	res := `
<b>%v</b>

Тут вы можете выбрать роль. При выборе роли, клавиатура будет изменена в соответствии с ролью`

	return fmt.Sprintf(res, buttons.Setting)
}
