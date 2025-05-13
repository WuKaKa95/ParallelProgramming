package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// Parameters
const (
	NrOfTravelers    = 15
	NrOfTraps        = 10
	NrOfWilds        = 10
	MinSteps         = 10
	MaxSteps         = 100
	MinDelay         = 10 * time.Millisecond
	MaxDelay         = 50 * time.Millisecond
	BoardWidth       = 15
	BoardHeight      = 15
	EnterTimeout     = 3 * MaxDelay
	WildEnterTimeout = 1 * MaxDelay
	TrapSleep        = 300 * time.Millisecond
	WildLifespan     = 800 * time.Millisecond
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

type WildType struct {
	ID       int
	Symbol   rune
	Position PositionType
}

var (
	startTime    = time.Now()
	wg           sync.WaitGroup
	wgPrint      sync.WaitGroup
	tracesCh     = make(chan []TraceType, NrOfTravelers+NrOfTraps+NrOfWilds)
	cellTokens   [BoardWidth][BoardHeight]chan rune
	wildRequests [BoardWidth][BoardHeight]chan struct{}
)

// Each cell is available if its channel holds a token.
func initCells() {
	for i := 0; i < BoardWidth; i++ {
		for j := 0; j < BoardHeight; j++ {
			cellTokens[i][j] = make(chan rune, 1)
			// Place one token in each cell, making the cell "free".
			cellTokens[i][j] <- 'F'
		}
	}
}

func trapCells() {
	for i := 0; i < NrOfTraps; i++ {
		X := rand.Intn(BoardWidth)
		Y := rand.Intn(BoardHeight)
		state := <-cellTokens[X][Y]
		switch state {
		case 'F':
			//We trap the cell
			cellTokens[X][Y] <- 'T'
			trace := make([]TraceType, 0, MaxSteps+1)
			trace = append(trace, TraceType{
				TimeStamp: 0.0,
				ID:        NrOfTravelers + NrOfWilds + i,
				Symbol:    '#',
				Position:  PositionType{X: X, Y: Y},
			})
			tracesCh <- trace
			wg.Done()
			break
		case 'T':
			//Cell already trapped, we try again
			cellTokens[X][Y] <- 'T'
			i--
			continue
		}
	}
}

func initWildRequests() {
	for x := 0; x < BoardWidth; x++ {
		for y := 0; y < BoardHeight; y++ {
			// buffered so that one nudge won't block a traveler
			wildRequests[x][y] = make(chan struct{}, 1)
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

// Printer goroutine: unchanged
func printer() {
	defer wgPrint.Done()
	for traces := range tracesCh {
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

// tryEnterCellLegal attempts to acquire the token from cellTokens[x][y] with a timeout.
func tryEnterCellLegal(pos PositionType) rune {
	timeout := time.After(EnterTimeout)
	for {
		select {
		case token := <-cellTokens[pos.X][pos.Y]:
			if token == 'W' {
				// Nudge the wild to move
				select {
				case wildRequests[pos.X][pos.Y] <- struct{}{}:
				default:
				}
				// Put the W back
				select {
				case cellTokens[pos.X][pos.Y] <- 'W':
				default:
				}
				continue
			}
			return token
		case <-timeout:
			return '0'
		}
	}
}

func tryEnterCellWild(pos PositionType) rune {
	select {
	case token := <-cellTokens[pos.X][pos.Y]:
		if token == 'F' {
			// Upon entering a free cell we mark it as occupied by Wild
			cellTokens[pos.X][pos.Y] <- 'W'
			return 'F'
		}
		if token == 'W' {
			// Consider the cells occupied by other wilds to be inaccessible
			cellTokens[pos.X][pos.Y] <- 'W'
			return '0'
		}
		return token
	case <-time.After(WildEnterTimeout):
		return '0'
	}
}

// leaveCell releases the token back into the cell's channel.
func leaveCell(pos PositionType, cellToken rune) {
	//non-blockingly overwrite if there is a token present
	select {
	case <-cellTokens[pos.X][pos.Y]:
	default:
	}
	cellTokens[pos.X][pos.Y] <- cellToken
}

func traveler(id int, seed int64, symbol rune) {
	defer wg.Done()
	r := rand.New(rand.NewSource(seed))

	traveler := TravelerType{
		ID:     id,
		Symbol: symbol,
		Position: PositionType{
			X: r.Intn(BoardWidth),
			Y: r.Intn(BoardHeight),
		},
	}

	traces := make([]TraceType, 0, MaxSteps+1)

	for {
		receivedToken := tryEnterCellLegal(traveler.Position)
		if receivedToken == 'F' {
			break
		}
		if receivedToken == 'T' {
			leaveCell(traveler.Position, 'T')
		}
		traveler.Position = PositionType{
			X: r.Intn(BoardWidth),
			Y: r.Intn(BoardHeight),
		}
		time.Sleep(1 * time.Millisecond)
	}

	traces = append(traces, TraceType{
		TimeStamp: time.Since(startTime),
		ID:        traveler.ID,
		Position:  traveler.Position,
		Symbol:    traveler.Symbol,
	})

	nrOfSteps := MinSteps + r.Intn(MaxSteps-MinSteps+1)
Alive:
	for step := 0; step < nrOfSteps; step++ {
		delay := MinDelay + time.Duration(r.Float64()*float64(MaxDelay-MinDelay))
		time.Sleep(delay)

		newPos := traveler.Position
		n := r.Intn(4)
		switch n {
		case 0:
			moveUp(&newPos)
		case 1:
			moveDown(&newPos)
		case 2:
			moveLeft(&newPos)
		case 3:
			moveRight(&newPos)
		}
		switch tryEnterCellLegal(newPos) {
		case '0':
			traveler.Symbol += ('a' - 'A')
			traces = append(traces, TraceType{
				TimeStamp: time.Since(startTime),
				ID:        traveler.ID,
				Position:  traveler.Position,
				Symbol:    traveler.Symbol,
			})
			break Alive
		case 'F':
			leaveCell(traveler.Position, 'F')
			traveler.Position = newPos
			traces = append(traces, TraceType{
				TimeStamp: time.Since(startTime),
				ID:        traveler.ID,
				Position:  traveler.Position,
				Symbol:    traveler.Symbol,
			})
			break
		case 'T':
			leaveCell(traveler.Position, 'F')
			traveler.Position = newPos
			traveler.Symbol += ('a' - 'A')
			traces = append(traces, TraceType{
				TimeStamp: time.Since(startTime),
				ID:        traveler.ID,
				Position:  traveler.Position,
				Symbol:    traveler.Symbol,
			})
			time.Sleep(TrapSleep)
			traveler.Symbol = '#'
			traces = append(traces, TraceType{
				TimeStamp: time.Since(startTime),
				ID:        traveler.ID,
				Position:  traveler.Position,
				Symbol:    traveler.Symbol,
			})
			leaveCell(traveler.Position, 'T')
			break Alive
		}
	}

	tracesCh <- traces
}

func wild(id int, seed int64, symbol rune) {
	defer wg.Done()
	r := rand.New(rand.NewSource(seed))
	var (
		timeAppear = time.Duration(rand.Float64()*float64((MaxSteps+MinSteps)/2)*float64((MaxDelay+MinDelay)/2)) - WildLifespan
	)
	traces := make([]TraceType, 0, MaxSteps+1)

	wild := WildType{
		ID:     id,
		Symbol: symbol,
		Position: PositionType{
			X: r.Intn(BoardWidth),
			Y: r.Intn(BoardHeight),
		},
	}
	time.Sleep(timeAppear)

	for {
		receivedToken := tryEnterCellWild(wild.Position)
		if receivedToken == 'F' {
			break
		}
		//don't want to start wilds in traps
		if receivedToken == 'T' {
			leaveCell(wild.Position, 'T')
		}
		wild.Position = PositionType{
			X: r.Intn(BoardWidth),
			Y: r.Intn(BoardHeight),
		}
		time.Sleep(1 * time.Millisecond)
	}

	traces = append(traces, TraceType{
		TimeStamp: time.Since(startTime),
		ID:        wild.ID,
		Position:  wild.Position,
		Symbol:    wild.Symbol,
	})
	nudgeCh := wildRequests[wild.Position.X][wild.Position.Y]

	timer := time.NewTimer(WildLifespan)
	defer timer.Stop()
Alive:
	for {
		select {
		case <-timer.C:
			break Alive
		case <-nudgeCh:
			// Try moves in up, right, down, left order
			for _, move := range []func(*PositionType){
				moveUp,
				moveRight,
				moveDown,
				moveLeft,
			} {
				newPos := wild.Position
				move(&newPos)

				switch token := tryEnterCellWild(newPos); token {
				case '0':
					//try next direction
					continue
				case 'F':
					traces = append(traces, TraceType{
						TimeStamp: time.Since(startTime),
						ID:        wild.ID,
						Position:  newPos,
						Symbol:    wild.Symbol,
					})
					leaveCell(wild.Position, 'F')
					wild.Position = newPos
					nudgeCh = wildRequests[wild.Position.X][wild.Position.Y]
					break
				case 'T':
					wild.Symbol = '+'
					traces = append(traces, TraceType{
						TimeStamp: time.Since(startTime),
						ID:        wild.ID,
						Position:  newPos,
						Symbol:    wild.Symbol,
					})
					leaveCell(wild.Position, 'F')
					wild.Position = newPos
					nudgeCh = wildRequests[wild.Position.X][wild.Position.Y]
					time.Sleep(TrapSleep)
					traces = append(traces, TraceType{
						TimeStamp: time.Since(startTime),
						ID:        wild.ID,
						Position:  wild.Position,
						Symbol:    '#',
					})
					leaveCell(wild.Position, 'T')
					break Alive
				}
				break
			}
			//if we've failed to move, we mark the cell as Wild-Occupied again
			select {
			case cellTokens[wild.Position.X][wild.Position.Y] <- 'W':
			default:
			}
		}
	}
	if wild.Symbol == '+' {
		traces = append(traces, TraceType{
			TimeStamp: time.Since(startTime),
			ID:        wild.ID,
			Position:  wild.Position,
			Symbol:    '#',
		})
		leaveCell(wild.Position, 'T')
	} else {
		leaveCell(wild.Position, 'F')
		wild.Position = PositionType{
			X: BoardWidth,
			Y: BoardHeight,
		}
		traces = append(traces, TraceType{
			TimeStamp: time.Since(startTime),
			ID:        wild.ID,
			Position:  wild.Position,
			Symbol:    wild.Symbol,
		})
	}
	tracesCh <- traces
}

func main() {
	fmt.Printf("-1 %d %d %d\n", NrOfTravelers+NrOfWilds+NrOfTraps, BoardWidth, BoardHeight)

	wgPrint.Add(1)
	go printer()

	// Initialize cell tokens for each board cell.
	initCells()
	// Trap some cells
	wg.Add(NrOfTraps)
	trapCells()
	// Initialize wilds
	initWildRequests()
	symbol := '0'
	for i := 0; i < NrOfWilds; i++ {
		wg.Add(1)
		go wild(NrOfTravelers+i, time.Now().UnixNano()+int64(i), symbol)
		symbol++
	}

	symbol = 'A'
	for i := 0; i < NrOfTravelers; i++ {
		wg.Add(1)
		go traveler(i, time.Now().UnixNano()+int64(i), symbol)
		symbol++
	}

	wg.Wait()
	close(tracesCh)
	wgPrint.Wait()
}
