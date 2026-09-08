package telegram

import (
	"context"
	"log/slog"
	"sync"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

const pendingWaitingCap = 32

type pendingUser struct {
	waiting  []func()
	draining bool
}

type PendingCommands struct {
	mu    sync.Mutex
	idle  *sync.Cond
	users map[int64]*pendingUser
}

func newPendingCommands() *PendingCommands {
	p := &PendingCommands{users: make(map[int64]*pendingUser)}
	p.idle = sync.NewCond(&p.mu)
	return p
}

func (p *PendingCommands) enqueue(userID int64, job func()) {
	if p == nil || job == nil {
		return
	}
	p.mu.Lock()
	u := p.users[userID]
	if u == nil {
		u = &pendingUser{}
		p.users[userID] = u
	}
	if len(u.waiting) >= pendingWaitingCap {
		u.waiting = u.waiting[1:]
		slog.Warn("pending queue full, dropping oldest", slog.Int64("user_id", userID))
	}
	u.waiting = append(u.waiting, job)
	start := !u.draining
	if start {
		u.draining = true
	}
	p.mu.Unlock()
	if start {
		go p.drain(userID)
	}
}

func (p *PendingCommands) drain(userID int64) {
	for {
		p.mu.Lock()
		u := p.users[userID]
		if u == nil || len(u.waiting) == 0 {
			if u != nil {
				u.draining = false
				delete(p.users, userID)
			}
			p.idle.Broadcast()
			p.mu.Unlock()
			return
		}
		job := u.waiting[0]
		u.waiting = u.waiting[1:]
		p.mu.Unlock()
		job()
	}
}

func (p *PendingCommands) wait(userID int64) {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	for {
		u := p.users[userID]
		if u == nil || !u.draining {
			return
		}
		p.idle.Wait()
	}
}

func (p *PendingCommands) middleware() bot.Middleware {
	return func(next bot.HandlerFunc) bot.HandlerFunc {
		return func(ctx context.Context, b *bot.Bot, update *models.Update) {
			from := sender(update)
			if from == nil || from.IsBot || from.ID == 0 {
				next(ctx, b, update)
				return
			}
			p.enqueue(from.ID, func() { next(ctx, b, update) })
		}
	}
}
