package util

import (
	"fmt"
	appModels "tonclient/internal/models"
	"tonclient/internal/services"
	"tonclient/internal/tonbot/buttons"

	"github.com/go-telegram/bot/models"
)

func generateNamePool(pool *appModels.Pool, aws *services.AdminWalletService) string {
	jettonData, err := aws.DataJetton(pool.JettonMaster)
	if err != nil {
		return "Без названия"
	}
	return fmt.Sprintf("%v (%d %v / %d%% / резерв %v)", jettonData.Name, pool.Period, SuffixDay(int(pool.Period)), pool.Reward, pool.Reserve)
}

func GeneratePoolButtons(pool *[]appModels.Pool, aws *services.AdminWalletService) []models.InlineKeyboardButton {
	res := make([]models.InlineKeyboardButton, 0, len(*pool))
	for _, p := range *pool {
		if !p.Id.Valid {
			continue
		}
		poolId := p.Id.Int64
		res = append(
			res,
			CreateDefaultButton(
				fmt.Sprintf("%v:%d", buttons.PoolDataButton, poolId),
				generateNamePool(&p, aws),
			),
		)
	}
	return res
}
