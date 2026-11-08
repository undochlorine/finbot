package text

import (
	"fmt"
	"strings"

	"finbot/internal/domain"
)

type Catalog struct {
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
	NeedTwoBanks        string
	AskTransferFrom     string
	SameBank            string
	CurrencyMismatch    string
	AskCurrency         string
	InvalidCurrency     string

	BankCreatedIncluded   string
	BankCreatedExcluded   string
	BankNameTakenFmt      string
	UnknownBankFmt        string
	AskAddAmountFmt       string
	AskSpendAmountFmt     string
	AskSetAmountFmt       string
	AddedFmt              string
	SpentFmt              string
	SetToFmt              string
	AskDeleteConfirmFmt   string
	BankDeletedFmt        string
	BankCardIncludedFmt   string
	BankCardExcludedFmt   string
	TotalFmt              string
	ToggledIncludedFmt    string
	ToggledExcludedFmt    string
	AskRenameNameFmt      string
	BankRenamedFmt        string
	AskTransferToFmt      string
	AskTransferAmountFmt  string
	TransferredFmt        string
	CurrencyAlreadyFmt    string
	AskCurrencyConfirmFmt string
	CurrencyKeptFmt       string
	CurrencyChangedFmt    string
	LanguageSetFmt        string
	LanguageNameEN        string
	LanguageNameRU        string
	LanguageNameUK        string
	LanguageNameMD        string
}

type LanguageOption struct {
	Code  string
	Label string
}

// AskLanguage is shown before a locale is known, so it is not catalog-keyed.
const AskLanguage = `Choose your language
Выберите язык
Оберіть мову
Алеӂець лимба`

func LanguageOptions() []LanguageOption {
	return []LanguageOption{
		{Code: domain.LocaleEN, Label: "English"},
		{Code: domain.LocaleRU, Label: "Русский"},
		{Code: domain.LocaleUK, Label: "Українська"},
		{Code: domain.LocaleMD, Label: "Moldovenească"},
	}
}

func (c Catalog) BankCreated(name, unit string, included bool) string {
	if included {
		return fmt.Sprintf(c.BankCreatedIncluded, name, unit)
	}
	return fmt.Sprintf(c.BankCreatedExcluded, name, unit)
}

func (c Catalog) BankNameTaken(name string) string {
	return fmt.Sprintf(c.BankNameTakenFmt, name)
}

func (c Catalog) UnknownBank(name string) string {
	return fmt.Sprintf(c.UnknownBankFmt, name)
}

func (c Catalog) AskAddAmount(name, unit string) string {
	return fmt.Sprintf(c.AskAddAmountFmt, name, unit)
}

func (c Catalog) AskSpendAmount(name, unit string) string {
	return fmt.Sprintf(c.AskSpendAmountFmt, name, unit)
}

func (c Catalog) AskSetAmount(name, unit string) string {
	return fmt.Sprintf(c.AskSetAmountFmt, name, unit)
}

func (c Catalog) Added(name, amount, balance string) string {
	return fmt.Sprintf(c.AddedFmt, amount, name, balance)
}

func (c Catalog) Spent(name, amount, balance string) string {
	return fmt.Sprintf(c.SpentFmt, amount, name, balance)
}

func (c Catalog) SetTo(name, balance string) string {
	return fmt.Sprintf(c.SetToFmt, name, balance)
}

func (c Catalog) AskDeleteConfirm(name string) string {
	return fmt.Sprintf(c.AskDeleteConfirmFmt, name)
}

func (c Catalog) BankDeleted(name string) string {
	return fmt.Sprintf(c.BankDeletedFmt, name)
}

func (c Catalog) FeedbackForward(userID int64, username, body string) string {
	if username == "" {
		return fmt.Sprintf("Feedback from %d:\n\n%s", userID, body)
	}
	return fmt.Sprintf("Feedback from %d (@%s):\n\n%s", userID, username, body)
}

func (c Catalog) BankCard(name, balance string, included bool) string {
	if included {
		return fmt.Sprintf(c.BankCardIncludedFmt, name, balance)
	}
	return fmt.Sprintf(c.BankCardExcludedFmt, name, balance)
}

func (c Catalog) Total(lines []string) string {
	return c.TotalFmt + "\n" + strings.Join(lines, "\n")
}

func (c Catalog) All(banks, total string) string {
	return banks + "\n\n" + total
}

func (c Catalog) Toggled(name, balance string, included bool) string {
	if included {
		return fmt.Sprintf(c.ToggledIncludedFmt, name, balance)
	}
	return fmt.Sprintf(c.ToggledExcludedFmt, name, balance)
}

func (c Catalog) AskRenameName(name string) string {
	return fmt.Sprintf(c.AskRenameNameFmt, name)
}

func (c Catalog) BankRenamed(oldName, newName string) string {
	return fmt.Sprintf(c.BankRenamedFmt, oldName, newName)
}

func (c Catalog) AskTransferTo(fromName string) string {
	return fmt.Sprintf(c.AskTransferToFmt, fromName)
}

func (c Catalog) AskTransferAmount(fromName, toName, unit string) string {
	return fmt.Sprintf(c.AskTransferAmountFmt, fromName, toName, unit)
}

func (c Catalog) Transferred(fromName, toName, amount, fromBalance, toBalance string) string {
	return fmt.Sprintf(c.TransferredFmt, amount, fromName, toName, fromName, fromBalance, toName, toBalance)
}

func (c Catalog) CurrencyAlready(name, unit string) string {
	return fmt.Sprintf(c.CurrencyAlreadyFmt, name, unit)
}

func (c Catalog) AskCurrencyConfirm(name, current, next string) string {
	return fmt.Sprintf(c.AskCurrencyConfirmFmt, name, current, next)
}

func (c Catalog) CurrencyKept(unit string) string {
	return fmt.Sprintf(c.CurrencyKeptFmt, unit)
}

func (c Catalog) CurrencyChanged(name, unit, balance string) string {
	return fmt.Sprintf(c.CurrencyChangedFmt, name, unit, balance)
}

func (c Catalog) LanguageSetTo(locale string) string {
	return fmt.Sprintf(c.LanguageSetFmt, c.languageName(locale))
}

func (c Catalog) languageName(locale string) string {
	switch locale {
	case domain.LocaleRU:
		return c.LanguageNameRU
	case domain.LocaleUK:
		return c.LanguageNameUK
	case domain.LocaleMD:
		return c.LanguageNameMD
	default:
		return c.LanguageNameEN
	}
}

var catalogs = map[string]Catalog{
	domain.LocaleEN: en,
	domain.LocaleRU: ru,
	domain.LocaleUK: uk,
	domain.LocaleMD: md,
}

func For(locale string) Catalog {
	if c, ok := catalogs[locale]; ok {
		return c
	}
	return en
}

// English catalog aliases. Handlers use For(locale); tests keep these names.
var (
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
	NeedTwoBanks        = en.NeedTwoBanks
	AskTransferFrom     = en.AskTransferFrom
	SameBank            = en.SameBank
	CurrencyMismatch    = en.CurrencyMismatch
	AskCurrency         = en.AskCurrency
	InvalidCurrency     = en.InvalidCurrency
)

func BankCreated(name, unit string, included bool) string {
	return en.BankCreated(name, unit, included)
}

func BankNameTaken(name string) string {
	return en.BankNameTaken(name)
}

func UnknownBank(name string) string {
	return en.UnknownBank(name)
}

func AskAddAmount(name, unit string) string {
	return en.AskAddAmount(name, unit)
}

func AskSpendAmount(name, unit string) string {
	return en.AskSpendAmount(name, unit)
}

func AskSetAmount(name, unit string) string {
	return en.AskSetAmount(name, unit)
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

func Total(lines []string) string {
	return en.Total(lines)
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

func AskTransferTo(fromName string) string {
	return en.AskTransferTo(fromName)
}

func AskTransferAmount(fromName, toName, unit string) string {
	return en.AskTransferAmount(fromName, toName, unit)
}

func Transferred(fromName, toName, amount, fromBalance, toBalance string) string {
	return en.Transferred(fromName, toName, amount, fromBalance, toBalance)
}

func LanguageSetTo(locale string) string {
	return en.LanguageSetTo(locale)
}

func CurrencyAlready(name, unit string) string {
	return en.CurrencyAlready(name, unit)
}

func AskCurrencyConfirm(name, current, next string) string {
	return en.AskCurrencyConfirm(name, current, next)
}

func CurrencyKept(unit string) string {
	return en.CurrencyKept(unit)
}

func CurrencyChanged(name, unit, balance string) string {
	return en.CurrencyChanged(name, unit, balance)
}
