package telegram

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/stretchr/testify/require"

	"finbot/internal/adapter/telegram/mocks"
	"finbot/internal/domain"
	"finbot/internal/text"
)

func TestPendingCommandsDrainsThreeJobsInOrder(t *testing.T) {
	p := newPendingCommands()
	var mu sync.Mutex
	got := make([]int, 0, 3)
	started := make(chan struct{})
	release := make(chan struct{})

	p.enqueue(telegramUserID, pendingJob{run: func() {
		close(started)
		<-release
		mu.Lock()
		got = append(got, 1)
		mu.Unlock()
	}})
	<-started
	p.enqueue(telegramUserID, pendingJob{run: func() {
		mu.Lock()
		got = append(got, 2)
		mu.Unlock()
	}})
	p.enqueue(telegramUserID, pendingJob{run: func() {
		mu.Lock()
		got = append(got, 3)
		mu.Unlock()
	}})
	close(release)
	p.wait(telegramUserID)

	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, []int{1, 2, 3}, got)
}

func TestPendingCommandsDropsOldestWaitingWhenFull(t *testing.T) {
	p := newPendingCommands()
	started := make(chan struct{})
	release := make(chan struct{})
	var mu sync.Mutex
	got := make([]int, 0, pendingWaitingCap)

	p.enqueue(telegramUserID, pendingJob{run: func() {
		close(started)
		<-release
	}})
	<-started
	for i := 1; i <= pendingWaitingCap+1; i++ {
		n := i
		p.enqueue(telegramUserID, pendingJob{run: func() {
			mu.Lock()
			got = append(got, n)
			mu.Unlock()
		}})
	}
	close(release)
	p.wait(telegramUserID)

	want := make([]int, pendingWaitingCap)
	for i := range want {
		want[i] = i + 2
	}
	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, want, got)
}

func TestPendingCommandsDoesNotBlockOtherUsers(t *testing.T) {
	p := newPendingCommands()
	const userA, userB int64 = 1, 2
	aStarted := make(chan struct{})
	aRelease := make(chan struct{})
	bDone := make(chan struct{})

	p.enqueue(userA, pendingJob{run: func() {
		close(aStarted)
		<-aRelease
	}})
	<-aStarted
	p.enqueue(userB, pendingJob{run: func() { close(bDone) }})

	select {
	case <-bDone:
	case <-time.After(time.Second):
		t.Fatal("user B waited on user A")
	}
	close(aRelease)
	p.wait(userA)
}

func TestPendingDrainCompactsHelpBurstToOneJob(t *testing.T) {
	p := newPendingCommands()
	p.mu.Lock()
	p.users[telegramUserID] = &pendingUser{draining: true}
	p.mu.Unlock()

	var mu sync.Mutex
	var ran int
	for range 3 {
		p.enqueue(telegramUserID, pendingJob{
			update: commandUpdate("/help"),
			run: func() {
				mu.Lock()
				ran++
				mu.Unlock()
			},
		})
	}
	go p.drain(telegramUserID)
	p.wait(telegramUserID)

	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, 1, ran)
}

func TestPendingCompactsWaitingAfterProcessedItem(t *testing.T) {
	p := newPendingCommands()
	started := make(chan struct{})
	release := make(chan struct{})
	var mu sync.Mutex
	var ran []string

	p.enqueue(telegramUserID, pendingJob{run: func() {
		close(started)
		<-release
		mu.Lock()
		ran = append(ran, "block")
		mu.Unlock()
	}})
	<-started
	for range 3 {
		p.enqueue(telegramUserID, pendingJob{
			update: commandUpdate("/help"),
			run: func() {
				mu.Lock()
				ran = append(ran, "help")
				mu.Unlock()
			},
		})
	}
	close(release)
	p.wait(telegramUserID)

	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, []string{"block", "help"}, ran)
}

func TestPendingMiddlewarePassesThroughWithoutUser(t *testing.T) {
	p := newPendingCommands()
	var nextCalled bool
	p.middleware()(func(context.Context, *bot.Bot, *models.Update) {
		nextCalled = true
	})(context.Background(), nil, &models.Update{ChannelPost: &models.Message{Text: "hi"}})
	require.True(t, nextCalled)
}

func TestPendingMiddlewareEnqueuesSlashTypedAndCallback(t *testing.T) {
	p := newPendingCommands()
	started := make(chan struct{})
	release := make(chan struct{})
	var mu sync.Mutex
	got := make([]string, 0, 3)

	kindOf := func(update *models.Update) string {
		switch {
		case update.CallbackQuery != nil:
			return "callback"
		case update.Message != nil && update.Message.Text != "" && update.Message.Text[0] != '/':
			return "typed"
		default:
			return "slash"
		}
	}
	next := p.middleware()(func(_ context.Context, _ *bot.Bot, update *models.Update) {
		kind := kindOf(update)
		if kind == "slash" {
			close(started)
			<-release
		}
		mu.Lock()
		got = append(got, kind)
		mu.Unlock()
	})

	next(context.Background(), nil, commandUpdate("/add"))
	<-started
	next(context.Background(), nil, commandUpdate("42.00"))
	next(context.Background(), nil, callbackUpdate(callbackAddPrefix+"7"))
	close(release)
	p.wait(telegramUserID)

	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, []string{"slash", "typed", "callback"}, got)
}

func TestPendingMoneyShortcutsApplyInEnqueueOrder(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	key := fsmKey(userID)
	bank := domain.Bank{ID: testBankID, UserID: userID, Name: "Holiday"}
	after100 := domain.Bank{ID: testBankID, Name: "Holiday", Balance: 10000}
	after150 := domain.Bank{ID: testBankID, Name: "Holiday", Balance: 15000}

	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})
	var mu sync.Mutex
	var addOrder []domain.Money

	b := newTestBot(t, ctx, func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
		svc.EXPECT().GetByName(anyCtx, userID, "Holiday").Return(bank, nil).Once()
		svc.EXPECT().Add(anyCtx, userID, testBankID, domain.Money(10000)).
			Run(func(context.Context, domain.UserID, int64, domain.Money) {
				mu.Lock()
				addOrder = append(addOrder, 10000)
				mu.Unlock()
				close(firstStarted)
				<-releaseFirst
			}).
			Return(after100, nil).Once()
		svc.EXPECT().GetByName(anyCtx, userID, "Holiday").Return(after100, nil).Once()
		svc.EXPECT().Add(anyCtx, userID, testBankID, domain.Money(5000)).
			Run(func(context.Context, domain.UserID, int64, domain.Money) {
				mu.Lock()
				addOrder = append(addOrder, 5000)
				mu.Unlock()
			}).
			Return(after150, nil).Once()
		cache.EXPECT().Delete(anyCtx, key).Return(nil)
		cache.EXPECT().Delete(anyCtx, key).Return(nil)
		expectSendMessage(t, client, nil)
		expectSendMessage(t, client, nil)
	})

	done := make(chan struct{})
	go func() {
		b.ProcessUpdate(ctx, commandUpdate("/add Holiday 100"))
		close(done)
	}()
	<-firstStarted
	b.inner.ProcessUpdate(ctx, commandUpdate("/add Holiday 50"))
	close(releaseFirst)
	<-done

	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, []domain.Money{10000, 5000}, addOrder)
}

func TestPendingHelpThenBanksRepliesInOrder(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	holiday := domain.Bank{ID: testBankID, UserID: userID, Name: "Holiday", Balance: 5000, IncludeInTotal: true}

	helpStarted := make(chan struct{})
	releaseHelp := make(chan struct{})
	listCalled := make(chan struct{})
	var mu sync.Mutex
	var sent []string

	b := newTestBot(t, ctx, func(svc *mocks.MockService, _ *mocks.MockCache, client *mocks.MockHTTPClient) {
		svc.EXPECT().List(anyCtx, userID).
			Run(func(context.Context, domain.UserID) { close(listCalled) }).
			Return([]domain.Bank{holiday}, nil)
		expectSendMessageCapture(t, client, &mu, &sent, helpStarted, releaseHelp)
		expectSendMessageCapture(t, client, &mu, &sent, nil, nil)
	})

	done := make(chan struct{})
	go func() {
		b.ProcessUpdate(ctx, commandUpdate("/help"))
		close(done)
	}()
	<-helpStarted
	b.inner.ProcessUpdate(ctx, commandUpdate("/banks"))
	select {
	case <-listCalled:
		t.Fatal("banks ran before help finished")
	default:
	}
	close(releaseHelp)
	<-done

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, sent, 2)
	require.Contains(t, sent[0], text.Help)
	require.Contains(t, sent[1], "Holiday")
}

func TestPendingBurstHelpAllHelpRepliesInSendOrder(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	holiday := domain.Bank{ID: testBankID, UserID: userID, Name: "Holiday", Balance: 5000, IncludeInTotal: true}

	var mu sync.Mutex
	var sent []string

	b := newTestBot(t, ctx, func(svc *mocks.MockService, _ *mocks.MockCache, client *mocks.MockHTTPClient) {
		svc.EXPECT().All(anyCtx, userID).Return([]domain.Bank{holiday}, holiday.Balance, nil)
		for range 3 {
			expectSendMessageCapture(t, client, &mu, &sent, nil, nil)
		}
	})

	b.pending.mu.Lock()
	b.pending.users[telegramUserID] = &pendingUser{draining: true}
	b.pending.mu.Unlock()
	for _, cmd := range []string{"/help", "/help", "/help", "/all", "/help"} {
		b.inner.ProcessUpdate(ctx, commandUpdate(cmd))
	}
	go b.pending.drain(telegramUserID)
	b.pending.wait(telegramUserID)

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, sent, 3)
	require.Contains(t, sent[0], text.Help)
	require.Contains(t, sent[1], "Holiday")
	require.NotContains(t, sent[1], "Finbot commands:")
	require.Contains(t, sent[2], text.Help)
}

func TestPendingHelpBurstProducesOneReply(t *testing.T) {
	ctx := context.Background()
	var mu sync.Mutex
	var sent []string

	b := newTestBot(t, ctx, func(_ *mocks.MockService, _ *mocks.MockCache, client *mocks.MockHTTPClient) {
		expectSendMessageCapture(t, client, &mu, &sent, nil, nil)
	})

	b.pending.mu.Lock()
	b.pending.users[telegramUserID] = &pendingUser{draining: true}
	b.pending.mu.Unlock()
	for range 3 {
		b.inner.ProcessUpdate(ctx, commandUpdate("/help"))
	}
	go b.pending.drain(telegramUserID)
	b.pending.wait(telegramUserID)

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, sent, 1)
	require.Contains(t, sent[0], text.Help)
}

func TestPendingDoesNotBlockOtherUsers(t *testing.T) {
	ctx := context.Background()
	userID := domain.UserID(telegramUserID)
	const otherUserID int64 = 99
	otherID := domain.UserID(otherUserID)
	key := fsmKey(userID)
	bank := domain.Bank{ID: testBankID, UserID: userID, Name: "Holiday"}
	after100 := domain.Bank{ID: testBankID, Name: "Holiday", Balance: 10000}

	aStarted := make(chan struct{})
	releaseA := make(chan struct{})

	b := newTestBot(t, ctx, func(svc *mocks.MockService, cache *mocks.MockCache, client *mocks.MockHTTPClient) {
		svc.EXPECT().
			UpsertUser(ctx, otherID, "bob").
			Return(domain.User{TelegramID: otherID, Username: "bob"}, nil)
		svc.EXPECT().GetByName(anyCtx, userID, "Holiday").Return(bank, nil)
		svc.EXPECT().Add(anyCtx, userID, testBankID, domain.Money(10000)).
			Run(func(context.Context, domain.UserID, int64, domain.Money) {
				close(aStarted)
				<-releaseA
			}).
			Return(after100, nil)
		cache.EXPECT().Delete(anyCtx, key).Return(nil)
		expectSendMessage(t, client, nil)
		expectSendMessage(t, client, nil)
	})

	aDone := make(chan struct{})
	go func() {
		b.ProcessUpdate(ctx, commandUpdate("/add Holiday 100"))
		close(aDone)
	}()
	<-aStarted

	helpDone := make(chan struct{})
	go func() {
		b.ProcessUpdate(ctx, commandUpdateFrom(otherUserID, "bob", "/help"))
		close(helpDone)
	}()
	select {
	case <-helpDone:
	case <-time.After(time.Second):
		t.Fatal("user B waited on user A")
	}
	close(releaseA)
	<-aDone
}

func commandUpdateFrom(userID int64, username, text string) *models.Update {
	return &models.Update{Message: &models.Message{
		From: &models.User{ID: userID, Username: username},
		Chat: models.Chat{ID: userID, Type: "private"},
		Text: text,
	}}
}
