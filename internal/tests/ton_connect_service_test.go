package tests

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"tonclient/internal/models"
	"tonclient/internal/services"

	"github.com/redis/go-redis/v9"
	"github.com/xssnick/tonutils-go/tvm/cell"
)

func redisInit() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
}

func TestTonConnectService_CreateSession(t *testing.T) {
	rdb := redisInit()
	err := rdb.Ping(context.Background()).Err()
	ss := InitAdminService()
	if err != nil {
		t.Fatal(err)
	}
	tcs := services.NewTonConnectService(rdb, ss)
	s, err := tcs.CreateSession()
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(s.ID)
}

func TestTonConnectService_SaveSession(t *testing.T) {
	rdb := redisInit()

	tcs := services.NewTonConnectService(rdb, InitAdminService())
	s, err := tcs.CreateSession()
	if err != nil {
		t.Fatal(err)
	}

	err = tcs.SaveSession("TEST", s)
	if err != nil {
		t.Fatal(err)
	}
}

func TestTonConnectServiceAndConncect_GenerateConnectUrls(t *testing.T) {
	rdb := redisInit()
	tcs := services.NewTonConnectService(rdb, InitAdminService())
	s, err := tcs.CreateSession()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := tcs.SaveSession("TEST", s)
		if err != nil {
			t.Fatal(err)
		}
	}()

	urls, err := tcs.GenerateConnectUrls(s)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(urls)

	_, err = tcs.Connect(s)
	if err != nil {
		t.Fatal(err)
	}

}

func TestTonConnectService_GetSession(t *testing.T) {
	rdb := redisInit()
	tcs := services.NewTonConnectService(rdb, InitAdminService())
	s, err := tcs.LoadSession("TEST")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(s.ID)
}

func TestTonConnectService_SendTransaction(t *testing.T) {
	rdb := redisInit()
	tcs := services.NewTonConnectService(rdb, InitAdminService())
	s, err := tcs.LoadSession("TEST")
	if err != nil {
		t.Fatal(err)
	}

	if s == nil {
		t.Fatal(errors.New("session not found"))
	}

	p := models.Pool{
		OwnerId:          1,
		Reserve:          1,
		JettonWallet:     "EQAPVdCkLAHYk0RXty5ucMNZhgX-wKe2mLBXp8A6YHm5z_os",
		Reward:           2,
		Period:           30,
		InsuranceCoating: 10,
		IsActive:         false,
		IsCommissionPaid: false,
		JettonMaster:     "EQAJKTfw3qP0OFUba-1l7rtA7_TzXd9Cbm4DjNCaioCdofF_",
	}

	data, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}

	boc, err := tcs.SendJettonTransaction(
		"test",
		p.JettonWallet,
		"UQD6A01mB8tAKJVekRrMjoA3l188LSCF2zrIHoH94tWhZGAO",
		"UQCrciOc9HE341fFtBs-WFuttXeciFDIvFwafCO4QQhAinLG",
		fmt.Sprint(p.Reserve),
		&models.Payload{
			OperationType: models.OP_ADMIN_CREATE_POOL,
			JettonMaster:  p.JettonMaster,
			Payload:       string(data),
		},
		s,
	)
	if err != nil {
		t.Fatal(err)
	}

	fromBOC, err := cell.FromBOC(boc)
	if err != nil {
		return
	}

	fmt.Println(fromBOC)
}
