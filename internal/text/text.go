package text

const Start = `Welcome to Finbot. Split money into named banks and track balances.
Some banks count toward your total; some do not.

Send /help to see all commands.`

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
/all - list banks and the total`
