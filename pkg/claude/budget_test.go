package claude

import (
	"math"
	"sync"
	"testing"
)

func TestNewBudgetTracker(t *testing.T) {
	t.Run("with nil config", func(t *testing.T) {
		bt := NewBudgetTracker(nil)
		if bt == nil {
			t.Fatal("NewBudgetTracker() returned nil")
		}
		if bt.TotalSpent() != 0 {
			t.Errorf("TotalSpent() = %v, want 0", bt.TotalSpent())
		}
	})

	t.Run("with config", func(t *testing.T) {
		config := &BudgetConfig{
			MaxBudgetUSD:     10.0,
			WarningThreshold: 0.8,
		}
		bt := NewBudgetTracker(config)
		if bt.Config().MaxBudgetUSD != 10.0 {
			t.Errorf("MaxBudgetUSD = %v, want 10.0", bt.Config().MaxBudgetUSD)
		}
	})
}

func TestBudgetTracker_AddSpend(t *testing.T) {
	t.Run("basic spending", func(t *testing.T) {
		bt := NewBudgetTracker(&BudgetConfig{MaxBudgetUSD: 10.0})

		err := bt.AddSpend("session1", 5.0)
		if err != nil {
			t.Errorf("AddSpend() returned error: %v", err)
		}
		if bt.TotalSpent() != 5.0 {
			t.Errorf("TotalSpent() = %v, want 5.0", bt.TotalSpent())
		}
		if bt.SessionSpent("session1") != 5.0 {
			t.Errorf("SessionSpent() = %v, want 5.0", bt.SessionSpent("session1"))
		}
	})

	t.Run("budget exceeded", func(t *testing.T) {
		bt := NewBudgetTracker(&BudgetConfig{MaxBudgetUSD: 5.0})

		err := bt.AddSpend("session1", 6.0)
		if err != ErrBudgetExceeded {
			t.Errorf("AddSpend() error = %v, want ErrBudgetExceeded", err)
		}
	})

	t.Run("no budget limit", func(t *testing.T) {
		bt := NewBudgetTracker(&BudgetConfig{})

		err := bt.AddSpend("session1", 1000.0)
		if err != nil {
			t.Errorf("AddSpend() returned error when no limit set: %v", err)
		}
	})

	t.Run("warning callback", func(t *testing.T) {
		warningCalled := false
		bt := NewBudgetTracker(&BudgetConfig{
			MaxBudgetUSD:     10.0,
			WarningThreshold: 0.5,
			OnBudgetWarning: func(current, max float64) {
				warningCalled = true
			},
		})

		_ = bt.AddSpend("session1", 6.0) // 60% of budget, exceeds 50% threshold
		if !warningCalled {
			t.Fatal("expected warning callback to be invoked")
		}
	})

	t.Run("exceeded callback", func(t *testing.T) {
		exceededCalled := false
		bt := NewBudgetTracker(&BudgetConfig{
			MaxBudgetUSD: 5.0,
			OnBudgetExceeded: func(current, max float64) {
				exceededCalled = true
			},
		})

		_ = bt.AddSpend("session1", 6.0)
		if !exceededCalled {
			t.Fatal("expected exceeded callback to be invoked")
		}
	})
}

func TestBudgetTracker_RemainingBudget(t *testing.T) {
	t.Run("with budget", func(t *testing.T) {
		bt := NewBudgetTracker(&BudgetConfig{MaxBudgetUSD: 10.0})
		_ = bt.AddSpend("session1", 3.0)

		remaining := bt.RemainingBudget()
		if remaining != 7.0 {
			t.Errorf("RemainingBudget() = %v, want 7.0", remaining)
		}
	})

	t.Run("no budget limit", func(t *testing.T) {
		bt := NewBudgetTracker(&BudgetConfig{})

		remaining := bt.RemainingBudget()
		if remaining != -1 {
			t.Errorf("RemainingBudget() = %v, want -1 (no limit)", remaining)
		}
	})

	t.Run("over budget", func(t *testing.T) {
		bt := NewBudgetTracker(&BudgetConfig{MaxBudgetUSD: 5.0})
		_ = bt.AddSpend("session1", 10.0)

		remaining := bt.RemainingBudget()
		if remaining != 0 {
			t.Errorf("RemainingBudget() = %v, want 0 (over budget)", remaining)
		}
	})
}

func TestBudgetTracker_CanSpend(t *testing.T) {
	bt := NewBudgetTracker(&BudgetConfig{MaxBudgetUSD: 10.0})
	_ = bt.AddSpend("session1", 5.0)

	tests := []struct {
		name   string
		amount float64
		want   bool
	}{
		{"Can spend within budget", 4.0, true},
		{"Can spend exact remaining", 5.0, true},
		{"Cannot exceed budget", 6.0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := bt.CanSpend(tt.amount); got != tt.want {
				t.Errorf("CanSpend(%v) = %v, want %v", tt.amount, got, tt.want)
			}
		})
	}
}

func TestBudgetTracker_Reset(t *testing.T) {
	bt := NewBudgetTracker(&BudgetConfig{MaxBudgetUSD: 10.0})
	_ = bt.AddSpend("session1", 5.0)
	_ = bt.AddSpend("session2", 3.0)

	bt.Reset()

	if bt.TotalSpent() != 0 {
		t.Errorf("TotalSpent() after Reset() = %v, want 0", bt.TotalSpent())
	}
	if bt.SessionSpent("session1") != 0 {
		t.Errorf("SessionSpent('session1') after Reset() = %v, want 0", bt.SessionSpent("session1"))
	}
}

func TestBudgetTracker_ResetSession(t *testing.T) {
	bt := NewBudgetTracker(&BudgetConfig{MaxBudgetUSD: 10.0})
	_ = bt.AddSpend("session1", 5.0)
	_ = bt.AddSpend("session2", 3.0)

	bt.ResetSession("session1")

	if bt.TotalSpent() != 3.0 {
		t.Errorf("TotalSpent() after ResetSession() = %v, want 3.0", bt.TotalSpent())
	}
	if bt.SessionSpent("session1") != 0 {
		t.Errorf("SessionSpent('session1') after ResetSession() = %v, want 0", bt.SessionSpent("session1"))
	}
	if bt.SessionSpent("session2") != 3.0 {
		t.Errorf("SessionSpent('session2') after ResetSession() = %v, want 3.0", bt.SessionSpent("session2"))
	}
}

func TestBudgetTracker_UpdateConfig(t *testing.T) {
	bt := NewBudgetTracker(&BudgetConfig{MaxBudgetUSD: 10.0})

	newConfig := &BudgetConfig{MaxBudgetUSD: 20.0}
	bt.UpdateConfig(newConfig)

	if bt.Config().MaxBudgetUSD != 20.0 {
		t.Errorf("MaxBudgetUSD after UpdateConfig() = %v, want 20.0", bt.Config().MaxBudgetUSD)
	}
}

func TestBudgetTracker_Concurrent(t *testing.T) {
	bt := NewBudgetTracker(&BudgetConfig{MaxBudgetUSD: 1000.0})

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(session int) {
			defer wg.Done()
			_ = bt.AddSpend("concurrent", 1.0)
		}(i)
	}
	wg.Wait()

	if bt.TotalSpent() != 100.0 {
		t.Errorf("TotalSpent() after concurrent adds = %v, want 100.0", bt.TotalSpent())
	}
}

func TestBudgetTracker_RecordSessionTotal(t *testing.T) {
	approx := func(got, want float64) bool { return math.Abs(got-want) < 1e-9 }

	t.Run("resumed session records only the increase", func(t *testing.T) {
		bt := NewBudgetTracker(&BudgetConfig{MaxBudgetUSD: 10.0})
		// The CLI reports a session's running total on each resumed run.
		for _, total := range []float64{0.10, 0.25, 0.30} {
			if err := bt.RecordSessionTotal("s1", "s1", total); err != nil {
				t.Fatalf("RecordSessionTotal(%v) error = %v", total, err)
			}
		}
		if !approx(bt.TotalSpent(), 0.30) {
			t.Errorf("TotalSpent() = %v, want 0.30", bt.TotalSpent())
		}
		if !approx(bt.SessionSpent("s1"), 0.30) {
			t.Errorf("SessionSpent() = %v, want 0.30", bt.SessionSpent("s1"))
		}
	})

	t.Run("fork starts from the parent's total", func(t *testing.T) {
		bt := NewBudgetTracker(&BudgetConfig{MaxBudgetUSD: 10.0})
		_ = bt.RecordSessionTotal("parent", "", 0.20)
		// The forked session's first total includes the parent's 0.20.
		_ = bt.RecordSessionTotal("fork", "parent", 0.25)
		if !approx(bt.TotalSpent(), 0.25) {
			t.Errorf("TotalSpent() = %v, want 0.25", bt.TotalSpent())
		}
		if !approx(bt.SessionSpent("fork"), 0.05) {
			t.Errorf("SessionSpent(fork) = %v, want 0.05", bt.SessionSpent("fork"))
		}
	})

	t.Run("unrelated sessions each count their total", func(t *testing.T) {
		bt := NewBudgetTracker(&BudgetConfig{MaxBudgetUSD: 10.0})
		_ = bt.RecordSessionTotal("a", "", 0.10)
		_ = bt.RecordSessionTotal("b", "", 0.15)
		if !approx(bt.TotalSpent(), 0.25) {
			t.Errorf("TotalSpent() = %v, want 0.25", bt.TotalSpent())
		}
	})

	t.Run("reset keeps reported totals", func(t *testing.T) {
		bt := NewBudgetTracker(&BudgetConfig{MaxBudgetUSD: 10.0})
		_ = bt.RecordSessionTotal("s1", "", 0.10)
		bt.Reset()
		_ = bt.RecordSessionTotal("s1", "s1", 0.12)
		if !approx(bt.TotalSpent(), 0.02) {
			t.Errorf("TotalSpent() = %v, want 0.02", bt.TotalSpent())
		}
	})

	t.Run("exceeding the budget returns ErrBudgetExceeded", func(t *testing.T) {
		bt := NewBudgetTracker(&BudgetConfig{MaxBudgetUSD: 0.20})
		_ = bt.RecordSessionTotal("s1", "", 0.15)
		if err := bt.RecordSessionTotal("s1", "s1", 0.25); err != ErrBudgetExceeded {
			t.Errorf("RecordSessionTotal() error = %v, want ErrBudgetExceeded", err)
		}
	})
}
