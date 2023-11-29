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

func makeCall(client *rpc.Client, world [][]byte, p stubs.Params) *stubs.Response {
	brorequest := stubs.Request{StartY: 0, EndY: p.ImageHeight, StartX: 0, EndX: p.ImageWidth, World: world, Turns: p.Turns}
	brosponse := new(stubs.Response)
	client.Call(stubs.ExecuteHandler, brorequest, brosponse)
	//I think will need a new stubs to pass appropriate values. Also not sure if all params are needed, + start x is useless even when parallelised
	//actually no, can probably just condense original stubs, which should be shortened anyway imo
	fmt.Println("brosponese - ", brosponse.Alives)
	return brosponse
}

type Broker struct {
}

//Basically GoLoperations

func (s *Broker) ExecuteGol(req stubs.Request, res *stubs.Response) (err error) {
	fmt.Println("in broker")
	var client *rpc.Client

	client, _ = rpc.Dial("tcp", "127.0.0.1:8030")
	World = req.World

	//finishedWorld := new([][]uint8)
	res = makeCall(client, World, stubs.Params{16, 1, 512, 512})
	fmt.Println(res.Alives)
	res.Turns = 123
	return
}

func main() {
	pAddr := flag.String("port", "8031", "Port to listen on")
	flag.Parse()
	rand.Seed(time.Now().UnixNano())
	rpc.Register(&Broker{})
	listener, _ := net.Listen("tcp", ":"+*pAddr)
	defer listener.Close()
	rpc.Accept(listener)
}
