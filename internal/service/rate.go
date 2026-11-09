package service

import (
	"context"
	"errors"
	"math/big"
)

// ErrRateUnavailable is a missing, non-positive, or unparsable rate, or a fetch with no cached table.
var ErrRateUnavailable = errors.New("exchange rate unavailable")

type RateQuote struct {
	From          string
	To            string
	DestPerSource *big.Rat
}

type RateProvider interface {
	Quote(ctx context.Context, from, to string) (RateQuote, error)
}
