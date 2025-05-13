package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// Constants
const (
	NrOfTravelers = 15
	MinSteps      = 10
	MaxSteps      = 100
	MinDelay      = 10 * time.Millisecond
	MaxDelay      = 50 * time.Millisecond
	BoardWidth    = 15
	BoardHeight   = 15
)

// PositionType represents a position on the board
type PositionType struct {
	X int
	Y int
}

// TraceType records traveler movements
type TraceType struct {
	TimeStamp time.Duration
	ID        int
	Position  PositionType
	Symbol    rune
}

// TravelerType represents a traveler
type TravelerType struct {
	ID       int
	Symbol   rune
	Position PositionType
}

// Global variables
var (
	startTime = time.Now()
	wg        sync.WaitGroup
	tracesCh  = make(chan []TraceType, NrOfTravelers)
)

// Movement functions
func moveDown(p *PositionType) {
	p.Y = (p.Y + 1) % BoardHeight
}

func moveUp(p *PositionType) {
	p.Y = (p.Y + BoardHeight - 1) % BoardHeight
}

func moveRight(p *PositionType) {
	p.X = (p.X + 1) % BoardWidth
}

func moveLeft(p *PositionType) {
	p.X = (p.X + BoardWidth - 1) % BoardWidth
}

// Printer goroutine
func printer() {
	for i := 0; i < NrOfTravelers; i++ {
		traces := <-tracesCh
		printTraces(traces)
	}
}

func printTrace(trace TraceType) {
	// Format duration with exactly 9 decimal places
	seconds := float64(trace.TimeStamp) / float64(time.Second)
	fmt.Printf("%.9f %d %d %d %c\n",
		seconds,
		trace.ID,
		trace.Position.X,
		trace.Position.Y,
		trace.Symbol)
}

func printTraces(traces []TraceType) {
	for _, trace := range traces {
		printTrace(trace)
	}
}

// Traveler goroutine
func traveler(id int, seed int64, symbol rune) {
	defer wg.Done()

	// Initialize random generator with unique seed
	r := rand.New(rand.NewSource(seed))

	// Initialize traveler
	traveler := TravelerType{
		ID:     id,
		Symbol: symbol,
		Position: PositionType{
			X: r.Intn(BoardWidth),
			Y: r.Intn(BoardHeight),
		},
	}

	// Create trace slice with initial capacity
	nrOfSteps := MinSteps + r.Intn(MaxSteps-MinSteps+1)
	traces := make([]TraceType, 0, nrOfSteps+1)

	// Store initial position
	traces = append(traces, TraceType{
		TimeStamp: time.Since(startTime),
		ID:        traveler.ID,
		Position:  traveler.Position,
		Symbol:    traveler.Symbol,
	})

	// Wait for all travelers to initialize
	<-time.After(1 * time.Millisecond) // Small delay to simulate entry Start

	// Movement loop
	for step := 0; step < nrOfSteps; step++ {
		delay := MinDelay + time.Duration(float64(MaxDelay-MinDelay)*r.Float64())
		time.Sleep(delay)

		// Random movement
		n := r.Intn(4)
		switch n {
		case 0:
			moveUp(&traveler.Position)
		case 1:
			moveDown(&traveler.Position)
		case 2:
			moveLeft(&traveler.Position)
		case 3:
			moveRight(&traveler.Position)
		default:
			fmt.Printf(" ?????????????? %d\n", n)
		}

		// Store trace
		traces = append(traces, TraceType{
			TimeStamp: time.Since(startTime),
			ID:        traveler.ID,
			Position:  traveler.Position,
			Symbol:    traveler.Symbol,
		})
	}

	// Send traces to printer
	tracesCh <- traces
}

func main() {
	// Print parameters for display script
	fmt.Printf("-1 %d %d %d\n", NrOfTravelers, BoardWidth, BoardHeight)

	// Start printer goroutine
	go printer()

	// Create and start travelers
	symbol := 'A'
	for i := 0; i < NrOfTravelers; i++ {
		wg.Add(1)
		// Use time.Now().UnixNano() with offset for different seeds
		go traveler(i, time.Now().UnixNano()+int64(i), symbol)
		symbol++
	}

	// Wait for all travelers to finish
	wg.Wait()

	// Give printer time to finish processing
	time.Sleep(100 * time.Millisecond)
}
