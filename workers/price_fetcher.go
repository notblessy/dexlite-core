package workers

import (
	"context"
	"log"
	"time"

	"github.com/notblessy/dexlite/models"
	"github.com/notblessy/dexlite/services"
	"gorm.io/gorm"
)

type PriceFetcher struct {
	db     *gorm.DB
	client *services.HyperLiquidClient
	coins  []string
}

func NewPriceFetcher(db *gorm.DB) *PriceFetcher {
	return &PriceFetcher{
		db:     db,
		client: services.NewHyperLiquidClient(),
		coins:  []string{"BTC", "ETH", "SOL", "ARB", "AVAX"},
	}
}

func (pf *PriceFetcher) Start(ctx context.Context) {
	// Then run every hour
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Price fetcher worker shutting down...")
			return
		case <-ticker.C:
			pf.FetchPrices()
		}
	}
}

// FetchPrices fetches and saves prices for all tracked coins
func (pf *PriceFetcher) FetchPrices() {
	pf.fetchPrices()
}

func (pf *PriceFetcher) fetchPrices() {
	log.Println("Starting price fetch for tracked coins...")

	for _, coin := range pf.coins {
		price, err := pf.client.GetPrice(coin)
		if err != nil {
			log.Printf("Error fetching price for %s: %v", coin, err)
			continue
		}

		coinPrice := models.CoinPrice{
			Coin:  coin,
			Price: price,
		}

		if err := pf.db.Create(&coinPrice).Error; err != nil {
			log.Printf("Error saving price for %s: %v", coin, err)
			continue
		}

		log.Printf("Successfully saved %s price: %.8f", coin, price)
	}

	log.Println("Price fetch completed")
}
