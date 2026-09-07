package telegram

import (
	"fmt"
	"time"

	"finbot/internal/domain"
)

const (
	fsmTTL = 10 * time.Minute

	flowNewBank = "newbank"
	stepName    = "name"
	stepInclude = "include"

	callbackNewBankPrefix     = "v1:newbank:"
	callbackNewBankIncludeYes = "v1:newbank:include:1"
	callbackNewBankIncludeNo  = "v1:newbank:include:0"
)

type fsmState struct {
	Flow string `json:"flow"`
	Step string `json:"step"`
	Name string `json:"name,omitempty"`
}

func fsmKey(userID domain.UserID) string {
	return fmt.Sprintf("fsm:%d", userID)
}
