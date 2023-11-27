package main

import (
	"net/rpc"
	"uk.ac.bris.cs/gameoflife/stubs"
)

var Tchan int
var Pause bool
var World [][]byte

func makeCall(client *rpc.Client, world [][]byte, p Params) *stubs.Response) {
	request := stubs.Request{StartY: 0, EndY: p.ImageHeight, StartX:0, EndX: p.ImageWidth, World: world, Turns: p.Turns}    response := new(stubs.Response)
	response := new(stubs.Response)
	//I think will need a new stubs to pass appropriate values. Also not sure if all params are needed, + start x is useless even when parallelised
	return response
}



func main() {

}
