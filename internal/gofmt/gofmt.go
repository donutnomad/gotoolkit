// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gofmt

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/scanner"
	"go/token"
	"io"
	"io/fs"

	"golang.org/x/sync/semaphore"
)

// 格式化配置
var (
	allErrors   = false // 报告所有错误
	rewriteRule = ""    // 重写规则
	simplifyAST = false // 简化代码
)

// Keep these in sync with go/format/format.go.
const (
	tabWidth    = 8
	printerMode = printer.UseSpaces | printer.TabIndent | printerNormalizeNumbers

	// printerNormalizeNumbers means to canonicalize number literal prefixes
	// and exponents while printing. See https://golang.org/doc/go1.13#gofmt.
	//
	// This value is defined in go/printer specifically for go/format and cmd/gofmt.
	printerNormalizeNumbers = 1 << 30
)

var (
	rewrite    func(*token.FileSet, *ast.File) *ast.File
	parserMode parser.Mode
)

func initParserMode() {
	parserMode = parser.ParseComments
	if allErrors {
		parserMode |= parser.AllErrors
	}
	// It's only -r that makes use of go/ast's object resolution,
	// so avoid the unnecessary work if the flag isn't used.
	if rewriteRule == "" {
		parserMode |= parser.SkipObjectResolution
	}
}

// A sequencer performs concurrent tasks that may write output, but emits that
// output in a deterministic order.
type sequencer struct {
	maxWeight int64
	sem       *semaphore.Weighted   // weighted by input bytes (an approximate proxy for memory overhead)
	prev      <-chan *reporterState // 1-buffered
}

// newSequencer returns a sequencer that allows concurrent tasks up to maxWeight
// and writes tasks' output to out and err.
func newSequencer(maxWeight int64, out, err io.Writer) *sequencer {
	sem := semaphore.NewWeighted(maxWeight)
	prev := make(chan *reporterState, 1)
	prev <- &reporterState{out: out, err: err}
	return &sequencer{
		maxWeight: maxWeight,
		sem:       sem,
		prev:      prev,
	}
}

// Add blocks until the sequencer has enough weight to spare, then adds f as a
// task to be executed concurrently.
//
// If the weight is either negative or larger than the sequencer's maximum
// weight, Add blocks until all other tasks have completed, then the task
// executes exclusively (blocking all other calls to Add until it completes).
//
// f may run concurrently in a goroutine, but its output to the passed-in
// reporter will be sequential relative to the other tasks in the sequencer.
//
// If f invokes a method on the reporter, execution of that method may block
// until the previous task has finished. (To maximize concurrency, f should
// avoid invoking the reporter until it has finished any parallelizable work.)
//
// If f returns a non-nil error, that error will be reported after f's output
// (if any) and will cause a nonzero final exit code.
func (s *sequencer) Add(weight int64, f func(*reporter) error) {
	if weight < 0 || weight > s.maxWeight {
		weight = s.maxWeight
	}
	if err := s.sem.Acquire(context.TODO(), weight); err != nil {
		// Change the task from "execute f" to "report err".
		weight = 0
		f = func(*reporter) error { return err }
	}

	r := &reporter{prev: s.prev}
	next := make(chan *reporterState, 1)
	s.prev = next

	// Start f in parallel: it can run until it invokes a method on r, at which
	// point it will block until the previous task releases the output state.
	go func() {
		if err := f(r); err != nil {
			r.Report(err)
		}
		next <- r.getState() // Release the next task.
		s.sem.Release(weight)
	}()
}

// AddReport prints an error to s after the output of any previously-added
// tasks, causing the final exit code to be nonzero.
func (s *sequencer) AddReport(err error) {
	s.Add(0, func(*reporter) error { return err })
}

// GetExitCode waits for all previously-added tasks to complete, then returns an
// exit code for the sequence suitable for passing to os.Exit.
func (s *sequencer) GetExitCode() int {
	c := make(chan int, 1)
	s.Add(0, func(r *reporter) error {
		c <- r.ExitCode()
		return nil
	})
	return <-c
}

// A reporter reports output, warnings, and errors.
type reporter struct {
	prev  <-chan *reporterState
	state *reporterState
}

// reporterState carries the state of a reporter instance.
//
// Only one reporter at a time may have access to a reporterState.
type reporterState struct {
	out, err io.Writer
	exitCode int
}

// getState blocks until any prior reporters are finished with the reporter
// state, then returns the state for manipulation.
func (r *reporter) getState() *reporterState {
	if r.state == nil {
		r.state = <-r.prev
	}
	return r.state
}

// Warnf emits a warning message to the reporter's error stream,
// without changing its exit code.
func (r *reporter) Warnf(format string, args ...any) {
	fmt.Fprintf(r.getState().err, format, args...)
}

// Write emits a slice to the reporter's output stream.
//
// Any error is returned to the caller, and does not otherwise affect the
// reporter's exit code.
func (r *reporter) Write(p []byte) (int, error) {
	return r.getState().out.Write(p)
}

// Report emits a non-nil error to the reporter's error stream,
// changing its exit code to a nonzero value.
func (r *reporter) Report(err error) {
	if err == nil {
		panic("Report with nil error")
	}
	st := r.getState()
	scanner.PrintError(st.err, err)
	st.exitCode = 2
}

func (r *reporter) ExitCode() int {
	return r.getState().exitCode
}

// If info == nil, we are formatting stdin instead of a file.
// If in == nil, the source is the contents of the file with the given filename.
func processFile(src []byte, info fs.FileInfo) ([]byte, error) {
	fileSet := token.NewFileSet()
	// If we are formatting stdin, we accept a program fragment in lieu of a
	// complete source file.
	fragmentOk := info == nil
	file, sourceAdj, indentAdj, err := parse(fileSet, "std-file", src, fragmentOk)
	if err != nil {
		return nil, err
	}

	if rewrite != nil {
		if sourceAdj == nil {
			file = rewrite(fileSet, file)
		} else {
		}
	}

	ast.SortImports(fileSet, file)

	if simplifyAST {
		simplify(file)
	}

	res, err := format(fileSet, file, sourceAdj, indentAdj, src, printer.Config{Mode: printerMode, Tabwidth: tabWidth})
	if err != nil {
		return nil, err
	}

	return res, err
}

func FormatBytes(src []byte) ([]byte, error) {
	initParserMode()
	initRewrite()
	return processFile(src, nil)
}
