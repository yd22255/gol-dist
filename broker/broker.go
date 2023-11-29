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
var World [][]byte
var p stubs.Params
var client *rpc.Client

func makeCall(client *rpc.Client, world [][]byte, p stubs.Params) *stubs.Response {
	brorequest := stubs.Request{StartY: 0, EndY: p.ImageHeight, StartX: 0, EndX: p.ImageWidth, World: world, Turns: p.Turns}
	brosponse := new(stubs.Response)
	client.Call(stubs.ExecuteHandler, brorequest, brosponse)
	//I think will need a new stubs to pass appropriate values. Also not sure if all params are needed, + start x is useless even when parallelised
	//actually no, can probably just condense original stubs, which should be shortened anyway imo
	//fmt.Println("brosponese - ", brosponse.Alives)
	return brosponse
}

type Broker struct {
}

//Basically GoLoperations

func (b *Broker) ExecuteGol(req stubs.Request, res *stubs.Response) (err error) {
	var client *rpc.Client
	client, _ = rpc.Dial("tcp", "127.0.0.1:8030")
	fmt.Println("in broker", req.Turns)
	//fmt.Println("WORLD --", len())
	for i := 0; i < req.Turns; i++ {
		brores := makeCall(client, req.World, stubs.Params{req.Turns, 1, req.EndX, req.EndY})
		//req.Turns = brores.Turns
		req.Alives = brores.Alives
		req.World = brores.World
		Tchan, Achan = i+1, brores.Alives
	}
	res.Alives = req.Alives
	res.World = req.World
	fmt.Println("alive --", len(res.Alives))
	//Clearly shit in res due to this print, wont go through to distributor though :/
	fmt.Println("returning")
	return
}

func (b *Broker) TickerInterface(req stubs.Request, res *stubs.Response) (err error) {
	fmt.Println("in ticker")
	res.Turns, res.Alives = Tchan, Achan
	return
}

func main() {
	pAddr := flag.String("brok", "8031", "Port to listen on")
	client, _ = rpc.Dial("tcp", "127.0.0.1:8030")
	flag.Parse()
	rand.Seed(time.Now().UnixNano())
	rpc.Register(&Broker{})
	listener, _ := net.Listen("tcp", ":"+*pAddr)
	//^^ Need to setup broker somehow since we can't build it ourselves. Probably make method in broker itself
	defer listener.Close()
	rpc.Accept(listener)
}
