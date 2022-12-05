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
	"strconv"
	"syscall"
	"time"

	_ "embed"

	"github.com/avast/retry-go/v4"
	"golang.org/x/term"
)

// will be overwritten by build
var version = "HEAD"

// environment variables to configure tool behavior
const (
	RetryAttempts = "RETRY_ATTEMPTS"
	RetryDelay    = "RETRY_DELAY"
	RetryBeQuiet  = "RETRY_BEQUIET"
)

//go:embed usage.tmpl
var usageTemplate string

type settings struct {
	beQuiet  bool
	attempts uint
	delay    time.Duration
}

var defaults = settings{
	beQuiet:  true,
	attempts: 3,
	delay:    30 * time.Second,
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

func usage() {
	tmpl, _ := template.New("usage").Parse(usageTemplate)
	data := map[string]any{
		"Name":           executableName(),
		"Version":        version,
		"EnvVarAttempts": RetryAttempts,
		"Attempts":       defaults.attempts,
		"EnvVarDelay":    RetryDelay,
		"Delay":          defaults.delay,
		"EnvVarBeQuiet":  RetryBeQuiet,
		"BeQuiet":        defaults.beQuiet,
	}

	_ = tmpl.Execute(os.Stderr, data)
}

func lookupAttempts() (uint, error) {
	if val, ok := os.LookupEnv(RetryAttempts); ok {
		attempts, err := strconv.ParseUint(val, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("cannot parse %q as a number", val)
		}

		return uint(attempts), nil
	}

	return defaults.attempts, nil
}

func lookupDelay() (time.Duration, error) {
	if val, ok := os.LookupEnv(RetryDelay); ok {
		delay, err := time.ParseDuration(val)
		if err != nil {
			return 0, fmt.Errorf("cannot parse %q as time duration", val)
		}

		return delay, nil
	}

	return defaults.delay, nil
}

func lookupBeQuiet() (bool, error) {
	if val, ok := os.LookupEnv(RetryBeQuiet); ok {
		beQuiet, err := strconv.ParseBool(val)
		if err != nil {
			return false, fmt.Errorf("cannot parse %q as boolean", val)
		}

		return beQuiet, nil
	}

	return defaults.beQuiet, nil
}

func Execute(ctx context.Context) (err error) {
	preferences.attempts, err = lookupAttempts()
	if err != nil {
		return err
	}

	preferences.delay, err = lookupDelay()
	if err != nil {
		return err
	}

	preferences.beQuiet, err = lookupBeQuiet()
	if err != nil {
		return err
	}

	var args []string
	for i, arg := range os.Args {
		switch {
		case i == 0:
			continue

		case arg == "--":
			continue

		default:
			args = append(args, arg)
		}
	}

	// Back out if nothing was specified (only binary name)
	if len(args) == 0 {
		usage()
		return fmt.Errorf("no command specified")
	}

	// In case data was piped into this process, mark that the input needs to
	// be buffered internally so that it can be used multiple times
	var buffer []byte
	var bufferStdin bool
	if bufferStdin = !term.IsTerminal(int(os.Stdin.Fd())); bufferStdin {
		buffer, err = io.ReadAll(os.Stdin)
		if err != nil {
			return err
		}
	}

	// Feed the command line arguments as-is into a command execution and
	// retry until the command succeeds, or maximum attempts is reached
	var cmdName, cmdArgs = args[0], args[1:]
	return retry.Do(
		func() error {
			var in io.Reader = os.Stdin
			if bufferStdin {
				in = bytes.NewReader(buffer)
			}

			command := exec.CommandContext(ctx, cmdName, cmdArgs...)
			command.Stdin = in
			command.Stdout = os.Stdout
			command.Stderr = os.Stderr

			return command.Run()
		},
		retry.OnRetry(func(n uint, err error) {
			if !preferences.beQuiet {
				fmt.Fprintf(os.Stderr, "command failed at attempt #%d: %v\n", n+1, err)
			}
		}),
		retry.Context(ctx),
		retry.Attempts(preferences.attempts),
		retry.Delay(preferences.delay),
		retry.DelayType(retry.BackOffDelay),
	)
}
