package tests

import (
	"fmt"
	"testing"
	"tonclient/internal/util"
)

func TestAddBonus(t *testing.T) {
	num := 2.
	res := util.RemoveZeroFloat(num)

	fmt.Printf("%f\n", num)
	fmt.Println(res)
}
