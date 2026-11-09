package fx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"strings"
	"time"

	"golang.org/x/sync/singleflight"

	"finbot/internal/ports"
	"finbot/internal/service"
)

const (
	cacheKey    = "fx:usd"
	userAgent   = "finbot"
	primaryURL  = "https://cdn.jsdelivr.net/npm/@fawazahmed0/currency-api@latest/v1/currencies/usd.min.json"
	fallbackURL = "https://latest.currency-api.pages.dev/v1/currencies/usd.min.json"
)

type Cache interface {
	Get(ctx context.Context, key string) ([]byte, bool, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
}

type Doer interface {
	Do(*http.Request) (*http.Response, error)
}

var (
	_ service.RateProvider = (*Provider)(nil)
	_ Doer                 = (*http.Client)(nil)
	_ Cache                = ports.Cache(nil)
)

var errBadRate = errors.New("bad rate")

type Provider struct {
	cache       Cache
	doer        Doer
	freshTTL    time.Duration
	cacheTTL    time.Duration
	httpTimeout time.Duration
	now         func() time.Time
	flight      singleflight.Group
}

type snapshot struct {
	FetchedAt time.Time
	Rates     map[string]string
}

type snapshotJSON struct {
	FetchedAt string            `json:"fetched_at"`
	Rates     map[string]string `json:"rates"`
}

func New(cfg Config, cache Cache, doer Doer) (*Provider, error) {
	if err := cfg.ValidateWithContext(context.Background()); err != nil {
		return nil, fmt.Errorf("fx config: %w", err)
	}
	fresh, cacheTTL, timeout, err := cfg.durations()
	if err != nil {
		return nil, fmt.Errorf("fx config: %w", err)
	}
	return &Provider{
		cache:       cache,
		doer:        doer,
		freshTTL:    fresh,
		cacheTTL:    cacheTTL,
		httpTimeout: timeout,
		now:         func() time.Time { return time.Now().UTC() },
	}, nil
}

func (p *Provider) Quote(ctx context.Context, from, to string) (service.RateQuote, error) {
	from = canonical(from)
	to = canonical(to)
	if err := ctx.Err(); err != nil {
		return service.RateQuote{}, fmt.Errorf("quote %s/%s: %w", from, to, err)
	}
	if from == to {
		return service.RateQuote{From: from, To: to, DestPerSource: big.NewRat(1, 1)}, nil
	}

	rates, err := p.rates(ctx)
	if err != nil {
		return service.RateQuote{}, quoteErr(ctx, from, to, err)
	}
	ratio, err := destPerSource(rates, from, to)
	if err != nil {
		return service.RateQuote{}, quoteErr(ctx, from, to, err)
	}
	return service.RateQuote{From: from, To: to, DestPerSource: ratio}, nil
}

func (p *Provider) rates(ctx context.Context) (map[string]string, error) {
	v, err, _ := p.flight.Do(cacheKey, func() (any, error) {
		return p.load(ctx)
	})
	if err != nil {
		return nil, fmt.Errorf("fx rates: %w", err)
	}
	rates, ok := v.(map[string]string)
	if !ok {
		return nil, fmt.Errorf("fx rates: empty")
	}
	return rates, nil
}

func (p *Provider) load(ctx context.Context) (map[string]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("fx rates: %w", err)
	}
	cached, ok := p.readCache(ctx)
	if ok && p.fresh(cached.FetchedAt) {
		return cached.Rates, nil
	}

	fetched, err := p.fetch(ctx)
	if err != nil {
		if ok && ctx.Err() == nil {
			return cached.Rates, nil
		}
		return nil, err
	}
	if err := p.writeCache(ctx, fetched); err != nil {
		slog.Error("fx cache set", slog.Any("err", err))
	}
	return fetched, nil
}

func (p *Provider) fresh(at time.Time) bool {
	return p.now().Sub(at) < p.freshTTL
}

func (p *Provider) readCache(ctx context.Context) (snapshot, bool) {
	raw, ok, err := p.cache.Get(ctx, cacheKey)
	if err != nil {
		slog.Error("fx cache get", slog.Any("err", err))
		return snapshot{}, false
	}
	if !ok {
		return snapshot{}, false
	}
	snap, err := decodeSnapshot(raw)
	if err != nil {
		slog.Error("fx cache decode", slog.Any("err", err))
		return snapshot{}, false
	}
	return snap, true
}

func (p *Provider) writeCache(ctx context.Context, rates map[string]string) error {
	raw, err := json.Marshal(snapshotJSON{
		FetchedAt: p.now().UTC().Format(time.RFC3339),
		Rates:     rates,
	})
	if err != nil {
		return fmt.Errorf("fx cache encode: %w", err)
	}
	if err := p.cache.Set(ctx, cacheKey, raw, p.cacheTTL); err != nil {
		return fmt.Errorf("fx cache set: %w", err)
	}
	return nil
}

func (p *Provider) fetch(ctx context.Context) (map[string]string, error) {
	rates, err := p.fetchOne(ctx, primaryURL)
	if err == nil {
		return rates, nil
	}
	rates, err = p.fetchOne(ctx, fallbackURL)
	if err != nil {
		return nil, fmt.Errorf("fx fetch: %w", err)
	}
	return rates, nil
}

func (p *Provider) fetchOne(ctx context.Context, rawURL string) (map[string]string, error) {
	ctx, cancel := context.WithTimeout(ctx, p.httpTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("fx request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := p.doer.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fx fetch: %w", err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			slog.Error("fx body close", slog.Any("err", cerr))
		}
	}()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fx fetch: %s", resp.Status)
	}
	return decodeFeed(resp.Body)
}

func decodeFeed(r io.Reader) (map[string]string, error) {
	dec := json.NewDecoder(r)
	dec.UseNumber()
	var body struct {
		USD map[string]json.Number `json:"usd"`
	}
	if err := dec.Decode(&body); err != nil {
		return nil, fmt.Errorf("fx json: %w", err)
	}
	rates := make(map[string]string, len(body.USD))
	for code, num := range body.USD {
		code = canonical(code)
		if code == "" {
			continue
		}
		rates[code] = num.String()
	}
	return rates, nil
}

func decodeSnapshot(raw []byte) (snapshot, error) {
	var body snapshotJSON
	if err := json.Unmarshal(raw, &body); err != nil {
		return snapshot{}, fmt.Errorf("fx cache json: %w", err)
	}
	at, err := time.Parse(time.RFC3339, body.FetchedAt)
	if err != nil {
		return snapshot{}, fmt.Errorf("fx cache time: %w", err)
	}
	if body.Rates == nil {
		return snapshot{}, fmt.Errorf("fx cache: rates missing")
	}
	return snapshot{FetchedAt: at, Rates: body.Rates}, nil
}

func destPerSource(rates map[string]string, from, to string) (*big.Rat, error) {
	fromRate, err := positiveRate(rates[from])
	if err != nil {
		return nil, err
	}
	toRate, err := positiveRate(rates[to])
	if err != nil {
		return nil, err
	}
	return new(big.Rat).Quo(toRate, fromRate), nil
}

func positiveRate(raw string) (*big.Rat, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, errBadRate
	}
	rate, ok := new(big.Rat).SetString(raw)
	if !ok || rate.Sign() <= 0 {
		return nil, errBadRate
	}
	return rate, nil
}

func canonical(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

func quoteErr(ctx context.Context, from, to string, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return fmt.Errorf("quote %s/%s: %w", from, to, ctxErr)
	}
	return fmt.Errorf("quote %s/%s: %w", from, to, errors.Join(service.ErrRateUnavailable, err))
}
