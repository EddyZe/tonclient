package util

import (
	"encoding/base64"
	"strconv"
)

func GenerateReferralTelegramCode(userChatId string) string {
	return base64.StdEncoding.EncodeToString([]byte(userChatId))
}

func DecodeReferralTelegramCode(code string) (int64, error) {
	res, err := base64.StdEncoding.DecodeString(code)
	if err != nil {
		return 0, err
	}

	id, err := strconv.ParseInt(string(res), 10, 64)
	if err != nil {
		return 0, err
	}

	return id, nil
}
