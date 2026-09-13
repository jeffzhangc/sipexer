package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

func newHistoryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "history",
		Aliases: []string{"h"},
		Short:   "Show and manage dial history",
	}
	cmd.AddCommand(newHistoryListCmd(), newHistoryClearCmd(), newHistoryAliasCmd())
	return cmd
}

func newHistoryListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List dial history (most recent first)",
		Args:    cobra.NoArgs,
		RunE: func(c *cobra.Command, args []string) error {
			app, err := loadApp()
			if err != nil {
				return err
			}
			entries := app.history.Entries()
			if len(entries) == 0 {
				fmt.Println("no history")
				return nil
			}
			for i, e := range entries {
				alias := ""
				if e.Alias != "" {
					alias = " [" + e.Alias + "]"
				}
				fmt.Printf("@%-3d %s  %s  %s  %s%s\n", i, e.When, e.Profile, e.Server, e.Target, alias)
			}
			return nil
		},
	}
}

func newHistoryClearCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "clear",
		Short: "Remove all dial history",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, args []string) error {
			app, err := loadApp()
			if err != nil {
				return err
			}
			// Interactive confirmation on a TTY.
			yes, yerr := promptConfirm("Remove all history entries? [y/N] ")
			if yerr == nil && !yes {
				fmt.Println("aborted")
				return nil
			}
			if err := app.history.Clear(); err != nil {
				return err
			}
			fmt.Println("history cleared")
			return nil
		},
	}
}

func newHistoryAliasCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "alias <@index> <name>",
		Short: "Assign a reusable alias to a history entry (most recent = @0)",
		Args:  cobra.ExactArgs(2),
		RunE: func(c *cobra.Command, args []string) error {
			app, err := loadApp()
			if err != nil {
				return err
			}
			idx, perr := parseIndex(args[0])
			if perr != nil {
				return perr
			}
			name := args[1]

			// Alias collision guard: number aliases in any profile, or an
			// existing history alias on a different entry.
			entries := app.history.Entries()
			for i, e := range entries {
				if e.Alias == name && i != idx {
					return fmt.Errorf("alias %q already used by entry @%d", name, i)
				}
			}
			for _, pr := range app.profiles.List() {
				for _, n := range pr.Numbers {
					if n.Alias == name {
						return fmt.Errorf("alias %q already used by number %s in profile %q", name, n.String(), pr.Name)
					}
				}
			}

			return app.history.SetAlias(idx, name)
		},
	}
}

func parseIndex(s string) (int, error) {
	if len(s) > 0 && s[0] == '@' {
		s = s[1:]
	}
	i, err := strconv.Atoi(s)
	if err != nil || i < 0 {
		return 0, fmt.Errorf("invalid history index %q (use @N, most recent = @0)", s)
	}
	return i, nil
}