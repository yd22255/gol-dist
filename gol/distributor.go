package gol

import (
	"flag"
	"fmt"
	"net/rpc"
	"os"
	"strconv"
	"time"
	"uk.ac.bris.cs/gameoflife/stubs"
)

var broker = flag.String("broker", "127.0.0.1:8031", "IP:port string to connect to as server")

type distributorChannels struct {
	events     chan<- Event
	ioCommand  chan<- ioCommand
	ioIdle     <-chan bool
	ioFilename chan<- string
	ioOutput   chan<- uint8
	ioInput    <-chan uint8
	KeyPresses <-chan rune
}

// function to output the inputted world as a .pgm file
func outputPGM(c distributorChannels, p Params, world [][]uint8) {
	filename := strconv.Itoa(p.ImageHeight) + "x" + strconv.Itoa(p.ImageWidth) + "x" + strconv.Itoa(p.Turns)
	c.ioCommand <- ioOutput
	c.ioFilename <- filename
	for i := 0; i < p.ImageWidth; i++ {
		for j := 0; j < p.ImageHeight; j++ {
			c.ioOutput <- world[i][j]
		}
	}
}

// call to broker
func makeCall(client *rpc.Client, world [][]byte, p Params) *stubs.Response {
	request := stubs.Request{StartY: 0, EndY: p.ImageHeight, StartX: 0, EndX: p.ImageWidth, World: world, Turns: p.Turns}
	response := new(stubs.Response)
	err := client.Call(stubs.BrokerTest, request, response)
	if err != nil {
		return nil
	}
	return response
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {
	flag.Parse()
	client, _ := rpc.Dial("tcp", *broker)
	defer func(client *rpc.Client) {
		err := client.Close()
		if err != nil {

		}
	}(client)

	// zero-turn world created from inputted file
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

	// goroutine runs both ticker and SDL keypress logic
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
				err := client.Call(stubs.TickInterface, tirequest, tiresponse)
				if err != nil {
					return
				}
				c.events <- AliveCellsCount{tiresponse.Turns, len(tiresponse.Alives)}
			case command := <-c.KeyPresses:
				switch command {
				case 's':
					// print PGM from current server state
					srequest := stubs.Request{}
					sresponse := new(stubs.Response)
					err := client.Call(stubs.PrintPGM, srequest, sresponse)
					if err != nil {
						return
					}
					outputPGM(c, p, sresponse.World)
				case 'q':
					// close controller client without causing error on GoL server
					qrequest := stubs.Request{}
					qresponse := new(stubs.Response)
					err := client.Call(stubs.ServerTicker, qrequest, qresponse)
					if err != nil {
						return
					}
					fmt.Println("quitting")
					c.events <- StateChange{qresponse.Turns, Quitting}
					done <- true
					close(c.events)
					os.Exit(1)
				case 'k':
					// shutdown all components of dist cleanly
					// output pgm of latest state too
					krequest := stubs.Request{}
					kresponse := new(stubs.Response)
					err := client.Call(stubs.PrintPGM, krequest, kresponse)
					if err != nil {
						return
					}
					outputPGM(c, p, kresponse.World)

					c.events <- StateChange{5, Quitting}
					done <- true
					killrequest := stubs.Request{}
					killresponse := new(stubs.Response)
					err = client.Call(stubs.KillServer, killrequest, killresponse)
					if err != nil {
						return
					}
					close(c.events)
					os.Exit(1)
				case 'p':
					// pause processing on AWS node + controller print current turn being processed (prolly yoink ticker code)
					pausereq := stubs.Request{Pausereq: true}
					pauseres := new(stubs.Response)
					err := client.Call(stubs.PauseTest, pausereq, pauseres)
					if err != nil {
						return
					}
					c.events <- StateChange{pauseres.Turns, Paused}
					isPaused := true
					fmt.Println("Paused!")

				nested:
					// runs this loop while paused
					// when keypress 'p' received again resume executing GoL
					for {
						select {
						case command := <-c.KeyPresses:
							if command == 'p' {
								//Put unpause code here
								fmt.Println("Unpaused!")
								pausereq1 := stubs.Request{Pausereq: false}
								pauseres1 := new(stubs.Response)
								err := client.Call(stubs.PauseTest, pausereq1, pauseres1)
								if err != nil {
									return
								}
								c.events <- StateChange{pauseres1.Turns, Executing}
								isPaused = false

							}
						}
						if !isPaused {
							break nested
						}
					}
				}
			}
		}
	}()

	finishedWorld := makeCall(client, worldslice, p)
	turn := 0

	// outputs final .pgm file
	outputPGM(c, p, finishedWorld.World)
	c.events <- FinalTurnComplete{p.Turns, finishedWorld.Alives}
	c.ioCommand <- ioCheckIdle
	Idle := <-c.ioIdle
	if Idle == true {
		c.events <- StateChange{turn, Quitting}
	}
	// Make sure that the Io has finished any output before exiting.

	done <- true
	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}
