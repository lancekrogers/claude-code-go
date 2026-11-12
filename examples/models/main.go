// Package main lists the models the installed Claude CLI can use.
//
// SupportedModels sends only an initialize control request to the CLI, so it
// makes no model call and costs nothing. Pass a Value to RunOptions.ModelAlias
// to select that model.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/lancekrogers/claude-code-go/pkg/claude"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := claude.NewClient("claude")
	models, err := client.SupportedModels(ctx, nil)
	switch {
	case errors.Is(err, exec.ErrNotFound):
		log.Fatal("claude CLI not found on PATH")
	case errors.Is(err, claude.ErrNoControlResponse):
		log.Fatalf("this claude CLI cannot list models; upgrade it: %v", err)
	case err != nil:
		log.Fatal(err)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "VALUE\tMODEL\tNAME\tEFFORT\tDESCRIPTION")
	for _, m := range models {
		levels := make([]string, len(m.SupportedEffortLevels))
		for i, level := range m.SupportedEffortLevels {
			levels[i] = string(level)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", m.Value, m.ResolvedModel, m.DisplayName, strings.Join(levels, ","), m.Description)
	}
	_ = w.Flush()
}
