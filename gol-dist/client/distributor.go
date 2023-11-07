package gol

import (
	"flag"
	"fmt"
	"net/rpc"
	"strconv"

	"uk.ac.bris.cs/gameoflife/gol-dist/stubs"
	"uk.ac.bris.cs/gameoflife/util"
	//"uk.ac.bris.cs/gameoflife/util"
)

type distributorChannels struct {
	events     chan<- Event
	ioCommand  chan<- ioCommand
	ioIdle     <-chan bool
	ioFilename chan<- string
	ioOutput   chan<- uint8
	ioInput    <-chan uint8
}

func calculateAliveCells(p Params, world [][]byte) []util.Cell {
	var alives []util.Cell
	//world[6][4] = 255 .
	for i := 0; i < p.ImageWidth; i++ {
		for j := 0; j < p.ImageHeight; j++ {
			if world[i][j] == 255 {
				alives = append(alives, util.Cell{j, i})
			}
		}
	}
	//alives = append(alives, cell{0, 15})
	//fmt.Println(alives)

	return alives
}

//where neighbour was

//where worker was

func makeCall(client *rpc.Client, world [][]byte, p Params, alives []util.Cell) *stubs.Response {
	request := stubs.Request{StartY: 0, EndY: p.ImageHeight, StartX: 0, EndX: p.ImageWidth, World: world, Turns: p.Turns}
	response := new(stubs.Response)
	client.Call(stubs.ExecuteHandler, request, response)
	return response
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {

	server := flag.String("server", "127.0.0.1:8030", "IP:port string to connect to as server")
	flag.Parse()
	client, _ := rpc.Dial("tcp", *server)
	defer client.Close()

	// TODO: Create a 2D slice to store the world.
	//make param string for filename and send it here
	fmt.Println(p.ImageWidth, p.ImageHeight)
	filename := strconv.Itoa(p.ImageHeight) + "x" + strconv.Itoa(p.ImageWidth)
	c.ioCommand <- ioInput
	c.ioFilename <- filename
	worldslice := make([][]uint8, p.ImageHeight)
	for i := range worldslice {
		worldslice[i] = make([]uint8, p.ImageWidth)
		//recievedByte := <-c.ioInput
		//worldslice[i] = append(worldslice[i], recievedByte)
	}
	for i := 0; i < p.ImageHeight; i++ {
		for j := 0; j < p.ImageWidth; j++ {
			worldslice[i][j] = <-c.ioInput
		}
	}

	// TODO: Execute all turns of the Game of Life.
	alives := calculateAliveCells(p, worldslice)
	finishedWorld := makeCall(client, worldslice, p, alives)
	lastalives := calculateAliveCells(p, finishedWorld.World)
	turn := 0
	// TODO: Report the final state using FinalTurnCompleteEvent.
	//pass down the events channel
	//close(c.ioOutput)
	c.events <- FinalTurnComplete{p.Turns, lastalives}

	// Make sure that the Io has finished any output before exiting.
	fmt.Println("pre idle")
	//c.ioCommand <- ioCheckIdle
	fmt.Println("idle1")
	//<-c.ioIdle
	fmt.Println("idle")
	c.events <- StateChange{turn, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}
