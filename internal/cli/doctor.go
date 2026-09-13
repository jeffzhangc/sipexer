package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/miconda/sipexer/internal/engine"
)

// newDoctorCmd builds the `sipx doctor` command, which reports the discovered
// engine binary, its executability, and its version output.
func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Report the discovered sipexer engine binary",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := engine.Locate(engineFlag)
			if err != nil {
				return err
			}

			info, statErr := os.Stat(path)
			fmt.Printf("engine path: %s\n", path)
			switch {
			case statErr != nil:
				fmt.Printf("  executable: no (%v)\n", statErr)
				return nil
			case info.IsDir():
				fmt.Printf("  executable: no (is a directory)\n")
				return nil
			case info.Mode()&0o111 == 0:
				fmt.Printf("  executable: no (missing execute bit)\n")
				return nil
			default:
				fmt.Printf("  executable: yes\n")
			}

			fmt.Printf("version: ")
			res := engine.Run(cmd.Context(), path, []string{"--version"})
			if res.StartErr != nil {
				fmt.Printf("(could not run: %v)\n", res.StartErr)
				return nil
			}
			return nil
		},
	}
}