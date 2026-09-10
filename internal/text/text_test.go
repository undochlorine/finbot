package text

import (
	"reflect"
	"strings"
	"testing"

	"finbot/internal/domain"
)

func TestForFallsBackToEnglish(t *testing.T) {
	en := For(domain.LocaleEN)
	if en.Start == "" || en.Start != Start {
		t.Fatalf("english catalog: %+v", en.Start)
	}
	got := For("fr")
	if got != en {
		t.Fatal("unknown locale should fall back to en")
	}
	if For("") != en {
		t.Fatal("empty locale should fall back to en")
	}
}

func TestHelpListsMVPCommands(t *testing.T) {
	commands := []struct {
		cmd  string
		desc string
	}{
		{"/start", CmdDescStart},
		{"/help", CmdDescHelp},
		{"/language", CmdDescLanguage},
		{"/newbank", CmdDescNewBank},
		{"/add", CmdDescAdd},
		{"/spend", CmdDescSpend},
		{"/set", CmdDescSet},
		{"/delete", CmdDescDelete},
		{"/bank", CmdDescBank},
		{"/toggle", CmdDescToggle},
		{"/rename", CmdDescRename},
		{"/transfer", CmdDescTransfer},
		{"/banks", CmdDescBanks},
		{"/total", CmdDescTotal},
		{"/all", CmdDescAll},
		{"/cancel", CmdDescCancel},
		{"/feedback", CmdDescFeedback},
	}
	for _, cmd := range commands {
		if !strings.Contains(Help, cmd.cmd+" - "+cmd.desc) {
			t.Errorf("help missing %s - %s", cmd.cmd, cmd.desc)
		}
	}
}

func TestFeedbackForwardOmitsEmptyUsername(t *testing.T) {
	got := FeedbackForward(42, "", "hi")
	if strings.Contains(got, "@") {
		t.Fatalf("unexpected @: %q", got)
	}
}

func TestStartPointsToHelp(t *testing.T) {
	if !strings.Contains(Start, "/help") {
		t.Fatal("start should point to /help")
	}
	if !strings.Contains(Start, "/newbank") {
		t.Fatal("start should mention a command shortcut")
	}
	if !strings.Contains(Start, "/rename") {
		t.Fatal("start should mention /rename")
	}
	if !strings.Contains(Start, "/transfer") {
		t.Fatal("start should mention /transfer")
	}
}

func TestHelpMentionsCommandShortcuts(t *testing.T) {
	if !strings.Contains(Help, "/newbank Holiday fund") {
		t.Fatal("help should show a command-plus-name example")
	}
	if !strings.Contains(Help, "spaces") {
		t.Fatal("help should mention that bank names may contain spaces")
	}
	if !strings.Contains(Help, "/add Holiday 100") {
		t.Fatal("help should show an /add shortcut")
	}
	if !strings.Contains(Help, "/spend Gifts 12.50") {
		t.Fatal("help should show a /spend shortcut")
	}
	if !strings.Contains(Help, "/set Live 0") {
		t.Fatal("help should show a /set shortcut")
	}
	if !strings.Contains(Help, "/delete Holiday") {
		t.Fatal("help should show a /delete shortcut")
	}
	if !strings.Contains(Help, "/bank Holiday") {
		t.Fatal("help should show a /bank shortcut")
	}
	if !strings.Contains(Help, "/toggle Holiday") {
		t.Fatal("help should show a /toggle shortcut")
	}
	if !strings.Contains(Help, "/rename Holiday") {
		t.Fatal("help should show a /rename shortcut")
	}
	if !strings.Contains(Help, "/rename Holiday Trips") {
		t.Fatal("help should show a /rename old-new shortcut")
	}
	if !strings.Contains(Help, "/transfer Holiday Gifts 50") {
		t.Fatal("help should show a /transfer shortcut")
	}
}

func TestOutcomeEmojis(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{
			name: "start first line",
			got:  strings.SplitN(Start, "\n", 2)[0],
			want: "👋 Welcome to Finbot. Split money into named banks and track balances.",
		},
		{
			name: "bank created included",
			got:  BankCreated("Holiday", true),
			want: "✅ Created bank \"Holiday\". It counts toward your total.",
		},
		{
			name: "bank created excluded",
			got:  BankCreated("Gifts", false),
			want: "✅ Created bank \"Gifts\". It does not count toward your total.",
		},
		{
			name: "added",
			got:  Added("Holiday", "100.00", "150.00"),
			want: "💸 Added 100.00 to \"Holiday\". Balance is 150.00.",
		},
		{
			name: "spent",
			got:  Spent("Gifts", "12.50", "87.50"),
			want: "💸 Spent 12.50 from \"Gifts\". Balance is 87.50.",
		},
		{
			name: "set",
			got:  SetTo("Live", "0.00"),
			want: "✅ Set \"Live\" to 0.00.",
		},
		{
			name: "deleted",
			got:  BankDeleted("Holiday"),
			want: "🗑️ Deleted bank \"Holiday\".",
		},
		{
			name: "toggled in",
			got:  Toggled("Gifts", "12.50", true),
			want: "✅ \"Gifts\" now counts toward your total. Balance is 12.50.",
		},
		{
			name: "toggled out",
			got:  Toggled("Holiday", "50.00", false),
			want: "✅ \"Holiday\" now does not count toward your total. Balance is 50.00.",
		},
		{
			name: "feedback thanks",
			got:  FeedbackThanks,
			want: "🙏 Thanks, I sent that to the admin.",
		},
		{
			name: "renamed",
			got:  BankRenamed("Holiday", "holiday"),
			want: "✏️ Renamed \"Holiday\" to \"holiday\".",
		},
		{
			name: "transferred",
			got:  Transferred("Holiday", "Gifts", "50.00", "50.00", "150.00"),
			want: "💸 Transferred 50.00 from \"Holiday\" to \"Gifts\".\nHoliday: 50.00.\nGifts: 150.00.",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("got %q, want %q", tt.got, tt.want)
			}
		})
	}
}

func TestQuietCopyHasNoEmoji(t *testing.T) {
	tests := []struct {
		name string
		got  string
	}{
		{name: "help", got: Help},
		{name: "cmd desc start", got: CmdDescStart},
		{name: "cmd desc help", got: CmdDescHelp},
		{name: "cmd desc language", got: CmdDescLanguage},
		{name: "cmd desc newbank", got: CmdDescNewBank},
		{name: "cmd desc add", got: CmdDescAdd},
		{name: "cmd desc spend", got: CmdDescSpend},
		{name: "cmd desc set", got: CmdDescSet},
		{name: "cmd desc delete", got: CmdDescDelete},
		{name: "cmd desc bank", got: CmdDescBank},
		{name: "cmd desc toggle", got: CmdDescToggle},
		{name: "cmd desc rename", got: CmdDescRename},
		{name: "cmd desc transfer", got: CmdDescTransfer},
		{name: "cmd desc banks", got: CmdDescBanks},
		{name: "cmd desc total", got: CmdDescTotal},
		{name: "cmd desc all", got: CmdDescAll},
		{name: "cmd desc cancel", got: CmdDescCancel},
		{name: "cmd desc feedback", got: CmdDescFeedback},
		{name: "new bank ask name", got: NewBankAskName},
		{name: "new bank ask include", got: NewBankAskInclude},
		{name: "invalid bank name", got: InvalidBankName},
		{name: "something went wrong", got: SomethingWentWrong},
		{name: "no banks", got: NoBanks},
		{name: "ask bank", got: AskBank},
		{name: "invalid amount", got: InvalidAmount},
		{name: "delete canceled", got: DeleteCancelled},
		{name: "canceled", got: Canceled},
		{name: "nothing to cancel", got: NothingToCancel},
		{name: "flow expired", got: FlowExpired},
		{name: "feedback ask", got: FeedbackAsk},
		{name: "feedback unavailable", got: FeedbackUnavailable},
		{name: "bank name taken", got: BankNameTaken("Holiday")},
		{name: "unknown bank", got: UnknownBank("Holiday")},
		{name: "ask add", got: AskAddAmount("Holiday")},
		{name: "ask spend", got: AskSpendAmount("Gifts")},
		{name: "ask set", got: AskSetAmount("Live")},
		{name: "ask delete", got: AskDeleteConfirm("Holiday")},
		{name: "ask rename", got: AskRenameName("Holiday")},
		{name: "need two banks", got: NeedTwoBanks},
		{name: "ask transfer from", got: AskTransferFrom},
		{name: "ask transfer to", got: AskTransferTo("Holiday")},
		{name: "ask transfer amount", got: AskTransferAmount("Holiday", "Gifts")},
		{name: "same bank", got: SameBank},
		{name: "currency mismatch", got: CurrencyMismatch},
		{name: "ask language", got: AskLanguage},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if pictograph := firstPictograph(tt.got); pictograph != -1 {
				t.Errorf("quiet copy has emoji %q in %q", string(pictograph), tt.got)
			}
		})
	}
}

func firstPictograph(s string) rune {
	for _, r := range s {
		switch {
		case r >= 0x1F300 && r <= 0x1FAFF:
			return r
		case r >= 0x2600 && r <= 0x27BF:
			return r
		case r == 0xFE0F || r == 0x200D:
			return r
		}
	}
	return -1
}

func TestBankCreated(t *testing.T) {
	tests := []struct {
		name     string
		bank     string
		included bool
		want     string
	}{
		{name: "included", bank: "Holiday", included: true, want: "counts toward your total"},
		{name: "excluded", bank: "Gifts", included: false, want: "does not count toward your total"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BankCreated(tt.bank, tt.included)
			if !strings.Contains(got, tt.bank) {
				t.Errorf("missing name %q in %q", tt.bank, got)
			}
			if !strings.Contains(got, tt.want) {
				t.Errorf("missing %q in %q", tt.want, got)
			}
		})
	}
}

func TestBankNameTaken(t *testing.T) {
	got := BankNameTaken("Holiday")
	if !strings.Contains(got, "Holiday") {
		t.Fatalf("missing name: %q", got)
	}
}

func TestMoneyCopy(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want []string
	}{
		{name: "unknown", got: UnknownBank("Holiday"), want: []string{"Holiday", "/banks"}},
		{name: "ask add", got: AskAddAmount("Holiday"), want: []string{"Holiday"}},
		{name: "ask spend", got: AskSpendAmount("Gifts"), want: []string{"Gifts"}},
		{name: "ask set", got: AskSetAmount("Live"), want: []string{"Live"}},
		{name: "added", got: Added("Holiday", "100.00", "150.00"), want: []string{"Holiday", "100.00", "150.00"}},
		{name: "spent", got: Spent("Gifts", "12.50", "87.50"), want: []string{"Gifts", "12.50", "87.50"}},
		{name: "set", got: SetTo("Live", "0.00"), want: []string{"Live", "0.00"}},
		{name: "ask delete", got: AskDeleteConfirm("Holiday"), want: []string{"Holiday"}},
		{name: "deleted", got: BankDeleted("Holiday"), want: []string{"Holiday"}},
		{name: "ask rename", got: AskRenameName("Holiday"), want: []string{"Holiday"}},
		{name: "renamed", got: BankRenamed("Holiday", "Trips"), want: []string{"Holiday", "Trips"}},
		{
			name: "transferred",
			got:  Transferred("Holiday", "Gifts", "50.00", "50.00", "150.00"),
			want: []string{"Holiday", "Gifts", "50.00", "150.00"},
		},
		{name: "need two banks", got: NeedTwoBanks, want: []string{"/newbank"}},
		{name: "ask transfer from", got: AskTransferFrom, want: []string{"from"}},
		{name: "ask transfer to", got: AskTransferTo("Holiday"), want: []string{"Holiday", "to"}},
		{name: "ask transfer amount", got: AskTransferAmount("Holiday", "Gifts"), want: []string{"Holiday", "Gifts"}},
		{name: "same bank", got: SameBank, want: []string{"different"}},
		{name: "currency mismatch", got: CurrencyMismatch, want: []string{"currenc"}},
		{name: "bank included", got: BankCard("Holiday", "50.00", true), want: []string{"Holiday", "50.00", "in total"}},
		{name: "bank excluded", got: BankCard("Gifts", "12.50", false), want: []string{"Gifts", "12.50", "not in total"}},
		{
			name: "toggled in",
			got:  Toggled("Gifts", "12.50", true),
			want: []string{"Gifts", "12.50", "counts toward your total"},
		},
		{
			name: "toggled out",
			got:  Toggled("Holiday", "50.00", false),
			want: []string{"Holiday", "50.00", "does not count toward your total"},
		},
		{name: "canceled", got: Canceled, want: []string{"Canceled"}},
		{name: "nothing to cancel", got: NothingToCancel, want: []string{"Nothing to cancel"}},
		{name: "expired", got: FlowExpired, want: []string{"expired", "Start over"}},
		{name: "feedback ask", got: FeedbackAsk, want: []string{"feedback"}},
		{name: "feedback thanks", got: FeedbackThanks, want: []string{"Thanks"}},
		{name: "feedback unavailable", got: FeedbackUnavailable, want: []string{"not available"}},
		{
			name: "feedback forward",
			got:  FeedbackForward(42, "alice", "please add history"),
			want: []string{"42", "alice", "please add history"},
		},
		{
			name: "feedback forward no username",
			got:  FeedbackForward(42, "", "please add history"),
			want: []string{"42", "please add history"},
		},
		{name: "total", got: Total("123.45"), want: []string{"123.45"}},
		{name: "empty total", got: Total("0.00"), want: []string{"0.00"}},
		{
			name: "all",
			got:  All("Holiday: 50.00 (in total)\nGifts: 12.50 (not in total)", Total("50.00")),
			want: []string{"Holiday", "50.00", "in total", "Gifts", "12.50", "not in total", "50.00"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, w := range tt.want {
				if !strings.Contains(tt.got, w) {
					t.Errorf("missing %q in %q", w, tt.got)
				}
			}
		})
	}
}

func TestCatalogsCompleteAndDistinct(t *testing.T) {
	enCat := For(domain.LocaleEN)
	for _, locale := range []string{domain.LocaleEN, domain.LocaleRU, domain.LocaleUK, domain.LocaleMD} {
		c := For(locale)
		v := reflect.ValueOf(c)
		typ := v.Type()
		for i := 0; i < v.NumField(); i++ {
			if v.Field(i).String() == "" {
				t.Errorf("%s %s is empty", locale, typ.Field(i).Name)
			}
		}
		for _, cmd := range []string{
			"/start", "/help", "/language", "/newbank", "/add", "/spend", "/set",
			"/delete", "/bank", "/toggle", "/rename", "/transfer", "/banks",
			"/total", "/all", "/cancel", "/feedback",
		} {
			if !strings.Contains(c.Help, cmd) {
				t.Errorf("%s help missing %s", locale, cmd)
			}
		}
		if strings.Contains(c.Help, "/язык") || strings.Contains(c.Help, "/мова") {
			t.Errorf("%s help translated a slash command", locale)
		}
	}
	if For(domain.LocaleRU).Start == enCat.Start {
		t.Fatal("russian start should differ from english")
	}
	if For(domain.LocaleUK).Start == enCat.Start {
		t.Fatal("ukrainian start should differ from english")
	}
	if For(domain.LocaleRU).Start == For(domain.LocaleUK).Start {
		t.Fatal("russian and ukrainian start should differ")
	}
	if For(domain.LocaleMD).Start == enCat.Start {
		t.Fatal("moldavian start should differ from english")
	}
}

func TestLanguageOptions(t *testing.T) {
	opts := LanguageOptions()
	if len(opts) != 4 {
		t.Fatalf("got %d options", len(opts))
	}
	want := []LanguageOption{
		{Code: domain.LocaleEN, Label: "English"},
		{Code: domain.LocaleRU, Label: "Русский"},
		{Code: domain.LocaleUK, Label: "Українська"},
		{Code: domain.LocaleMD, Label: "Moldovenească"},
	}
	for i, opt := range want {
		if opts[i] != opt {
			t.Errorf("option %d: %+v, want %+v", i, opts[i], opt)
		}
	}
}

func TestLanguageSetTo(t *testing.T) {
	tests := []struct {
		locale string
		want   string
	}{
		{locale: domain.LocaleEN, want: "Language set to English."},
		{locale: domain.LocaleRU, want: "Язык изменён на русский."},
		{locale: domain.LocaleUK, want: "Мову змінено на українську."},
	}
	for _, tt := range tests {
		t.Run(tt.locale, func(t *testing.T) {
			got := For(tt.locale).LanguageSetTo(tt.locale)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
