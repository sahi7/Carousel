package payroll

import (
    "context"
    "encoding/json"
    "fmt"
    "log"

    "github.com/confluentinc/confluent-kafka-go/v2/kafka"
    "carousel/internal/config"
    "carousel/internal/db"
    "carousel/internal/services"
)

// Handler processes payroll events
type Handler struct {
    config    config.Config
    db        *db.PayrollDB
    processor *Processor
}

// NewHandler creates a payroll handler
func NewHandler(cfg config.Config) services.EventHandler {
    baseDB, err := db.NewPostgres(cfg.PostgresDSN, cfg.RedisAddr)
    if err != nil {
        log.Fatalf("Failed to initialize database: %v", err)
    }
    dbClient := db.NewPayrollDB(baseDB) // Initialize PayrollDB
    return &Handler{
        config:    cfg,
        db:        dbClient,
        processor: NewProcessor(dbClient, cfg),
    }
}

// Handle processes a payroll.generate event
func (h *Handler) Handle(ctx context.Context, msg *kafka.Message) error {
    var event struct {
        PeriodID int64 `json:"period_id"`
        Month    int   `json:"month"`
        Year     int   `json:"year"`
        RestaurantID int64 `json:"restaurant_id"`
        BranchID int64 `json:"branch_id"`
        CompanyID int64 `json:"company_id"`
    }
    if err := json.Unmarshal(msg.Value, &event); err != nil {
        return fmt.Errorf("Handle: failed to parse event: %w", err)
    }

    periodEvent := map[string]interface{}{
        "period_id":  event.PeriodID,
        "month":      event.Month,
        "year":       event.Year,
        "restaurant_id": event.RestaurantID,
        "branch_id":     event.BranchID,
        "company_id":     event.CompanyID,
    }
    if err := h.processor.ProcessPayroll(ctx, periodEvent); err != nil {
        return err
    }
    fmt.Printf("periodEvent2: %+v\n", periodEvent)
    return nil
    // return h.processor.ProcessPayroll(ctx, periodEvent)
}