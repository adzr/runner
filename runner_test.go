/*
Copyright 2018 Ahmed Zaher

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package runner

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockRunner struct {
	mock.Mock
}

func (m *mockRunner) Start() error {
	return m.Called().Error(0)
}

func (m *mockRunner) Stop() error {
	return m.Called().Error(0)
}

func (m *mockRunner) Check() error {
	return m.Called().Error(0)
}

func (m *mockRunner) OnError(err error) {
	m.Called(err)
}

const hcInterval = time.Second

func TestRunNone(t *testing.T) {
	Run(hcInterval)
	Run(hcInterval, nil, nil)
}

func TestRunSIGINT(t *testing.T) {
	r := &mockRunner{}
	r.On("Start").Once().Return(nil)
	r.On("Stop").Once().Return(nil)
	r.On("Check").Return(nil)

	signalMyselfIn100MS(syscall.SIGINT, t)

	Run(hcInterval, r)
}

func TestRunSIGTERM(t *testing.T) {
	r := &mockRunner{}
	r.On("Start").Once().Return(nil)
	r.On("Stop").Once().Return(nil)
	r.On("Check").Return(nil)

	signalMyselfIn100MS(syscall.SIGTERM, t)

	Run(hcInterval, r)
}

func TestRunSIGQUIT(t *testing.T) {
	r := &mockRunner{}
	r.On("Start").Once().Return(nil)
	r.On("Stop").Once().Return(nil)
	r.On("Check").Return(nil)

	signalMyselfIn100MS(syscall.SIGQUIT, t)

	s := captureRunOutput(hcInterval, r)

	if !strings.HasPrefix(s, "goroutine profile:") {
		t.Errorf("expected to have output that starts with 'goroutine profile:' but found %v", s)
	}
}

func TestRunErrorStart(t *testing.T) {
	var errString = "start error"
	r := &mockRunner{}
	r.On("Start").Once().Return(errors.New(errString))
	r.On("Stop").Once().Return(nil)
	r.On("Check").Return(nil)
	r.On("OnError", mock.Anything).Once().Run(func(args mock.Arguments) {
		assert.EqualError(t, args.Error(0), errString)
	})

	signalMyselfIn100MS(syscall.SIGTERM, t)

	Run(hcInterval, r)
}

func TestRunErrorStop(t *testing.T) {
	var errString = "stop error"
	r := &mockRunner{}
	r.On("Start").Once().Return(nil)
	r.On("Stop").Once().Return(errors.New(errString))
	r.On("Check").Return(nil)
	r.On("OnError", mock.Anything).Once().Run(func(args mock.Arguments) {
		assert.EqualError(t, args.Error(0), errString)
	})

	signalMyselfIn100MS(syscall.SIGTERM, t)

	Run(hcInterval, r)
}

func TestRunErrorCheck(t *testing.T) {
	var errString = "check error"
	r := &mockRunner{}
	r.On("Start").Once().Return(nil)
	r.On("Stop").Once().Return(nil)
	r.On("Check").Return(errors.New(errString))
	r.On("OnError", mock.Anything).Once().Run(func(args mock.Arguments) {
		assert.EqualError(t, args.Error(0), errString)
	})

	signalMyselfIn100MS(syscall.SIGTERM, t)

	Run(hcInterval, r)
}

func captureRunOutput(hcInterval time.Duration, r ...Runnable) string {

	stdOutReader, stdOutWriter, _ := os.Pipe()

	out := os.Stdout

	defer func(out *os.File) {
		os.Stdout = out
	}(out)

	os.Stdout = stdOutWriter

	Run(hcInterval, r...)

	stdOutWriter.Close()

	var bufOut bytes.Buffer
	io.Copy(&bufOut, stdOutReader)
	return bufOut.String()
}

func signalMyselfIn100MS(sig syscall.Signal, t *testing.T) {
	signalMyselfAsync(sig, 100*time.Millisecond, t)
}

func signalMyselfAsync(sig syscall.Signal, delay time.Duration, t *testing.T) {
	go func(s syscall.Signal, d time.Duration, t *testing.T) {
		time.Sleep(delay)
		if err := signalMyself(s); err != nil {
			t.Error(err.Error())
		}
	}(sig, delay, t)
}

func signalMyself(sig syscall.Signal) error {
	var (
		err error
		p   *os.Process
	)

	pid := syscall.Getpid()

	if p, err = os.FindProcess(pid); err != nil {
		return fmt.Errorf("failed to find current process (%v), %v", pid, err.Error())
	}

	return p.Signal(sig)
}
