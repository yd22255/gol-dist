package gol

import (
	"flag"
	"math/rand"
	"net"
	"net/rpc"
	"time"

	"uk.ac.bris.cs/gameoflife/gol-dist/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

func neighbour(req stubs.Request, y, x int) int {
	//Check neighbours for individual cell. Find way to implement for loop for open grid checking
	count := 0
	//z := util.Cell{x, y}
	var left, right, up, down int = 0, 0, 0, 0
	if x == 0 {
		left = req.EndX - 1
	} else {
		left = x - 1
	}
	if x == req.EndX-1 {
		right = 0
	} else {
		right = x + 1
	}
	if y == 0 {
		up = req.EndY - 1
	} else {
		up = y - 1
	}
	if y == req.EndY-1 {
		down = 0
	} else {
		down = y + 1
	}
	//TODO : Run foreach on each neighbour - Likely need array of neighbours which isnt hard
	if util.Cell.In(util.Cell{right, y}, req.Alives) {
		count++
	}
	if util.Cell.In(util.Cell{right, down}, req.Alives) {
		count++
	}
	if util.Cell.In(util.Cell{right, up}, req.Alives) {
		count++
	}
	if util.Cell.In(util.Cell{left, y}, req.Alives) {
		count++
	}
	if util.Cell.In(util.Cell{left, down}, req.Alives) {
		count++
	}
	if util.Cell.In(util.Cell{left, up}, req.Alives) {
		count++
	}
	if util.Cell.In(util.Cell{x, down}, req.Alives) {
		count++
	}
	if util.Cell.In(util.Cell{x, up}, req.Alives) {
		count++
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
	for i := req.StartX; i < req.EndX; i++ {
		for j := req.StartY; j < req.EndY; j++ {
			count := neighbour(req, i, j)
			if req.World[i][j] == 255 {
				if count != 2 && count != 3 {
					req.World[i][j] = 0
				}
			} else {
				if count == 3 {
					req.World[i][j] = 255
				}
			}
		}
	}
	return req.World
}

type GolOperations struct{}

func (g *GolOperations) ExecuteWorker(req stubs.Request, res *stubs.Response) (err error) {
	for i := 0; i < req.Turns; i++ {
		res.World = ExecuteGol(req)
		req.Alives = calculateAliveCells(req)
	}

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
