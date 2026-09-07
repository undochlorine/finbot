package telegram

import (
	"fmt"
	"time"

	"finbot/internal/domain"
)

const (
	commandStart  = "start"
	commandHelp   = "help"
	commandCancel = "cancel"

	fsmTTL = 10 * time.Minute

	stepName    = "name"
	stepInclude = "include"
	stepBank    = "bank"
	stepAmount  = "amount"
	stepConfirm = "confirm"

	callbackNewBankPrefix     = "v1:" + domain.CommandNewBank + ":"
	callbackNewBankIncludeYes = callbackNewBankPrefix + stepInclude + ":1"
	callbackNewBankIncludeNo  = callbackNewBankPrefix + stepInclude + ":0"

	callbackAddPrefix    = "v1:" + domain.CommandAdd + ":"
	callbackSpendPrefix  = "v1:" + domain.CommandSpend + ":"
	callbackSetPrefix    = "v1:" + domain.CommandSet + ":"
	callbackDeletePrefix = "v1:" + domain.CommandDelete + ":"
	callbackDeleteYes    = callbackDeletePrefix + domain.Yes + ":"
	callbackDeleteNo     = callbackDeletePrefix + domain.No + ":"
	callbackBankPrefix   = "v1:" + domain.CommandBank + ":"
	callbackTogglePrefix = "v1:" + domain.CommandToggle + ":"
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
