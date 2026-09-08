package text

import (
	"fmt"

	"finbot/internal/domain"
)

type Catalog struct {
	CmdDescStart    string
	CmdDescHelp     string
	CmdDescNewBank  string
	CmdDescAdd      string
	CmdDescSpend    string
	CmdDescSet      string
	CmdDescDelete   string
	CmdDescBank     string
	CmdDescToggle   string
	CmdDescRename   string
	CmdDescBanks    string
	CmdDescTotal    string
	CmdDescAll      string
	CmdDescCancel   string
	CmdDescFeedback string

	Start               string
	Help                string
	NewBankAskName      string
	NewBankAskInclude   string
	Yes                 string
	No                  string
	InvalidBankName     string
	SomethingWentWrong  string
	NoBanks             string
	AskBank             string
	InvalidAmount       string
	DeleteCancelled     string
	Canceled            string
	NothingToCancel     string
	FlowExpired         string
	FeedbackAsk         string
	FeedbackThanks      string
	FeedbackUnavailable string
}

func (c Catalog) BankCreated(name string, included bool) string {
	if included {
		return "✅ Created bank \"" + name + "\". It counts toward your total."
	}
	return "✅ Created bank \"" + name + "\". It does not count toward your total."
}

func (c Catalog) BankNameTaken(name string) string {
	return "You already have a bank named \"" + name + "\". Choose a different name."
}

func (c Catalog) UnknownBank(name string) string {
	return "I don't know a bank named \"" + name + "\". Send /banks to see your list."
}

func (c Catalog) AskAddAmount(name string) string {
	return "How much should I add to \"" + name + "\"?"
}

func (c Catalog) AskSpendAmount(name string) string {
	return "How much should I spend from \"" + name + "\"?"
}

func (c Catalog) AskSetAmount(name string) string {
	return "What should \"" + name + "\" be set to?"
}

func (c Catalog) Added(name, amount, balance string) string {
	return "💸 Added " + amount + " to \"" + name + "\". Balance is " + balance + "."
}

func (c Catalog) Spent(name, amount, balance string) string {
	return "💸 Spent " + amount + " from \"" + name + "\". Balance is " + balance + "."
}

func (c Catalog) SetTo(name, balance string) string {
	return "✅ Set \"" + name + "\" to " + balance + "."
}

func (c Catalog) AskDeleteConfirm(name string) string {
	return "Delete \"" + name + "\"? This cannot be undone."
}

func (c Catalog) BankDeleted(name string) string {
	return "🗑️ Deleted bank \"" + name + "\"."
}

func (c Catalog) FeedbackForward(userID int64, username, body string) string {
	if username == "" {
		return fmt.Sprintf("Feedback from %d:\n\n%s", userID, body)
	}
	return fmt.Sprintf("Feedback from %d (@%s):\n\n%s", userID, username, body)
}

func (c Catalog) BankCard(name, balance string, included bool) string {
	if included {
		return name + ": " + balance + " (in total)"
	}
	return name + ": " + balance + " (not in total)"
}

func (c Catalog) Total(amount string) string {
	return "Total: " + amount
}

func (c Catalog) All(banks, total string) string {
	return banks + "\n\n" + total
}

func (c Catalog) Toggled(name, balance string, included bool) string {
	if included {
		return "✅ \"" + name + "\" now counts toward your total. Balance is " + balance + "."
	}
	return "✅ \"" + name + "\" now does not count toward your total. Balance is " + balance + "."
}

func (c Catalog) AskRenameName(name string) string {
	return "What should \"" + name + "\" be called?"
}

func (c Catalog) BankRenamed(oldName, newName string) string {
	return "✏️ Renamed \"" + oldName + "\" to \"" + newName + "\"."
}

var en = Catalog{
	CmdDescStart:    "welcome",
	CmdDescHelp:     "this list",
	CmdDescNewBank:  "create a bank",
	CmdDescAdd:      "add money to a bank",
	CmdDescSpend:    "subtract money from a bank",
	CmdDescSet:      "set a bank's balance",
	CmdDescDelete:   "delete a bank",
	CmdDescBank:     "show one bank",
	CmdDescToggle:   "include or exclude a bank from the total",
	CmdDescRename:   "rename a bank",
	CmdDescBanks:    "list all banks",
	CmdDescTotal:    "sum of banks included in the total",
	CmdDescAll:      "list banks and the total",
	CmdDescCancel:   "cancel the current step",
	CmdDescFeedback: "send feedback to the admin",

	Start: `👋 Welcome to Finbot. Split money into named banks and track balances.
Some banks count toward your total; some do not.

Send /help to see all commands. You can type a bank name after a command
(names may contain spaces), for example /newbank Holiday or /rename Holiday.`,

	Help: `Finbot commands:

/start - welcome
/help - this list
/newbank - create a bank
/add - add money to a bank
/spend - subtract money from a bank
/set - set a bank's balance
/delete - delete a bank
/bank - show one bank
/toggle - include or exclude a bank from the total
/rename - rename a bank
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
Rename a bank: /rename Holiday, or /rename Holiday Trips.`,

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
}

var catalogs = map[string]Catalog{
	domain.LocaleEN: en,
}

func For(locale string) Catalog {
	if c, ok := catalogs[locale]; ok {
		return c
	}
	return en
}

// English catalog aliases. Handlers use For(locale); tests and boot-time menu keep these names.
var (
	CmdDescStart    = en.CmdDescStart
	CmdDescHelp     = en.CmdDescHelp
	CmdDescNewBank  = en.CmdDescNewBank
	CmdDescAdd      = en.CmdDescAdd
	CmdDescSpend    = en.CmdDescSpend
	CmdDescSet      = en.CmdDescSet
	CmdDescDelete   = en.CmdDescDelete
	CmdDescBank     = en.CmdDescBank
	CmdDescToggle   = en.CmdDescToggle
	CmdDescRename   = en.CmdDescRename
	CmdDescBanks    = en.CmdDescBanks
	CmdDescTotal    = en.CmdDescTotal
	CmdDescAll      = en.CmdDescAll
	CmdDescCancel   = en.CmdDescCancel
	CmdDescFeedback = en.CmdDescFeedback

	Start               = en.Start
	Help                = en.Help
	NewBankAskName      = en.NewBankAskName
	NewBankAskInclude   = en.NewBankAskInclude
	Yes                 = en.Yes
	No                  = en.No
	InvalidBankName     = en.InvalidBankName
	SomethingWentWrong  = en.SomethingWentWrong
	NoBanks             = en.NoBanks
	AskBank             = en.AskBank
	InvalidAmount       = en.InvalidAmount
	DeleteCancelled     = en.DeleteCancelled
	Canceled            = en.Canceled
	NothingToCancel     = en.NothingToCancel
	FlowExpired         = en.FlowExpired
	FeedbackAsk         = en.FeedbackAsk
	FeedbackThanks      = en.FeedbackThanks
	FeedbackUnavailable = en.FeedbackUnavailable
)

func BankCreated(name string, included bool) string {
	return en.BankCreated(name, included)
}

func BankNameTaken(name string) string {
	return en.BankNameTaken(name)
}

func UnknownBank(name string) string {
	return en.UnknownBank(name)
}

func AskAddAmount(name string) string {
	return en.AskAddAmount(name)
}

func AskSpendAmount(name string) string {
	return en.AskSpendAmount(name)
}

func AskSetAmount(name string) string {
	return en.AskSetAmount(name)
}

func Added(name, amount, balance string) string {
	return en.Added(name, amount, balance)
}

func Spent(name, amount, balance string) string {
	return en.Spent(name, amount, balance)
}

func SetTo(name, balance string) string {
	return en.SetTo(name, balance)
}

func AskDeleteConfirm(name string) string {
	return en.AskDeleteConfirm(name)
}

func BankDeleted(name string) string {
	return en.BankDeleted(name)
}

func FeedbackForward(userID int64, username, body string) string {
	return en.FeedbackForward(userID, username, body)
}

func BankCard(name, balance string, included bool) string {
	return en.BankCard(name, balance, included)
}

func Total(amount string) string {
	return en.Total(amount)
}

func All(banks, total string) string {
	return en.All(banks, total)
}

func Toggled(name, balance string, included bool) string {
	return en.Toggled(name, balance, included)
}

func AskRenameName(name string) string {
	return en.AskRenameName(name)
}

func BankRenamed(oldName, newName string) string {
	return en.BankRenamed(oldName, newName)
}
