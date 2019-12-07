package main

import (
	"fmt"
	"strconv"
	"strings"
)

func sendWorld(p golParams, world [][]byte, d distributorChans) {
	d.io.command <- ioOutput
	d.io.filename <- strings.Join([]string{strconv.Itoa(p.imageWidth), strconv.Itoa(p.imageHeight) + "-" + strconv.Itoa(p.turns)}, "x")

	for y := range world {
		for x := range world[y] {
			d.io.outputVal <- world[y][x]
		}
	}
}

func worker(haloHeight int, in <-chan byte, out chan<- byte, p golParams) {
	workerWorld := make([][]byte, haloHeight)
	for i := range workerWorld {
		workerWorld[i] = make([]byte, p.imageWidth)
	}
	for {
		for y := 0; y < haloHeight; y++ {
			for x := 0; x < p.imageWidth; x++ {
				//fmt.Println("hi")
				workerWorld[y][x] = <-in
			}
		}

		for y := 1; y < haloHeight-1; y++ {
			for x := 0; x < p.imageWidth; x++ {
				xRight:= x+1
                                						                        						xLeft := x-1

                                                        						if xRight >= p.imageWidth {
                                                        							xRight %= p.imageWidth
                                                        						}
                                                        						if xLeft < 0 {
                                                        							xLeft += p.imageWidth
                                                        						}
                                                        						count := 0
                                                        						count = int(workerWorld[y-1][xLeft]) +
                                                        								int(workerWorld[y-1][x]) +
                                                        								int(workerWorld[y-1][xRight]) +
                                                        								int(workerWorld[y][xLeft]) +
                                                        								int(workerWorld[y][xRight]) +
                                                        								int(workerWorld[y+1][xLeft]) +
                                                        								int(workerWorld[y+1][x]) +
                                                        								int(workerWorld[y+1][xRight])
                                                        						count /= 255
				if count == 3 || (workerWorld[y][x] == 0xFF && count == 2) {
					out <- 0xFF
				} else {
					out <- 0
				}
			}
		}
	}

}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p golParams, d distributorChans, alive chan []cell) {

	// Create the 2D slice to store the world.
	world := make([][]byte, p.imageHeight)
	for i := range world {
		world[i] = make([]byte, p.imageWidth)
	}

	// Request the io goroutine to read in the image with the given filename.
	d.io.command <- ioInput
	d.io.filename <- strings.Join([]string{strconv.Itoa(p.imageWidth), strconv.Itoa(p.imageHeight)}, "x")

	// The io goroutine sends the requested image byte by byte, in rows.
	for y := 0; y < p.imageHeight; y++ {
		for x := 0; x < p.imageWidth; x++ {
			val := <-d.io.inputVal
			if val != 0 {
				fmt.Println("Alive cell at", x, y)
				world[y][x] = val
			}
		}
	}
	threadHeight := p.imageHeight / p.threads
	in := make([]chan byte, p.threads)
	out := make([] chan byte, p.threads)
	// Calculate the new state of Game of Life after the given number of turns.
	for i := 0; i < p.threads; i++ {
		in[i] = make(chan byte, p.imageHeight)
		out[i] = make(chan byte, p.imageHeight)
	}
	for i := 0; i < p.threads; i++ {
		go worker(threadHeight+2, in[i], out[i], p)
	}
	for turns := 0; turns < p.turns; turns++ {
		for i := 0; i < p.threads; i++ {
			for y := 0; y < (threadHeight)+2; y++ {
				for x := 0; x < p.imageWidth; x++ {
					proposedY := y + (i * threadHeight) - 1

					if proposedY < 0 {
						proposedY += p.imageHeight
					}
					proposedY %= p.imageHeight
					in[i] <- world[proposedY][x]
				}

			}
		}
		for i := 0; i < p.threads; i++ {
			for y := 0; y < threadHeight; y++ {
				for x := 0; x < p.imageWidth; x++ {
					world[y+(i*(threadHeight))][x] = <-out[i]

				}
			}
		}
	}

	// Create an empty slice to store coordinates of cells that are still alive after p.turns are done.
	var finalAlive []cell
	// Go through the world and append the cells that are still alive.
	for y := 0; y < p.imageHeight; y++ {
		for x := 0; x < p.imageWidth; x++ {
			if world[y][x] != 0 {
				finalAlive = append(finalAlive, cell{x: x, y: y})
			}
		}
	}

	sendWorld(p, world, d)

	// Make sure that the Io has finished any output before exiting.
	d.io.command <- ioCheckIdle
	<-d.io.idle

	// Return the coordinates of cells that are still alive.
	alive <- finalAlive
}
