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

type Cell struct {
	sync.Mutex
	occupied bool
}

type Board struct {
	cells [BoardWidth][BoardHeight]*Cell
}

var (
	startTime = time.Now()
	board     = NewBoard()
	wg        sync.WaitGroup
	tracesCh  = make(chan []Trace, NrOfTravelers)
)

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

func NewBoard() *Board {
	b := &Board{}
	for x := 0; x < BoardWidth; x++ {
		for y := 0; y < BoardHeight; y++ {
			b.cells[x][y] = &Cell{}
		}
	}
	return b
}

func (b *Board) acquire(x, y int) bool {
	cell := b.cells[x][y]
	cell.Lock()
	defer cell.Unlock()

	if !cell.occupied {
		cell.occupied = true
		return true
	}
	return false
}

func (b *Board) move(oldX, oldY, newX, newY int) bool {
	firstX, firstY := oldX, oldY
	secondX, secondY := newX, newY
	if oldX > newX || (oldX == newX && oldY > newY) {
		firstX, firstY, secondX, secondY = newX, newY, oldX, oldY
	}

	b.cells[firstX][firstY].Lock()
	b.cells[secondX][secondY].Lock()
	defer b.cells[secondX][secondY].Unlock()
	defer b.cells[firstX][firstY].Unlock()

	if !b.cells[newX][newY].occupied {
		b.cells[oldX][oldY].occupied = false
		b.cells[newX][newY].occupied = true
		return true
	}
	return false
}

func traveler(id int, seed int64, symbol rune) {
	defer wg.Done()
	r := rand.New(rand.NewSource(seed))
	traces := make([]Trace, 0, MaxSteps+1)

	pos := Position{X: id % BoardWidth, Y: id % BoardHeight}

	for !board.acquire(pos.X, pos.Y) {
		time.Sleep(MinDelay)
	}

	var direction int
	if id%2 == 0 {
		direction = r.Intn(2) // 0 = up, 1 = down
	} else {
		direction = 2 + r.Intn(2) // 2 = left, 3 = right
	}

	traces = append(traces, Trace{
		TimeStamp: time.Since(startTime),
		ID:        id,
		Position:  pos,
		Symbol:    symbol,
	})

	nrSteps := MinSteps + r.Intn(MaxSteps-MinSteps+1)
	timeoutDuration := time.Duration(TimeoutMulti * float64(MaxDelay))

	for step := 0; step < nrSteps; step++ {
		time.Sleep(MinDelay + time.Duration(float64(MaxDelay-MinDelay)*r.Float64()))

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

