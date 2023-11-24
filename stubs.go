package stubs

import (
	//"toby/workspace/golcw/gol-dist/gol"
	"uk.ac.bris.cs/gameoflife/util"
)

var ExecuteHandler = "GolOperations.ExecuteWorker"

//var PremiumReverseHandler = "SecretStringOperations.FastReverse"

type Params struct {
	Turns       int
	Threads     int
	ImageWidth  int
	ImageHeight int
}

type Response struct {
	World [][]byte
}

type Request struct {
	StartY, EndY, StartX, EndX int
	Alives                     []util.Cell
	World                      [][]byte
	Turns                      int
}
