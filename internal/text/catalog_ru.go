package text

var ru = Catalog{
	Start: `👋 Добро пожаловать в Finbot. Распределяйте деньги по банкам с названиями и следите за балансами.
Некоторые банки входят в общий итог, некоторые — нет.

Отправьте /help, чтобы увидеть все команды. Название банка можно указать сразу после команды
(в названии могут быть пробелы), например /newbank Holiday, /rename Holiday или /transfer Holiday Gifts 50.`,

	Help: `Команды Finbot:

/start - приветствие
/help - этот список
/language - сменить язык
/newbank - создать банк
/add - добавить деньги в банк
/spend - списать деньги из банка
/set - задать баланс банка
/delete - удалить банк
/bank - показать один банк
/toggle - включить или исключить банк из итога
/rename - переименовать банк
/transfer - перевести деньги между банками
/banks - список всех банков
/total - сумма банков, входящих в итог
/all - список банков и итог
/cancel - отменить текущий шаг
/feedback - отправить отзыв администратору

Подсказки можно пропускать, указывая детали сразу после команды. В названиях банков могут быть пробелы.
Пример: /newbank Holiday fund
После того как имя принято, бот спросит, учитывать ли банк в итоге.
Команды с суммой: /add Holiday 100, /spend Gifts 12.50, /set Live 0.
Удаление всё равно требует подтверждения: /delete Holiday.
Показать один банк: /bank Holiday.
Включить или исключить банк из итога: /toggle Holiday.
Переименовать банк: /rename Holiday или /rename Holiday Trips.
Перевести между банками: /transfer Holiday Gifts 50.`,

	NewBankAskName:      "Как назвать этот банк?",
	NewBankAskInclude:   "Учитывать этот банк в итоге?",
	Yes:                 "Да",
	No:                  "Нет",
	InvalidBankName:     "Название банка не может быть пустым. Отправьте /newbank, чтобы попробовать снова.",
	SomethingWentWrong:  "Что-то пошло не так. Попробуйте ещё раз.",
	NoBanks:             "У вас пока нет банков. Создайте банк командой /newbank.",
	AskBank:             "Какой банк?",
	InvalidAmount:       "Эта сумма не подходит. Отправьте число, например 100 или 12.50.",
	DeleteCancelled:     "Хорошо, ничего не удалено.",
	Canceled:            "Отменено.",
	NothingToCancel:     "Нечего отменять.",
	FlowExpired:         "Этот шаг истёк. Начните заново с команды.",
	FeedbackAsk:         "Какой у вас отзыв? Отправьте его сообщением.",
	FeedbackThanks:      "🙏 Спасибо, отзыв отправлен администратору.",
	FeedbackUnavailable: "Сейчас отзывы недоступны.",
	NeedTwoBanks:        "Для перевода нужно как минимум два банка. Создайте ещё один командой /newbank.",
	AskTransferFrom:     "С какого банка перевести?",
	SameBank:            "Выберите другой банк для перевода.",
	CurrencyMismatch:    "У этих банков разные валюты. Переводить можно только в одной валюте.",

	BankCreatedIncluded:  `✅ Создан банк «%s». Он учитывается в итоге.`,
	BankCreatedExcluded:  `✅ Создан банк «%s». Он не учитывается в итоге.`,
	BankNameTakenFmt:     `У вас уже есть банк с названием «%s». Выберите другое имя.`,
	UnknownBankFmt:       `Я не знаю банк с названием «%s». Отправьте /banks, чтобы увидеть список.`,
	AskAddAmountFmt:      `Сколько добавить в «%s»?`,
	AskSpendAmountFmt:    `Сколько списать из «%s»?`,
	AskSetAmountFmt:      `Какой баланс установить для «%s»?`,
	AddedFmt:             `💸 Добавлено %s в «%s». Баланс: %s.`,
	SpentFmt:             `💸 Списано %s из «%s». Баланс: %s.`,
	SetToFmt:             `✅ Банк «%s» установлен на %s.`,
	AskDeleteConfirmFmt:  `Удалить «%s»? Это нельзя отменить.`,
	BankDeletedFmt:       `🗑️ Банк «%s» удалён.`,
	BankCardIncludedFmt:  `%s: %s (в итоге)`,
	BankCardExcludedFmt:  `%s: %s (не в итоге)`,
	TotalFmt:             `Итог: %s`,
	ToggledIncludedFmt:   `✅ «%s» теперь учитывается в итоге. Баланс: %s.`,
	ToggledExcludedFmt:   `✅ «%s» теперь не учитывается в итоге. Баланс: %s.`,
	AskRenameNameFmt:     `Как назвать «%s»?`,
	BankRenamedFmt:       `✏️ Банк «%s» переименован в «%s».`,
	AskTransferToFmt:     `Перевести из «%s» в какой банк?`,
	AskTransferAmountFmt: `Сколько перевести из «%s» в «%s»?`,
	TransferredFmt:       "💸 Переведено %s из «%s» в «%s».\n%s: %s.\n%s: %s.",
	LanguageSetFmt:       "Язык изменён на %s.",
	LanguageNameEN:       "английский",
	LanguageNameRU:       "русский",
	LanguageNameUK:       "украинский",
	LanguageNameMD:       "молдавский",
}
