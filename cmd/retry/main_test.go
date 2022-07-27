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
	"bytes"
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Retry tool", func() {
	Context("use cases", func() {
		It("should fail if no command is provided", func() {
			Expect(that(
				retry(),
			)).ToNot(Succeed())
		})

		It("should succeed if command returns zero exit code", func() {
			Expect(that(
				retry("true"),
			)).To(Succeed())
		})

		It("should print the version info", func() {
			Expect(that(
				retry("--version"),
			)).To(BeNil())
		})

		It("should ignore unknown flags that are probably flags for the command to be retried", func() {
			Expect(that(
				retry("true", "--flag"),
			)).To(BeNil())
		})

		It("should fail after all attempts if the tool never return a non-zero exit code", func() {
			Expect(that(
				retry("--delay", "25ms", "--", "false"),
				withOutput(GinkgoWriter),
			)).ToNot(BeNil())
		})

		It("should not produce additional output if quiet flag is used", func() {
			var buf bytes.Buffer
			Expect(that(
				retry("--quiet", "--delay", "25ms", "--", "false"),
				withOutput(&buf),
			)).ToNot(BeNil())
			Expect(buf.Len()).To(BeZero())
		})

		It("should cancel the execution if the context is canceled", func() {
			ctx, cancel := context.WithCancel(context.Background())
			start := time.Now()

			go func() {
				time.Sleep(time.Second)
				cancel()
			}()

			Expect(that(
				withContext(ctx),
				retry("sleep", "60"),
			)).ToNot(Succeed())

			Expect(time.Now()).Should(BeTemporally("<", start.Add(60*time.Second)))
		})
	})
})
