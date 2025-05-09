package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"

	"stonk-risk-management/pkg/database"
	"stonk-risk-management/pkg/models"

	"github.com/dgraph-io/badger/v3"
)

// App struct
type App struct {
	ctx                context.Context
	db                 *database.DB
	riskRepository     *database.RiskRepository
	stockRepository    *database.StockRepository
	tradeRepository    *database.TradeRepository
	positionRepository *database.PositionRepository
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Setup database
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Failed to get user home directory: %v", err)
	}

	dbPath := filepath.Join(homeDir, ".options-risk-management")
	db, err := database.New(dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	a.db = db
	a.riskRepository = database.NewRiskRepository(db)
	a.stockRepository = database.NewStockRepository(db)
	a.tradeRepository = database.NewTradeRepository(db)
	a.positionRepository = database.NewPositionRepository(db)
}

// shutdown is called when the app is closing
func (a *App) shutdown(ctx context.Context) {
	if a.db != nil {
		_ = a.db.Close()
	}
}

// GetRiskAssessments returns all risk assessments
func (a *App) GetRiskAssessments() ([]*models.RiskAssessment, error) {
	return a.riskRepository.GetAll()
}

// SaveRiskAssessment saves a risk assessment
func (a *App) SaveRiskAssessment(assessment *models.RiskAssessment) error {
	return a.riskRepository.Save(assessment)
}

// DeleteRiskAssessment deletes a risk assessment
func (a *App) DeleteRiskAssessment(id string) error {
	return a.riskRepository.Delete(id)
}

// GetStockRatings returns all stock ratings
func (a *App) GetStockRatings() ([]*models.StockRating, error) {
	return a.stockRepository.GetAll()
}

// GetStockRatingsByDate returns stock ratings for a specific date
func (a *App) GetStockRatingsByDate(date time.Time) ([]*models.StockRating, error) {
	return a.stockRepository.GetByDate(date)
}

// SaveStockRating saves a stock rating
func (a *App) SaveStockRating(rating *models.StockRating) error {
	return a.stockRepository.Save(rating)
}

// DeleteStockRating deletes a stock rating
func (a *App) DeleteStockRating(id string) error {
	return a.stockRepository.Delete(id)
}

// GetLatestMarketRating returns the most recent market rating
func (a *App) GetLatestMarketRating() (*models.StockRating, error) {
	// Get all ratings for the "MARKET" symbol
	ratings, err := a.stockRepository.GetBySymbol("MARKET")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch market ratings: %w", err)
	}

	// If no ratings found, return nil
	if len(ratings) == 0 {
		return nil, nil
	}

	// Sort by date (descending) to get the most recent one
	sort.Slice(ratings, func(i, j int) bool {
		return ratings[i].Date.After(ratings[j].Date)
	})

	// Return the first (most recent) rating
	return ratings[0], nil
}

// GetLatestSectorRating returns the most recent rating for a specific sector
func (a *App) GetLatestSectorRating(sector string) (*models.StockRating, error) {
	if sector == "" {
		return nil, fmt.Errorf("sector cannot be empty")
	}

	// Get all ratings
	allRatings, err := a.stockRepository.GetAll()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ratings: %w", err)
	}

	// Filter for sector ratings (where symbol is "SECTOR" and sector matches)
	var sectorRatings []*models.StockRating
	for _, rating := range allRatings {
		if rating.Symbol == "SECTOR" && rating.Sector == sector {
			sectorRatings = append(sectorRatings, rating)
		}
	}

	// If no ratings found, return nil
	if len(sectorRatings) == 0 {
		return nil, nil
	}

	// Sort by date (descending) to get the most recent one
	sort.Slice(sectorRatings, func(i, j int) bool {
		return sectorRatings[i].Date.After(sectorRatings[j].Date)
	})

	// Return the first (most recent) rating
	return sectorRatings[0], nil
}

// GetLatestStockRating returns the most recent rating for a specific stock symbol
func (a *App) GetLatestStockRating(symbol string) (*models.StockRating, error) {
	if symbol == "" {
		return nil, fmt.Errorf("symbol cannot be empty")
	}

	// Get all ratings for the provided symbol
	ratings, err := a.stockRepository.GetBySymbol(symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ratings for symbol %s: %w", symbol, err)
	}

	// If no ratings found, return nil
	if len(ratings) == 0 {
		return nil, nil
	}

	// Sort by date (descending) to get the most recent one
	sort.Slice(ratings, func(i, j int) bool {
		return ratings[i].Date.After(ratings[j].Date)
	})

	// Return the first (most recent) rating
	return ratings[0], nil
}

// GetTrades returns all trades
func (a *App) GetTrades() ([]*models.Trade, error) {
	return a.tradeRepository.GetAll()
}

// SaveTrade saves a trade
func (a *App) SaveTrade(trade *models.Trade) error {
	// Basic validation before saving
	if trade.Symbol == "" || trade.Sector == "" || trade.Strategy == "" || trade.Type == "" {
		return fmt.Errorf("invalid trade data: missing required fields")
	}

	// For backward compatibility, always set legNumber to 1
	trade.LegNumber = 1

	return a.tradeRepository.Save(trade)
}

// DeleteTrade deletes all legs associated with a trade ID
func (a *App) DeleteTrade(id string) error {
	return a.tradeRepository.Delete(id)
}

// RunDatabaseMaintenance performs database maintenance tasks including garbage collection
// Returns a success message or error message
func (a *App) RunDatabaseMaintenance() string {
	if a.db == nil {
		return "Error: Database not initialized"
	}

	err := a.db.RunGC()
	if err != nil {
		if err == badger.ErrNoRewrite {
			return "No garbage collection needed at this time. Database is already optimized."
		}
		return fmt.Sprintf("Error during garbage collection: %v", err)
	}

	return "Database maintenance completed successfully. Freed up unused space."
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

// GetPositionSettings returns the saved position settings
func (a *App) GetPositionSettings() (*models.PositionSettings, error) {
	return a.positionRepository.GetSettings()
}

// SavePositionSettings saves the position settings
func (a *App) SavePositionSettings(settings *models.PositionSettings) error {
	return a.positionRepository.SaveSettings(settings)
}
