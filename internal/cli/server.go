package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/miconda/sipexer/internal/store"
)

func newServerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "server",
		Aliases: []string{"servers"},
		Short:   "Manage service addresses of a phone profile",
	}
	cmd.AddCommand(newServerAddCmd(), newServerRMCmd(), newServerListCmd())
	return cmd
}

func newServerAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <profile> <proto://host:port> [--default]",
		Short: "Add a service address to a profile",
		Args:  cobra.ExactArgs(2),
		RunE: func(c *cobra.Command, args []string) error {
			app, err := loadApp()
			if err != nil {
				return err
			}
			pr, ok := app.profiles.Get(args[0])
			if !ok {
				return fmt.Errorf("profile %q does not exist", args[0])
			}
			def, _ := c.Flags().GetBool("default")
			// If the default flag was given, unset other defaults and mark this one.
			if def {
				for i := range pr.Servers {
					pr.Servers[i].Default = false
				}
			}
			// Duplicate address guard.
			for _, s := range pr.Servers {
				if s.Addr == args[1] {
					return fmt.Errorf("server %q already exists in profile %q", args[1], args[0])
				}
			}
			pr.Servers = append(pr.Servers, store.Server{Addr: args[1], Default: def})
			return app.profiles.Edit(pr)
		},
	}
	cmd.Flags().BoolP("default", "d", false, "mark this server as the default (and clear existing defaults)")
	return cmd
}

func newServerRMCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "rm <profile> <proto://host:port>",
		Aliases: []string{"remove", "del"},
		Short:   "Remove a service address from a profile",
		Args:    cobra.ExactArgs(2),
		RunE: func(c *cobra.Command, args []string) error {
			app, err := loadApp()
			if err != nil {
				return err
			}
			pr, ok := app.profiles.Get(args[0])
			if !ok {
				return fmt.Errorf("profile %q does not exist", args[0])
			}
			found := false
			out := pr.Servers[:0]
			for _, s := range pr.Servers {
				if s.Addr == args[1] {
					found = true
					continue
				}
				out = append(out, s)
			}
			if !found {
				return fmt.Errorf("server %q not found in profile %q", args[1], args[0])
			}
			pr.Servers = out
			// If the default was removed and another exists, promote the first.
			if len(pr.Servers) > 0 {
				hasDefault := false
				for _, s := range pr.Servers {
					if s.Default {
						hasDefault = true
					}
				}
				if !hasDefault {
					pr.Servers[0].Default = true
				}
			}
			return app.profiles.Edit(pr)
		},
	}
}

func newServerListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list <profile>",
		Aliases: []string{"ls"},
		Short:   "List service addresses of a profile",
		Args:    cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			app, err := loadApp()
			if err != nil {
				return err
			}
			pr, ok := app.profiles.Get(args[0])
			if !ok {
				return fmt.Errorf("profile %q does not exist", args[0])
			}
			if len(pr.Servers) == 0 {
				fmt.Println("(no servers)")
				return nil
			}
			for _, s := range pr.Servers {
				def := ""
				if s.Default {
					def = " (default)"
				}
				fmt.Printf("  %s%s\n", s.Addr, def)
			}
			return nil
		},
	}
}