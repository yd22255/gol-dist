package gol

import (
	"flag"
	"fmt"
	"net/rpc"
	"strconv"
	"time"

	"uk.ac.bris.cs/gameoflife/stubs"
)

var server = flag.String("server", "127.0.0.1:8030", "IP:port string to connect to as server")

type distributorChannels struct {
	events     chan<- Event
	ioCommand  chan<- ioCommand
	ioIdle     <-chan bool
	ioFilename chan<- string
	ioOutput   chan<- uint8
	ioInput    <-chan uint8
}

func outputPGM(c distributorChannels, p Params, world [][]uint8) {
	fmt.Println("inpgm")
	filename := strconv.Itoa(p.ImageHeight) + "x" + strconv.Itoa(p.ImageWidth) + "x" + strconv.Itoa(p.Turns)
	c.ioCommand <- ioOutput
	c.ioFilename <- filename
	count := 0
	for i := 0; i < p.ImageWidth; i++ {
		for j := 0; j < p.ImageHeight; j++ {
			count++
			fmt.Println(i, j)
			c.ioOutput <- world[i][j]
		}
	}
	fmt.Println(count)
}

//where neighbour was

//where worker was

func makeCall(client *rpc.Client, world [][]byte, p Params) *stubs.Response {
	fmt.Println("turns:", p.Turns)
	request := stubs.Request{StartY: 0, EndY: p.ImageHeight, StartX: 0, EndX: p.ImageWidth, World: world, Turns: p.Turns}
	response := new(stubs.Response)
	client.Call(stubs.ExecuteHandler, request, response)
	return response
}

func makeTicker(client *rpc.Client, world [][]byte, done chan bool, c distributorChannels) {
	fmt.Println("ticker start")
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				tirequest := stubs.Request{}
				tiresponse := new(stubs.Response)

				client.Call(stubs.ServerTicker, tirequest, tiresponse)
				fmt.Println(tiresponse.Turns, len(tiresponse.Alives))
				//fmt.Println(response.Turns, response.Alives)
				c.events <- AliveCellsCount{tiresponse.Turns, len(tiresponse.Alives)}
			}
		}
	}()
	return
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {
	///this bit can't be in distributor bc it loops
	flag.Parse()
	client, _ := rpc.Dial("tcp", *server)
	defer client.Close()
	/// but i dont know where to put it in that case given i'm not meant to have a client program
	//i think it might work if i close the server at the end of the distributor? but idk how to do that and then get it running again

	// TODO: Create a 2D slice to store the world.
	//make param string for filename and send it here
	fmt.Println(p.ImageWidth, p.ImageHeight)
	filename := strconv.Itoa(p.ImageHeight) + "x" + strconv.Itoa(p.ImageWidth)
	c.ioCommand <- ioInput
	c.ioFilename <- filename
	worldslice := make([][]uint8, p.ImageHeight)
	for i := range worldslice {
		worldslice[i] = make([]uint8, p.ImageWidth)
	}
	for i := 0; i < p.ImageHeight; i++ {
		for j := 0; j < p.ImageWidth; j++ {
			worldslice[i][j] = <-c.ioInput
		}
	}

	done := make(chan bool, 1)

	// TODO: Execute all turns of the Game of Life.
	go makeTicker(client, worldslice, done, c)
	finishedWorld := makeCall(client, worldslice, p)
	turn := 0
	// TODO: Report the final state using FinalTurnCompleteEvent.
	//pass down the events channel
	//close(c.ioOutput)
	c.events <- FinalTurnComplete{p.Turns, finishedWorld.Alives}
	outputPGM(c, p, finishedWorld.World)
	// Make sure that the Io has finished any output before exiting.
	fmt.Println("pre idle")
	//c.ioCommand <- ioCheckIdle
	fmt.Println("idle1")
	//<-c.ioIdle
	fmt.Println("idle")
	c.events <- StateChange{turn, Quitting}
	done <- true

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}
