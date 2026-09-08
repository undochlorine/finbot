package service

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	"finbot/internal/domain"
	"finbot/internal/service/mocks"
)

func TestCreateBank(t *testing.T) {
	now := fixedNow()
	ctx := context.Background()

	tests := []struct {
		name    string
		bank    string
		include bool
		setup   func(*mocks.MockBankRepository, *mocks.MockClock)
		want    domain.Bank
		wantErr error
	}{
		{
			name:    "stores typed casing",
			bank:    "Holiday",
			include: true,
			setup: func(banks *mocks.MockBankRepository, clock *mocks.MockClock) {
				in := sampleBank(userA, 0, "Holiday", 0, true)
				out := in
				out.ID = 1
				banks.EXPECT().GetByName(ctx, userA, "Holiday").Return(domain.Bank{}, domain.ErrBankNotFound)
				clock.EXPECT().Now().Return(now)
				banks.EXPECT().Create(ctx, userA, in).Return(out, nil)
			},
			want: sampleBank(userA, 1, "Holiday", 0, true),
		},
		{
			name:    "unicode name",
			bank:    "Подарки",
			include: false,
			setup: func(banks *mocks.MockBankRepository, clock *mocks.MockClock) {
				in := sampleBank(userA, 0, "Подарки", 0, false)
				out := in
				out.ID = 1
				banks.EXPECT().GetByName(ctx, userA, "Подарки").Return(domain.Bank{}, domain.ErrBankNotFound)
				clock.EXPECT().Now().Return(now)
				banks.EXPECT().Create(ctx, userA, in).Return(out, nil)
			},
			want: sampleBank(userA, 1, "Подарки", 0, false),
		},
		{
			name:    "empty name",
			bank:    "",
			wantErr: domain.ErrInvalidBankName,
		},
		{
			name:    "whitespace name",
			bank:    "   ",
			wantErr: domain.ErrInvalidBankName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, banks, ops, clock := newBankSvc(t)
			if tt.setup != nil {
				tt.setup(banks, clock)
			}

			got, err := svc.CreateBank(ctx, userA, tt.bank, tt.include)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.True(t, banks.AssertNotCalled(t, "Create"))
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
			require.True(t, ops.AssertNotCalled(t, "Append"))
		})
	}
}

func TestCreateBankDuplicate(t *testing.T) {
	ctx := context.Background()
	existing := sampleBank(userA, 1, "Holiday", 10000, true)

	tests := []struct {
		name string
		dup  string
	}{
		{name: "same spelling", dup: "Holiday"},
		{name: "different case", dup: "holiday"},
		{name: "unicode case", dup: "подарки"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, banks, _, _ := newBankSvc(t)
			found := existing
			if tt.dup == "подарки" {
				found = sampleBank(userA, 1, "Подарки", 10000, true)
			}
			banks.EXPECT().GetByName(ctx, userA, tt.dup).Return(found, nil)

			_, err := svc.CreateBank(ctx, userA, tt.dup, false)
			require.ErrorIs(t, err, domain.ErrBankNameTaken)
			require.True(t, banks.AssertNotCalled(t, "Create"))
			require.True(t, banks.AssertNotCalled(t, "Update"))
		})
	}
}

func TestAddSpendSet(t *testing.T) {
	ctx := context.Background()
	now := fixedNow()

	tests := []struct {
		name    string
		start   domain.Money
		op      string
		amount  domain.Money
		unknown bool
		noIO    bool
		want    domain.Money
		wantErr error
	}{
		{name: "add increases", start: 10000, op: domain.CommandAdd, amount: 2500, want: 12500},
		{name: "spend decreases", start: 10000, op: domain.CommandSpend, amount: 2500, want: 7500},
		{name: "spend below zero allowed", start: 100, op: domain.CommandSpend, amount: 250, want: -150},
		{name: "set replaces", start: 10000, op: domain.CommandSet, amount: 0, want: 0},
		{name: "set negative allowed", start: 10000, op: domain.CommandSet, amount: -500, want: -500},
		{name: "add negative is invalid", start: 10000, op: domain.CommandAdd, amount: -1, noIO: true, wantErr: domain.ErrInvalidAmount},
		{name: "spend negative is invalid", start: 10000, op: domain.CommandSpend, amount: -1, noIO: true, wantErr: domain.ErrInvalidAmount},
		{name: "add overflow rejected", start: math.MaxInt64, op: domain.CommandAdd, amount: 1, wantErr: domain.ErrInvalidAmount},
		{name: "spend underflow rejected", start: math.MinInt64, op: domain.CommandSpend, amount: 1, wantErr: domain.ErrInvalidAmount},
		{name: "add unknown bank", op: domain.CommandAdd, amount: 100, unknown: true, wantErr: domain.ErrBankNotFound},
		{name: "spend unknown bank", op: domain.CommandSpend, amount: 100, unknown: true, wantErr: domain.ErrBankNotFound},
		{name: "set unknown bank", op: domain.CommandSet, amount: 100, unknown: true, wantErr: domain.ErrBankNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, banks, ops, clock := newBankSvc(t)
			start := sampleBank(userA, 1, "Live", tt.start, true)
			id := start.ID
			if !tt.noIO {
				switch {
				case tt.unknown:
					id = 99
					banks.EXPECT().GetByID(ctx, userA, id).Return(domain.Bank{}, domain.ErrBankNotFound)
				case tt.wantErr == domain.ErrInvalidAmount:
					banks.EXPECT().GetByID(ctx, userA, id).Return(start, nil)
				default:
					banks.EXPECT().GetByID(ctx, userA, id).Return(start, nil)
					clock.EXPECT().Now().Return(now)
					updated := start
					updated.Balance = tt.want
					updated.UpdatedAt = now
					banks.EXPECT().Update(ctx, userA, updated).Return(updated, nil)
					after := tt.want
					ops.EXPECT().Append(ctx, matchOp(userA, moneyOpType(tt.op), tt.amount, id, &after, start.Name, now)).Return(nil)
				}
			}

			got, err := applyMoneyOp(ctx, svc, userA, id, tt.op, tt.amount)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.True(t, banks.AssertNotCalled(t, "Update"))
				require.True(t, ops.AssertNotCalled(t, "Append"))
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got.Balance)
		})
	}
}

func TestDelete(t *testing.T) {
	ctx := context.Background()
	now := fixedNow()
	bank := sampleBank(userA, 1, "Live", 10000, true)

	t.Run("removes bank and appends delete", func(t *testing.T) {
		svc, banks, ops, clock := newBankSvc(t)
		banks.EXPECT().GetByID(ctx, userA, bank.ID).Return(bank, nil)
		clock.EXPECT().Now().Return(now)
		ops.EXPECT().Append(ctx, matchOp(userA, domain.OperationDelete, bank.Balance, bank.ID, nil, bank.Name, now)).Return(nil)
		banks.EXPECT().Delete(ctx, userA, bank.ID).Return(nil)

		require.NoError(t, svc.Delete(ctx, userA, bank.ID))
	})

	t.Run("unknown bank does not append", func(t *testing.T) {
		svc, banks, ops, _ := newBankSvc(t)
		banks.EXPECT().GetByID(ctx, userA, int64(99)).Return(domain.Bank{}, domain.ErrBankNotFound)

		err := svc.Delete(ctx, userA, 99)
		require.ErrorIs(t, err, domain.ErrBankNotFound)
		require.True(t, ops.AssertNotCalled(t, "Append"))
		require.True(t, banks.AssertNotCalled(t, "Delete"))
	})
}

func TestTotalToggleAndEmptyList(t *testing.T) {
	ctx := context.Background()
	now := fixedNow()
	included := sampleBank(userA, 1, "Live", 10000, true)
	excluded := sampleBank(userA, 2, "Gifts", 2500, false)

	t.Run("empty list and total", func(t *testing.T) {
		svc, banks, _, _ := newBankSvc(t)
		banks.EXPECT().List(ctx, userA).Return([]domain.Bank{}, nil).Twice()
		banks.EXPECT().TotalIncluded(ctx, userA).Return(domain.Money(0), nil).Twice()

		list, err := svc.List(ctx, userA)
		require.NoError(t, err)
		require.Empty(t, list)

		total, err := svc.Total(ctx, userA)
		require.NoError(t, err)
		require.Equal(t, domain.Money(0), total)

		all, allTotal, err := svc.All(ctx, userA)
		require.NoError(t, err)
		require.Empty(t, all)
		require.Equal(t, domain.Money(0), allTotal)
	})

	t.Run("excluded omitted from total", func(t *testing.T) {
		svc, banks, _, _ := newBankSvc(t)
		list := []domain.Bank{included, excluded}
		banks.EXPECT().List(ctx, userA).Return(list, nil)
		banks.EXPECT().TotalIncluded(ctx, userA).Return(included.Balance, nil)

		all, total, err := svc.All(ctx, userA)
		require.NoError(t, err)
		require.Equal(t, list, all)
		require.Equal(t, included.Balance, total)
	})

	t.Run("only excluded is zero total", func(t *testing.T) {
		svc, banks, _, _ := newBankSvc(t)
		banks.EXPECT().TotalIncluded(ctx, userA).Return(domain.Money(0), nil)

		total, err := svc.Total(ctx, userA)
		require.NoError(t, err)
		require.Equal(t, domain.Money(0), total)
	})

	t.Run("toggle excluded into total without changing balance", func(t *testing.T) {
		svc, banks, ops, clock := newBankSvc(t)
		banks.EXPECT().GetByID(ctx, userA, excluded.ID).Return(excluded, nil)
		clock.EXPECT().Now().Return(now)
		updated := excluded
		updated.IncludeInTotal = true
		updated.UpdatedAt = now
		banks.EXPECT().Update(ctx, userA, updated).Return(updated, nil)

		got, err := svc.Toggle(ctx, userA, excluded.ID)
		require.NoError(t, err)
		require.True(t, got.IncludeInTotal)
		require.Equal(t, excluded.Balance, got.Balance)
		require.True(t, ops.AssertNotCalled(t, "Append"))
	})

	t.Run("toggle included out of total without changing balance", func(t *testing.T) {
		svc, banks, ops, clock := newBankSvc(t)
		banks.EXPECT().GetByID(ctx, userA, included.ID).Return(included, nil)
		clock.EXPECT().Now().Return(now)
		updated := included
		updated.IncludeInTotal = false
		updated.UpdatedAt = now
		banks.EXPECT().Update(ctx, userA, updated).Return(updated, nil)

		got, err := svc.Toggle(ctx, userA, included.ID)
		require.NoError(t, err)
		require.False(t, got.IncludeInTotal)
		require.Equal(t, included.Balance, got.Balance)
		require.True(t, ops.AssertNotCalled(t, "Append"))
	})
}

func TestRename(t *testing.T) {
	ctx := context.Background()
	now := fixedNow()
	holiday := sampleBank(userA, 1, "Holiday", 10000, true)
	gifts := sampleBank(userA, 2, "Gifts", 2500, false)

	tests := []struct {
		name    string
		bankID  int64
		newName string
		setup   func(*mocks.MockBankRepository, *mocks.MockOperationRepository, *mocks.MockClock)
		want    string
		wantErr error
	}{
		{
			name:    "persists typed spelling",
			bankID:  holiday.ID,
			newName: "Trips",
			setup: func(banks *mocks.MockBankRepository, ops *mocks.MockOperationRepository, clock *mocks.MockClock) {
				banks.EXPECT().GetByID(ctx, userA, holiday.ID).Return(holiday, nil)
				banks.EXPECT().GetByName(ctx, userA, "Trips").Return(domain.Bank{}, domain.ErrBankNotFound)
				clock.EXPECT().Now().Return(now)
				updated := holiday
				updated.Name = "Trips"
				updated.UpdatedAt = now
				banks.EXPECT().Update(ctx, userA, updated).Return(updated, nil)
				after := holiday.Balance
				ops.EXPECT().Append(ctx, matchOp(userA, domain.OperationRename, 0, holiday.ID, &after, "Holiday -> Trips", now)).Return(nil)
			},
			want: "Trips",
		},
		{
			name:    "recases the same bank",
			bankID:  holiday.ID,
			newName: "holiday",
			setup: func(banks *mocks.MockBankRepository, ops *mocks.MockOperationRepository, clock *mocks.MockClock) {
				banks.EXPECT().GetByID(ctx, userA, holiday.ID).Return(holiday, nil)
				banks.EXPECT().GetByName(ctx, userA, "holiday").Return(holiday, nil)
				clock.EXPECT().Now().Return(now)
				updated := holiday
				updated.Name = "holiday"
				updated.UpdatedAt = now
				banks.EXPECT().Update(ctx, userA, updated).Return(updated, nil)
				after := holiday.Balance
				ops.EXPECT().Append(ctx, matchOp(userA, domain.OperationRename, 0, holiday.ID, &after, "Holiday -> holiday", now)).Return(nil)
			},
			want: "holiday",
		},
		{
			name:    "name taken by another bank",
			bankID:  holiday.ID,
			newName: "gifts",
			setup: func(banks *mocks.MockBankRepository, ops *mocks.MockOperationRepository, _ *mocks.MockClock) {
				banks.EXPECT().GetByID(ctx, userA, holiday.ID).Return(holiday, nil)
				banks.EXPECT().GetByName(ctx, userA, "gifts").Return(gifts, nil)
			},
			wantErr: domain.ErrBankNameTaken,
		},
		{
			name:    "unknown bank",
			bankID:  99,
			newName: "Trips",
			setup: func(banks *mocks.MockBankRepository, ops *mocks.MockOperationRepository, _ *mocks.MockClock) {
				banks.EXPECT().GetByID(ctx, userA, int64(99)).Return(domain.Bank{}, domain.ErrBankNotFound)
			},
			wantErr: domain.ErrBankNotFound,
		},
		{
			name:    "empty name",
			bankID:  holiday.ID,
			newName: "",
			wantErr: domain.ErrInvalidBankName,
		},
		{
			name:    "whitespace name",
			bankID:  holiday.ID,
			newName: "   ",
			wantErr: domain.ErrInvalidBankName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, banks, ops, clock := newBankSvc(t)
			if tt.setup != nil {
				tt.setup(banks, ops, clock)
			}

			got, err := svc.Rename(ctx, userA, tt.bankID, tt.newName)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.True(t, banks.AssertNotCalled(t, "Update"))
				require.True(t, ops.AssertNotCalled(t, "Append"))
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got.Name)
			require.Equal(t, holiday.Balance, got.Balance)
		})
	}
}

func TestTransfer(t *testing.T) {
	ctx := context.Background()
	now := fixedNow()
	from := sampleBank(userA, 1, "Holiday", 10000, true)
	to := sampleBank(userA, 2, "Gifts", 2500, false)
	eur := from
	eur.Currency = "EUR"

	tests := []struct {
		name     string
		fromID   int64
		toID     int64
		amount   domain.Money
		setup    func(*mocks.MockBankRepository, *mocks.MockOperationRepository, *mocks.MockClock)
		wantFrom domain.Money
		wantTo   domain.Money
		wantErr  error
	}{
		{
			name:   "moves money and writes one transfer row",
			fromID: from.ID,
			toID:   to.ID,
			amount: 2500,
			setup: func(banks *mocks.MockBankRepository, ops *mocks.MockOperationRepository, clock *mocks.MockClock) {
				banks.EXPECT().GetByID(ctx, userA, from.ID).Return(from, nil)
				banks.EXPECT().GetByID(ctx, userA, to.ID).Return(to, nil)
				clock.EXPECT().Now().Return(now).Times(2)
				debited := from
				debited.Balance = 7500
				debited.UpdatedAt = now
				credited := to
				credited.Balance = 5000
				credited.UpdatedAt = now
				banks.EXPECT().Update(ctx, userA, debited).Return(debited, nil)
				banks.EXPECT().Update(ctx, userA, credited).Return(credited, nil)
				after := domain.Money(7500)
				ops.EXPECT().Append(ctx, matchOp(userA, domain.OperationTransfer, 2500, from.ID, &after, "2", now)).Return(nil)
			},
			wantFrom: 7500,
			wantTo:   5000,
		},
		{
			name:   "overdraft allowed",
			fromID: from.ID,
			toID:   to.ID,
			amount: 12500,
			setup: func(banks *mocks.MockBankRepository, ops *mocks.MockOperationRepository, clock *mocks.MockClock) {
				banks.EXPECT().GetByID(ctx, userA, from.ID).Return(from, nil)
				banks.EXPECT().GetByID(ctx, userA, to.ID).Return(to, nil)
				clock.EXPECT().Now().Return(now).Times(2)
				debited := from
				debited.Balance = -2500
				debited.UpdatedAt = now
				credited := to
				credited.Balance = 15000
				credited.UpdatedAt = now
				banks.EXPECT().Update(ctx, userA, debited).Return(debited, nil)
				banks.EXPECT().Update(ctx, userA, credited).Return(credited, nil)
				after := domain.Money(-2500)
				ops.EXPECT().Append(ctx, matchOp(userA, domain.OperationTransfer, 12500, from.ID, &after, "2", now)).Return(nil)
			},
			wantFrom: -2500,
			wantTo:   15000,
		},
		{
			name:    "same bank",
			fromID:  from.ID,
			toID:    from.ID,
			amount:  100,
			wantErr: domain.ErrSameBank,
		},
		{
			name:    "zero amount",
			fromID:  from.ID,
			toID:    to.ID,
			amount:  0,
			wantErr: domain.ErrInvalidAmount,
		},
		{
			name:    "negative amount",
			fromID:  from.ID,
			toID:    to.ID,
			amount:  -1,
			wantErr: domain.ErrInvalidAmount,
		},
		{
			name:   "different currency",
			fromID: from.ID,
			toID:   to.ID,
			amount: 100,
			setup: func(banks *mocks.MockBankRepository, _ *mocks.MockOperationRepository, _ *mocks.MockClock) {
				banks.EXPECT().GetByID(ctx, userA, from.ID).Return(from, nil)
				banks.EXPECT().GetByID(ctx, userA, to.ID).Return(eur, nil)
			},
			wantErr: domain.ErrCurrencyMismatch,
		},
		{
			name:   "unknown from bank",
			fromID: 99,
			toID:   to.ID,
			amount: 100,
			setup: func(banks *mocks.MockBankRepository, _ *mocks.MockOperationRepository, _ *mocks.MockClock) {
				banks.EXPECT().GetByID(ctx, userA, int64(99)).Return(domain.Bank{}, domain.ErrBankNotFound)
			},
			wantErr: domain.ErrBankNotFound,
		},
		{
			name:   "unknown to bank",
			fromID: from.ID,
			toID:   99,
			amount: 100,
			setup: func(banks *mocks.MockBankRepository, _ *mocks.MockOperationRepository, _ *mocks.MockClock) {
				banks.EXPECT().GetByID(ctx, userA, from.ID).Return(from, nil)
				banks.EXPECT().GetByID(ctx, userA, int64(99)).Return(domain.Bank{}, domain.ErrBankNotFound)
			},
			wantErr: domain.ErrBankNotFound,
		},
		{
			name:   "debit overflow rejected",
			fromID: from.ID,
			toID:   to.ID,
			amount: 1,
			setup: func(banks *mocks.MockBankRepository, _ *mocks.MockOperationRepository, _ *mocks.MockClock) {
				start := from
				start.Balance = math.MinInt64
				banks.EXPECT().GetByID(ctx, userA, from.ID).Return(start, nil)
				banks.EXPECT().GetByID(ctx, userA, to.ID).Return(to, nil)
			},
			wantErr: domain.ErrInvalidAmount,
		},
		{
			name:   "credit overflow rejected",
			fromID: from.ID,
			toID:   to.ID,
			amount: 1,
			setup: func(banks *mocks.MockBankRepository, _ *mocks.MockOperationRepository, _ *mocks.MockClock) {
				start := to
				start.Balance = math.MaxInt64
				banks.EXPECT().GetByID(ctx, userA, from.ID).Return(from, nil)
				banks.EXPECT().GetByID(ctx, userA, to.ID).Return(start, nil)
			},
			wantErr: domain.ErrInvalidAmount,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, banks, ops, clock := newBankSvc(t)
			if tt.setup != nil {
				tt.setup(banks, ops, clock)
			}

			gotFrom, gotTo, err := svc.Transfer(ctx, userA, tt.fromID, tt.toID, tt.amount)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.True(t, banks.AssertNotCalled(t, "Update"))
				require.True(t, ops.AssertNotCalled(t, "Append"))
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantFrom, gotFrom.Balance)
			require.Equal(t, tt.wantTo, gotTo.Balance)
		})
	}
}

func TestToggleUnknownBank(t *testing.T) {
	svc, banks, _, _ := newBankSvc(t)
	ctx := context.Background()
	banks.EXPECT().GetByID(ctx, userA, int64(99)).Return(domain.Bank{}, domain.ErrBankNotFound)

	_, err := svc.Toggle(ctx, userA, 99)
	require.ErrorIs(t, err, domain.ErrBankNotFound)
	require.True(t, banks.AssertNotCalled(t, "Update"))
}

func TestBanksIsolatedByUser(t *testing.T) {
	ctx := context.Background()
	a := sampleBank(userA, 1, "Holiday", 10000, true)
	b := sampleBank(userB, 2, "Holiday", 2500, false)

	tests := []struct {
		name  string
		setup func(*mocks.MockBankRepository)
		run   func(*testing.T, *Service)
	}{
		{
			name: "list does not leak",
			setup: func(banks *mocks.MockBankRepository) {
				banks.EXPECT().List(ctx, userB).Return([]domain.Bank{b}, nil)
			},
			run: func(t *testing.T, svc *Service) {
				list, err := svc.List(ctx, userB)
				require.NoError(t, err)
				require.Equal(t, []domain.Bank{b}, list)
			},
		},
		{
			name: "cannot get other user's bank",
			setup: func(banks *mocks.MockBankRepository) {
				banks.EXPECT().GetByID(ctx, userB, a.ID).Return(domain.Bank{}, domain.ErrBankNotFound)
			},
			run: func(t *testing.T, svc *Service) {
				_, err := svc.Get(ctx, userB, a.ID)
				require.ErrorIs(t, err, domain.ErrBankNotFound)
			},
		},
		{
			name: "cannot add to other user's bank",
			setup: func(banks *mocks.MockBankRepository) {
				banks.EXPECT().GetByID(ctx, userB, a.ID).Return(domain.Bank{}, domain.ErrBankNotFound)
			},
			run: func(t *testing.T, svc *Service) {
				_, err := svc.Add(ctx, userB, a.ID, 1)
				require.ErrorIs(t, err, domain.ErrBankNotFound)
			},
		},
		{
			name: "cannot delete other user's bank",
			setup: func(banks *mocks.MockBankRepository) {
				banks.EXPECT().GetByID(ctx, userB, a.ID).Return(domain.Bank{}, domain.ErrBankNotFound)
			},
			run: func(t *testing.T, svc *Service) {
				err := svc.Delete(ctx, userB, a.ID)
				require.ErrorIs(t, err, domain.ErrBankNotFound)
			},
		},
		{
			name: "cannot rename other user's bank",
			setup: func(banks *mocks.MockBankRepository) {
				banks.EXPECT().GetByID(ctx, userB, a.ID).Return(domain.Bank{}, domain.ErrBankNotFound)
			},
			run: func(t *testing.T, svc *Service) {
				_, err := svc.Rename(ctx, userB, a.ID, "Trips")
				require.ErrorIs(t, err, domain.ErrBankNotFound)
			},
		},
		{
			name: "cannot transfer other user's bank",
			setup: func(banks *mocks.MockBankRepository) {
				banks.EXPECT().GetByID(ctx, userB, a.ID).Return(domain.Bank{}, domain.ErrBankNotFound)
			},
			run: func(t *testing.T, svc *Service) {
				_, _, err := svc.Transfer(ctx, userB, a.ID, b.ID, 100)
				require.ErrorIs(t, err, domain.ErrBankNotFound)
			},
		},
		{
			name: "totals are per user",
			setup: func(banks *mocks.MockBankRepository) {
				banks.EXPECT().TotalIncluded(ctx, userA).Return(a.Balance, nil)
				banks.EXPECT().TotalIncluded(ctx, userB).Return(domain.Money(0), nil)
			},
			run: func(t *testing.T, svc *Service) {
				totalA, err := svc.Total(ctx, userA)
				require.NoError(t, err)
				totalB, err := svc.Total(ctx, userB)
				require.NoError(t, err)
				require.Equal(t, a.Balance, totalA)
				require.Equal(t, domain.Money(0), totalB)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, banks, _, _ := newBankSvc(t)
			tt.setup(banks)
			tt.run(t, svc)
		})
	}
}

func applyMoneyOp(
	ctx context.Context,
	svc *Service,
	user domain.UserID,
	id int64,
	op string,
	amount domain.Money,
) (domain.Bank, error) {
	switch op {
	case domain.CommandAdd:
		return svc.Add(ctx, user, id, amount)
	case domain.CommandSpend:
		return svc.Spend(ctx, user, id, amount)
	case domain.CommandSet:
		return svc.Set(ctx, user, id, amount)
	default:
		return domain.Bank{}, errors.New("unknown op")
	}
}

func moneyOpType(op string) domain.OperationType {
	switch op {
	case domain.CommandAdd:
		return domain.OperationAdd
	case domain.CommandSpend:
		return domain.OperationSpend
	default:
		return domain.OperationSet
	}
}
