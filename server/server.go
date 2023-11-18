package main

import (
	"flag"
	"math/rand"
	"net"
	"net/rpc"
	"time"

	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

var Achan []util.Cell
var Tchan int

func neighbour(req stubs.Request, y, x int) int {
	//Check neighbours for individual cell. Find way to implement for loop for open grid checking
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
			if req.World[i][j] == 255 {
				alives = append(alives, util.Cell{j, i})
			}
		}
	}

	return alives
}

/** Super-Secret `reversing a string' method we can't allow clients to see. **/
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

func (g *GolOperations) ExecuteWorker(req stubs.Request, res *stubs.Response) (err error) {
	//res.Alives = calculateAliveCells(req)
	for i := 0; i < req.Turns; i++ {
		req.World = ExecuteGol(req)
		req.Alives = calculateAliveCells(req)
		Achan = req.Alives
		Tchan = req.Turns
	}
	res.World = req.World
	return
}

func (g *GolOperations) ServerTicker(req stubs.Request, res stubs.Response) (err error) {
	res.Alives = Achan
	res.Turns = Tchan
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
