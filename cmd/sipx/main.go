package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/miconda/sipexer/internal/cli"
)

// sipx is the wrapped CLI built on cobra/pflag that manages phone profiles
// and dial history, and drives the original sipexer binary as a subprocess.
func main() {
	err := cli.Execute()
	var ee *cli.EngineExit
	if errors.As(err, &ee) {
		// Forward the engine's exit code verbatim (design.md D0). Note that
		// sipexer exits with the final SIP response code on a successful
		// call, so a code >=200 here is an outcome, not an error.
		os.Exit(ee.Code)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "sipx:", err)
		os.Exit(1)
	}
}