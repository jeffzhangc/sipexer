// Package cli implements the cobra command tree for the sipx wrapped binary.
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// engineFlag is the --engine path override, resolved before dial/doctor runs.
var engineFlag string

// rootCmd is the sipx root command.
var rootCmd = &cobra.Command{
	Use:   "sipx",
	Short: "SIP phone wrapper around sipexer",
	Long: `sipx manages named SIP phone configurations and dial history, and drives
the original sipexer binary as a subprocess to send SIP requests.

Configure profiles with 'sipx phone', manage service addresses with
'sipx server', dial with 'sipx dial', and review past calls with
'sipx history'.`,
	Version: "2.0.0-wrapped",
}

// Execute runs the sipx command tree. Any engine-exit error is translated
// into the engine's exit code by the caller.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&engineFlag, "engine", "", "path to the sipexer engine binary (else $SIPEXER_BIN, then alongside sipx, then PATH)")
	rootCmd.PersistentFlags().StringVar(&configDirFlag, "config-dir", "", "directory for profiles.json and history.json (else $SIPEXER_CONFIG_DIR, then XDG/gome config)")
	rootCmd.AddCommand(
		newDoctorCmd(),
		newPhoneCmd(),
		newServerCmd(),
		newHistoryCmd(),
		newDialCmd(),
	)

	// --version is handled by cobra (Version field); silence the "unknown
	// flag" for --version by letting cobra's version flag be registered.
	rootCmd.SetVersionTemplate(fmt.Sprintf("%s v{{.Version}}\n", rootCmd.Name()))
}