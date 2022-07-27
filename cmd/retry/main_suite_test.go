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

package main_test

import (
	"context"
	"io"
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/homeport/retry/cmd/retry"
)

func TestRetry(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Retry")
}

type settings struct {
	ctx    context.Context
	output io.Writer
	args   []string
}

type testOption func(*settings)

func retry(args ...string) testOption {
	return func(s *settings) {
		s.args = args
	}
}

func withOutput(w io.Writer) testOption {
	return func(s *settings) {
		s.output = w
	}
}

func withContext(ctx context.Context) testOption {
	return func(s *settings) {
		s.ctx = ctx
	}
}

func that(options ...testOption) error {
	stdin, stdout, stderr, args := os.Stdin, os.Stdout, os.Stderr, os.Args
	defer func() {
		os.Stdin = stdin
		os.Stdout = stdout
		os.Stderr = stderr
		os.Args = args
	}()

	var cfg = settings{}
	for _, option := range options {
		option(&cfg)
	}

	if cfg.ctx == nil {
		cfg.ctx = context.Background()
	}

	if cfg.output == nil {
		cfg.output = GinkgoWriter
	}

	r, w, err := os.Pipe()
	Expect(err).ToNot(HaveOccurred())

	os.Stdout = w
	os.Stderr = w
	os.Args = append([]string{"retry"}, cfg.args...)
	err = Execute(cfg.ctx)

	w.Close()

	_, copyErr := io.Copy(cfg.output, r)
	Expect(copyErr).ToNot(HaveOccurred())

	return err
}
