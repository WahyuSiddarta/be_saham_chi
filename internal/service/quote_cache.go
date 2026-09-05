package service

import (
	"sync"
	"time"

	"github.com/WahyuSiddarta/be_saham_chi/internal/repository"
)

type cachedQuote struct {
	quote     repository.MarketPrice
	expiresAt time.Time
}

type quoteCache struct {
	mu     sync.Mutex
	quotes map[string]cachedQuote
}

func newQuoteCache() *quoteCache {
	return &quoteCache{
		quotes: make(map[string]cachedQuote),
	}
}

func (c *quoteCache) Get(symbol string, now time.Time) (repository.MarketPrice, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	cached, ok := c.quotes[symbol]
	if !ok || cached.expiresAt.IsZero() || !now.Before(cached.expiresAt) {
		return repository.MarketPrice{}, false
	}

	return cached.quote, true
}

func (c *quoteCache) Set(symbol string, quote repository.MarketPrice, expiresAt time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.quotes[symbol] = cachedQuote{
		quote:     quote,
		expiresAt: expiresAt,
	}
}
