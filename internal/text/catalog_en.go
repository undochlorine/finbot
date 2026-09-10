package text

// Command menu descriptions stay English. Telegram setMyCommands is a single bot-wide list.
const (
	CmdDescStart    = "welcome"
	CmdDescHelp     = "this list"
	CmdDescLanguage = "change language"
	CmdDescNewBank  = "create a bank"
	CmdDescAdd      = "add money to a bank"
	CmdDescSpend    = "subtract money from a bank"
	CmdDescSet      = "set a bank's balance"
	CmdDescDelete   = "delete a bank"
	CmdDescBank     = "show one bank"
	CmdDescToggle   = "include or exclude a bank from the total"
	CmdDescRename   = "rename a bank"
	CmdDescTransfer = "move money between banks"
	CmdDescBanks    = "list all banks"
	CmdDescTotal    = "sum of banks included in the total"
	CmdDescAll      = "list banks and the total"
	CmdDescCancel   = "cancel the current step"
	CmdDescFeedback = "send feedback to the admin"
)

var en = Catalog{
	Start: `👋 Welcome to Finbot. Split money into named banks and track balances.
Some banks count toward your total; some do not.

Send /help to see all commands. You can type a bank name after a command
(names may contain spaces), for example /newbank Holiday, /rename Holiday, or /transfer Holiday Gifts 50.`,

	Help: `Finbot commands:

/start - welcome
/help - this list
/language - change language
/newbank - create a bank
/add - add money to a bank
/spend - subtract money from a bank
/set - set a bank's balance
/delete - delete a bank
/bank - show one bank
/toggle - include or exclude a bank from the total
/rename - rename a bank
/transfer - move money between banks
/banks - list all banks
/total - sum of banks included in the total
/all - list banks and the total
/cancel - cancel the current step
/feedback - send feedback to the admin

You can skip prompts by typing details after a command. Bank names may contain spaces.
Example: /newbank Holiday fund
After the name is accepted, the bot asks whether the bank counts in your total.
Money shortcuts: /add Holiday 100, /spend Gifts 12.50, /set Live 0.
Delete still asks you to confirm: /delete Holiday.
Show one bank: /bank Holiday.
Toggle whether a bank counts in the total: /toggle Holiday.
Rename a bank: /rename Holiday, or /rename Holiday Trips.
Transfer between banks: /transfer Holiday Gifts 50.`,

	NewBankAskName:      "What should this bank be called?",
	NewBankAskInclude:   "Count this bank in your total?",
	Yes:                 "Yes",
	No:                  "No",
	InvalidBankName:     "Bank name can't be empty. Send /newbank to try again.",
	SomethingWentWrong:  "Something went wrong. Try again.",
	NoBanks:             "You have no banks yet. Create one with /newbank.",
	AskBank:             "Which bank?",
	InvalidAmount:       "That amount isn't valid. Send a number like 100 or 12.50.",
	DeleteCancelled:     "Okay, I didn't delete anything.",
	Canceled:            "Canceled.",
	NothingToCancel:     "Nothing to cancel.",
	FlowExpired:         "This step expired. Start over with a command.",
	FeedbackAsk:         "What's your feedback? Send it as a message.",
	FeedbackThanks:      "🙏 Thanks, I sent that to the admin.",
	FeedbackUnavailable: "Feedback is not available right now.",
	NeedTwoBanks:        "You need at least two banks to transfer. Create another with /newbank.",
	AskTransferFrom:     "Transfer from which bank?",
	SameBank:            "Choose a different bank to transfer to.",
	CurrencyMismatch:    "Those banks use different currencies. Transfers must be the same currency.",

	BankCreatedIncluded:  `✅ Created bank "%s". It counts toward your total.`,
	BankCreatedExcluded:  `✅ Created bank "%s". It does not count toward your total.`,
	BankNameTakenFmt:     `You already have a bank named "%s". Choose a different name.`,
	UnknownBankFmt:       `I don't know a bank named "%s". Send /banks to see your list.`,
	AskAddAmountFmt:      `How much should I add to "%s"?`,
	AskSpendAmountFmt:    `How much should I spend from "%s"?`,
	AskSetAmountFmt:      `What should "%s" be set to?`,
	AddedFmt:             `💸 Added %s to "%s". Balance is %s.`,
	SpentFmt:             `💸 Spent %s from "%s". Balance is %s.`,
	SetToFmt:             `✅ Set "%s" to %s.`,
	AskDeleteConfirmFmt:  `Delete "%s"? This cannot be undone.`,
	BankDeletedFmt:       `🗑️ Deleted bank "%s".`,
	BankCardIncludedFmt:  `%s: %s (in total)`,
	BankCardExcludedFmt:  `%s: %s (not in total)`,
	TotalFmt:             `Total: %s`,
	ToggledIncludedFmt:   `✅ "%s" now counts toward your total. Balance is %s.`,
	ToggledExcludedFmt:   `✅ "%s" now does not count toward your total. Balance is %s.`,
	AskRenameNameFmt:     `What should "%s" be called?`,
	BankRenamedFmt:       `✏️ Renamed "%s" to "%s".`,
	AskTransferToFmt:     `Transfer from "%s" to which bank?`,
	AskTransferAmountFmt: `How much should I transfer from "%s" to "%s"?`,
	TransferredFmt:       "💸 Transferred %s from \"%s\" to \"%s\".\n%s: %s.\n%s: %s.",
	LanguageSetFmt:       "Language set to %s.",
	LanguageNameEN:       "English",
	LanguageNameRU:       "Russian",
	LanguageNameUK:       "Ukrainian",
	LanguageNameMD:       "Moldavian",
}
