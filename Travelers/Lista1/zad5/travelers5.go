package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
	"unicode"
)

const (
	NrOfTravelers = 15
	MinSteps      = 10
	MaxSteps      = 100
	MinDelay      = 10 * time.Millisecond
	MaxDelay      = 50 * time.Millisecond
	BoardWidth    = 15
	BoardHeight   = 15
	TimeoutMulti  = 2
)

type Position struct {
	X int
	Y int
}

type Trace struct {
	TimeStamp time.Duration
	ID        int
	Position  Position
	Symbol    rune
}

type Traveler struct {
	ID        int
	Symbol    rune
	Position  Position
	Direction int
}

type Board struct {
	sync.Mutex
	occupied [BoardWidth][BoardHeight]bool
}

var (
	startTime = time.Now()
	board     = &Board{}
	wg        sync.WaitGroup
	tracesCh  = make(chan []Trace, NrOfTravelers)
)

// Movement functions
func moveDown(p *Position)  { p.Y = (p.Y + 1) % BoardHeight }
func moveUp(p *Position)    { p.Y = (p.Y + BoardHeight - 1) % BoardHeight }
func moveRight(p *Position) { p.X = (p.X + 1) % BoardWidth }
func moveLeft(p *Position)  { p.X = (p.X + BoardWidth - 1) % BoardWidth }

func printer() {
	for i := 0; i < NrOfTravelers; i++ {
		traces := <-tracesCh
		printTraces(traces)
	}
}

func printTrace(trace Trace) {
	seconds := float64(trace.TimeStamp) / float64(time.Second)
	fmt.Printf("%.9f %d %d %d %c\n",
		seconds, trace.ID, trace.Position.X, trace.Position.Y, trace.Symbol)
}

func printTraces(traces []Trace) {
	for _, t := range traces {
		printTrace(t)
	}
}

func (b *Board) acquire(x, y int) bool {
	b.Lock()
	defer b.Unlock()
	if !b.occupied[x][y] {
		b.occupied[x][y] = true
		return true
	}
	return false
}

func (b *Board) move(oldX, oldY, newX, newY int) bool {
	b.Lock()
	defer b.Unlock()
	if !b.occupied[newX][newY] {
		b.occupied[oldX][oldY] = false
		b.occupied[newX][newY] = true
		return true
	}
	return false
}

func traveler(id int, seed int64, symbol rune) {
	defer wg.Done()
	r := rand.New(rand.NewSource(seed))
	traces := make([]Trace, 0, MaxSteps+1)

	// Initialize position on diagonal
	pos := Position{X: id % BoardWidth, Y: id % BoardHeight}

	// Acquire initial position
	for !board.acquire(pos.X, pos.Y) {
		time.Sleep(MinDelay)
	}

	// Set initial direction
	var direction int
	if id%2 == 0 { // Vertical
		direction = r.Intn(2) // 0=up, 1=down
	} else { // Horizontal
		direction = 2 + r.Intn(2) // 2=left, 3=right
	}

	// Initial trace
	traces = append(traces, Trace{
		TimeStamp: time.Since(startTime),
		ID:        id,
		Position:  pos,
		Symbol:    symbol,
	})

	nrSteps := MinSteps + r.Intn(MaxSteps-MinSteps+1)
	timeoutDuration := time.Duration(TimeoutMulti * float64(MaxDelay))

	for step := 0; step < nrSteps; step++ {
		delay := MinDelay + time.Duration(float64(MaxDelay-MinDelay)*r.Float64())
		time.Sleep(delay)

		newPos := pos
		switch direction {
		case 0:
			moveUp(&newPos)
		case 1:
			moveDown(&newPos)
		case 2:
			moveLeft(&newPos)
		case 3:
			moveRight(&newPos)
		}

		moveStart := time.Now()
		timeout := false
		for {
			if board.move(pos.X, pos.Y, newPos.X, newPos.Y) {
				pos = newPos
				break
			}

			if time.Since(moveStart) > timeoutDuration {
				symbol = unicode.ToLower(symbol)
				timeout = true
				break
			}
			time.Sleep(MinDelay + time.Duration(r.Float64())*MaxDelay)
		}

		traces = append(traces, Trace{
			TimeStamp: time.Since(startTime),
			ID:        id,
			Position:  pos,
			Symbol:    symbol,
		})

		if timeout {
			break
		}
	}

	tracesCh <- traces
}

func main() {
	fmt.Printf("-1 %d %d %d\n", NrOfTravelers, BoardWidth, BoardHeight)
	go printer()

	symbol := 'A'
	for i := 0; i < NrOfTravelers; i++ {
		wg.Add(1)
		go traveler(i, time.Now().UnixNano()+int64(i), symbol)
		symbol++
	}

	wg.Wait()
	time.Sleep(100 * time.Millisecond)
}
