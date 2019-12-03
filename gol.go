package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func sendWorldtoPGM(p golParams, world [][]byte, d distributorChans, turn int) {
	d.io.command <- ioOutput
	d.io.filename <- strings.Join([]string{strconv.Itoa(p.imageWidth), strconv.Itoa(p.imageHeight), "Turn:" + strconv.Itoa(turn)}, "x")

	for y := range world {
		for x := range world[y] {
			d.io.outputVal <- world[y][x]
		}
	}
}

func getWorldfromPGM(p golParams, d distributorChans) [][]byte {
	world := buildWorld(p, p.imageHeight)

	// Request the io goroutine to read in the image with the given filename.
	d.io.command <- ioInput
	d.io.filename <- strings.Join([]string{strconv.Itoa(p.imageWidth), strconv.Itoa(p.imageHeight)}, "x")

	// The io goroutine sends the requested image byte by byte, in rows.
	for y := 0; y < p.imageHeight; y++ {
		for x := 0; x < p.imageWidth; x++ {
			val := <-d.io.inputVal
			if val != 0 {
				world[y][x] = val
			}
		}
	}
	return world
}

func buildWorld(p golParams, haloHeight int) [][]byte {
	workerWorld := make([][]byte, haloHeight)
	for i := range workerWorld {
		workerWorld[i] = make([]byte, p.imageWidth)
	}
	return workerWorld
}

func countAliveCells(p golParams, world [][]byte, in []chan byte, out []chan byte, threadHeight int, extra int) {
	go notifyWorkers(p, in, 0xAB)
	world = getWorldFromWorkers(p, world, out, threadHeight, extra)
	alive := 0
	for y := 0; y < p.imageHeight; y++ {
		for x := 0; x < p.imageWidth; x++ {
			if world[y][x] == 0xFF {
				alive ++
			}
		}
	}
	fmt.Println("Number of Alive Cells:", alive)
}

func isAlive(imageWidth, x, y int, world [][]byte) bool {
	x += imageWidth
	x %= imageWidth
	if world[y][x] != 0xFF {
		return false
	} else {
		return true
	}
}

func giveWorldToWorkers(p golParams, world [][]byte, in []chan byte, threadHeight int, extra int) {
	for i := 0; i < p.threads; i++ {
		yBound := threadHeight + 2
		if i == p.threads-1 && !powerOfTwo(p) {
			yBound += extra
		}
		for y := 0; y < yBound; y++ {
			proposedY := y + (i * threadHeight) - 1
			if proposedY < 0 {
				proposedY += p.imageHeight
			}
			proposedY %= p.imageHeight
			for x := 0; x < p.imageWidth; x++ {
				in[i] <- world[proposedY][x]
			}
		}
	}
}

func sendWorldFromWorkers(p golParams, world [][]byte, out chan<- byte, height int) {
	for y := 1; y < height-1; y++ {
		for x := 0; x < p.imageWidth; x++ {
			out <- world[y][x]
		}
	}
}

func getWorldFromWorkers(p golParams, world [][]byte, out []chan byte, threadHeight int, extra int) [][]byte {
	for i := 0; i < p.threads; i++ {
		for y := 0; y < threadHeight; y++ {
			for x := 0; x < p.imageWidth; x++ {
				world[y+(i*(threadHeight))][x] = <-out[i]
			}
		}
	}
	if !powerOfTwo(p) {
		for e := 0; e < extra; e++ {
			for x := 0; x < p.imageWidth; x++ {
				world[e+(p.threads*(threadHeight))][x] = <-out[p.threads-1]
			}
		}
	}
	return world
}

func notifyWorkers(p golParams, in []chan byte, key byte) {
	for i := 0; i < p.threads; i++ {
		in[i] <- key
	}
}

func worker(haloHeight int, in <-chan byte, out chan<- byte, p golParams, sending []chan byte, receiving [2]chan byte) {
	workerWorld := buildWorld(p, haloHeight)
	for y := 0; y < haloHeight; y++ {
		for x := 0; x < p.imageWidth; x++ {
			workerWorld[y][x] = <-in
		}
	}
	temp := buildWorld(p, haloHeight)
loop:
	for {
		select {
		case val := <-in:
			if val == 0xAA {
				for x := 0; x < p.imageWidth; x++ {
					sending[0] <- workerWorld[1][x]
					sending[1] <- workerWorld[haloHeight-2][x]
				}
				for x := 0; x < p.imageWidth; x++ {
					workerWorld[0][x] = <-receiving[0]
					workerWorld[haloHeight-1][x] = <-receiving[1]
				}

				//GOL Logic
				for y := 1; y < haloHeight-1; y++ {
					for x := 0; x < p.imageWidth; x++ {
						count := 0
						for i := -1; i <= 1; i++ {
							for j := -1; j <= 1; j++ {
								if (j != 0 || i != 0) && isAlive(p.imageWidth, x+i, y+j, workerWorld) {
									count++
								}
							}
						}
						if count == 3 || (isAlive(p.imageWidth, x, y, workerWorld) && count == 2) {
							temp[y][x] = 0xFF
						} else {
							temp[y][x] = 0
						}
					}
				}

				tmp := workerWorld
				workerWorld = temp
				temp = tmp
			}

			if val == 0xAB {
				sendWorldFromWorkers(p, workerWorld, out, haloHeight)
			}
			if val == 0xAC {
				sendWorldFromWorkers(p, workerWorld, out, haloHeight)
				break loop
			}
			if val == 0xAD {
			loop1:
				for {
					val := <-in
					if val == 0xAD {
						break loop1
					}
				}
			}
		}
	}
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p golParams, d distributorChans, alive chan []cell, in []chan byte, out []chan byte) {

	// Create the 2D slice to store the world.
	world := getWorldfromPGM(p, d)
	threadHeight := p.imageHeight / p.threads
	extra := p.imageHeight % p.threads
	giveWorldToWorkers(p, world, in, threadHeight, extra)
	ticker := time.NewTicker(2 * time.Second)

loop1:
	for turn := 0; turn < p.turns; turn++ {
		notifyWorkers(p, in, 0xAA)
	loop2:
		for {
			select {
			case keyValue := <-d.key:
				char := string(keyValue)
				if char == "s" {
					fmt.Println("S Pressed")
					go notifyWorkers(p, in, 0xAB)
					world = getWorldFromWorkers(p, world, out, threadHeight, extra)
					go sendWorldtoPGM(p, world, d, turn)
					break loop2
				}
				if char == "q" {
					fmt.Println("Q pressed, breaking from program")
					go notifyWorkers(p, in, 0xAC)
					break loop1
				}
				if char == "p" {
					fmt.Println("P pressed, pausing at turn" + strconv.Itoa(turn))
					go notifyWorkers(p, in, 0xAD)
					for {
						keyValue := <-d.key
						char := string(keyValue)
						if char == "p" {
							fmt.Println("Continuing")
							go notifyWorkers(p, in, 0xAD)
							break loop2
						}
					}
				}
			case <-ticker.C:
				go countAliveCells(p, world, in, out, threadHeight, extra)
				break loop2
			default:
				break loop2
			}

		}

	}
	go notifyWorkers(p, in, 0xAB)
	world = getWorldFromWorkers(p, world, out, threadHeight, extra)
	go sendWorldtoPGM(p, world, d, p.turns)

	// Create an empty slice to store coordinates of cells that are still alive after p.turns are done.
	var finalAlive []cell
	// Go through the world and append the cellsbreak loop that are still alive.
	for y := 0; y < p.imageHeight; y++ {
		for x := 0; x < p.imageWidth; x++ {
			if world[y][x] != 0 {
				finalAlive = append(finalAlive, cell{x: x, y: y})
			}
		}
	}

	// Make sure that the Io has finished any output before exiting.
	d.io.command <- ioCheckIdle
	<-d.io.idle

	// Return the coordinates of cells that are still alive.
	alive <- finalAlive
}
