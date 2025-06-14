package tests

import (
	"context"
	"fmt"
	"log"
	"testing"
	"tonclient/internal/config"
	"tonclient/internal/util"

	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/ton"
)

func TestApi(t *testing.T) {
	//UQD6A01mB8tAKJVekRrMjoA3l188LSCF2zrIHoH94tWhZGAO

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := liteclient.NewConnectionPool()
	if err := client.AddConnectionsFromConfigUrl(ctx, config.CONFIG_TON_MAINNET_URL); err != nil {
		log.Fatalln("Failed to add connections to config server:", err)
	}

	api := ton.NewAPIClient(client)
	adr, err := address.ParseAddr("UQD6A01mB8tAKJVekRrMjoA3l188LSCF2zwrIHoH94tWhZGAO")
	if err != nil {
		log.Fatalln("Failed to parse address:", err)
	}
	master, err := api.GetMasterchainInfo(ctx)
	if err != nil {
		log.Fatalln("Failed to get masterchain info:", err)
	}

	acc, err := api.GetAccount(ctx, master, adr)
	if err != nil {
		log.Fatalln("Failed to get account balance:", err)
	}

	fmt.Println(acc.State.Balance)
}

func TestZerosToK(t *testing.T) {
	num := 30_000_000

	fmt.Println(util.ReplaceThreeZerosToK(int64(num)))
}
