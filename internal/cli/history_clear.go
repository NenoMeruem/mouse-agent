package cli

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/meruem/prompt-builder-agent/internal/app"
	"github.com/spf13/cobra"
)

var historyClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear run history",
	RunE:  historyClearCommand,
}

var clearBefore string // e.g. "30d", "7d", "1h"
var clearAll bool

func init() {
	historyCmd.AddCommand(historyClearCmd)
	historyClearCmd.Flags().StringVar(&clearBefore, "before", "30d", "Clear records older than this duration (e.g. 30d, 7d, 1h)")
	historyClearCmd.Flags().BoolVar(&clearAll, "all", false, "Clear all history records")
}

func historyClearCommand(cmd *cobra.Command, args []string) error {
	if clearAll {
		if err := app.GlobalContext.HistoryStore.ClearAll(); err != nil {
			return fmt.Errorf("cannot clear history: %w", err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "History cleared.\n")
		return nil
	}

	d, err := parseDuration(clearBefore)
	if err != nil {
		return fmt.Errorf("invalid --before value: %w", err)
	}
	cutoff := time.Now().Add(-d)
	if err := app.GlobalContext.HistoryStore.Clear(cutoff); err != nil {
		return fmt.Errorf("cannot clear history: %w", err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "History cleared (records before %s).\n", cutoff.Format("2006-01-02"))
	return nil
}

// parseDuration parses a duration string supporting Nd (days), Nh (hours),
// and falls back to standard Go duration notation (e.g. "1h30m").
func parseDuration(s string) (time.Duration, error) {
	if strings.HasSuffix(s, "d") {
		days, err := strconv.Atoi(strings.TrimSuffix(s, "d"))
		if err != nil {
			return 0, fmt.Errorf("invalid duration: %s", s)
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	if strings.HasSuffix(s, "h") {
		hours, err := strconv.Atoi(strings.TrimSuffix(s, "h"))
		if err != nil {
			return 0, fmt.Errorf("invalid duration: %s", s)
		}
		return time.Duration(hours) * time.Hour, nil
	}
	return time.ParseDuration(s)
}
