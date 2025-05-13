package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// Parameters
const (
	NrOfTravelers = 15
	MinSteps      = 10
	MaxSteps      = 100
	MinDelay      = 10 * time.Millisecond
	MaxDelay      = 50 * time.Millisecond
	BoardWidth    = 15
	BoardHeight   = 15
	EnterTimeout  = 3 * MaxDelay
)

type PositionType struct {
	X int
	Y int
}

type TraceType struct {
	TimeStamp time.Duration
	ID        int
	Position  PositionType
	Symbol    rune
}

type TravelerType struct {
	ID       int
	Symbol   rune
	Position PositionType
}

var (
	startTime  = time.Now()
	wg         sync.WaitGroup
	tracesCh   = make(chan []TraceType, NrOfTravelers)
	cellTokens [BoardWidth][BoardHeight]chan struct{}
)

// Each cell is available if its channel holds a token.
func initCells() {
	for i := 0; i < BoardWidth; i++ {
		for j := 0; j < BoardHeight; j++ {
			cellTokens[i][j] = make(chan struct{}, 1)
			// Place one token in each cell, making the cell "free".
			cellTokens[i][j] <- struct{}{}
		}
	}
}

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

func printer() {
	defer wg.Done()
	for i := 0; i < NrOfTravelers; i++ {
		traces := <-tracesCh
		printTraces(traces)
	}
}

func printTrace(trace TraceType) {
	seconds := float64(trace.TimeStamp) / float64(time.Second)
	fmt.Printf("%.9f %d %d %d %c\n", seconds, trace.ID, trace.Position.X, trace.Position.Y, trace.Symbol)
}

func printTraces(traces []TraceType) {
	for _, trace := range traces {
		printTrace(trace)
	}
}

// tryEnterCell attempts to acquire the token from cellTokens[x][y] with a timeout.
func tryEnterCell(pos PositionType) bool {
	select {
	case <-cellTokens[pos.X][pos.Y]:
		// Successfully acquired the token: cell is now locked.
		return true
	case <-time.After(EnterTimeout):
		// Timed out trying to enter.
		return false
	}
}

// leaveCell releases the token back into the cell's channel.
func leaveCell(pos PositionType) {
	cellTokens[pos.X][pos.Y] <- struct{}{}
}

func traveler(id int, seed int64, symbol rune) {
	defer wg.Done()
	r := rand.New(rand.NewSource(seed))

	// Decide fixed movement direction
	var moveFunc func(*PositionType)
	if id%2 == 0 {
		// Even ID → vertical
		if r.Intn(2) == 0 {
			moveFunc = moveUp
		} else {
			moveFunc = moveDown
		}
	} else {
		// Odd ID → horizontal
		if r.Intn(2) == 0 {
			moveFunc = moveLeft
		} else {
			moveFunc = moveRight
		}
	}

	// Set diagonal starting position
	traveler := TravelerType{
		ID:     id,
		Symbol: symbol,
		Position: PositionType{
			X: id,
			Y: id,
		},
	}

	traces := make([]TraceType, 0, MaxSteps+1)

	tryEnterCell(traveler.Position) // Always true at startup

	traces = append(traces, TraceType{
		TimeStamp: time.Since(startTime),
		ID:        traveler.ID,
		Position:  traveler.Position,
		Symbol:    traveler.Symbol,
	})

	nrOfSteps := MinSteps + r.Intn(MaxSteps-MinSteps+1)
	for step := 0; step < nrOfSteps; step++ {
		delay := MinDelay + time.Duration(r.Float64()*float64(MaxDelay-MinDelay))
		time.Sleep(delay)

		newPos := traveler.Position
		moveFunc(&newPos)

		if tryEnterCell(newPos) {
			leaveCell(traveler.Position)
			traveler.Position = newPos
			traces = append(traces, TraceType{
				TimeStamp: time.Since(startTime),
				ID:        traveler.ID,
				Position:  traveler.Position,
				Symbol:    traveler.Symbol,
			})
		} else {
			traveler.Symbol += ('a' - 'A')
			traces = append(traces, TraceType{
				TimeStamp: time.Since(startTime),
				ID:        traveler.ID,
				Position:  traveler.Position,
				Symbol:    traveler.Symbol,
			})
			break
		}
	}

	tracesCh <- traces
}

func main() {
	fmt.Printf("-1 %d %d %d\n", NrOfTravelers, BoardWidth, BoardHeight)

	// Initialize cell tokens for each board cell.
	initCells()

	wg.Add(1)
	go printer()

	// Launch traveler goroutines.
	symbol := 'A'
	for i := 0; i < NrOfTravelers; i++ {
		wg.Add(1)
		go traveler(i, time.Now().UnixNano()+int64(i), symbol)
		symbol++
	}

	wg.Wait()
}

