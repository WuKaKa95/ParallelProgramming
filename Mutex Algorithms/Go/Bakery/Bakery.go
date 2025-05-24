package main

import (
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

// Number of processes
const NrOfProcesses = 15

// Step counts
const (
	MinSteps = 50
	MaxSteps = 100
)

// Random delays
var (
	MinDelay          = 10 * time.Millisecond
	MaxDelay          = 50 * time.Millisecond
	MinCriticalDelay  = 1 * time.Millisecond
	MaxCriticalDelay  = 5 * time.Millisecond
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

// Bakery algorithm shared arrays
var (
	choosing [NrOfProcesses]uint32 // atomic bool: 0 = false, 1 = true
	ticket   [NrOfProcesses]uint32
)

// MaxTicketTracker
type maxTracker struct {
	mu   sync.Mutex
	gmax int
}

func (m *maxTracker) Update(v int) {
	m.mu.Lock()
	if v > m.gmax {
		m.gmax = v
	}
	m.mu.Unlock()
}

func (m *maxTracker) Get() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.gmax
}

var tracker maxTracker

// Global start time
var startTime time.Time

// Printer collects and prints trace sequences
func printer(wg *sync.WaitGroup, reports <-chan TraceSeq) {
	defer wg.Done()

	// collect reports
	for i := 0; i < NrOfProcesses; i++ {
		seq := <-reports
		for _, t := range seq.Traces {
			fmt.Printf("%.6f %d %d %d %c\n",
				t.TimeStamp.Seconds(), t.Id, t.Pos.X, t.Pos.Y, t.Symbol)
		}
	}

	// print parameters line
	fmt.Printf("-1 %d %d %d ",
		NrOfProcesses, NrOfProcesses, len(stateNames))
	for _, name := range stateNames {
		fmt.Printf("%s;", name)
	}
	fmt.Printf("MAX_TICKET=%d;\n", tracker.Get())
}

// One process/task
func processTask(id int, seed int64, symbol rune, start <-chan struct{},
	reportCh chan<- TraceSeq, wg *sync.WaitGroup) {

	defer wg.Done()
	rng := rand.New(rand.NewSource(seed))

	// number of steps
	nSteps := MinSteps + rng.Intn(MaxSteps-MinSteps+1)

	// prepare trace buffer
	seq := TraceSeq{
		Traces: make([]Trace, 0, nSteps+5),
	}

	// record helper
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

	localMax := 0

	for step := 0; step < nSteps; step++ {
		// Local section
		time.Sleep(MinDelay +
			time.Duration(rng.Float64()*float64(MaxDelay-MinDelay)))

		// Entry protocol (Bakery)
		record(EntryProtocol)
		atomic.StoreUint32(&choosing[id], 1)
		// find max ticket
		var maxT uint32
		for k := 0; k < NrOfProcesses; k++ {
			t := atomic.LoadUint32(&ticket[k])
			if t > maxT {
				maxT = t
			}
		}
		myTicket := maxT + 1
		atomic.StoreUint32(&ticket[id], myTicket)
		if int(myTicket) > localMax {
			localMax = int(myTicket)
		}
		atomic.StoreUint32(&choosing[id], 0)

		// wait for others
		for k := 0; k < NrOfProcesses; k++ {
			if k == id {
				continue
			}
			for atomic.LoadUint32(&choosing[k]) == 1 {
			}
			for {
				tk := atomic.LoadUint32(&ticket[k])
				if tk == 0 {
					break
				}
				if tk < myTicket || (tk == myTicket && k < id) {
					// keep waiting
				} else {
					break
				}
			}
		}

		// Critical section
		record(CriticalSection)
		time.Sleep(MinCriticalDelay +
			time.Duration(rng.Float64()*float64(MaxCriticalDelay-MinCriticalDelay)))

		// Exit protocol
		record(ExitProtocol)
		time.Sleep(ExitProtocolDelay)
		atomic.StoreUint32(&ticket[id], 0)
		// Back to local
		record(LocalSection)
	}

	tracker.Update(localMax)
	reportCh <- seq
}

func main() {
	startTime = time.Now()

	startCh := make(chan struct{})
	reportCh := make(chan TraceSeq, NrOfProcesses)

	var wg sync.WaitGroup

	// start printer
	wg.Add(1)
	go printer(&wg, reportCh)

	// start processes
	for i := 0; i < NrOfProcesses; i++ {
		seed := time.Now().UnixNano() + int64(i)*997
		symbol := rune('A' + (i % 26))
		wg.Add(1)
		go processTask(i, seed, symbol, startCh, reportCh, &wg)
	}

	// kick off
	close(startCh)

	// wait completion
	wg.Wait()
}
