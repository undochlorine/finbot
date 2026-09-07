package telegram

import (
	"fmt"
	"time"

	"finbot/internal/domain"
)

const (
	fsmTTL = 10 * time.Minute

	flowNewBank = "newbank"
	flowAdd     = "add"
	flowSpend   = "spend"
	flowSet     = "set"

	stepName    = "name"
	stepInclude = "include"
	stepBank    = "bank"
	stepAmount  = "amount"

	callbackNewBankPrefix     = "v1:newbank:"
	callbackNewBankIncludeYes = "v1:newbank:include:1"
	callbackNewBankIncludeNo  = "v1:newbank:include:0"

	callbackAddPrefix   = "v1:add:"
	callbackSpendPrefix = "v1:spend:"
	callbackSetPrefix   = "v1:set:"
)

type fsmState struct {
	Flow   string `json:"flow"`
	Step   string `json:"step"`
	Name   string `json:"name,omitempty"`
	BankID int64  `json:"bank_id,omitempty"`
}

func fsmKey(userID domain.UserID) string {
	return fmt.Sprintf("fsm:%d", userID)
}
