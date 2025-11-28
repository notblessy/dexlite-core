package workers

import (
	"context"
	"log"
	"time"

	"github.com/notblessy/dexlite/models"
	"gorm.io/gorm"
)

type CleanupWorker struct {
	db *gorm.DB
}

func NewCleanupWorker(db *gorm.DB) *CleanupWorker {
	return &CleanupWorker{
		db: db,
	}
}

func (cw *CleanupWorker) Start(ctx context.Context) {
	// Run immediately on start
	cw.cleanup()

	// Then run every hour (to keep data fresh)
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Cleanup worker shutting down...")
			return
		case <-ticker.C:
			cw.cleanup()
		}
	}
}

func (cw *CleanupWorker) cleanup() {
	log.Println("Starting cleanup of old coin prices...")

	// Delete records older than 2 days
	cutoff := time.Now().AddDate(0, 0, -2)
	
	result := cw.db.Where("created_at < ?", cutoff).Delete(&models.CoinPrice{})
	if result.Error != nil {
		log.Printf("Error during cleanup: %v", result.Error)
		return
	}

	log.Printf("Cleanup completed. Deleted %d records older than %s", result.RowsAffected, cutoff.Format(time.RFC3339))
}

