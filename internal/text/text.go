package text

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

You can skip prompts by typing details after a command. Bank names may contain spaces.
Example: /newbank Holiday fund
After the name is accepted, the bot asks whether the bank counts in your total.`

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
