package fx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net/http"
	"os"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"finbot/internal/adapter/fx/mocks"
	"finbot/internal/service"
)

const (
	testFresh = "1h"
	testCache = "24h"
	testHTTP  = "3s"
)

var fixedNow = time.Date(2026, 11, 9, 18, 0, 0, 0, time.UTC)

func TestQuoteMatchesFixture(t *testing.T) {
	raw := readFixture(t)
	rats, text := fixtureRates(t, raw)
	tests := []struct {
		from string
		to   string
	}{
		{from: "USD", to: "UAH"},
		{from: "EUR", to: "UAH"},
		{from: "UAH", to: "USD"},
	}
	for _, tt := range tests {
		t.Run(tt.from+"/"+tt.to, func(t *testing.T) {
			cache := mocks.NewMockCache(t)
			doer := mocks.NewMockDoer(t)
			p := newProvider(t, cache, doer)
			cache.EXPECT().Get(mock.Anything, cacheKey).Return(nil, false, nil).Once()
			doer.EXPECT().Do(mock.Anything).RunAndReturn(func(req *http.Request) (*http.Response, error) {
				checkFetch(t, req, primaryURL, p.httpTimeout)
				return okResp(raw), nil
			}).Once()
			cache.EXPECT().
				Set(mock.Anything, cacheKey, mock.Anything, mustDuration(t, testCache)).
				RunAndReturn(func(_ context.Context, _ string, value []byte, _ time.Duration) error {
					assertStored(t, value, text)
					return nil
				}).Once()

			got, err := p.Quote(context.Background(), tt.from, tt.to)
			require.NoError(t, err)
			require.Equal(t, tt.from, got.From)
			require.Equal(t, tt.to, got.To)
			require.NotNil(t, got.DestPerSource)
			require.Zero(t, got.DestPerSource.Cmp(new(big.Rat).Quo(rats[tt.to], rats[tt.from])))
		})
	}
}

func TestQuoteEqualCodesSkipIO(t *testing.T) {
	tests := []struct {
		from string
		to   string
	}{
		{from: "USD", to: "USD"},
		{from: "usd", to: " USD "},
	}
	for _, tt := range tests {
		t.Run(tt.from+"/"+tt.to, func(t *testing.T) {
			cache := mocks.NewMockCache(t)
			doer := mocks.NewMockDoer(t)
			p := newProvider(t, cache, doer)

			got, err := p.Quote(context.Background(), tt.from, tt.to)
			require.NoError(t, err)
			require.Equal(t, "USD", got.From)
			require.Equal(t, "USD", got.To)
			require.Zero(t, got.DestPerSource.Cmp(big.NewRat(1, 1)))
			cache.AssertNotCalled(t, "Get", mock.Anything, mock.Anything)
			doer.AssertNotCalled(t, "Do", mock.Anything)
		})
	}
}

func TestQuoteFreshCacheSkipsHTTP(t *testing.T) {
	cache := mocks.NewMockCache(t)
	doer := mocks.NewMockDoer(t)
	p := newProvider(t, cache, doer)
	rates := map[string]string{"USD": "1", "UAH": "45.28"}
	at := fixedNow.Add(-mustDuration(t, testFresh) + time.Second)
	cache.EXPECT().Get(mock.Anything, cacheKey).Return(cacheBody(t, at, rates), true, nil).Once()

	got, err := p.Quote(context.Background(), "USD", "UAH")
	require.NoError(t, err)
	require.NotNil(t, got.DestPerSource)
	require.Zero(t, got.DestPerSource.Cmp(ratio("45.28", "1")))
	doer.AssertNotCalled(t, "Do", mock.Anything)
	cache.AssertNotCalled(t, "Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestQuoteStaleCacheSurvivesHTTPFailure(t *testing.T) {
	cache := mocks.NewMockCache(t)
	doer := mocks.NewMockDoer(t)
	p := newProvider(t, cache, doer)
	rates := map[string]string{"USD": "1", "UAH": "45.28"}
	at := fixedNow.Add(-mustDuration(t, testFresh))
	cache.EXPECT().Get(mock.Anything, cacheKey).Return(cacheBody(t, at, rates), true, nil).Once()
	doer.EXPECT().Do(mock.Anything).Return(nil, errors.New("dial")).Times(2)

	got, err := p.Quote(context.Background(), "USD", "UAH")
	require.NoError(t, err)
	require.NotNil(t, got.DestPerSource)
	require.Zero(t, got.DestPerSource.Cmp(ratio("45.28", "1")))
	cache.AssertNotCalled(t, "Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestQuoteEmptyCacheBothFetchesFail(t *testing.T) {
	cache := mocks.NewMockCache(t)
	doer := mocks.NewMockDoer(t)
	p := newProvider(t, cache, doer)
	cache.EXPECT().Get(mock.Anything, cacheKey).Return(nil, false, nil).Once()
	doer.EXPECT().Do(mock.Anything).Return(nil, errors.New("dial")).Times(2)

	_, err := p.Quote(context.Background(), "USD", "UAH")
	require.ErrorIs(t, err, service.ErrRateUnavailable)
	cache.AssertNotCalled(t, "Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestQuotePrimaryFailureUsesFallback(t *testing.T) {
	raw := readFixture(t)
	rats, text := fixtureRates(t, raw)
	tests := []struct {
		name    string
		primary func() (*http.Response, error)
	}{
		{name: "transport", primary: func() (*http.Response, error) { return nil, errors.New("dial") }},
		{name: "non-200", primary: func() (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusBadGateway,
				Status:     "502 Bad Gateway",
				Body:       io.NopCloser(strings.NewReader("no")),
				Header:     make(http.Header),
			}, nil
		}},
		{name: "invalid json", primary: func() (*http.Response, error) { return okResp([]byte("{")), nil }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := mocks.NewMockCache(t)
			doer := mocks.NewMockDoer(t)
			p := newProvider(t, cache, doer)
			cache.EXPECT().Get(mock.Anything, cacheKey).Return(nil, false, nil).Once()
			doer.EXPECT().Do(mock.Anything).RunAndReturn(func(req *http.Request) (*http.Response, error) {
				checkFetch(t, req, primaryURL, p.httpTimeout)
				return tt.primary()
			}).Once()
			doer.EXPECT().Do(mock.Anything).RunAndReturn(func(req *http.Request) (*http.Response, error) {
				checkFetch(t, req, fallbackURL, p.httpTimeout)
				return okResp(raw), nil
			}).Once()
			cache.EXPECT().
				Set(mock.Anything, cacheKey, mock.Anything, mustDuration(t, testCache)).
				RunAndReturn(func(_ context.Context, _ string, value []byte, _ time.Duration) error {
					assertStored(t, value, text)
					return nil
				}).Once()

			got, err := p.Quote(context.Background(), "EUR", "UAH")
			require.NoError(t, err)
			require.Zero(t, got.DestPerSource.Cmp(new(big.Rat).Quo(rats["UAH"], rats["EUR"])))
		})
	}
}

func TestQuoteMissingCodeDoesNotFallback(t *testing.T) {
	raw := readFixture(t)
	cache := mocks.NewMockCache(t)
	doer := mocks.NewMockDoer(t)
	p := newProvider(t, cache, doer)
	cache.EXPECT().Get(mock.Anything, cacheKey).Return(nil, false, nil).Once()
	doer.EXPECT().Do(mock.Anything).RunAndReturn(func(req *http.Request) (*http.Response, error) {
		require.Equal(t, primaryURL, req.URL.String())
		return okResp(raw), nil
	}).Once()
	cache.EXPECT().Set(mock.Anything, cacheKey, mock.Anything, mustDuration(t, testCache)).Return(nil).Once()

	_, err := p.Quote(context.Background(), "USD", "GBP")
	require.ErrorIs(t, err, service.ErrRateUnavailable)
}

func TestQuoteRejectsUnusableRate(t *testing.T) {
	tests := []struct {
		name  string
		rates map[string]string
		from  string
		to    string
	}{
		{name: "missing", rates: map[string]string{"USD": "1"}, from: "USD", to: "UAH"},
		{name: "zero", rates: map[string]string{"USD": "1", "UAH": "0"}, from: "USD", to: "UAH"},
		{name: "negative", rates: map[string]string{"USD": "1", "UAH": "-1"}, from: "USD", to: "UAH"},
		{name: "unparsable", rates: map[string]string{"USD": "1", "UAH": "nope"}, from: "USD", to: "UAH"},
		{name: "zero source", rates: map[string]string{"USD": "1", "UAH": "0"}, from: "UAH", to: "USD"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := mocks.NewMockCache(t)
			doer := mocks.NewMockDoer(t)
			p := newProvider(t, cache, doer)
			at := fixedNow.Add(-mustDuration(t, testFresh) + time.Second)
			cache.EXPECT().Get(mock.Anything, cacheKey).Return(cacheBody(t, at, tt.rates), true, nil).Once()

			_, err := p.Quote(context.Background(), tt.from, tt.to)
			require.ErrorIs(t, err, service.ErrRateUnavailable)
			doer.AssertNotCalled(t, "Do", mock.Anything)
		})
	}
}

func TestQuoteSingleflightSharesOneFetch(t *testing.T) {
	raw := readFixture(t)
	rats, _ := fixtureRates(t, raw)
	synctest.Test(t, func(t *testing.T) {
		cache := mocks.NewMockCache(t)
		doer := mocks.NewMockDoer(t)
		p := newProvider(t, cache, doer)
		release := make(chan struct{})
		cache.EXPECT().Get(mock.Anything, cacheKey).Return(nil, false, nil).Once()
		doer.EXPECT().Do(mock.Anything).RunAndReturn(func(req *http.Request) (*http.Response, error) {
			require.Equal(t, primaryURL, req.URL.String())
			<-release
			return okResp(raw), nil
		}).Once()
		cache.EXPECT().Set(mock.Anything, cacheKey, mock.Anything, mustDuration(t, testCache)).Return(nil).Once()

		type gotQuote struct {
			quote service.RateQuote
			err   error
		}
		out := make(chan gotQuote, 2)
		go func() {
			quote, err := p.Quote(context.Background(), "USD", "UAH")
			out <- gotQuote{quote: quote, err: err}
		}()
		go func() {
			quote, err := p.Quote(context.Background(), "EUR", "UAH")
			out <- gotQuote{quote: quote, err: err}
		}()
		synctest.Wait()
		close(release)
		synctest.Wait()

		for range 2 {
			got := <-out
			require.NoError(t, got.err)
			require.Zero(t, got.quote.DestPerSource.Cmp(new(big.Rat).Quo(rats[got.quote.To], rats[got.quote.From])))
		}
	})
}

func newProvider(t *testing.T, cache Cache, doer Doer) *Provider {
	t.Helper()
	p, err := New(Config{FreshTTL: testFresh, CacheTTL: testCache, HTTPTimeout: testHTTP}, cache, doer)
	require.NoError(t, err)
	p.now = func() time.Time { return fixedNow }
	return p
}

func checkFetch(t *testing.T, req *http.Request, rawURL string, timeout time.Duration) {
	t.Helper()
	require.Equal(t, http.MethodGet, req.Method)
	require.Equal(t, rawURL, req.URL.String())
	require.Equal(t, userAgent, req.Header.Get("User-Agent"))
	deadline, ok := req.Context().Deadline()
	require.True(t, ok)
	require.WithinDuration(t, time.Now().Add(timeout), deadline, time.Second)
}

func okResp(body []byte) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Body:       io.NopCloser(bytes.NewReader(body)),
		Header:     make(http.Header),
	}
}

func cacheBody(t *testing.T, at time.Time, rates map[string]string) []byte {
	t.Helper()
	raw, err := json.Marshal(struct {
		FetchedAt string            `json:"fetched_at"`
		Rates     map[string]string `json:"rates"`
	}{
		FetchedAt: at.UTC().Format(time.RFC3339),
		Rates:     rates,
	})
	require.NoError(t, err)
	return raw
}

func assertStored(t *testing.T, raw []byte, want map[string]string) {
	t.Helper()
	var snap struct {
		FetchedAt string            `json:"fetched_at"`
		Rates     map[string]string `json:"rates"`
	}
	require.NoError(t, json.Unmarshal(raw, &snap))
	require.Equal(t, fixedNow.UTC().Format(time.RFC3339), snap.FetchedAt)
	require.Equal(t, want, snap.Rates)
}

func fixtureRates(t *testing.T, raw []byte) (map[string]*big.Rat, map[string]string) {
	t.Helper()
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var body struct {
		USD map[string]json.Number `json:"usd"`
	}
	require.NoError(t, dec.Decode(&body))
	rats := make(map[string]*big.Rat, len(body.USD))
	text := make(map[string]string, len(body.USD))
	for code, num := range body.USD {
		code = strings.ToUpper(code)
		rate, ok := new(big.Rat).SetString(num.String())
		require.True(t, ok)
		rats[code] = rate
		text[code] = num.String()
	}
	return rats, text
}

func readFixture(t *testing.T) []byte {
	t.Helper()
	raw, err := os.ReadFile("testdata/usd.min.json")
	require.NoError(t, err)
	return raw
}

func ratio(to, from string) *big.Rat {
	toRate, ok := new(big.Rat).SetString(to)
	if !ok {
		panic(to)
	}
	fromRate, ok := new(big.Rat).SetString(from)
	if !ok {
		panic(from)
	}
	return new(big.Rat).Quo(toRate, fromRate)
}

func mustDuration(t *testing.T, raw string) time.Duration {
	t.Helper()
	d, err := time.ParseDuration(raw)
	require.NoError(t, err)
	return d
}
