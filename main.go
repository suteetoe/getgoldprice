// main.go
package main

import (
	"getgoldprice/internal/getgoldprice"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// PriceCache holds the latest gold price in memory
type PriceCache struct {
	mu       sync.RWMutex
	headline *getgoldprice.Headline
	lastErr  error
}

// Global price cache
var priceCache = &PriceCache{}

func main() {
	// Start background price fetcher
	go startPriceFetcher()

	// Create Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Routes
	e.GET("/goldprice", getGoldPriceHandler)
	e.GET("/health", healthHandler)

	// Start server
	log.Println("Starting server on :8080...")
	e.Logger.Fatal(e.Start(":8080"))
}

// startPriceFetcher runs in background and fetches gold prices every 5 seconds
func startPriceFetcher() {
	log.Println("Starting background price fetcher...")
	getprice := getgoldprice.GetGoldPrice{}

	// Fetch initial price
	updatePrice(&getprice)

	// Set up ticker for every 5 seconds
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		updatePrice(&getprice)
	}
}

// updatePrice fetches the latest price and updates the cache
func updatePrice(getprice *getgoldprice.GetGoldPrice) {
	headline, err := getprice.GetLastPrice()

	priceCache.mu.Lock()
	defer priceCache.mu.Unlock()

	if err != nil {
		log.Printf("Error fetching gold price: %v", err)
		priceCache.lastErr = err
	} else {
		priceCache.headline = headline
		priceCache.lastErr = nil
		log.Printf("Price updated: Bar Sell: %.2f, Bar Buy: %.2f (Fetched: %s)",
			headline.BarSell, headline.BarBuy, headline.Fetched.Format("15:04:05"))
	}
}

// getGoldPriceHandler serves the cached gold price
func getGoldPriceHandler(c echo.Context) error {
	priceCache.mu.RLock()
	defer priceCache.mu.RUnlock()

	// Check if we have cached data
	if priceCache.headline == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error":   "Price data not available yet",
			"message": "Please wait for the first price fetch to complete",
		})
	}

	// Check if last fetch had an error
	if priceCache.lastErr != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error":      "Failed to fetch latest price",
			"lastError":  priceCache.lastErr.Error(),
			"cachedData": priceCache.headline,
		})
	}

	// Return the cached headline data as JSON
	return c.JSON(http.StatusOK, priceCache.headline)
}

// healthHandler provides health check endpoint
func healthHandler(c echo.Context) error {
	priceCache.mu.RLock()
	defer priceCache.mu.RUnlock()

	status := "healthy"
	if priceCache.headline == nil {
		status = "initializing"
	} else if priceCache.lastErr != nil {
		status = "degraded"
	}

	healthInfo := map[string]interface{}{
		"status":    status,
		"hasData":   priceCache.headline != nil,
		"lastError": nil,
	}

	if priceCache.lastErr != nil {
		healthInfo["lastError"] = priceCache.lastErr.Error()
	}

	if priceCache.headline != nil {
		healthInfo["lastFetch"] = priceCache.headline.Fetched
		healthInfo["dataAge"] = time.Since(priceCache.headline.Fetched).String()
	}

	return c.JSON(http.StatusOK, healthInfo)
}
