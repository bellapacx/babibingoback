package repository

import (
	"babibingo/internal/models"
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// internal/repository/db.go

func InitDB(dsn string) (*gorm.DB, error) {
    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
    if err != nil {
        return nil, fmt.Errorf("failed to connect to database: %w", err)
    }

    // ✅ AutoMigrate for all models
    if err := db.AutoMigrate(
        &models.User{},
        &models.Game{},
        &models.Card{},
        &models.GamePlayer{},
    ); err != nil {
        return nil, fmt.Errorf("failed to migrate: %w", err)
    }

    // ✅ Manually handle Transaction to avoid constraint conflicts
    if err := ensureTransactionTable(db); err != nil {
        return nil, err
    }

    return db, nil
}

func ensureTransactionTable(db *gorm.DB) error {
    // Check if transactions table exists
    var tableExists bool
    db.Raw("SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'transactions')").Scan(&tableExists)

    if !tableExists {
        // Create the table with AutoMigrate
        return db.AutoMigrate(&models.Transaction{})
    }

    // Check if the correct constraint exists
    var constraintExists bool
    db.Raw(`
        SELECT EXISTS (
            SELECT 1 FROM pg_constraint 
            WHERE conname = 'unique_transactions_reference' 
            AND conrelid = 'transactions'::regclass
        )
    `).Scan(&constraintExists)

    // Add the constraint if it doesn't exist
    if !constraintExists {
        // First check if any other unique constraint exists
        var otherConstraint string
        db.Raw(`
            SELECT conname FROM pg_constraint 
            WHERE conrelid = 'transactions'::regclass 
            AND contype = 'u' 
            AND pg_get_constraintdef(oid) LIKE '%reference%'
            LIMIT 1
        `).Scan(&otherConstraint)

        if otherConstraint != "" {
            // Drop the old constraint
            db.Exec(fmt.Sprintf("ALTER TABLE transactions DROP CONSTRAINT IF EXISTS %s", otherConstraint))
        }

        // Add the new constraint
        if err := db.Exec("ALTER TABLE transactions ADD CONSTRAINT unique_transactions_reference UNIQUE (reference)").Error; err != nil {
            // If error is because of duplicates, log and continue
            return fmt.Errorf("failed to add unique constraint: %w", err)
        }
    }

    return nil
}