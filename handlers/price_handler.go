package handlers

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/notblessy/dexlite/models"
	"gorm.io/gorm"
)

type PriceHandler struct {
	db *gorm.DB
}

func NewPriceHandler(db *gorm.DB) *PriceHandler {
	return &PriceHandler{
		db: db,
	}
}

type PriceResponse struct {
	Coin      string    `json:"coin"`
	Price     float64   `json:"price"`
	CreatedAt time.Time `json:"created_at"`
}

type PriceComparisonResponse struct {
	Coin   string          `json:"coin"`
	Prices []PriceResponse `json:"prices"`
	Count  int64           `json:"count"`
}

// GetPriceComparison returns prices for a coin within the last 24 hours
// GET /api/prices/:coin
func (h *PriceHandler) GetPriceComparison(c echo.Context) error {
	coin := c.Param("coin")
	if coin == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "coin symbol is required",
		})
	}

	// Calculate 24 hours ago
	twentyFourHoursAgo := time.Now().Add(-24 * time.Hour)

	var prices []models.CoinPrice
	var count int64

	// Query prices for the coin within the last 24 hours
	query := h.db.Where("coin = ? AND created_at >= ?", coin, twentyFourHoursAgo)

	// Count first
	if err := query.Model(&models.CoinPrice{}).Count(&count).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to count prices",
		})
	}

	// Then fetch the data
	if err := query.Order("created_at DESC").Find(&prices).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to fetch prices",
		})
	}

	// Convert to response format
	priceResponses := make([]PriceResponse, len(prices))
	for i, price := range prices {
		priceResponses[i] = PriceResponse{
			Coin:      price.Coin,
			Price:     price.Price,
			CreatedAt: price.CreatedAt,
		}
	}

	response := PriceComparisonResponse{
		Coin:   coin,
		Prices: priceResponses,
		Count:  count,
	}

	return c.JSON(http.StatusOK, response)
}
