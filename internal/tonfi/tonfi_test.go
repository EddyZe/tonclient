package tonfi

import (
	"fmt"
	"testing"
)

func TestTonfi(t *testing.T) {
	res, err := GetAssetByAddr("EQAJKTfw3qP0OFUba-1l7rtA7_TzXd9Cbm4DjNCaioCdofF_")
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Println(res)
}
