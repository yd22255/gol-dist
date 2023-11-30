package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net"
	"net/rpc"
	"os"
	"time"

	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

var World [][]byte

// neighbour to check how many of a cells bordering cells are alive.
func neighbour(req stubs.Request, y, x int) int {
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

// function to calculate alive cells for GOL logic and tests to function.
func calculateAliveCells(req stubs.Request) []util.Cell {
	var alives []util.Cell
	for i := 0; i < req.EndX; i++ {
		for j := 0; j < req.EndY; j++ {
			if req.World[i][j] == 255 {
				alives = append(alives, util.Cell{j, i})
			}
		}
	}
	return alives
}

// ExecuteGol to compute main GoL logic
// updates world based on alive cells through neighbour and returns
func ExecuteGol(req stubs.Request) [][]byte {
	//Feed in horizontal strips.
	newWorld := make([][]uint8, req.EndY)
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

// FindAlives required to pass back alive cells without entering progressing a turn
func (g *GolOperations) FindAlives(req stubs.Request, res *stubs.Response) (err error) {
	fmt.Println(res.Alives)
	res.Alives = calculateAliveCells(req)

	return
}

// ExecuteWorker interacts between server and broker to pass back a completed turn
func (g *GolOperations) ExecuteWorker(req stubs.Request, res *stubs.Response) (err error) {
	fmt.Println("exectued")
	req.Alives = calculateAliveCells(req)
	req.World = ExecuteGol(req)
	req.Alives = calculateAliveCells(req)
	World = req.World
	fmt.Println("returning")
	res.World = req.World
	res.Alives = calculateAliveCells(req)
	return
}

// PrintPGM kills server upon keypress
func (g *GolOperations) PrintPGM(req stubs.Request, res *stubs.Response) (err error) {
	res.World = World
	os.Exit(1)
	return

}

//KillServer kills server upon keypress
func (g *GolOperations) KillServer(req stubs.Request, res *stubs.Response) (err error) {
	os.Exit(1)
	return
}

func main() {
	pAddr := flag.String("port", "8030", "Port to listen on")
	flag.Parse()
	rand.Seed(time.Now().UnixNano())
	err := rpc.Register(&GolOperations{})
	if err != nil {
		return
	}
	listener, _ := net.Listen("tcp", ":"+*pAddr)
	defer func(listener net.Listener) {
		err := listener.Close()
		if err != nil {

		}
	}(listener)
	rpc.Accept(listener)
}
