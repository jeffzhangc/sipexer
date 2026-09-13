package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/miconda/sipexer/internal/dial"
	"github.com/miconda/sipexer/internal/engine"
	"github.com/miconda/sipexer/internal/store"
)

func newDialCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dial [profile] [target]",
		Short: "Dial a target through a profile (history is recorded)",
		Long: `Place an outbound SIP call through a configured profile.

The optional target is resolved in this order:
  @N        history entry N (most recent = @0)
  alias     a configured number alias, then a history-entry alias
  number    a raw number or SIP URI

With no target and a terminal, sipx shows the profile's recent history and
prompts for a choice. Raw engine flags go after '--', e.g.
  sipx dial work 1001 -- -vl 3

Call teardown: the engine sends BYE automatically when the session wait (-sw)
expires after a successful INVITE. Interrupting a dial with Ctrl-C kills the
engine without a BYE, leaving the gateway to time the dialog out — prefer -sw
for graceful hangup.`,
		Args: cobra.ArbitraryArgs, // passthrough after "--" may contain anything
		// Engine exit codes are surfaced by main, not as cobra errors; and a
		// dial failure should never dump the usage text.
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(c *cobra.Command, args []string) error {
			return runDial(c, args)
		},
	}

	// per-dial overrides for the engine
	f := cmd.Flags()
	f.StringP("server", "s", "", "override the profile's service address")
	f.StringP("from", "f", "", "override the profile's from-number (user)")
	f.String("method", "", "override the profile's default method")
	f.Int("sw", 0, "override session wait (-sw)")
	f.Int("vl", 0, "override verbosity (-vl)")
	f.Int("cd", 0, "override call duration (-cd)")
	f.Int("rt", 0, "override ring time (-rt)")
	f.Bool("co", false, "override color output (-co)")
	f.Bool("no-record", false, "do not record this dial in history")

	return cmd
}

// runDial implements the core dial flow (design.md D2/D7/D3).
func runDial(c *cobra.Command, args []string) error {
	app, err := loadApp()
	if err != nil {
		return err
	}

	// Raw engine passthrough after "--" is NOT positional; collect it verbatim
	// and slice it off the trailing args (cobra includes it as trailing args).
	passthrough := afterDoubleDash(c)
	if len(passthrough) > 0 && len(args) >= len(passthrough) {
		args = args[:len(args)-len(passthrough)]
	}

	var profileName, targetArg string
	// The reserved names "default" and "current" always refer to the current
	// profile, so `sipx dial default 1001` is explicit shorthand.
	if len(args) > 0 && (args[0] == "default" || args[0] == "current") {
		def, ok := app.profiles.DefaultProfile()
		if !ok {
			return fmt.Errorf("no default profile set; run 'sipx phone default <name>'")
		}
		profileName = def.Name
		args = args[1:]
	}
	// args may be [target] or [profile target]
	switch len(args) {
	case 1:
		// Could be profile name or a raw target.
		if _, ok := app.profiles.Get(args[0]); ok {
			profileName = args[0]
		} else {
			targetArg = args[0]
		}
	case 2:
		profileName, targetArg = args[0], args[1]
	}
	if profileName == "" {
		// No profile given: use the default ("current") profile.
		if def, ok := app.profiles.DefaultProfile(); ok {
			profileName = def.Name
		} else {
			names := app.profiles.Names()
			if len(names) == 0 {
				return fmt.Errorf("no profiles configured; add one with 'sipx phone add'")
			}
			return fmt.Errorf("multiple profiles and no default set: 'sipx phone default <name>', or dial one of %v", names)
		}
	}
	pr, ok := app.profiles.Get(profileName)
	if !ok {
		return fmt.Errorf("profile %q does not exist", profileName)
	}

	// Resolve the dial.
	resolved, err := dial.ResolveTarget(pr, app.history, targetArg)
	if err != nil {
		return err
	}

	// If no target was given, run the interactive picker (TTY only).
	if resolved == nil {
		if isTerminal() {
			resolved, err = pickTarget(app, pr)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("no target given and stdin is not a terminal; provide a number, @<index>, an alias, or a history alias")
		}
	}

	// Server: resolved or profile default / CLI override.
	server := resolved.Server
	if server == "" {
		server, err = app.profiles.SelectServer(pr)
		if err != nil {
			return err
		}
	}
	if c.Flags().Changed("server") {
		server, _ = c.Flags().GetString("server")
	}

	// From-number for the dial: resolved number, else CLI override, else profile default.
	from := resolved.Target
	if c.Flags().Changed("from") {
		f, _ := c.Flags().GetString("from")
		from = store.Number{User: f}
	} else if from.User == "" {
		from, err = app.profiles.SelectFromNumber(pr, "")
		if err != nil {
			return err
		}
	}

	// Determine per-dial overrides (CLI flags beat profile defaults).
	overrides, overrideErr := dialOverrides(c)
	if overrideErr != nil {
		return overrideErr
	}
	merged := mergeProfileOverrides(pr, overrides)

	// Engine discovery.
	bin, err := engine.Locate(engineFlag)
	if err != nil {
		return err
	}

	// Build the argv and run the engine.
	argv := dial.BuildArguments(merged, from, resolved.To, server, pr.ExtraDialFlags, passthrough)
	fmt.Fprintf(c.ErrOrStderr(), "sipx: dial %q through %q -> %s (%v)\n", resolved.To, profileName, server, argv)
	res := engine.Run(c.Context(), bin, argv)
	if res.StartErr != nil {
		return fmt.Errorf("run engine: %w", res.StartErr)
	}

	// Record history (attempt, regardless of engine exit code).
	if !mustNoRecord(c) {
		e := store.Entry{
			Profile:  profileName,
			Server:   server,
			FromUser: from.User,
			Target:   resolved.To,
		}
		if err := app.history.Append(e); err != nil {
			fmt.Fprintf(c.ErrOrStderr(), "warning: could not record history: %v\n", err)
		}
	}

	return engineExitError(res)
}

// engineExitError converts a non-zero engine result into an error that main
// translates into the engine's exit code (design.md D0/D7). Note sipexer's
// own convention: on a successful call the engine exits with the SIP final
// response code (e.g. 200), so a forwarded non-zero is not necessarily a
// failure.
func engineExitError(res engine.Result) error {
	if res.ExitCode != 0 {
		return &EngineExit{Code: res.ExitCode}
	}
	return nil
}

// EngineExit is an error carrying the engine's exit code so main can exit
// with it verbatim without printing a usage dump.
type EngineExit struct{ Code int }

func (e *EngineExit) Error() string {
	if e.Code >= 200 && e.Code < 300 {
		return fmt.Sprintf("engine finished, last SIP response %d", e.Code)
	}
	return fmt.Sprintf("engine exited with code %d", e.Code)
}

func mustNoRecord(c *cobra.Command) bool {
	v, _ := c.Flags().GetBool("no-record")
	return v
}

// afterDoubleDash finds the first standalone "--" in the original argv and
// returns everything after it verbatim. We scan os.Args directly because
// cobra's ArgsLenAtDash() semantics differ across versions and cobra may
// already have stripped flags by the time RunE runs.
func afterDoubleDash(c *cobra.Command) []string {
	all := os.Args[1:]
	for i := 0; i < len(all); i++ {
		if all[i] == "--" {
			return all[i+1:]
		}
	}
	return nil
}

	// dialOverrides collects raw CLI flag overrides for this dial into an
	// Overrides struct (nil pointers mean "not overridden").
	func dialOverrides(c *cobra.Command) (dial.Overrides, error) {
		var o dial.Overrides
		f := c.Flags()

		fn := func(name string, dst **int) {
			if f.Changed(name) {
				v, _ := f.GetInt(name)
				*dst = &v
			}
		}
		fn("sw", &o.SessionWait)
		fn("vl", &o.Verbosity)
		fn("cd", &o.CallDuration)
		fn("rt", &o.RingTime)

		if f.Changed("co") {
			v, _ := f.GetBool("co")
			if v {
				b := true
				o.ColorOutput = &b
			}
		}
		if f.Changed("method") {
			m, _ := f.GetString("method")
			o.Method = &m
		}
		return o, nil
}

// mergeProfileOverrides applies per-dial CLI overrides on top of the profile's
// typed defaults, returning an effective profile for BuildArguments.
func mergeProfileOverrides(p store.Profile, o dial.Overrides) store.Profile {
	if o.Method != nil {
		p.Method = *o.Method
	}
	if o.SessionWait != nil {
		p.DialDefaults.SessionWaitMs = o.SessionWait
	}
	if o.Verbosity != nil {
		p.DialDefaults.Verbosity = o.Verbosity
	}
	if o.CallDuration != nil {
		p.DialDefaults.CallDurationMs = o.CallDuration
	}
	if o.RingTime != nil {
		p.DialDefaults.RingTimeMs = o.RingTime
	}
	if o.ColorOutput != nil {
		p.DialDefaults.ColorOutput = o.ColorOutput
	}
	return p
}