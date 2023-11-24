package stubs

import (
	"uk.ac.bris.cs/gameoflife/util"
)

var ExecuteHandler = "GolOperations.ExecuteWorker"
var ServerTicker = "GolOperations.ServerTicker"
var PauseFunc = "GolOperations.PauseFunc"
var PrintPGM = "GolOperations.PrintPGM"
var KillServer = "GolOperations.KillServer"

type Params struct {
	Turns       int
	Threads     int
	ImageWidth  int
	ImageHeight int
}

type Response struct {
	World  [][]byte
	Alives []util.Cell
	Turns  int
}

type Request struct {
	StartY, EndY, StartX, EndX int
	Alives                     []util.Cell
	World                      [][]byte
	Turns                      int
	Pausereq                   bool
}
