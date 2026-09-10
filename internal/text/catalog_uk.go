package text

var uk = Catalog{
	Start: `👋 Вітаємо у Finbot. Розподіляйте гроші по банках із назвами та стежте за балансами.
Деякі банки входять до загального підсумку, деякі — ні.

Надішліть /help, щоб побачити всі команди. Назву банку можна вказати одразу після команди
(у назві можуть бути пробіли), наприклад /newbank Holiday, /rename Holiday або /transfer Holiday Gifts 50.`,

	Help: `Команди Finbot:

/start - вітання
/help - цей список
/language - змінити мову
/newbank - створити банк
/add - додати гроші до банку
/spend - списати гроші з банку
/set - задати баланс банку
/delete - видалити банк
/bank - показати один банк
/toggle - враховувати або не враховувати банк у підсумку
/rename - перейменувати банк
/transfer - переказати гроші між банками
/banks - список усіх банків
/total - сума банків, які входять до підсумку
/all - список банків і підсумок
/cancel - скасувати поточний крок
/feedback - надіслати відгук адміністратору

Підказки можна пропускати, вказуючи деталі одразу після команди. У назвах банків можуть бути пробіли.
Приклад: /newbank Holiday fund
Після того як назву прийнято, бот запитає, чи враховувати банк у підсумку.
Команди із сумою: /add Holiday 100, /spend Gifts 12.50, /set Live 0.
Видалення все одно потребує підтвердження: /delete Holiday.
Показати один банк: /bank Holiday.
Враховувати або не враховувати банк у підсумку: /toggle Holiday.
Перейменувати банк: /rename Holiday або /rename Holiday Trips.
Переказати між банками: /transfer Holiday Gifts 50.`,

	NewBankAskName:      "Як назвати цей банк?",
	NewBankAskInclude:   "Враховувати цей банк у підсумку?",
	Yes:                 "Так",
	No:                  "Ні",
	InvalidBankName:     "Назва банку не може бути порожньою. Надішліть /newbank, щоб спробувати знову.",
	SomethingWentWrong:  "Щось пішло не так. Спробуйте ще раз.",
	NoBanks:             "У вас ще немає банків. Створіть банк командою /newbank.",
	AskBank:             "Який банк?",
	InvalidAmount:       "Ця сума не підходить. Надішліть число, наприклад 100 або 12.50.",
	DeleteCancelled:     "Гаразд, нічого не видалено.",
	Canceled:            "Скасовано.",
	NothingToCancel:     "Немає чого скасовувати.",
	FlowExpired:         "Цей крок минув. Почніть знову з команди.",
	FeedbackAsk:         "Який у вас відгук? Надішліть його повідомленням.",
	FeedbackThanks:      "🙏 Дякую, відгук надіслано адміністратору.",
	FeedbackUnavailable: "Зараз відгуки недоступні.",
	NeedTwoBanks:        "Для переказу потрібно щонайменше два банки. Створіть ще один командою /newbank.",
	AskTransferFrom:     "З якого банку переказати?",
	SameBank:            "Виберіть інший банк для переказу.",
	CurrencyMismatch:    "У цих банків різні валюти. Переказувати можна лише в одній валюті.",

	BankCreatedIncluded:  `✅ Створено банк «%s». Він враховується в підсумку.`,
	BankCreatedExcluded:  `✅ Створено банк «%s». Він не враховується в підсумку.`,
	BankNameTakenFmt:     `У вас уже є банк із назвою «%s». Виберіть іншу назву.`,
	UnknownBankFmt:       `Я не знаю банк із назвою «%s». Надішліть /banks, щоб побачити список.`,
	AskAddAmountFmt:      `Скільки додати до «%s»?`,
	AskSpendAmountFmt:    `Скільки списати з «%s»?`,
	AskSetAmountFmt:      `Який баланс встановити для «%s»?`,
	AddedFmt:             `💸 Додано %s до «%s». Баланс: %s.`,
	SpentFmt:             `💸 Списано %s з «%s». Баланс: %s.`,
	SetToFmt:             `✅ Банк «%s» встановлено на %s.`,
	AskDeleteConfirmFmt:  `Видалити «%s»? Це не можна скасувати.`,
	BankDeletedFmt:       `🗑️ Банк «%s» видалено.`,
	BankCardIncludedFmt:  `%s: %s (у підсумку)`,
	BankCardExcludedFmt:  `%s: %s (не в підсумку)`,
	TotalFmt:             `Підсумок: %s`,
	ToggledIncludedFmt:   `✅ «%s» тепер враховується в підсумку. Баланс: %s.`,
	ToggledExcludedFmt:   `✅ «%s» тепер не враховується в підсумку. Баланс: %s.`,
	AskRenameNameFmt:     `Як назвати «%s»?`,
	BankRenamedFmt:       `✏️ Банк «%s» перейменовано на «%s».`,
	AskTransferToFmt:     `Переказати з «%s» до якого банку?`,
	AskTransferAmountFmt: `Скільки переказати з «%s» до «%s»?`,
	TransferredFmt:       "💸 Переказано %s з «%s» до «%s».\n%s: %s.\n%s: %s.",
	LanguageSetFmt:       "Мову змінено на %s.",
	LanguageNameEN:       "англійську",
	LanguageNameRU:       "російську",
	LanguageNameUK:       "українську",
	LanguageNameMD:       "молдавську",
}
