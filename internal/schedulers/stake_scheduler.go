package schedulers

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
	"tonclient/internal/models"
	"tonclient/internal/services"
	"tonclient/internal/tonfi"
	"tonclient/internal/util"

	"github.com/go-telegram/bot"
)

type StakeScheduler struct {
	b           *bot.Bot
	ss          *services.StakeService
	us          *services.UserService
	ps          *services.PoolService
	rs          *services.ReferalService
	aws         *services.AdminWalletService
	ws          *services.WalletTonService
	ts          *services.TelegramService
	closedStake chan *models.NotificationStake
}

func NewStakScheduler(
	b *bot.Bot,
	ss *services.StakeService,
	us *services.UserService,
	ps *services.PoolService,
	rs *services.ReferalService,
	aws *services.AdminWalletService,
	ws *services.WalletTonService,
	ts *services.TelegramService,
	closeStaked chan *models.NotificationStake,
) *StakeScheduler {
	return &StakeScheduler{
		b:           b,
		ss:          ss,
		us:          us,
		ps:          ps,
		rs:          rs,
		aws:         aws,
		ws:          ws,
		closedStake: closeStaked,
		ts:          ts,
	}
}

func (s *StakeScheduler) AddStakeBonusActiveStakes() func() {
	return func() {
		stakes := s.ss.GetAllIsStatus(true)
		for _, stake := range *stakes {
			pool, err := s.ps.GetId(stake.PoolId)
			if err != nil {
				continue
			}
			if stake.EndDate.Before(time.Now()) {
				if stake.IsActive {
					jettonData, err := tonfi.GetAssetByAddr(pool.JettonMaster)
					if err != nil {
						continue
					}
					stake.IsActive = false
					stake.CloseDate = time.Now()
					currentPrice := util.GetCurrentPriceJettonAddr(pool.JettonMaster)
					stake.JettonPriceClosed = currentPrice
					err = s.ss.Update(&stake)
					if err != nil {
						continue
					}
					if s.closedStake != nil {
						profit := stake.Balance - stake.Amount
						s.closedStake <- &models.NotificationStake{
							Stake: &stake,
							Msg: fmt.Sprintf("✅ Стейк с токеном %v был закрыт.\n\n Заработано: %v %v.\n Общий баланс: %v %v\n Теперь вы можете вывести токены или получить компенсацию, если она полагается.",
								jettonData.DisplayName,
								profit,
								jettonData.DisplayName,
								stake.Balance,
								jettonData.DisplayName,
							),
						}
					}
					log.Println("проверка кол-во стейкаов")
					stakesCountUser := s.ss.CountUser(stake.UserId)
					tgStaker, err := s.ts.GetByUserId(stake.UserId)
					if err != nil {
						continue
					}
					if stakesCountUser == 1 {
						u, err := s.us.GetById(stake.UserId)
						if err == nil {
							log.Println("отправка бонуса")
							if u.RefererId.Valid && u.RefererId.Int64 != 0 {
								go func() {
									if err := s.sendBonus(
										s.b,
										uint64(u.RefererId.Int64),
										&stake,
										tgStaker,
									); err != nil {
										log.Println("Failed to send bonus:", err)
										return
									}
								}()
							}
						}
					}
				}
				continue
			}
			bonusPercent := float64(pool.Reward) / 100
			amountBonus := stake.Amount * bonusPercent
			rewardAllTime := amountBonus * float64(pool.Period)
			if stake.Balance < rewardAllTime+stake.Amount {
				stake.Balance += amountBonus
			}
			if err := s.ss.Update(&stake); err != nil {
				continue
			}
		}
	}
}

func (s *StakeScheduler) sendBonus(b *bot.Bot, referalId uint64, stake *models.Stake, tgStaker *models.Telegram) error {
	u, err := s.us.GetByTelegramChatId(referalId)
	if err != nil {
		log.Println("Failed to get user :", err)
		return err
	}
	w, err := s.ws.GetByUserId(uint64(u.Id.Int64))
	if err != nil {
		log.Println("Failed to get user:", err)
		return err
	}
	jettonAdminAddr := os.Getenv("JETTON_CONTRACT_ADMIN_JETTON")
	if jettonAdminAddr == "" {
		return err
	}
	bonus := os.Getenv("REFERAL_BONUS")
	if bonus == "" {
		bonus = "2"
	}
	bonusNum, err := strconv.ParseFloat(bonus, 64)
	if err != nil {
		log.Println("Failed to parse bonus:", err)
		return err
	}
	decimal := os.Getenv("JETTON_DECIMAL")
	if decimal == "" {
		decimal = "9"
	}
	decimalNum, err := strconv.Atoi(decimal)
	if err != nil {
		log.Println("Failed to parse decimal:", err)
		return err
	}
	bonusAmount := stake.Amount * (bonusNum / 100)
	if _, err := s.aws.SendJetton(
		jettonAdminAddr,
		w.Addr,
		"",
		bonusAmount,
		decimalNum,
	); err != nil {
		log.Println("Failed to send bonus:", err)
		return err
	}
	tokenName := os.Getenv("JETTON_NAME_COIN")
	if tokenName == "" {
		tokenName = "NESTRAH"
	}

	if tgStaker != nil {
		if _, err := util.SendTextMessage(
			b,
			referalId,
			fmt.Sprintf(
				"✅ Вы получили бонус %v %v, за пользователя %v. Токены были отправлены на привязанный кошелек",
				util.RemoveZeroFloat(bonusAmount),
				tokenName,
				tgStaker.Username,
			),
		); err != nil {
			log.Print("Failed to send bonus:", err)
			return err
		}
	}

	if err := s.rs.Save(&models.Referral{
		ReferrerUserId: u.Id,
		ReferralUserId: sql.NullInt64{
			Int64: int64(stake.UserId),
			Valid: true,
		},
		FirstStakeId: stake.Id,
		RewardGiven:  true,
		RewardAmount: bonusAmount,
	}); err != nil {
		log.Println("Failed to save referral:", err)
		return err
	}

	return nil
}
