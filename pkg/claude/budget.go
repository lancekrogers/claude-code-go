package claude

import (
	"errors"
	"sync"
)

// ErrBudgetExceeded is returned when the budget limit is exceeded
var ErrBudgetExceeded = errors.New("budget limit exceeded")

// BudgetConfig controls spending limits and notifications
type BudgetConfig struct {
	// MaxBudgetUSD is the maximum allowed spend in USD
	MaxBudgetUSD float64
	// WarningThreshold is the percentage (0.0-1.0) at which to emit warnings
	WarningThreshold float64
	// OnBudgetWarning is called when spending exceeds the warning threshold
	OnBudgetWarning func(current, max float64)
	// OnBudgetExceeded is called when spending exceeds the budget
	OnBudgetExceeded func(current, max float64)
}

// BudgetTracker tracks cumulative spending across sessions
type BudgetTracker struct {
	mu             sync.RWMutex
	totalSpent     float64
	sessionSpent   map[string]float64
	config         *BudgetConfig
	warningEmitted bool
}

// NewBudgetTracker creates a new BudgetTracker with the given configuration
func NewBudgetTracker(config *BudgetConfig) *BudgetTracker {
	if config == nil {
		config = &BudgetConfig{}
	}
	return &BudgetTracker{
		sessionSpent: make(map[string]float64),
		config:       config,
	}
}

// TotalSpent returns the total amount spent across all sessions
func (bt *BudgetTracker) TotalSpent() float64 {
	bt.mu.RLock()
	defer bt.mu.RUnlock()
	return bt.totalSpent
}

// SessionSpent returns the amount spent in a specific session
func (bt *BudgetTracker) SessionSpent(sessionID string) float64 {
	bt.mu.RLock()
	defer bt.mu.RUnlock()
	return bt.sessionSpent[sessionID]
}

// RemainingBudget returns the remaining budget, or -1 if no limit is set
func (bt *BudgetTracker) RemainingBudget() float64 {
	if bt.config.MaxBudgetUSD <= 0 {
		return -1
	}
	bt.mu.RLock()
	defer bt.mu.RUnlock()
	remaining := bt.config.MaxBudgetUSD - bt.totalSpent
	if remaining < 0 {
		return 0
	}
	return remaining
}

// CanSpend checks if the given amount can be spent within the budget
func (bt *BudgetTracker) CanSpend(amount float64) bool {
	if bt.config.MaxBudgetUSD <= 0 {
		return true // No limit set
	}
	bt.mu.RLock()
	defer bt.mu.RUnlock()
	return bt.totalSpent+amount <= bt.config.MaxBudgetUSD
}

// AddSpend adds spending to the tracker and returns an error if budget is exceeded
func (bt *BudgetTracker) AddSpend(sessionID string, amount float64) error {
	bt.mu.Lock()
	defer bt.mu.Unlock()

	bt.totalSpent += amount
	bt.sessionSpent[sessionID] += amount

	// Check warning threshold
	if bt.config.MaxBudgetUSD > 0 && bt.config.WarningThreshold > 0 && !bt.warningEmitted {
		warningAmount := bt.config.MaxBudgetUSD * bt.config.WarningThreshold
		if bt.totalSpent >= warningAmount {
			bt.warningEmitted = true
			if bt.config.OnBudgetWarning != nil {
				// Call callback outside of lock to prevent deadlocks
				go bt.config.OnBudgetWarning(bt.totalSpent, bt.config.MaxBudgetUSD)
			}
		}
	}

	// Check if budget exceeded
	if bt.config.MaxBudgetUSD > 0 && bt.totalSpent > bt.config.MaxBudgetUSD {
		if bt.config.OnBudgetExceeded != nil {
			go bt.config.OnBudgetExceeded(bt.totalSpent, bt.config.MaxBudgetUSD)
		}
		return ErrBudgetExceeded
	}

	return nil
}

// Reset resets the tracker to zero spending
func (bt *BudgetTracker) Reset() {
	bt.mu.Lock()
	defer bt.mu.Unlock()
	bt.totalSpent = 0
	bt.sessionSpent = make(map[string]float64)
	bt.warningEmitted = false
}

// ResetSession resets spending for a specific session
func (bt *BudgetTracker) ResetSession(sessionID string) {
	bt.mu.Lock()
	defer bt.mu.Unlock()
	if spent, ok := bt.sessionSpent[sessionID]; ok {
		bt.totalSpent -= spent
		delete(bt.sessionSpent, sessionID)
	}
}

// Config returns the budget configuration
func (bt *BudgetTracker) Config() *BudgetConfig {
	return bt.config
}

// UpdateConfig updates the budget configuration
func (bt *BudgetTracker) UpdateConfig(config *BudgetConfig) {
	bt.mu.Lock()
	defer bt.mu.Unlock()
	bt.config = config
	bt.warningEmitted = false // Reset warning state when config changes
}
