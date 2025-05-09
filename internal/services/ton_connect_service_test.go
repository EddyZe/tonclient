package services

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"testing"

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
	if err != nil {
		t.Fatal(err)
	}
	tcs := NewTonConnectService(rdb)
	s, err := tcs.CreateSession()
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(s.ID)
}

func TestTonConnectService_SaveSession(t *testing.T) {
	rdb := redisInit()
	tcs := NewTonConnectService(rdb)
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
	tcs := NewTonConnectService(rdb)
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

	err = tcs.Connect(s)
	if err != nil {
		t.Fatal(err)
	}

}

func TestTonConnectService_GetSession(t *testing.T) {
	rdb := redisInit()
	tcs := NewTonConnectService(rdb)
	s, err := tcs.LoadSession("TEST")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(s.ID)
}

func TestTonConnectService_SendTransaction(t *testing.T) {
	rdb := redisInit()
	tcs := NewTonConnectService(rdb)
	s, err := tcs.LoadSession("TEST")
	if err != nil {
		t.Fatal(err)
	}

	if s == nil {
		t.Fatal(errors.New("session not found"))
	}

	boc, err := tcs.SendJettonTransaction(
		"EQAPVdCkLAHYk0RXty5ucMNZhgX-wKe2mLBXp8A6YHm5z_os",
		"UQAdpNJR-hZ72cPb70eFuQU3VDx8EcLsOEgm7K0Puh9cHA1d",
		"UQCrciOc9HE341fFtBs-WFuttXeciFDIvFwafCO4QQhAinLG",
		"2",
		base64.StdEncoding.EncodeToString([]byte("dfhdfgbdhfgbdfgdfgdfgdfgdgfdfgdfiugdifghdifghdiyfghidfyghidfyghidyfhgidyfhgidyfhgidyfghifffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffdfhgidfhgidhfgidhfigdhfighdfig")),
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
