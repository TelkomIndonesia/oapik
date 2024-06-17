package cmd

import (
	"fmt"
	"os"
)

func write(dest string, content []byte) (err error) {
	switch dest {
	case "-":
		_, err = os.Stdout.Write(content)
		if err != nil {
			return fmt.Errorf("failed to write to stdout: %w", err)
		}
	default:
		err = os.WriteFile(dest, content, 0644)
		if err != nil {
			return fmt.Errorf("failed to write to %s: %w", dest, err)
		}
	}
	return
}
