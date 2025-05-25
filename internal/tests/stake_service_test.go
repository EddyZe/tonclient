package tests

import (
	"context"
	"log"
	"testing"
	"time"
	"tonclient/internal/models"
	"tonclient/internal/repositories"
	"tonclient/internal/schedulers"
	"tonclient/internal/services"
)

func TestAddBonus(t *testing.T) {
	db, err := InitDBDefault()
	if err != nil {
		t.Fatal(err)
	}
	pr := repositories.NewPoolRepository(db.Db)
	ur := repositories.NewUserRepository(db.Db)
	sr := repositories.NewStakeRepository(db.Db)
	us := services.NewUserService(ur)
	ps := services.NewPoolService(pr, us)
	ss := services.NewStakeService(sr, us, ps)
	ch := make(chan *models.NotificationStake)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	f := schedulers.AddStakeBonusActiveStakes(ss, ps, ch)
	go f()
	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				continue
			}
			log.Println(msg)
		case <-ctx.Done():
			return
		}
	}
}
