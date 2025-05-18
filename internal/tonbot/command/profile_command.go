package command

import (
	"context"
	"fmt"
	appModels "tonclient/internal/models"
	"tonclient/internal/services"
	"tonclient/internal/tonbot/buttons"
	"tonclient/internal/util"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type ProfileCommand struct {
	b   *bot.Bot
	us  *services.UserService
	ws  *services.WalletTonService
	aws *services.AdminWalletService
	ps  *services.PoolService
	ss  *services.StakeService
	tcs *services.TonConnectService
}

func NewProfileCommand(b *bot.Bot, us *services.UserService, ws *services.WalletTonService,
	aws *services.AdminWalletService, ps *services.PoolService, ss *services.StakeService) *ProfileCommand {
	return &ProfileCommand{
		b:   b,
		us:  us,
		ws:  ws,
		aws: aws,
		ps:  ps,
		ss:  ss,
	}
}

func (c *ProfileCommand) Execute(ctx context.Context, msg *models.Message) {
	chatId := msg.Chat.ID
	user, err := c.us.GetByTelegramChatId(uint64(chatId))
	if err != nil {
		if err.Error() == "user not found" {
			if _, er := util.SendTextMessage(c.b, uint64(chatId), "❌ Профиль не найден. Введите команду /start, чтобы создать профиль!"); err != nil {
				log.Error("Failed send message", er)
			}
		}
		log.Error("Failed to get user by telegram chat id", "chatId", chatId, "err", err)
		return
	}

	var tonAddr string
	w, err := c.ws.GetByUserId(uint64(user.Id.Int64))
	if err != nil {
		log.Error(err)
		tonAddr = "Не указан"
	} else {
		tonAddr = w.Addr
	}

	activeStakes := c.ss.GetStakesUserIdStatus(uint64(user.Id.Int64), true)
	text := c.generateMessage(user, tonAddr, activeStakes)
	setWalAddrBtn := util.CreateDefaultButton(buttons.SetNumberWalletId, buttons.SetNumberWallet)

	markup := util.MenuWithBackButton(buttons.DefCloseId, buttons.DefCloseText, setWalAddrBtn)

	if _, err = util.SendTextMessageMarkup(c.b, uint64(chatId), text, markup); err != nil {
		log.Error("Failed send message", err)
		return
	}
}

func (c *ProfileCommand) generateMessage(u *appModels.User, tonAddr string, stakes *[]appModels.Stake) string {
	stakesTokenSum := 0.

	for _, s := range *stakes {
		stakesTokenSum += s.Amount
	}

	countActiveStake := len(*stakes)

	text := `
<b>👤 Ваш профиль NESTRAH</b>

<b>TON-адрес</b>: %v
<b>Дата регистрации</b>: %v
<b>Стейкнутые токены</b>: %v %v
<b>Общий застейконый объем</b>: %v токенов
`
	res := fmt.Sprintf(
		text,
		tonAddr,
		u.CreatedAt.Format("02 Jan 2006"),
		countActiveStake,
		util.SuffixPol(countActiveStake),
		stakesTokenSum,
	)

	return res
}
