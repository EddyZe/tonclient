package command

import (
	"context"
	"tonclient/internal/tonbot/buttons"
	"tonclient/internal/util"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type Info struct {
	b *bot.Bot
}

func NewInfoCommand(b *bot.Bot) *Info {
	return &Info{b: b}
}

func (c *Info) Execute(ctx context.Context, msg *models.Message) {
	chatId := msg.Chat.ID

	btn1 := util.CreateDefaultButton(buttons.RoleButtonUserId, buttons.RoleButtonUserText)
	btn2 := util.CreateDefaultButton(buttons.RoleButtonOwnerTokensId, buttons.RoleButtonOwnerTokensText)

	markup := util.CreateInlineMarup(2, btn1, btn2)

	if _, err := util.SendTextMessageMarkup(
		c.b,
		uint64(chatId),
		c.generateInfo(),
		markup,
	); err != nil {
		log.Infoln(err)
	}
}

func (c *Info) generateInfo() string {
	return `
<b>Что такое стейкинг?</b>
Стейкинг — это процесс блокировки ваших криптоактивов на определенный срок. 
Взамен вы получаете вознаграждение — это похоже на процент по вкладу в банке, но с криптовалютной доходностью.

<b>Как это работает в NESTRAH?</b>
Вы выбираете <b>пул</b>
— Замораживаете токены на срок (<b>7/30/60/365 дней</b>).

<b>Активируется защита</b>
— Если цена токена упадет ниже установленного порога (например, -10%), вы получите компенсацию.

🔒 <b>Компенсация при падении цены:</b>
Если токен потеряет в стоимости — мы компенсируем <b>до 90%</b> от вашей замороженной суммы.

<b>Автоматические выплаты:</b>
Все расчеты выполняют смарт-контракты TON. <b>Никакого ручного вмешательства!</b>

💎 <b>Надежность</b>
<b>Резервы пула:</b>
Токены для выплат хранятся на защищенных кошельках. Вы всегда можете проверить резерв.

⚡️ <b>Прозрачность</b>
Цена токена фиксируется при старте стейкинга и проверяется через <b>независимые ораклы</b> (Ston.fi, TonAPI).

<b>Почему это выгодно?</b>
<b>Пассивный доход + гарантия компенсации</b> = уверенность даже при падении цены.

💡 <b>Пример:</b>
Вы застейкали <b>1000 токенов</b> по цене <b>1 TON</b> за штуку.
Через 30 дней получили <b>300 токенов</b> награды (1% в день).

Цена упала до <b>0.7 TON (-30%)</b>, но вы получили компенсацию <b>300 токенов</b> (30% от стейка).
<b>Итог: 1300 токенов вместо возможных 700!</b>

🚀 <b>Готовы начать?</b>
Выберите пул, в нем выберите условия — ваши токены будут работать даже в медвежьем рынке!

➖➖➖➖➖➖➖➖➖
❓ <b>Остались вопросы?</b>
Задайте их в поддержке: @NestrahDev
`
}
