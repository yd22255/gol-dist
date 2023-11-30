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

var globalTurns int
var globalAlives []util.Cell
var Pause bool
var globalWorld [][]uint8
var KillVar bool

//var _ *rpc.Client

// call to server to access and run GoL logic
func makeCall(client *rpc.Client, world [][]byte, p stubs.Params) *stubs.Response {
	brorequest := stubs.Request{StartY: 0, EndY: p.ImageHeight, StartX: 0, EndX: p.ImageWidth, World: world, Turns: p.Turns}
	brosponse := new(stubs.Response)
	err := client.Call(stubs.ExecuteHandler, brorequest, brosponse)
	if err != nil {
		return nil
	}
	return brosponse
}

type Broker struct {
}

// ExecuteGol function to call server and run GoL logic
// with multiple AWS nodes, would split the workload across the server with different ports
func (b *Broker) ExecuteGol(req stubs.Request, res *stubs.Response) (err error) {
	KillVar = false
	Pause = false
	var client *rpc.Client
	client, _ = rpc.Dial("tcp", "127.0.0.1:8030")

	// prerequisite for testing with zero-turn games
	if req.Turns == 0 {
		turnres := new(stubs.Response)
		err := client.Call(stubs.FindAlives, req, turnres)
		if err != nil {
			return err
		}
		res.Alives = turnres.Alives
		res.Turns = req.Turns
		res.World = req.World

	}

	// makes a new call for each turn
	// in case of multiple nodes will call these nodes, wait for responses from all of them and then call again
	for i := 0; i < req.Turns; i++ {
		if KillVar == true {
			client.Call(stubs.KillServer, req, res)
			return
		}
		brores := makeCall(client, req.World, stubs.Params{Turns: req.Turns, Threads: 1, ImageWidth: req.EndX, ImageHeight: req.EndY})
		req.Alives = brores.Alives
		req.World = brores.World
		//update the requirements for the next loop
		globalTurns, globalAlives = i+1, brores.Alives
		globalWorld = brores.World
		//update the ticker's values
		res.Alives = brores.Alives
		res.World = brores.World
		//update the return values
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

func (b *Broker) SendKillServer(req stubs.Request, res *stubs.Response) (err error) {
	KillVar = true
	//turn the kill variable on to os.Exit the server
	return
}

func (b *Broker) KillBroker(req stubs.Request, res *stubs.Response) (err error) {
	//kill the broker cleanly
	os.Exit(1)
	return
}

// TickerInterface sends the current turn and alive number of cells back to the controller upon every tick
func (b *Broker) TickerInterface(req stubs.Request, res *stubs.Response) (err error) {
	res.Turns, res.Alives, res.World = globalTurns, globalAlives, globalWorld
	return
}

// PauseFunc to pause the broker from a call in the controller
// unpauses upon being called again
func (b *Broker) PauseFunc(req stubs.Request, res *stubs.Response) (err error) {
	Pause = !Pause
	res.Turns = globalTurns
	return
}

func main() {
	pAddr := flag.String("broker", "8031", "Port to listen on")
	flag.Parse()
	rand.Seed(time.Now().UnixNano())
	err := rpc.Register(&Broker{})
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
