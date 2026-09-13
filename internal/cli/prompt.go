package cli

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

// promptConfirm asks a yes/no question on the terminal. Returns an error when
// stdin is not a terminal (callers decide whether that should fail or skip).
func promptConfirm(question string) (bool, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return false, fmt.Errorf("interactive confirmation requires a terminal")
	}
	fmt.Fprint(os.Stderr, question)
	var reply string
	_, err := fmt.Fscanln(os.Stdin, &reply)
	if err != nil {
		// EOF with empty reply counts as "no".
		return false, nil
	}
	switch reply {
	case "y", "Y", "yes", "YES":
		return true, nil
	default:
		return false, nil
	}
}