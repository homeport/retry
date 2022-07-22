// Copyright © 2022 The Homeport Team
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package cmd

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
	"time"

	_ "embed"

	"github.com/avast/retry-go/v4"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var version = "HEAD"

//go:embed examples.tmpl
var examplesTemplate string

type options struct {
	showVersion bool
	quiet       bool
	attempts    uint
	delay       time.Duration
}

var defaults = options{
	showVersion: false,
	quiet:       false,
	attempts:    3,
	delay:       2 * time.Second,
}

var rootCmdSettings options

var rootCmd = &cobra.Command{
	Use:           fmt.Sprintf("%s [%s flags] [--] command [command flags] [command arguments] [...]", executableName(), executableName()),
	Short:         "Tool to retry a command in case it fails",
	Long:          `Tool to retry a command in case it fails.`,
	Example:       examples(),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		// Show version number and bail out if version flag was specified
		if rootCmdSettings.showVersion {
			_, err = fmt.Println(version)
			return err
		}

		// Back out if nothing was specified
		if len(args) == 0 {
			return cmd.Usage()
		}

		// In case data was piped into this process, create a temporary
		// buffer to store the Stdin to have copies for each retry
		var input []byte
		if !term.IsTerminal(int(os.Stdin.Fd())) {
			input, err = io.ReadAll(os.Stdin)
			if err != nil {
				return err
			}
		}

		// Enable or disable output in case of an error and a retry, even with
		// no quiet flag, there is no additional output if the command succeeds
		var onRetry retry.OnRetryFunc
		if rootCmdSettings.quiet {
			onRetry = func(_ uint, _ error) {}
		} else {
			onRetry = func(n uint, err error) { fmt.Fprintf(os.Stderr, "command failed at attempt #%d: %v\n", n+1, err) }
		}

		// Feed the command line arguments as-is into a command execution and
		// retry until the command succeeds, or maximum attempts is reached
		var cmdName, cmdArgs = args[0], args[1:]
		return retry.Do(
			func() error {
				command := exec.CommandContext(cmd.Context(), cmdName, cmdArgs...)
				command.Stdin = bytes.NewReader(input)
				command.Stdout = os.Stdout
				command.Stderr = os.Stderr

				return command.Run()
			},
			retry.OnRetry(onRetry),
			retry.Context(cmd.Context()),
			retry.Attempts(rootCmdSettings.attempts),
			retry.Delay(rootCmdSettings.delay),
			retry.DelayType(retry.BackOffDelay),
		)
	},
}

func init() {
	rootCmd.Flags().SortFlags = false
	rootCmd.Flags().UintVar(&rootCmdSettings.attempts, "attempts", defaults.attempts, "Number of attempts")
	rootCmd.Flags().DurationVar(&rootCmdSettings.delay, "delay", defaults.delay, "Initial delay between attempts")
	rootCmd.Flags().BoolVar(&rootCmdSettings.quiet, "quiet", defaults.quiet, "Disable output for failed attempts")
	rootCmd.Flags().BoolVar(&rootCmdSettings.showVersion, "version", defaults.showVersion, "Show tool version")
}

func executableName() string {
	var result = "retry"
	if executable, err := os.Executable(); err == nil {
		result = filepath.Clean(filepath.Base(executable))
	}

	return result
}

func examples() string {
	tmpl, err := template.New("examples").Parse(examplesTemplate)
	if err != nil {
		log.Fatalln(err)
	}

	data := map[string]any{
		"Name":     executableName(),
		"Attempts": defaults.attempts,
		"Delay":    defaults.delay,
	}

	var buf bytes.Buffer
	if err = tmpl.Execute(&buf, data); err != nil {
		log.Fatalln(err)
	}

	return buf.String()
}

func Execute() error {
	rootCmdSettings = defaults
	return rootCmd.Execute()
}
