package command

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	appMoels "tonclient/internal/models"
	"tonclient/internal/services"
	"tonclient/internal/tonbot/buttons"
	"tonclient/internal/util"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

type OpenPoolInfoCommand struct {
	b  *bot.Bot
	ps *services.PoolService
	us *services.UserService
	ss *services.StakeService
}

func NewPoolInfo(b *bot.Bot, ps *services.PoolService, us *services.UserService,
	ss *services.StakeService) *OpenPoolInfoCommand {

	return &OpenPoolInfoCommand{
		b:  b,
		ps: ps,
		us: us,
		ss: ss,
	}
}

func (c *OpenPoolInfoCommand) Execute(ctx context.Context, callback *models.CallbackQuery) {
	if err := util.CheckTypeMessage(c.b, callback); err != nil {
		log.Error("[OpenPoolInfoCommand.Execute]", err)
		return
	}

	data := callback.Data
	msg := callback.Message.Message
	chatId := msg.Chat.ID

	splitData := strings.Split(data, ":")
	poolIdStr := splitData[1]

	user, err := c.us.GetByTelegramChatId(uint64(chatId))
	if err != nil {
		log.Error("[OpenPoolInfoCommand.Execute]", err)
		if _, err := util.SendTextMessage(c.b, uint64(chatId), "❌ Ваш аккаунт не кативирован, чтобы активировать аккаунт введите команду /start"); err != nil {
			log.Error("[OpenPoolInfoCommand.Execute]", err)
		}
		return
	}

	poolId, err := strconv.ParseInt(poolIdStr, 10, 64)
	if err != nil {
		log.Error("[OpenPoolInfoCommand.Execute]", err)
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			"❌ Что-то пошло не так, попробуйте снова",
		); err != nil {
			log.Error("[OpenPoolInfoCommand.Execute]", err)
		}
		return
	}

	pool, err := c.ps.GetId(uint64(poolId))
	if err != nil {
		log.Error("[OpenPoolInfoCommand.Execute]", err)
		if _, err := util.SendTextMessage(
			c.b,
			uint64(chatId),
			"❌ Не смог найти выбранный пул. Возможно он был удален. Выберите другой",
		); err != nil {
			log.Error("[OpenPoolInfoCommand.Execute]", err)
		}
		return
	}

	poolInfo := c.info(pool)
	dataBtn := fmt.Sprintf("%v:%v", buttons.CreateStakeId, poolId)
	btn := util.CreateDefaultButton(dataBtn, buttons.StakePoolTokensText)
	var markup *models.InlineKeyboardMarkup

	if pool.OwnerId == uint64(user.Id.Int64) {
		//TODO сделать меню для владельца пула
		markup = util.CreateInlineMarup(1, util.CreateDefaultButton("1", "Test"))
	} else {
		markup = util.MenuWithBackButton(buttons.BackPoolListId, buttons.BackPoolList, btn)
	}
	if err := util.EditTextMessageMarkup(
		ctx,
		c.b,
		uint64(chatId),
		msg.ID,
		poolInfo,
		markup,
	); err != nil {
		log.Error("[OpenPoolInfoCommand.Execute]", err)
		return
	}

}

func (c *OpenPoolInfoCommand) info(p *appMoels.Pool) string {
	allStakesPool := c.ss.GetPoolStakes(uint64(p.Id.Int64))
	var sumAmount float64

	if allStakesPool != nil {
		for _, stake := range *allStakesPool {
			sumAmount += stake.Amount
		}
	}

	foramter := message.NewPrinter(language.English)
	ut := foramter.Sprintf("%.2f", sumAmount)
	reserve := foramter.Sprintf("%.2f", p.Reserve)

	i := `
<b> Описание пула: </b>

<b>📈 Доходность: </b>
%v%% в день начисляется на ваш застейканый баланс.

<b>⏳Срок холда:</b>
%v %v без возможности досрочного вывода

<b>🛡️ Страховка:</b>
Если цена токена упадёт более чем на %v%% за время холда — вам будет выплачена компенсация.

<b>💸 Максимальная компенсация:</b>
До %v%% от вашей стейкнутой суммы.

🔒 Резерв пула:
 •	Заблокировано участниками: %v токенов
 •	Доступно для новых стейков: %v токенов
`

	res := fmt.Sprintf(i, p.Reward, p.Period, util.SuffixDay(int(p.Period)), p.InsuranceCoating, p.MaxCompensationPercent, ut, reserve)
	return res
}
