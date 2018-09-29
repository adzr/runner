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
	"os"
	"os/signal"
	"runtime/pprof"
	"sync"
	"syscall"
	"time"
)

// Runnable represents anything that can be run in a separate go-routine,
// this is an ideal interface for a service type components that can run
// concurrently in the same application.
type Runnable interface {

	// Start is being called in a separate go-routine as soon as this object
	// is passed to the Run() function to switch the implementation
	// to "running" mode.
	// It must return an error if it fails to start for any reason, nil otherwise.
	Start() error

	// Stop is being called in a separate go-routine once one of the termination
	// signals (syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT) has been sent
	// to the application.
	// It must return an error if it fails to stop for any reason, nil otherwise.
	Stop() error

	// Check is being called in a separate go-routine after all the "Runnable"s
	// have started, this is called periodically every certain time duration
	// passed as a parameter to the Run() function, the periodic call stops when
	// Stop() method is called.
	// It must return an error if the check fails for any reason, nil otherwise.
	Check() error

	// OnError handles any error returned by the other methods.
	OnError(err error)
}

func assertRunnables(runnables []Runnable) []Runnable {
	if r := make([]Runnable, 0); runnables != nil {
		for _, runnable := range runnables {
			if runnable != nil {
				r = append(r, runnable)
			}
		}
		return r
	}
	return make([]Runnable, 0)
}

func closeChannles(ch ...chan bool) {
	for _, c := range ch {
		close(c)
	}
}

func handleStop(stop <-chan bool, stopped chan<- bool, hcStop chan<- bool, hcStopped <-chan bool, rs []Runnable) {
	var killing sync.WaitGroup

	// at the end of this go-routine, notify that stopping is finished.
	defer func(d chan<- bool) { d <- true }(stopped)

	// now wait for the kill signal to initiate the stopping procedures.
	<-stop

	// stop health checks.
	hcStop <- true

	// wait for health checks to stop.
	<-hcStopped

	// ok, we received the signal, now stop all the runnables asynchronously and wait for them to finish.
	for _, runnable := range rs {
		killing.Add(1)
		go func(r Runnable, wg *sync.WaitGroup) {
			defer wg.Done()
			if err := r.Stop(); err != nil {
				r.OnError(err)
			}
		}(runnable, &killing)
	}

	// still waiting for all the runnables to stop.
	killing.Wait()
}

func handleSignals(s chan os.Signal, stop chan<- bool) {
	// register the termination signals.
	signal.Notify(s, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	// at the end of this go-routine stop channel must receive a signal so the stopping go-routine
	// can continue execution.
	defer func(s chan<- bool) { s <- true }(stop)

	// now wait for the system signal.
	switch <-s {
	case syscall.SIGQUIT:
		pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)
	}
}

func handleChecks(interval time.Duration, start <-chan bool, stop <-chan bool, stopped chan<- bool, rs []Runnable) {
	defer func(s chan<- bool) { s <- true }(stopped)
	<-start
	var done = false
	for !done {
		select {
		case <-stop:
			done = true
		}

		for _, r := range rs {
			go func(r Runnable) {
				if err := r.Check(); err != nil {
					r.OnError(err)
				}
			}(r)
		}

		time.Sleep(interval)
	}
}

// Run receives Runnable implementations and a health check time interval
// then and calls the Start() method of each Runnable in a separate go-routine
// and then the Check() method of each in a separate go-routine periodically
// based on the time interval received, and then the function blocks waiting
// for one of the system signals (syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT).
// If a signal is received then it doesn't call Runnable.Check() anymore and
// calls the Stop() method of each Runnable in a separate go-routine and wait
// them all to finish and finally returns.
func Run(hcInterval time.Duration, runnables ...Runnable) {

	// omit nil runnable interfaces.
	runnables = assertRunnables(runnables)

	// if no interfaces are found then return.
	if len(runnables) == 0 {
		return
	}

	// define some channels to control the flow.
	var (
		// this is to capture the system signals.
		sigs = make(chan os.Signal, 1)

		// this is to signal the health check to start running.
		running = make(chan bool, 1)

		// this is to signal the stopping go-routine to initiate the stopping procedures.
		stop = make(chan bool, 1)

		// this is to signal this function that the stopping go-routine has finished execution.
		stopped = make(chan bool, 1)

		// this is to signal the health check go-routine to stop.
		hcStop = make(chan bool, 1)

		// this is to notify that health check go-routine is stopped.
		hcStopped = make(chan bool, 1)
	)

	// make sure to close all channels before exiting the function.
	defer closeChannles(running, stop, stopped, hcStop, hcStopped)

	// at the end of this function block until the stopping go-routine has finished execution.
	defer func(s <-chan bool) { <-s }(stopped)

	// call the stopping go-routine and make it wait for the stop channel signal before going on.
	go handleStop(stop, stopped, hcStop, hcStopped, runnables)

	// call the system signals subscriber go-routine and make it wait for system termination signal.
	go handleSignals(sigs, stop)

	// ok, now call the health check go-routine to report any failures.
	go handleChecks(hcInterval, running, hcStop, hcStopped, runnables)

	// almost done here, now start all the runnables asynchronously and wait for the done signal.
	for _, runnable := range runnables {
		go func(r Runnable) {
			if err := r.Start(); err != nil {
				r.OnError(err)
			}
		}(runnable)
	}

	// finally, start running health checks.
	running <- true
}
