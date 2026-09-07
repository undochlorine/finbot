package text

const (
	CmdDescStart   = "welcome"
	CmdDescHelp    = "this list"
	CmdDescNewBank = "create a bank"
	CmdDescAdd     = "add money to a bank"
	CmdDescSpend   = "subtract money from a bank"
	CmdDescSet     = "set a bank's balance"
	CmdDescDelete  = "delete a bank"
	CmdDescBank    = "show one bank"
	CmdDescToggle  = "include or exclude a bank from the total"
	CmdDescBanks   = "list all banks"
	CmdDescTotal   = "sum of banks included in the total"
	CmdDescAll     = "list banks and the total"
	CmdDescCancel  = "cancel the current step"
)

const Start = `Welcome to Finbot. Split money into named banks and track balances.
Some banks count toward your total; some do not.

Send /help to see all commands. You can type a bank name after a command
(names may contain spaces), for example /newbank Holiday.`

const Help = `Finbot commands:

/start - welcome
/help - this list
/newbank - create a bank
/add - add money to a bank
/spend - subtract money from a bank
/set - set a bank's balance
/delete - delete a bank
/bank - show one bank
/toggle - include or exclude a bank from the total
/banks - list all banks
/total - sum of banks included in the total
/all - list banks and the total
/cancel - cancel the current step

You can skip prompts by typing details after a command. Bank names may contain spaces.
Example: /newbank Holiday fund
After the name is accepted, the bot asks whether the bank counts in your total.
Money shortcuts: /add Holiday 100, /spend Gifts 12.50, /set Live 0.
Delete still asks you to confirm: /delete Holiday.
Show one bank: /bank Holiday.
Toggle whether a bank counts in the total: /toggle Holiday.`

const NewBankAskName = "What should this bank be called?"

const NewBankAskInclude = "Count this bank in your total?"

const Yes = "Yes"

const No = "No"

const InvalidBankName = "Bank name can't be empty. Send /newbank to try again."

const SomethingWentWrong = "Something went wrong. Try again."

func BankCreated(name string, included bool) string {
	if included {
		return "Created bank \"" + name + "\". It counts toward your total."
	}
	return "Created bank \"" + name + "\". It does not count toward your total."
}

func BankNameTaken(name string) string {
	return "You already have a bank named \"" + name + "\". Choose a different name."
}

const NoBanks = "You have no banks yet. Create one with /newbank."

const AskBank = "Which bank?"

const InvalidAmount = "That amount isn't valid. Send a number like 100 or 12.50."

func UnknownBank(name string) string {
	return "I don't know a bank named \"" + name + "\". Send /banks to see your list."
}

func AskAddAmount(name string) string {
	return "How much should I add to \"" + name + "\"?"
}

func AskSpendAmount(name string) string {
	return "How much should I spend from \"" + name + "\"?"
}

func AskSetAmount(name string) string {
	return "What should \"" + name + "\" be set to?"
}

func Added(name, amount, balance string) string {
	return "Added " + amount + " to \"" + name + "\". Balance is " + balance + "."
}

func Spent(name, amount, balance string) string {
	return "Spent " + amount + " from \"" + name + "\". Balance is " + balance + "."
}

func SetTo(name, balance string) string {
	return "Set \"" + name + "\" to " + balance + "."
}

func AskDeleteConfirm(name string) string {
	return "Delete \"" + name + "\"? This cannot be undone."
}

func BankDeleted(name string) string {
	return "Deleted bank \"" + name + "\"."
}

const DeleteCancelled = "Okay, I didn't delete anything."

const Canceled = "Canceled."

const NothingToCancel = "Nothing to cancel."

const FlowExpired = "This step expired. Start over with a command."

func BankCard(name, balance string, included bool) string {
	if included {
		return name + ": " + balance + " (in total)"
	}
	return name + ": " + balance + " (not in total)"
}

func Total(amount string) string {
	return "Total: " + amount
}

func All(banks, total string) string {
	return banks + "\n\n" + total
}

func Toggled(name, balance string, included bool) string {
	if included {
		return "\"" + name + "\" now counts toward your total. Balance is " + balance + "."
	}
	return "\"" + name + "\" now does not count toward your total. Balance is " + balance + "."
}
