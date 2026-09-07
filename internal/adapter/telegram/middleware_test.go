package telegram

import (
	"context"
	"errors"
	"testing"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/stretchr/testify/require"

	"finbot/internal/adapter/telegram/mocks"
	"finbot/internal/domain"
)

func TestActivityMiddleware(t *testing.T) {
	ctx := context.Background()
	from := models.User{ID: telegramUserID, Username: "alice"}
	upsertErr := errors.New("db down")

	tests := []struct {
		name       string
		update     *models.Update
		upsertErr  error
		wantID     domain.UserID
		wantName   string
		wantUpsert bool
		wantNext   bool
	}{
		{
			name:       "message upserts user",
			update:     &models.Update{Message: &models.Message{From: &from}},
			wantID:     domain.UserID(telegramUserID),
			wantName:   "alice",
			wantUpsert: true,
			wantNext:   true,
		},
		{
			name:       "edited message upserts user",
			update:     &models.Update{EditedMessage: &models.Message{From: &from}},
			wantID:     domain.UserID(telegramUserID),
			wantName:   "alice",
			wantUpsert: true,
			wantNext:   true,
		},
		{
			name:       "callback query upserts user",
			update:     &models.Update{CallbackQuery: &models.CallbackQuery{From: from}},
			wantID:     domain.UserID(telegramUserID),
			wantName:   "alice",
			wantUpsert: true,
			wantNext:   true,
		},
		{
			name:       "empty username still upserts",
			update:     &models.Update{Message: &models.Message{From: &models.User{ID: telegramUserID}}},
			wantID:     domain.UserID(telegramUserID),
			wantUpsert: true,
			wantNext:   true,
		},
		{
			name:     "message without from continues",
			update:   &models.Update{Message: &models.Message{Text: "/start"}},
			wantNext: true,
		},
		{
			name: "bot sender skipped",
			update: &models.Update{Message: &models.Message{
				From: &models.User{ID: telegramUserID, IsBot: true},
			}},
			wantNext: true,
		},
		{
			name:     "zero user id skipped",
			update:   &models.Update{Message: &models.Message{From: &models.User{Username: "alice"}}},
			wantNext: true,
		},
		{
			name:     "nil update continues",
			wantNext: true,
		},
		{
			name:     "channel post without user continues",
			update:   &models.Update{ChannelPost: &models.Message{Text: "hi"}},
			wantNext: true,
		},
		{
			name:       "upsert error does not call next",
			update:     &models.Update{Message: &models.Message{From: &from}},
			upsertErr:  upsertErr,
			wantID:     domain.UserID(telegramUserID),
			wantName:   "alice",
			wantUpsert: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := mocks.NewMockService(t)
			if tt.wantUpsert {
				svc.EXPECT().
					UpsertUser(ctx, tt.wantID, tt.wantName).
					Return(domain.User{TelegramID: tt.wantID, Username: tt.wantName}, tt.upsertErr)
			}

			var nextCalled bool
			activityMiddleware(svc)(func(context.Context, *bot.Bot, *models.Update) {
				nextCalled = true
			})(ctx, nil, tt.update)

			require.Equal(t, tt.wantNext, nextCalled)
		})
	}
}

func TestActivityMiddlewareEveryUpdateUpserts(t *testing.T) {
	ctx := context.Background()
	svc := mocks.NewMockService(t)
	svc.EXPECT().
		UpsertUser(ctx, domain.UserID(telegramUserID), "alice").
		Return(domain.User{TelegramID: domain.UserID(telegramUserID), Username: "alice"}, nil).
		Times(2)

	var nextCount int
	h := activityMiddleware(svc)(func(context.Context, *bot.Bot, *models.Update) {
		nextCount++
	})
	upd := &models.Update{Message: &models.Message{
		From: &models.User{ID: telegramUserID, Username: "alice"},
	}}
	h(ctx, nil, upd)
	h(ctx, nil, upd)

	require.Equal(t, 2, nextCount)
}
