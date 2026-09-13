package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/miconda/sipexer/internal/store"
)

// parseNumber parses "alias=user@domain" / "user@domain" / "user".
// Returns (Number, alias) or an error.
func parseNumber(s string) (store.Number, string, error) {
	var alias string
	if idx := strings.Index(s, "="); idx >= 0 {
		alias = strings.TrimSpace(s[:idx])
		s = strings.TrimSpace(s[idx+1:])
	}
	if s == "" {
		return store.Number{}, "", fmt.Errorf("empty number")
	}
	at := strings.Index(s, "@")
	if at < 0 {
		return store.Number{User: s, Alias: alias}, alias, nil
	}
	return store.Number{User: s[:at], Domain: s[at+1:], Alias: alias}, alias, nil
}

// promptPassword reads a password from the TTY without echo. Non-TTY stdin
// returns an error (callers turn it into guidance).
func promptPassword(prompt string) (string, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return "", fmt.Errorf("cannot read password: stdin is not a terminal (pass --auth-password or set it in the profile)")
	}
	fmt.Fprint(os.Stderr, prompt)
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", fmt.Errorf("read password: %w", err)
	}
	return string(b), nil
}

// applyProfileFlags reads the phone-add/edit flag set into a Profile. Only
// flags the user actually changed are applied (so edit preserves the rest).
func applyProfileFlags(cmd *cobra.Command, pr *store.Profile, isAdd bool) error {
	f := cmd.Flags()
	var err error

	parseNums := func() error {
		vals, _ := f.GetStringArray("number")
		pr.Numbers = nil
		for _, v := range vals {
			n, _, perr := parseNumber(v)
			if perr != nil {
				return perr
			}
			pr.Numbers = append(pr.Numbers, n)
		}
		return nil
	}
	if isAdd || f.Changed("number") {
		if err = parseNums(); err != nil {
			return err
		}
	}

	parseServers := func() {
		vals, _ := f.GetStringArray("server")
		pr.Servers = nil
		for _, v := range vals {
			pr.Servers = append(pr.Servers, store.Server{Addr: v})
		}
	}
	if isAdd || f.Changed("server") {
		parseServers()
	}
	if f.Changed("server-default") && isAdd && len(pr.Servers) > 0 {
		pr.Servers[0].Default = true
	}

	if f.Changed("auth-user") {
		pr.AuthUser, _ = f.GetString("auth-user")
	}
	if f.Changed("ha1") {
		pr.HA1 = store.NewString(pr.HA1.Plain())
	}
	if f.Changed("auth-password") {
		pr.AuthPass = store.NewString(mustGetString(f, "auth-password"))
	}

	// typed defaults: int pointers
	for _, opt := range []struct {
		name string
		dst  **int
	}{
		{"sw", &pr.DialDefaults.SessionWaitMs},
		{"cd", &pr.DialDefaults.CallDurationMs},
		{"rt", &pr.DialDefaults.RingTimeMs},
		{"timeout", &pr.DialDefaults.TimeoutMs},
		{"timeout-connect", &pr.DialDefaults.TimeoutConnectMs},
		{"timeout-write", &pr.DialDefaults.TimeoutWriteMs},
		{"vl", &pr.DialDefaults.Verbosity},
	} {
		if isAdd || f.Changed(opt.name) {
			if v, err := f.GetInt(opt.name); err == nil && v != 0 {
				val := v
				*opt.dst = &val
			} else if err != nil {
				return err
			}
		}
	}

	// typed defaults: bool pointers
	for _, opt := range []struct {
		name string
		dst  **bool
	}{
		{"co", &pr.DialDefaults.ColorOutput},
		{"com", &pr.DialDefaults.ColorMessage},
		{"ti", &pr.DialDefaults.TLSInsecure},
	} {
		if isAdd || f.Changed(opt.name) {
			v, _ := f.GetBool(opt.name)
			if v {
				val := true
				*opt.dst = &val
			}
		}
	}

	// string call defaults
	for _, opt := range []struct {
		name string
		dst  *string
	}{
		{"ex", &pr.Expires},
		{"ua", &pr.UserAgent},
		{"cu", &pr.ContactURI},
		{"ct", &pr.ContentType},
		{"mb", &pr.Body},
		{"method", &pr.Method},
	} {
		if isAdd || f.Changed(opt.name) {
			*opt.dst = mustGetString(f, opt.name)
		}
	}

	// scenario toggles
	for _, opt := range []struct {
		name string
		dst  *bool
	}{
		{"register-first", &pr.RegisterFirst},
		{"set-user", &pr.SetUser},
		{"contact-build", &pr.ContactBuild},
	} {
		if isAdd || f.Changed(opt.name) {
			v, _ := f.GetBool(opt.name)
			*opt.dst = v
		}
	}

	if isAdd || f.Changed("xh") {
		pr.ExtraHeaders, _ = f.GetStringArray("xh")
	}
	if isAdd || f.Changed("extra-dial-flag") {
		pr.ExtraDialFlags, _ = f.GetStringArray("extra-dial-flag")
	}

	return nil
}

func mustGetString(f interface{ GetString(string) (string, error) }, name string) string {
	v, _ := f.GetString(name)
	return v
}

// registerPhoneFlags adds the shared phone add/edit flags.
func registerPhoneFlags(cmd *cobra.Command) {
	f := cmd.Flags()
	f.StringArray("number", nil, "registered number in the profile: [alias=]user[@domain] (repeatable)")
	f.StringArray("server", nil, "service address proto://host:port (repeatable)")
	f.BoolP("server-default", "d", false, "mark the first --server as default")
	f.String("auth-user", "", "SIP authentication username")
	f.String("auth-password", "", "SIP authentication password (prompted if empty and stdin is a terminal)")
	f.String("ha1", "", "HA1 digest of the password (used with --auth-user)")
	f.Int("sw", 0, "session wait milliseconds (-sw)")
	f.Int("cd", 0, "call duration milliseconds (-cd)")
	f.Int("rt", 0, "ring time milliseconds (-rt)")
	f.Int("timeout", 0, "receive timeout ms (-timeout)")
	f.Int("timeout-connect", 0, "connect timeout ms (-timeout-connect)")
	f.Int("timeout-write", 0, "write timeout ms (-timeout-write)")
	f.Int("vl", 0, "verbosity 0..3 (-vl)")
	f.Bool("co", false, "color output (-co)")
	f.Bool("com", false, "color SIP message output (-com)")
	f.Bool("ti", false, "skip TLS certificate validation (-ti)")
	f.String("ex", "", "expires header value (-ex)")
	f.String("ua", "", "user agent (-ua)")
	f.String("cu", "", "contact header URI (-cu)")
	f.String("ct", "", "content type (-ct)")
	f.String("mb", "", "message body (-mb)")
	f.String("method", "", "SIP method, e.g. INVITE/REGISTER/OPTIONS (-i/-r/-o equivalent)")
	f.Bool("register-first", false, "send REGISTER before the request (-register-first)")
	f.Bool("set-user", false, "set R-URI user to To-URI user for the proxy (-su)")
	f.Bool("contact-build", false, "build Contact header from the local address (-cb)")
	f.StringArray("xh", nil, "extra header in `name:body` form (repeatable, -xh)")
	f.StringArray("extra-dial-flag", nil, "additional raw engine flag string, appended verbatim on every dial")
}

func newPhoneCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "phone",
		Aliases: []string{"profiles"},
		Short:   "Manage SIP phone profiles",
	}

	cmd.AddCommand(newPhoneAddCmd(), newPhoneEditCmd(), newPhoneListCmd(), newPhoneRMCmd(), newPhoneShowCmd(), newPhoneDefaultCmd())
	return cmd
}

// newPhoneDefaultCmd marks the "current" profile used by bare `sipx dial`.
func newPhoneDefaultCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "default [name]",
		Aliases: []string{"current", "use"},
		Short:   "Set or show the default profile used by 'sipx dial' without a name",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			app, err := loadApp()
			if err != nil {
				return err
			}
			if len(args) == 0 {
				if def, ok := app.profiles.DefaultProfile(); ok {
					mark := ""
					if !def.Default && len(app.profiles.List()) == 1 {
						mark = " (implicit: only profile)"
					}
					fmt.Printf("current profile: %s%s\n", def.Name, mark)
				} else {
					fmt.Println("no default profile set")
				}
				return nil
			}
			if err := app.profiles.SetDefault(args[0]); err != nil {
				return err
			}
			fmt.Printf("current profile set to: %s\n", args[0])
			return nil
		},
	}
}

func newPhoneAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Create a new phone profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			app, err := loadApp()
			if err != nil {
				return err
			}
			name := args[0]
			if _, ok := app.profiles.Get(name); ok {
				return fmt.Errorf("profile %q already exists (use 'sipx phone edit')", name)
			}
			pr := store.Profile{Name: name}
			if err := applyProfileFlags(c, &pr, true); err != nil {
				return err
			}
			if len(pr.Numbers) == 0 {
				fmt.Fprintf(os.Stderr, "warning: profile has no registered numbers yet; add with --number\n")
			}
			if len(pr.Servers) == 0 {
				fmt.Fprintf(os.Stderr, "warning: profile has no service address yet; add with --server\n")
			}
			// Prompt for password if not provided and we have an auth-user.
			if pr.AuthUser != "" && pr.AuthPass.Plain() == "" && pr.HA1.Plain() == "" {
				if c.Flags().Changed("auth-password") || c.Flags().Changed("ha1") {
					_ = c // password came via flag; nothing to prompt
				} else {
					pw, perr := promptPassword(fmt.Sprintf("Password for %s: ", pr.AuthUser))
					if perr != nil {
						return perr
					}
					pr.AuthPass = store.NewString(pw)
				}
			}
			return app.profiles.Add(pr)
		},
	}
	registerPhoneFlags(cmd)
	return cmd
}

func newPhoneEditCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit <name>",
		Short: "Edit an existing phone profile (fields not given are preserved)",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			app, err := loadApp()
			if err != nil {
				return err
			}
			pr, ok := app.profiles.Get(args[0])
			if !ok {
				return fmt.Errorf("profile %q does not exist (use add)", args[0])
			}
			if err := applyProfileFlags(c, &pr, false); err != nil {
				return err
			}
			// Prompt for a new password when editing and none is set: the
			// user may press enter to keep whatever was stored.
			if pr.AuthUser != "" && pr.AuthPass.Plain() == "" && pr.HA1.Plain() == "" && c.Flags().Changed("auth-user") {
				pw, perr := promptPassword(fmt.Sprintf("Password for %s (enter to skip): ", pr.AuthUser))
				if perr != nil {
					return perr
				}
				if pw != "" {
					pr.AuthPass = store.NewString(pw)
				}
			}
			return app.profiles.Edit(pr)
		},
	}
	registerPhoneFlags(cmd)
	return cmd
}

func newPhoneListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List phone profiles (secrets redacted)",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, args []string) error {
			app, err := loadApp()
			if err != nil {
				return err
			}
			names := app.profiles.Names()
			if len(names) == 0 {
				fmt.Println("no profiles")
				return nil
			}
			for _, name := range names {
				pr, _ := app.profiles.Get(name)
				mark := ""
				if pr.Default {
					mark = " *"
				}
				fmt.Printf("%-16s%s auth=%s numbers=%d servers=%s\n", pr.Name, mark, redact(pr.AuthPass), len(pr.Numbers), serverSummary(pr.Servers))
			}
			return nil
		},
	}
}

func newPhoneRMCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "rm <name>",
		Aliases: []string{"remove", "del"},
		Short:   "Remove a phone profile",
		Args:    cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			app, err := loadApp()
			if err != nil {
				return err
			}
			return app.profiles.Remove(args[0])
		},
	}
}

func newPhoneShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <name>",
		Short: "Show a phone profile's details",
		Args:  cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			app, err := loadApp()
			if err != nil {
				return err
			}
			pr, ok := app.profiles.Get(args[0])
			if !ok {
				return fmt.Errorf("profile %q does not exist", args[0])
			}
			fmt.Printf("name:         %s\n", pr.Name)
			fmt.Printf("auth user:    %s\n", pr.AuthUser)
			fmt.Printf("auth secret:  %s\n", redact(pr.AuthPass))
			fmt.Println("numbers:")
			for _, n := range pr.Numbers {
				fmt.Printf("  %s%s\n", n.String(), aliasSuffix(n))
			}
			fmt.Println("servers:")
			for _, s := range pr.Servers {
				def := ""
				if s.Default {
					def = " (default)"
				}
				fmt.Printf("  %s%s\n", s.Addr, def)
			}
			if pr.Method != "" {
				fmt.Printf("method:       %s\n", pr.Method)
			}
			fmt.Printf("dial flags:   %d extra, %d after-dash\n", len(pr.ExtraDialFlags), len(pr.ExtraFlagsAfterDash))
			return nil
		},
	}
}

func redact(s store.String) string {
	if s.Plain() == "" {
		return "-"
	}
	return "••••••"
}

func aliasSuffix(n store.Number) string {
	if n.Alias != "" {
		return " (alias " + n.Alias + ")"
	}
	return ""
}

func serverSummary(ss []store.Server) string {
	var out []string
	for _, s := range ss {
		if s.Default {
			out = append(out, s.Addr+"*")
		} else {
			out = append(out, s.Addr)
		}
	}
	if len(out) == 0 {
		return "-"
	}
	return strings.Join(out, ", ")
}