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

var server = flag.String("server", "127.0.0.1:8030", "IP:port string to connect to as server")

type distributorChannels struct {
	events     chan<- Event
	ioCommand  chan<- ioCommand
	ioIdle     <-chan bool
	ioFilename chan<- string
	ioOutput   chan<- uint8
	ioInput    <-chan uint8
	KeyPresses <-chan rune
}

func outputPGM(c distributorChannels, p Params, world [][]uint8) {
	fmt.Println(world)
	filename := strconv.Itoa(p.ImageHeight) + "x" + strconv.Itoa(p.ImageWidth) + "x" + strconv.Itoa(p.Turns)
	c.ioCommand <- ioOutput
	c.ioFilename <- filename
	for i := 0; i < p.ImageWidth; i++ {
		for j := 0; j < p.ImageHeight; j++ {
			c.ioOutput <- world[i][j]
		}
	}
}

func makeCall(client *rpc.Client, world [][]uint8, p Params) *stubs.Response {
	//make the initial GOL call to the server which starts the processing off.
	request := stubs.Request{StartY: 0, EndY: p.ImageHeight, StartX: 0, EndX: p.ImageWidth, World: world, Turns: p.Turns}
	response := new(stubs.Response)
	client.Call(stubs.ExecuteHandler, request, response)

	return response
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {
	//create the client and parse the server flag.
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
	}
	for i := 0; i < p.ImageHeight; i++ {
		for j := 0; j < p.ImageWidth; j++ {
			worldslice[i][j] = <-c.ioInput
		}
	}

	done := make(chan bool, 1) //setup a kill switch for the for-select statement

	// TODO: Execute all turns of the Game of Life.
	go func() {
		//first we set up a ticker goroutine to execute while the gol logic is going on.
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				tirequest := stubs.Request{}
				tiresponse := new(stubs.Response)
				//make a new rpc call every 2 seconds to access the current state of affairs.
				client.Call(stubs.ServerTicker, tirequest, tiresponse)
				c.events <- AliveCellsCount{tiresponse.Turns, len(tiresponse.Alives)}
				//then report it to the events channel.
			case command := <-c.KeyPresses:
				switch command {
				case 's':
					//print PGM from current server state
					srequest := stubs.Request{}
					sresponse := new(stubs.Response)
					client.Call(stubs.PrintPGM, srequest, sresponse)
					outputPGM(c, p, sresponse.World)
				case 'q':
					//close controller client without causing an error on the GoL server
					qrequest := stubs.Request{}
					qresponse := new(stubs.Response)
					client.Call(stubs.ServerTicker, qrequest, qresponse)
					fmt.Println("Quitting")
					c.events <- StateChange{qresponse.Turns, Quitting}
					done <- true
					close(c.events)
					os.Exit(1)
				case 'k':
					//Shutdown all components of dist cleanly.
					//Output pgm of latest state before closing
					krequest := stubs.Request{}
					kresponse := new(stubs.Response)
					client.Call(stubs.PrintPGM, krequest, kresponse)
					outputPGM(c, p, kresponse.World)
					//then send the events the signifier
					c.events <- StateChange{5, Quitting}
					done <- true
					//kill the server before the controller, to ensure we dont have issues
					//with unanchored return statements
					killrequest := stubs.Request{}
					killresponse := new(stubs.Response)
					client.Call(stubs.KillServer, killrequest, killresponse)
					close(c.events)
					//and finally close events before killing the controller
					os.Exit(1)
				case 'p':
					//Pause processing on AWS node + controller print the current turn being processed
					pausereq := stubs.Request{Pausereq: true}
					pauseres := new(stubs.Response)
					client.Call(stubs.PauseFunc, pausereq, pauseres)
					c.events <- StateChange{pauseres.Turns, Paused}

					isPaused := true
					fmt.Println("Paused!")
				nested:
					for {
						//Resume after p pressed again. the for-select here prevents the
						//ticker from sending anything during the pause, thus keeping the program
						//completely suspended
						select {
						case command := <-c.KeyPresses:
							if command == 'p' {
								//Unpause the server
								fmt.Println("Unpaused!")
								pausereq1 := stubs.Request{Pausereq: false}
								pauseres1 := new(stubs.Response)
								client.Call(stubs.PauseFunc, pausereq1, pauseres1)
								c.events <- StateChange{pauseres1.Turns, Executing}
								isPaused = false

							}
						}
						if !isPaused {
							//unpausing break of the for-select.
							break nested
						}
					}
				}
			}
		}
	}()
	finishedWorld := makeCall(client, worldslice, p)
	turn := 0
	// TODO: Report the final state using FinalTurnCompleteEvent.
	c.events <- FinalTurnComplete{p.Turns, finishedWorld.Alives}
	outputPGM(c, p, finishedWorld.World)
	// Make sure that the IO has finished any output before exiting.
	fmt.Println("pre idle")
	fmt.Println("idle1")
	fmt.Println("idle")
	c.events <- StateChange{turn, Quitting}
	done <- true //kill the ticker
	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}
