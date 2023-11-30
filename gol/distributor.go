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
func outputPGM(c distributorChannels, p Params, world [][]uint8, turnsFinished int) {
	filename := strconv.Itoa(p.ImageHeight) + "x" + strconv.Itoa(p.ImageWidth) + "x" + strconv.Itoa(turnsFinished)
	c.ioCommand <- ioOutput
	c.ioFilename <- filename
	for i := 0; i < p.ImageWidth; i++ {
		for j := 0; j < p.ImageHeight; j++ {
			c.ioOutput <- world[i][j]
		}
	}
}

// Make the initial call to broker which starts the process off
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
	//create the client and parse the server
	client, _ := rpc.Dial("tcp", *broker)
	defer func(client *rpc.Client) {
		err := client.Close()
		if err != nil {

		}
	}(client)

	//TODO: Create a 2D slice to store the world.
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

	done := make(chan bool, 1) //setup a kill switch for the for-select statement
	//TODO: Execute all turns of the Game of Life
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
				//make a new rpc call every 2 seconds to access the current state of the Game
				err := client.Call(stubs.TickInterface, tirequest, tiresponse)
				if err != nil {
					return
				}
				c.events <- AliveCellsCount{tiresponse.Turns, len(tiresponse.Alives)}
				//then report it to the events channel
			case command := <-c.KeyPresses:
				switch command {
				case 's':
					// print PGM from current server state
					tirequest := stubs.Request{}
					tiresponse := new(stubs.Response)
					client.Call(stubs.TickInterface, tirequest, tiresponse)
					outputPGM(c, p, tiresponse.World, tiresponse.Turns)
				case 'q':
					// close controller client without causing error on GoL server
					qrequest := stubs.Request{}
					qresponse := new(stubs.Response)
					client.Call(stubs.TickInterface, qrequest, qresponse)
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
					client.Call(stubs.TickInterface, krequest, kresponse)
					outputPGM(c, p, kresponse.World, kresponse.Turns)
					//kill the server before the broker, to ensure we dont have issues with unanchored
					//return statements
					brequest := stubs.Request{}
					bresponse := new(stubs.Response)
					client.Call(stubs.SendKillServer, brequest, bresponse)
					time.Sleep(1 * time.Second)
					sresponse := new(stubs.Response)
					client.Call(stubs.KillBroker, brequest, sresponse)
					//then send the events channel the signifier
					c.events <- StateChange{kresponse.Turns, Quitting}
					done <- true
					close(c.events)
					//and close events before killing the controller
					os.Exit(1)

				case 'p':
					// pause processing on AWS node + controller print current turn being processed
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
					// runs this loop while paused, preventing the ticker from sending anything during this pause
					//and so keeping the GOL record suspended
					// when keypress 'p' received again resume executing GoL
					for {
						select {
						case command := <-c.KeyPresses:
							if command == 'p' {
								//UNpause on another 'p' press
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
							//unpausing break of the for-select
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
	outputPGM(c, p, finishedWorld.World, finishedWorld.Turns)
	//TODO: Report the final state using FinalTurnCompleteEvent.
	c.events <- FinalTurnComplete{p.Turns, finishedWorld.Alives}
	c.ioCommand <- ioCheckIdle
	Idle := <-c.ioIdle
	if Idle == true {
		c.events <- StateChange{turn, Quitting}
	}
	// Make sure that the Io has finished any output before exiting.

	done <- true //kill the tickerc
	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}
