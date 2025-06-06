package main

import (
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

// Number of processes
const NrOfProcesses = 2

// Step counts
const (
	MinSteps = 50
	MaxSteps = 100
)

// Random delays
var (
	MinDelay          = 10 * time.Millisecond
	MaxDelay          = 50 * time.Millisecond
	MinCriticalDelay  = 10 * time.Millisecond
	MaxCriticalDelay  = 50 * time.Millisecond
	ExitProtocolDelay = 1 * time.Microsecond
)

// Process states
type ProcessState int

const (
	LocalSection ProcessState = iota
	EntryProtocol
	CriticalSection
	ExitProtocol
)

// For printing the board we assign each state a row:
var stateNames = [...]string{
	"LocalSection",
	"EntryProtocol",
	"CriticalSection",
	"ExitProtocol",
}

// Position on board
type Position struct {
	X, Y int
}

// Trace of one event
type Trace struct {
	TimeStamp time.Duration
	Id        int
	Pos       Position
	Symbol    rune
}

// A full trace sequence
type TraceSeq struct {
	Traces []Trace
}

var (
	flags [NrOfProcesses]uint32 // 0=false, 1=true
	turn  uint32
)

// Global start time
var startTime time.Time

// Printer collects and prints trace sequences
func printer(wg *sync.WaitGroup, reports <-chan TraceSeq) {
	defer wg.Done()

	// collect NrOfProcesses reports
	for i := 0; i < NrOfProcesses; i++ {
		seq := <-reports
		for _, t := range seq.Traces {
			fmt.Printf("%.6f %d %d %d %c\n",
				t.TimeStamp.Seconds(), t.Id, t.Pos.X, t.Pos.Y, t.Symbol)
		}
	}

	// print the parameters line
	fmt.Printf("-1 %d %d %d ",
		NrOfProcesses, NrOfProcesses, len(stateNames))
	for _, name := range stateNames {
		fmt.Printf("%s;", name)
	}
	fmt.Println("EXTRA_LABEL;")
}

// One process/task
func processTask(id int, seed int64, symbol rune, start <-chan struct{},
	reportCh chan<- TraceSeq, wg *sync.WaitGroup) {

	defer wg.Done()
	rng := rand.New(rand.NewSource(seed))

	// determine number of steps
	nSteps := MinSteps + rng.Intn(MaxSteps-MinSteps+1)

	// prepare trace buffer
	seq := TraceSeq{
		Traces: make([]Trace, 0, nSteps/4+5),
	}

	// helper to record a trace
	record := func(st ProcessState) {
		now := time.Since(startTime)
		seq.Traces = append(seq.Traces, Trace{
			TimeStamp: now,
			Id:        id,
			Pos: Position{
				X: id,
				Y: int(st),
			},
			Symbol: symbol,
		})
	}

	// initial position
	record(LocalSection)

	// wait for global start
	<-start

	for step := 0; step < nSteps/4; step++ {
		// Local section
		time.Sleep(MinDelay +
			time.Duration(rng.Float64()*float64(MaxDelay-MinDelay)))

		// Entry protocol (Dekker)
		record(EntryProtocol)
		other := 1 - id
		atomic.StoreUint32(&flags[id], 1)
		for atomic.LoadUint32(&flags[other]) == 1 {
			if atomic.LoadUint32(&turn) != uint32(id) {
				atomic.StoreUint32(&flags[id], 0)
				for atomic.LoadUint32(&turn) != uint32(id) {
				}
				atomic.StoreUint32(&flags[id], 1)
			}
		}

		// Critical section
		record(CriticalSection)
		time.Sleep(MinCriticalDelay +
			time.Duration(rng.Float64()*float64(MaxCriticalDelay-MinCriticalDelay)))

		// Exit protocol (Dekker)
		record(ExitProtocol)
		time.Sleep(ExitProtocolDelay)
		atomic.StoreUint32(&flags[id], 0)
		atomic.StoreUint32(&turn, uint32(other))

		// Back to local
		record(LocalSection)
	}

	// send report
	reportCh <- seq
}

func main() {
	startTime = time.Now()

	// Channel to signal start
	startCh := make(chan struct{})

	// Channel for reports
	reportCh := make(chan TraceSeq, NrOfProcesses)

	var wg sync.WaitGroup

	// Launch printer
	wg.Add(1)
	go printer(&wg, reportCh)

	// Launch processes
	for i := 0; i < NrOfProcesses; i++ {
		seed := time.Now().UnixNano() + int64(i)*100
		symbol := rune('A' + i)
		wg.Add(1)
		go processTask(i, seed, symbol, startCh, reportCh, &wg)
	}

	// Kick off all processes
	close(startCh)

	// Wait for everything to finish
	wg.Wait()
}
