package sync

import (
	"fmt"
	"strings"
	"time"

	"github.com/vaultenv/vaultenv-cli/internal/ui"
)

// ConflictStrategy defines how to resolve conflicts
type ConflictStrategy string

const (
	StrategyPrompt ConflictStrategy = "prompt"
	StrategyOurs   ConflictStrategy = "ours"
	StrategyTheirs ConflictStrategy = "theirs"
	StrategyNewest ConflictStrategy = "newest"
)

// Change represents a change to a variable
type Change struct {
	Author    string
	Timestamp time.Time
	Action    string // "set", "delete"
	Value     string
}

// Conflict represents a merge conflict between local and remote values
type Conflict struct {
	Environment  string
	Variable     string
	LocalValue   string
	RemoteValue  string
	BaseValue    string // Value before both changes
	LocalChange  Change
	RemoteChange Change
}

// Resolution represents how a conflict was resolved
type Resolution struct {
	Value  string
	Source string // "local", "remote", "manual", "merged"
	Skip   bool
}

// ConflictResolver handles conflict resolution with various strategies
type ConflictResolver struct {
	strategy ConflictStrategy
}

// NewConflictResolver creates a new conflict resolver
func NewConflictResolver(strategy ConflictStrategy) *ConflictResolver {
	return &ConflictResolver{
		strategy: strategy,
	}
}

// ResolveConflict determines how to handle a specific conflict
func (cr *ConflictResolver) ResolveConflict(conflict Conflict) (Resolution, error) {
	// Check if conflict can be auto-resolved
	if resolution, canResolve := cr.canAutoResolve(conflict); canResolve {
		return resolution, nil
	}

	// Otherwise use configured strategy
	switch cr.strategy {
	case StrategyOurs:
		return Resolution{Value: conflict.LocalValue, Source: "local"}, nil
	case StrategyTheirs:
		return Resolution{Value: conflict.RemoteValue, Source: "remote"}, nil
	case StrategyNewest:
		if conflict.LocalChange.Timestamp.After(conflict.RemoteChange.Timestamp) {
			return Resolution{Value: conflict.LocalValue, Source: "local"}, nil
		}
		return Resolution{Value: conflict.RemoteValue, Source: "remote"}, nil
	case StrategyPrompt:
		return cr.promptUser(conflict)
	default:
		return Resolution{}, fmt.Errorf("unknown strategy: %s", cr.strategy)
	}
}

// canAutoResolve checks if conflict has an obvious resolution
func (cr *ConflictResolver) canAutoResolve(conflict Conflict) (Resolution, bool) {
	// Same value - not really a conflict
	if conflict.LocalValue == conflict.RemoteValue {
		return Resolution{Value: conflict.LocalValue, Source: "merged"}, true
	}

	// One side just deleted
	if conflict.LocalValue == "" && conflict.RemoteValue != "" {
		// Local deleted, remote modified - needs manual review
		return Resolution{}, false
	}
	if conflict.RemoteValue == "" && conflict.LocalValue != "" {
		// Remote deleted, local modified - needs manual review
		return Resolution{}, false
	}

	// Check if one side didn't actually change
	if conflict.LocalValue == conflict.BaseValue && conflict.RemoteValue != conflict.BaseValue {
		// Only remote changed, take remote
		return Resolution{Value: conflict.RemoteValue, Source: "remote"}, true
	}
	if conflict.RemoteValue == conflict.BaseValue && conflict.LocalValue != conflict.BaseValue {
		// Only local changed, take local
		return Resolution{Value: conflict.LocalValue, Source: "local"}, true
	}

	// Both sides changed to different values - needs resolution
	return Resolution{}, false
}

// promptUser interactively resolves conflict
func (cr *ConflictResolver) promptUser(conflict Conflict) (Resolution, error) {
	ui.Header(fmt.Sprintf("Conflict in %s: %s", conflict.Environment, conflict.Variable))

	// Show the conflict details
	ui.Info("Base value: %s", maskValue(conflict.BaseValue))
	ui.Info("Local:  %s (modified %s by %s)",
		maskValue(conflict.LocalValue),
		conflict.LocalChange.Timestamp.Format("Jan 2 15:04"),
		conflict.LocalChange.Author)
	ui.Info("Remote: %s (modified %s by %s)",
		maskValue(conflict.RemoteValue),
		conflict.RemoteChange.Timestamp.Format("Jan 2 15:04"),
		conflict.RemoteChange.Author)

	// Offer resolution options
	choice := ui.Select("How would you like to resolve this conflict?", []string{
		"Keep local value",
		"Keep remote value",
		"Show full values and decide",
		"Enter new value",
		"Skip this variable",
	})

	switch choice {
	case 0:
		return Resolution{Value: conflict.LocalValue, Source: "local"}, nil
	case 1:
		return Resolution{Value: conflict.RemoteValue, Source: "remote"}, nil
	case 2:
		// Show unmasked values for decision
		return cr.showAndDecide(conflict)
	case 3:
		// Prompt for new value
		newValue := ui.PromptMasked("Enter new value: ")
		return Resolution{Value: newValue, Source: "manual"}, nil
	case 4:
		return Resolution{Skip: true}, nil
	default:
		return Resolution{}, fmt.Errorf("invalid choice")
	}
}

// showAndDecide shows full values and prompts for decision
func (cr *ConflictResolver) showAndDecide(conflict Conflict) (Resolution, error) {
	ui.Warning("Showing full values:")
	ui.Info("Base value: %s", conflict.BaseValue)
	ui.Info("Local:  %s", conflict.LocalValue)
	ui.Info("Remote: %s", conflict.RemoteValue)

	choice := ui.Select("Select which value to keep:", []string{
		"Keep local value",
		"Keep remote value",
		"Enter new value",
		"Skip",
	})

	switch choice {
	case 0:
		return Resolution{Value: conflict.LocalValue, Source: "local"}, nil
	case 1:
		return Resolution{Value: conflict.RemoteValue, Source: "remote"}, nil
	case 2:
		newValue := ui.PromptMasked("Enter new value: ")
		return Resolution{Value: newValue, Source: "manual"}, nil
	case 3:
		return Resolution{Skip: true}, nil
	default:
		return Resolution{}, fmt.Errorf("invalid choice")
	}
}

// maskValue masks sensitive values for display
func maskValue(value string) string {
	if value == "" {
		return "(empty)"
	}
	if len(value) <= 4 {
		return "****"
	}
	// Show first 2 and last 2 characters
	return value[:2] + strings.Repeat("*", len(value)-4) + value[len(value)-2:]
}

// ConflictSet represents a collection of conflicts
type ConflictSet struct {
	Conflicts []Conflict
	Strategy  ConflictStrategy
}

// ResolveAll resolves all conflicts in the set
func (cs *ConflictSet) ResolveAll() (map[string]Resolution, error) {
	resolver := NewConflictResolver(cs.Strategy)
	resolutions := make(map[string]Resolution)

	ui.Info("Resolving %d conflicts with strategy: %s", len(cs.Conflicts), cs.Strategy)

	for i, conflict := range cs.Conflicts {
		ui.Progress(i+1, len(cs.Conflicts), "Resolving conflicts")

		resolution, err := resolver.ResolveConflict(conflict)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve conflict for %s/%s: %w",
				conflict.Environment, conflict.Variable, err)
		}

		if !resolution.Skip {
			key := fmt.Sprintf("%s/%s", conflict.Environment, conflict.Variable)
			resolutions[key] = resolution
		}
	}

	return resolutions, nil
}

// Summary generates a summary of resolutions
func (cs *ConflictSet) Summary(resolutions map[string]Resolution) string {
	local, remote, manual, skipped := 0, 0, 0, 0

	for _, resolution := range resolutions {
		switch resolution.Source {
		case "local":
			local++
		case "remote":
			remote++
		case "manual":
			manual++
		}
	}

	// Count skipped
	skipped = len(cs.Conflicts) - len(resolutions)

	return fmt.Sprintf("Resolved %d conflicts: %d local, %d remote, %d manual, %d skipped",
		len(cs.Conflicts), local, remote, manual, skipped)
}