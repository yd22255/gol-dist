package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net"
	"net/rpc"
	"time"
	"uk.ac.bris.cs/gameoflife/stubs"
)

var Tchan int
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
	brores := makeCall(client, req.World, stubs.Params{req.Turns, 1, req.EndX, req.EndY})
	res.Turns = brores.Turns
	res.Alives = brores.Alives
	res.World = brores.World
	fmt.Println("alive --", len(res.Alives))
	//Clearly shit in res due to this print, wont go through to distributor though :/
	fmt.Println("returning")
	return
}

func (b *Broker) TickerInterface(req stubs.Request, res *stubs.Response) (err error) {
	fmt.Println("in ticker")

	tiresponse := new(stubs.Response)

	client.Call(stubs.ServerTicker, req, tiresponse)

	fmt.Println(tiresponse.Turns, len(tiresponse.Alives))
	res.Turns = tiresponse.Turns
	res.Alives = tiresponse.Alives
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
