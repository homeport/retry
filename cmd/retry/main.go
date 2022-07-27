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

package main

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	_ "embed"

	"github.com/avast/retry-go/v4"
	"github.com/spf13/pflag"
	"golang.org/x/term"
)

// will be overwritten by build
var version string = "HEAD"

//go:embed usage.tmpl
var usageTemplate string

type settings struct {
	showVersion bool
	quiet       bool
	attempts    uint
	delay       time.Duration
}

var defaults = settings{
	showVersion: false,
	quiet:       false,
	attempts:    3,
	delay:       2 * time.Second,
}

var preferences settings

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		cancel()
		os.Exit(1)
	}()

	if err := Execute(ctx); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func executableName() string {
	var result = "retry"
	if executable, err := os.Executable(); err == nil {
		result = filepath.Clean(filepath.Base(executable))
	}

	return result
}

func Execute(ctx context.Context) (err error) {
	preferences = defaults

	// Setup command line flag set that does not fail on unknown flags
	// and with custom usage output based on file template
	fs := pflag.NewFlagSet(os.Args[0], pflag.ContinueOnError)
	fs.SortFlags = false
	fs.ParseErrorsWhitelist = pflag.ParseErrorsWhitelist{UnknownFlags: true}
	fs.Usage = func() {
		tmpl, _ := template.New("usage").Parse(usageTemplate)
		data := map[string]any{
			"Name":     executableName(),
			"Flags":    fs.FlagUsages(),
			"Attempts": defaults.attempts,
			"Delay":    defaults.delay,
		}

		_ = tmpl.Execute(os.Stdout, data)
	}

	// Command line flags
	fs.UintVar(&preferences.attempts, "attempts", defaults.attempts, "Number of attempts")
	fs.DurationVar(&preferences.delay, "delay", defaults.delay, "Initial delay between attempts")
	fs.BoolVar(&preferences.quiet, "quiet", defaults.quiet, "Disable output for failed attempts")
	fs.BoolVar(&preferences.showVersion, "version", defaults.showVersion, "Show tool version")
	_ = fs.Parse(os.Args[1:])

	// Slice with all non-flag arguments (everything but the specified flags)
	args := fs.Args()

	// Show version number and bail out if version flag was specified
	if preferences.showVersion {
		_, err = fmt.Printf("%s\n", version)
		return err
	}

	// Back out if nothing was specified (only binary name)
	if len(args) == 0 {
		fs.Usage()
		return fmt.Errorf("no command specified")
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
	if preferences.quiet {
		onRetry = func(_ uint, _ error) {}
	} else {
		onRetry = func(n uint, err error) { fmt.Fprintf(os.Stderr, "command failed at attempt #%d: %v\n", n+1, err) }
	}

	// Feed the command line arguments as-is into a command execution and
	// retry until the command succeeds, or maximum attempts is reached
	var cmdName, cmdArgs = args[0], args[1:]
	return retry.Do(
		func() error {
			command := exec.CommandContext(ctx, cmdName, cmdArgs...)
			command.Stdin = bytes.NewReader(input)
			command.Stdout = os.Stdout
			command.Stderr = os.Stderr

			return command.Run()
		},
		retry.OnRetry(onRetry),
		retry.Context(ctx),
		retry.Attempts(preferences.attempts),
		retry.Delay(preferences.delay),
		retry.DelayType(retry.BackOffDelay),
	)
}
