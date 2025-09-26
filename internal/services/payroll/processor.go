package payroll

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	// "github.com/jackc/pgx/v5"
	"carousel/internal/config"
	"carousel/internal/db"
	"carousel/internal/models"
	"golang.org/x/sync/semaphore"
)

// Processor handles payroll computation
type Processor struct {
	db            *db.PayrollDB
	config        config.Config
	kafkaProducer *kafka.Producer
	sem           *semaphore.Weighted
}

// NewProcessor creates a new processor
func NewProcessor(db *db.PayrollDB, cfg config.Config) *Processor {
	producer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": cfg.KafkaBootstrapServers,
	})
	if err != nil {
		log.Fatalf("Failed to create Kafka producer: %v", err)
	}
	return &Processor{
		db:            db,
		config:        cfg,
		kafkaProducer: producer,
		sem:           semaphore.NewWeighted(int64(cfg.MaxWorkers)), // e.g., 100 workers
	}
}

// ProcessPayroll processes the payroll for a period
// ProcessPayroll processes the payroll for a period
func (p *Processor) ProcessPayroll(ctx context.Context, periodEvent map[string]interface{}) error {
	// Log event reception
	var branchID *int64
	if id, ok := periodEvent["branch_id"].(int64); ok {
		branchID = &id
	}

	if err := p.db.LogActivity(ctx, "event_received", map[string]interface{}{"event": periodEvent}, branchID, nil); err != nil {
		log.Printf("Failed to log event reception: %v", err)
	}

	periodID, _ := periodEvent["period_id"].(int64)
	offset := 0
	var allUserIDs []int64

	for {
		users, err := p.db.FetchUsers(ctx, periodEvent, offset)
		if err != nil {
			return fmt.Errorf("failed to fetch users: %w", err)
		}
		if len(users) == 0 {
			break
		}
		fmt.Println("Users:", users)

		// Log user fetching
		if err := p.db.LogActivity(ctx, "users_fetched", map[string]interface{}{"offset": offset, "count": len(users), "scope": getScope(periodEvent)}, nil, nil); err != nil {
			log.Printf("Failed to log users fetched: %v", err)
		}

		rules, ruleIDs, err := p.db.FetchRules(ctx, periodEvent)
		if err != nil {
			return fmt.Errorf("failed to fetch rules: %w", err)
		} else {
			fmt.Println("Rules:", rules, "Rule IDs:", ruleIDs)
		}

		mappedTargets, unmappedTargets, targets, err := p.db.FetchRuleTargets(ctx, ruleIDs, users)
		if err != nil {
			return fmt.Errorf("failed to fetch rule targets: %w", err)
		} else {
			fmt.Println("Mapped Targets:", mappedTargets)
			fmt.Println("Unmapped Targets:", unmappedTargets)
			fmt.Println("All Targets:", targets)
		}
		// fmt.Println("Mapped Targets:", mappedTargets, unmappedTargets)

		matchedOverrides, processedOverrides, overrides, updatedRules, err := p.db.FetchOverrides(ctx, ruleIDs, periodID, rules, branchID)
		if err != nil {
			return fmt.Errorf("failed to fetch overrides: %w", err)
		}
		fmt.Println("Matched Overrides:", matchedOverrides)
		fmt.Println("processedOverrides:", processedOverrides)
		fmt.Println("updatedRules:", updatedRules)

		// Log data fetching
		if err := p.db.LogActivity(ctx, "data_fetched", map[string]interface{}{"type": "rules/targets/overrides", "count": len(rules) + len(targets) + len(overrides)}, nil, nil); err != nil {
			log.Printf("Failed to log data fetched: %v", err)
		}

		var wg sync.WaitGroup
		resultChan := make(chan struct {
			user          models.CustomUser
			payrollRecord models.Record
			auditLogs     []models.AuditLog
			err           error
		}, len(users))

		for _, user := range users {
			wg.Add(1)
			if err := p.sem.Acquire(ctx, 1); err != nil {
				wg.Done()
				continue
			}
			go func(u models.CustomUser) {
				defer wg.Done()
				defer p.sem.Release(1)
				payrollRecord, auditLogs, err := computePayrollRecord(u, updatedRules, mappedTargets, unmappedTargets, matchedOverrides, processedOverrides, ruleIDs, periodID, branchID)
				resultChan <- struct {
					user          models.CustomUser
					payrollRecord models.Record
					auditLogs     []models.AuditLog
					err           error
				}{u, payrollRecord, auditLogs, err}
			}(user)
		}

		go func() {
			wg.Wait()
			close(resultChan)
		}()

		for res := range resultChan {
			if res.err != nil {
				log.Printf("Error processing user %d: %v", res.user.ID, res.err)
				continue
			}
			if err := p.db.StorePayrollRecord(ctx, res.payrollRecord); err != nil {
				log.Printf("Error storing payroll record for user %d: %v", res.user.ID, err)
				continue
			}
			for _, auditLog := range res.auditLogs {
				err := p.db.LogActivity(ctx, auditLog.ActivityType, auditLog.Details, auditLog.BranchID, auditLog.UserID)
				if err != nil {
					log.Printf("Failed to log activity for user %d: %v", res.user.ID, err)
				}
			}
			allUserIDs = append(allUserIDs, res.user.ID)
		}

		offset += 500
	}

	// Publish payroll.generated event
	if err := p.publishGeneratedEvent(ctx, periodID, allUserIDs, periodEvent); err != nil {
		log.Printf("Failed to publish payroll.generated: %v", err)
		return err
	}

	return nil
}

// computePayrollRecord computes the effective payroll record for a user
func computePayrollRecord(user models.CustomUser, updatedRules []models.Rule, mappedTargets []models.MappedTarget,
	unmappedTargets []models.UnmappedTarget, matchedOverrides []models.MatchedOverride,
	processedOverrides []models.Override, ruleIDs []int64, periodID int64, branchID *int64) (models.Record, []models.AuditLog, error) {

	var auditLogs []models.AuditLog
	auditLog := models.AuditLog{
		ActivityType: "payroll_computation",
		Timestamp:    time.Now(),
		Details:      map[string]interface{}{"user_id": user.ID},
		BranchID:     branchID,
		UserID:       &user.ID,
	}
	auditLogs = append(auditLogs, auditLog)

	// Handle empty ruleIDs case: use only base salary
	if len(ruleIDs) == 0 {
		bonusDetails, _ := json.Marshal(map[string]float64{})
		deductionDetails, _ := json.Marshal(map[string]float64{})
		var branchIDVal int64
		if branchID != nil {
			branchIDVal = *branchID
		}
		return models.Record{
			UserID:         user.ID,
			BranchID:       branchIDVal,
			PeriodID:       periodID,
			BaseSalary:     user.Salary,
			TotalBonus:     0.0,
			TotalDeduction: 0.0,
			NetPay:         user.Salary,
			Status:         "generated",
			GeneratedAt:    time.Now(),
			BonusDetails:   bonusDetails,
			DeductionDetails: deductionDetails,
		}, auditLogs, nil
	}
	
	// Build rule map for quick lookup (keep as reference, avoid mutation)
	ruleMap := make(map[int64]models.Rule)
	for _, rule := range updatedRules {
		// Create a copy to ensure ruleMap isn’t modified externally
		ruleMap[rule.ID] = models.Rule{
			ID:    rule.ID,
			Name:  rule.Name,
			Type:  rule.Type,
			Amount: rule.Amount,
		}
		fmt.Printf("Initial rule map for rule %d, Amount: %.2f\n", rule.ID, rule.Amount)
	}

	// Gather effective rules for the user with independent copies
	effectiveRules := make(map[int64]models.Rule)

	// Apply mapped targets (specific to user)
	for _, target := range mappedTargets {
		if contains(target.AssociatedUserIDs, user.ID) {
			if rule, exists := ruleMap[target.RuleID]; exists {
				effectiveRules[target.RuleID] = models.Rule{
					ID:    rule.ID,
					Name:  rule.Name,
					Type:  rule.Type,
					Amount: rule.Amount,
				}
				fmt.Printf("Mapped target for user %d, rule %d, initial Amount: %.2f\n", user.ID, rule.ID, rule.Amount)
			}
		}
	}

	// Apply unmapped targets (general rules for all users)
	for _, target := range unmappedTargets {
		if contains(target.UserIDs, user.ID) || len(target.UserIDs) == 0 { // Apply to all if no specific users
			for _, ruleID := range target.RuleIDs {
				if rule, exists := ruleMap[ruleID]; exists {
					effectiveRules[ruleID] = models.Rule{
						ID:    rule.ID,
						Name:  rule.Name,
						Type:  rule.Type,
						Amount: rule.Amount,
					}
					fmt.Printf("Unmapped target for user %d, rule %d, initial Amount: %.2f\n", user.ID, rule.ID, rule.Amount)
				}
			}
		}
	}

	// Apply matched overrides (user-specific, override wins conflicts)
	for _, override := range matchedOverrides {
		if override.UserID == user.ID {
			if rule, exists := effectiveRules[override.Override.RuleID]; exists {
				newRule := models.Rule{
					ID:    rule.ID,
					Name:  rule.Name,
					Type:  rule.Type,
					Amount: rule.Amount, // Start with current value
				}
				fmt.Printf("Before override for user %d, rule %d, Amount: %.2f\n", user.ID, newRule.ID, newRule.Amount)
				switch override.Override.OverrideType {
				case "replace":
					if override.Override.Amount != 0 { // Use 0 as unset value check
						newRule.Amount = override.Override.Amount
					}
					effectiveRules[override.Override.RuleID] = newRule
				case "add":
					if override.Override.Amount != 0 { // Check if override provides a value
						newRule.Amount += override.Override.Amount
					}
					effectiveRules[override.Override.RuleID] = newRule
				case "subtract":
					if override.Override.Amount != 0 { // Check if override provides a value
						newRule.Amount -= override.Override.Amount
					}
					effectiveRules[override.Override.RuleID] = newRule
				}
				fmt.Printf("After override for user %d, rule %d, Amount: %.2f\n", user.ID, newRule.ID, newRule.Amount)
			}
		}
	}

	// Apply processed overrides (general, apply to all users with existing rules)
	for _, override := range processedOverrides {
		if rule, exists := effectiveRules[override.RuleID]; exists {
			newRule := models.Rule{
				ID:    rule.ID,
				Name:  rule.Name,
				Type:  rule.Type,
				Amount: rule.Amount, // Start with current value
			}
			fmt.Printf("Before processed override for user %d, rule %d, Amount: %.2f\n", user.ID, newRule.ID, newRule.Amount)
			switch override.OverrideType {
			case "add":
				if override.Amount != 0 { // Check if override provides a value
					newRule.Amount += override.Amount
				}
				effectiveRules[override.RuleID] = newRule
			case "subtract":
				if override.Amount != 0 { // Check if override provides a value
					newRule.Amount -= override.Amount
				}
				effectiveRules[override.RuleID] = newRule
			}
			fmt.Printf("After processed override for user %d, rule %d, Amount: %.2f\n", user.ID, newRule.ID, newRule.Amount)
		}
	}

	// Calculate totals
	var totalBonus, totalDeduction float64
	bonusDetailsMap := make(map[string]float64)
	deductionDetailsMap := make(map[string]float64)

	for _, rule := range effectiveRules {
		if rule.Amount != 0 { // Skip if amount is unset (0 as default)
			if rule.Type == "bonus" {
				totalBonus += rule.Amount
				bonusDetailsMap[rule.Name] = rule.Amount
			} else if rule.Type == "deduction" {
				totalDeduction += rule.Amount
				deductionDetailsMap[rule.Name] = rule.Amount
			}
		}
	}

	// Marshal JSON details
	bonusDetails, err := json.Marshal(bonusDetailsMap)
	if err != nil {
		return models.Record{}, auditLogs, fmt.Errorf("failed to marshal bonus details: %w", err)
	}
	deductionDetails, err := json.Marshal(deductionDetailsMap)
	if err != nil {
		return models.Record{}, auditLogs, fmt.Errorf("failed to marshal deduction details: %w", err)
	}

	// Compute net pay
	netPay := user.Salary + totalBonus - totalDeduction

	// Create payroll record
	var branchIDVal int64
	if branchID != nil {
		branchIDVal = *branchID
	}
	payrollRecord := models.Record{
		UserID:         user.ID,
		BranchID:       branchIDVal,
		PeriodID:       periodID,
		BaseSalary:     user.Salary,
		TotalBonus:     totalBonus,
		TotalDeduction: totalDeduction,
		NetPay:         netPay,
		Status:         "generated",
		GeneratedAt:    time.Now(),
		BonusDetails:   bonusDetails,
		DeductionDetails: deductionDetails,
	}

	return payrollRecord, auditLogs, nil
}

// contains checks if an ID is in a slice of int64
func contains(ids []int64, id int64) bool {
	for _, i := range ids {
		if i == id {
			return true
		}
	}
	return false
}

// publishGeneratedEvent publishes the payroll.generated event
func (p *Processor) publishGeneratedEvent(ctx context.Context, periodID int64, userIDs []int64, periodEvent map[string]interface{}) error {
	if p.config.KafkaTopicGenerated == "" {
        return fmt.Errorf("KafkaTopicGenerated is not configured")
    }
	var senderID *int64
	var Notify string
	if id, ok := periodEvent["sender"].(int64); ok {
		senderID = &id
	}
	if id, ok := periodEvent["notify"].(string); ok {
		Notify = id
	}
	event := struct {
		PeriodID int64   `json:"period_id"`
		UserIDs  []int64 `json:"user_ids"`
		Sender   int64   `json:"sender"`
		Notify   string  `json:"notify"`
	}{
		PeriodID: periodID,
		UserIDs:  userIDs,
		Sender:   *senderID,
		Notify:  Notify,
	}
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return err
	}

	msg := &kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &p.config.KafkaTopicGenerated, Partition: kafka.PartitionAny},
		Value:          eventJSON,
	}
	if err := p.kafkaProducer.Produce(msg, nil); err != nil {
		return err
	}
	p.kafkaProducer.Flush(1000)

	// Log event publishing
	if err := p.db.LogActivity(ctx, "event_published", map[string]interface{}{"topic": p.config.KafkaTopicGenerated, "user_ids": userIDs}, nil, nil); err != nil {
		log.Printf("Failed to log event published: %v", err)
	}
	fmt.Printf("eventJSON Producer: %s\n", eventJSON)
	return nil
}

func getScope(event map[string]interface{}) string {
	if _, ok := event["branch_id"].(int64); ok {
		return "branch"
	} else if _, ok := event["restaurant_id"].(int64); ok {
		return "restaurant"
	} else if _, ok := event["company_id"].(int64); ok {
		return "company"
	}
	return "unknown"
}