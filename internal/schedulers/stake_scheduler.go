package schedulers

import (
	"fmt"
	"strconv"
	"time"
	"tonclient/internal/models"
	"tonclient/internal/services"
	"tonclient/internal/tonfi"
)

func AddStakeBonusActiveStakes(s *services.StakeService, ps *services.PoolService, closedStake chan *models.NotificationStake) func() {
	return func() {
		stakes := s.GetAllIsStatus(true)
		for _, stake := range *stakes {
			pool, err := ps.GetId(stake.PoolId)
			if err != nil {
				continue
			}
			period := time.Duration(pool.Period) * (time.Hour * 24)
			if stake.StartDate.Add(period).Before(time.Now()) {
				if stake.IsActive {
					jettonData, err := tonfi.GetAssetByAddr(pool.JettonMaster)
					if err != nil {
						continue
					}
					stake.IsActive = false
					currentPrice, err := strconv.ParseFloat(jettonData.DexPriceUsd, 64)
					if err != nil {
						currentPrice = 0.
					}
					stake.JettonPriceClosed = currentPrice
					err = s.Update(&stake)
					if err != nil {
						continue
					}
					if closedStake != nil {
						profit := stake.Balance - stake.Amount
						closedStake <- &models.NotificationStake{
							Stake: &stake,
							Msg: fmt.Sprintf("✅ Стейк с токеном %v был закрыт.\n\n Заработано: %v %v.\n Общий баланс: %v %v\n Теперь вы можете вывести токены или возместить ущерб, если он был получен.",
								jettonData.DisplayName,
								profit,
								jettonData.DisplayName,
								stake.Balance,
								jettonData.DisplayName,
							),
						}
					}
				}
				continue
			}
			bonusPrecient := pool.Reward / 100
			amountBonus := stake.Amount * float64(bonusPrecient)
			stake.Balance += amountBonus
			if err := s.Update(&stake); err != nil {
				continue
			}
		}
	}
}
