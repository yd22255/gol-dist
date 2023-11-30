package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net"
	"net/rpc"
	"time"
	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

var Tchan int
var Achan []util.Cell
var Pause bool
var client *rpc.Client

// call to server to access and run GoL logic
func makeCall(client *rpc.Client, world [][]byte, p stubs.Params) *stubs.Response {
	brorequest := stubs.Request{StartY: 0, EndY: p.ImageHeight, StartX: 0, EndX: p.ImageWidth, World: world, Turns: p.Turns}
	brosponse := new(stubs.Response)
	client.Call(stubs.ExecuteHandler, brorequest, brosponse)
	return brosponse
}

type Broker struct {
}

// ExecuteGol function to call server and run GoL logic
// with multiple AWS nodes, would split the workload across the server with different ports
func (b *Broker) ExecuteGol(req stubs.Request, res *stubs.Response) (err error) {
	Pause = false
	var client *rpc.Client
	fmt.Println(req.World)
	client, _ = rpc.Dial("tcp", "127.0.0.1:8030")
	fmt.Println("in broker", req.Turns)

	// prerequisite for testing with zero-turn games
	if req.Turns == 0 {
		turnres := new(stubs.Response)
		client.Call(stubs.FindAlives, req, turnres)
		fmt.Println("hello - ", turnres)
		res.Alives = turnres.Alives
		res.Turns = req.Turns
		res.World = req.World

	}

	// makes a new call for each turn
	// in case of multiple nodes will call these nodes, wait for responses from all of them and then call again
	for i := 0; i < req.Turns; i++ {
		brores := makeCall(client, req.World, stubs.Params{Turns: req.Turns, Threads: 1, ImageWidth: req.EndX, ImageHeight: req.EndY})
		req.Alives = brores.Alives
		req.World = brores.World
		Tchan, Achan = i+1, brores.Alives
		res.Alives = brores.Alives
		res.World = brores.World
		fmt.Println("turn", Tchan)
	nested:
		// paused state implemented here to stop world updates until un-paused
		for Pause == true {
			if Pause == false {
				break nested
			}
		}

	}
	return
}

// TickerInterface to send the current turn and alive number of cells back to the controller
func (b *Broker) TickerInterface(req stubs.Request, res *stubs.Response) (err error) {
	fmt.Println("in ticker")
	res.Turns, res.Alives = Tchan, Achan
	return
}

// PauseFunc to pause the broker from a call in the controller
// unpauses upon being called again
func (b *Broker) PauseFunc(req stubs.Request, res *stubs.Response) (err error) {
	Pause = !Pause
	res.Turns = Tchan
	fmt.Println("paused status -- ", Pause)
	return
}

func main() {
	pAddr := flag.String("brok", "8031", "Port to listen on")
	client, _ = rpc.Dial("tcp", "127.0.0.1:8030")
	flag.Parse()
	rand.Seed(time.Now().UnixNano())
	rpc.Register(&Broker{})
	listener, _ := net.Listen("tcp", ":"+*pAddr)
	defer listener.Close()
	rpc.Accept(listener)
}
