package payroll

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/jackc/pgx/v5"
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

// ProcessPayroll processes the payroll for a given period and scope
func (p *Processor) ProcessPayroll(ctx context.Context, periodEvent map[string]interface{}) error {
	// Log event reception
	if err := p.db.LogActivity(ctx, "event_received", map[string]interface{}{"event": periodEvent}, nil, nil); err != nil {
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

		// Log user fetching
		if err := p.db.LogActivity(ctx, "users_fetched", map[string]interface{}{"offset": offset, "count": len(users), "scope": getScope(periodEvent)}, nil, nil); err != nil {
			log.Printf("Failed to log users fetched: %v", err)
		}

		rules, err := p.db.FetchRules(ctx)
		if err != nil {
			return fmt.Errorf("failed to fetch rules: %w", err)
		}
		targets, err := p.db.FetchRuleTargets(ctx)
		if err != nil {
			return fmt.Errorf("failed to fetch rule targets: %w", err)
		}
		overrides, err := p.db.FetchOverrides(ctx, periodID)
		if err != nil {
			return fmt.Errorf("failed to fetch overrides: %w", err)
		}

		// Log data fetching
		if err := p.db.LogActivity(ctx, "data_fetched", map[string]interface{}{"type": "rules/targets/overrides", "count": len(rules) + len(targets) + len(overrides)}, nil, nil); err != nil {
			log.Printf("Failed to log data fetched: %v", err)
		}

		var wg sync.WaitGroup
		resultChan := make(chan struct {
			user           models.CustomUser
			effectiveRules []models.EffectiveRule
			auditLogs      []models.AuditLog
			err            error
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
				effectiveRules, auditLogs := computeEffectiveRules(u, rules, targets, overrides, periodID)
				resultChan <- struct {
					user           models.CustomUser
					effectiveRules []models.EffectiveRule
					auditLogs      []models.AuditLog
					err            error
				}{u, effectiveRules, auditLogs, nil}
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
			if err := p.storePayroll(ctx, res.user, periodID, res.effectiveRules, res.auditLogs); err != nil {
				log.Printf("Error storing payroll for user %d: %v", res.user.ID, err)
				continue
			}
			allUserIDs = append(allUserIDs, res.user.ID)
		}

		offset += 500
	}

	// Publish payroll.generated event
	if err := p.publishGeneratedEvent(ctx, periodID, allUserIDs); err != nil {
		log.Printf("Failed to publish payroll.generated: %v", err)
		return err
	}

	return nil
}

// computeEffectiveRules computes the effective rules for a user
func computeEffectiveRules(user models.CustomUser, rules []models.Rule, targets []models.RuleTarget, overrides []models.Override, periodID int64) ([]models.EffectiveRule, []models.AuditLog) {
	effectiveRules := []models.EffectiveRule{}
	auditLogs := []models.AuditLog{}

	overrideMap := make(map[int64]models.Override)
	for _, o := range overrides {
		if o.UserID == user.ID {
			overrideMap[o.RuleID] = o
		}
	}

	for _, rule := range rules {
		applies := false
		for _, target := range targets {
			if target.RuleID == rule.ID && (target.BranchID == nil || contains(user.Branches, *target.BranchID)) {
				if (target.TargetType == "role" && target.TargetValue == user.Role) ||
					(target.TargetType == "user" && fmt.Sprint(target.TargetValue) == fmt.Sprint(user.ID)) {
					applies = true
					break
				}
			}
		}
		if !applies {
			continue
		}

		if override, exists := overrideMap[rule.ID]; exists {
			if override.BranchID != nil && !contains(user.Branches, *override.BranchID) {
				continue
			}
			amount := rule.Amount
			percentage := rule.Percentage
			if override.OverrideType == "remove" {
				auditLogs = append(auditLogs, models.AuditLog{
					ActivityType: "rule_removed",
					Timestamp:    time.Now(),
					Details:      map[string]interface{}{"rule_id": rule.ID, "notes": fmt.Sprintf("Removed rule %s", rule.Name)},
					BranchID:     &user.Branches[0], // Use first branch for simplicity
					UserID:       &user.ID,
				})
				continue
			}

			if override.OverrideType == "replace" {
				if override.Amount != nil {
					amount = *override.Amount
				}
				if override.Percentage != nil {
					percentage = override.Percentage
				}
			} else if override.OverrideType == "add" {
				if override.Amount != nil {
					amount += *override.Amount
				}
				if override.Percentage != nil && rule.Percentage != nil {
					*percentage += *override.Percentage
				}
			}
			effectiveRules = append(effectiveRules, models.EffectiveRule{
				PeriodID:   periodID,
				UserID:     user.ID,
				BranchID:   user.Branches[0], // Use first branch for simplicity
				RuleID:     rule.ID,
				Amount:     amount,
				Percentage: percentage,
				Source:     rule.Scope,
				Type:       rule.Type,
			})
			auditLogs = append(auditLogs, models.AuditLog{
				ActivityType: "rule_overridden",
				Timestamp:    time.Now(),
				Details:      map[string]interface{}{"rule_id": rule.ID, "amount": amount, "notes": fmt.Sprintf("Overrode rule %s", rule.Name)},
				BranchID:     &user.Branches[0],
				UserID:       &user.ID,
			})
		} else {
			effectiveRules = append(effectiveRules, models.EffectiveRule{
				PeriodID:   periodID,
				UserID:     user.ID,
				BranchID:   user.Branches[0],
				RuleID:     rule.ID,
				Amount:     rule.Amount,
				Percentage: rule.Percentage,
				Source:     rule.Scope,
			})
			auditLogs = append(auditLogs, models.AuditLog{
				ActivityType: "rule_applied",
				Timestamp:    time.Now(),
				Details:      map[string]interface{}{"rule_id": rule.ID, "amount": rule.Amount, "notes": fmt.Sprintf("Applied rule %s", rule.Name)},
				BranchID:     &user.Branches[0],
				UserID:       &user.ID,
			})
		}
	}

	return effectiveRules, auditLogs
}

// storePayroll stores the computed payroll data
func (p *Processor) storePayroll(ctx context.Context, user models.CustomUser, periodID int64, effectiveRules []models.EffectiveRule, auditLogs []models.AuditLog) error {
	tx, err := p.db.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	totalBonus, totalDeduction := 0.0, 0.0
	for _, er := range effectiveRules {
		amount := er.Amount
		if er.Percentage != nil {
			amount += user.Salary * (*er.Percentage / 100)
		}
		if er.Type == "bonus" {
			totalBonus += amount
		} else {
			totalDeduction += amount
		}
	}
	netPay := user.Salary + totalBonus - totalDeduction

	// Insert Record
	var recordID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO payroll_record (user_id, branch_id, period_id, base_salary, total_bonus, total_deduction, net_pay, status, generated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'generated', $8)
		RETURNING id
	`, user.ID, user.Branches[0], periodID, user.Salary, totalBonus, totalDeduction, netPay, time.Now()).Scan(&recordID)
	if err != nil {
		return err
	}

	// Bulk insert EffectiveRules
	_, err = tx.CopyFrom(ctx, pgx.Identifier{"payroll_effectiverule"}, []string{"period_id", "user_id", "branch_id", "rule_id", "amount", "percentage", "source", "created_at"},
		pgx.CopyFromSlice(len(effectiveRules), func(i int) ([]interface{}, error) {
			return []interface{}{periodID, user.ID, user.Branches[0], effectiveRules[i].RuleID, effectiveRules[i].Amount, effectiveRules[i].Percentage, effectiveRules[i].Source, time.Now()}, nil
		}))
	if err != nil {
		return err
	}

	// Bulk insert Components
	_, err = tx.CopyFrom(ctx, pgx.Identifier{"payroll_component"}, []string{"record_id", "rule_id", "amount"},
		pgx.CopyFromSlice(len(effectiveRules), func(i int) ([]interface{}, error) {
			amount := effectiveRules[i].Amount
			if effectiveRules[i].Percentage != nil {
				amount += user.Salary * (*effectiveRules[i].Percentage / 100)
			}
			return []interface{}{recordID, effectiveRules[i].RuleID, amount}, nil
		}))
	if err != nil {
		return err
	}

	// Log audit activities
	for _, al := range auditLogs {
		if err := p.db.LogActivity(ctx, al.ActivityType, al.Details, al.BranchID, al.UserID); err != nil {
			log.Printf("Failed to log audit for user %d: %v", user.ID, err)
		}
	}

	// Log storage
	if err := p.db.LogActivity(ctx, "data_stored", map[string]interface{}{"record_id": recordID, "user_id": user.ID}, &user.Branches[0], &user.ID); err != nil {
		log.Printf("Failed to log data stored: %v", err)
	}

	return tx.Commit(ctx)
}

// publishGeneratedEvent publishes the payroll.generated event
func (p *Processor) publishGeneratedEvent(ctx context.Context, periodID int64, userIDs []int64) error {
	if p.config.KafkaTopicGenerated == "" {
        return fmt.Errorf("KafkaTopicGenerated is not configured")
    }
	event := struct {
		PeriodID int64   `json:"period_id"`
		UserIDs  []int64 `json:"user_ids"`
	}{
		PeriodID: periodID,
		UserIDs:  userIDs,
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

	return nil
}

func contains(slice []int64, item int64) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
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