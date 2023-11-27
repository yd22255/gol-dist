package main

import (
	"flag"
	"math/rand"
	"net"
	"net/rpc"
	"os"
	"time"

	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

var globalAlives []util.Cell
var globalTurns int
var globalPause bool
var globalWorld [][]uint8

func neighbour(req stubs.Request, y, x int) int {
	//Check neighbours for individual cell.
	count := 0
	edgex := [3]int{0, req.EndX - 1, 0}
	edgey := [3]int{0, req.EndY - 1, 0}
	adjacent := [6]int{x, x - 1, x + 1, y, y - 1, y + 1}

	for i := 0; i < len(edgex)-1; i++ {
		if x == edgex[i] {
			adjacent[i+1] = edgex[i+1]
		}
	}
	for i := 0; i < len(edgey)-1; i++ {
		if y == edgey[i] {
			adjacent[4+i] = edgey[i+1]
		}
	}
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if adjacent[i] != x || adjacent[j+3] != y {
				if req.World[adjacent[i]][adjacent[j+3]] == 255 {
					count++
				}
			}
		}
	}
	return count
}

func calculateAliveCells(req stubs.Request) []util.Cell {
	var alives []util.Cell
	for i := 0; i < req.EndX; i++ {
		for j := 0; j < req.EndY; j++ {
			//foreach cell in req.World
			if req.World[i][j] == 255 {
				//if that cell is alive, add it to the current alives list.
				alives = append(alives, util.Cell{j, i})
			}
		}
	}

	return alives
}

func ExecuteGol(req stubs.Request) [][]uint8 {
	//Actually execute a turn on the world itself
	newWorld := make([][]uint8, req.EndY)
	//create a newWorld as not to create issues during processing of the turn
	for i := range newWorld {
		newWorld[i] = make([]uint8, req.EndX)
		for j := range newWorld[i] {
			var x uint8
			x = req.World[i][j]
			newWorld[i][j] = x
		}
	}
	for i := req.StartX; i < req.EndX; i++ {
		for j := req.StartY; j < req.EndY; j++ {
			count := neighbour(req, j, i)
			if req.World[i][j] == 255 {
				if count != 2 && count != 3 {
					newWorld[i][j] = 0
				}
			} else {
				if count == 3 {
					newWorld[i][j] = 255
				}
			}
		}
	}

	return newWorld
}

type GolOperations struct {
}

func (g *GolOperations) ExecuteWorker(req stubs.Request, res *stubs.Response) (err error) {
	globalPause = false
	req.Alives = calculateAliveCells(req)
	globalAlives = req.Alives
	globalTurns = 0
	for i := 0; i < req.Turns; i++ {
		//execute values
		req.World = ExecuteGol(req)
		req.Alives = calculateAliveCells(req)
		//update globals so other operations can access them
		globalAlives = req.Alives
		globalTurns = globalTurns + 1
		globalWorld = req.World
		for globalPause == true {
			//busy-waiting, not the most elegant solution but it works.
		}
	}
	//give the return variables their requisite values
	res.World = req.World
	res.Alives = calculateAliveCells(req)
	return
}

func (g *GolOperations) ServerTicker(req stubs.Request, res *stubs.Response) (err error) {
	//return alives/turns for the ticker update.
	res.Alives = globalAlives
	res.Turns = globalTurns
	return
}

func (g *GolOperations) PauseFunc(req stubs.Request, res *stubs.Response) (err error) {
	//update the global pause, has dual purpose as a pause/unpause button
	if req.Pausereq == true {
		globalPause = true
	} else if req.Pausereq == false {
		globalPause = false
	}
	res.Turns = globalTurns
	return
}

func (g *GolOperations) PrintPGM(req stubs.Request, res *stubs.Response) (err error) {
	//return the current state of the world so we can print it via PGM.
	res.World = globalWorld
	return
}

func (g *GolOperations) KillServer(req stubs.Request, res *stubs.Response) (err error) {
	//kill the server gracefully.
	os.Exit(1)
	return
}

func main() {
	pAddr := flag.String("port", "8030", "Port to listen on")
	flag.Parse()
	rand.Seed(time.Now().UnixNano())
	rpc.Register(&GolOperations{})
	listener, _ := net.Listen("tcp", ":"+*pAddr)
	defer listener.Close()
	rpc.Accept(listener)
}
