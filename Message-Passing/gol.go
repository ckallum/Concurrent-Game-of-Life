package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func buildWorld(p golParams, height int) [][]byte {
	workerWorld := make([][]byte, height)
	for i := range workerWorld {
		workerWorld[i] = make([]byte, p.imageWidth)
	}
	return workerWorld
}

func countAliveCells(p golParams, world [][]byte, workerHeight int) int {
	alive := 0
	for y := 1; y < workerHeight-1; y++ {
		for x := 0; x < p.imageWidth; x++ {
			if world[y][x] == 0xFF {
				alive++
			}
		}
	}
	return alive
}

func totalAliveCells(p golParams, keyChannels []chan int, out []chan byte, threadHeight int, extra int) {
	notifyWorkers(p, keyChannels, 5)
	alive := 0
	for i := 0; i < p.threads; i++ {
		alive += <-keyChannels[i]
	}
	fmt.Println("Number of Alive Cells:", alive)
}

func sendWorldToPGM(p golParams, world [][] byte, d distributorChans, turn int) {
	d.io.command <- ioOutput
	d.io.filename <- strings.Join([]string{strconv.Itoa(p.imageWidth), strconv.Itoa(p.imageHeight), "Turn:" + strconv.Itoa(turn)}, "x")
	for y := range world {
		for x := range world[y] {
			d.io.outputVal <- world[y][x]
		}
	}
}

func getWorldFromPGM(p golParams, d distributorChans) [][]byte {
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

func giveWorldToWorkers(p golParams, world [][]byte, in []chan byte, threadHeight int, extra int, isP bool) {
	for i := 0; i < p.threads; i++ {
		go sendToWorker(p, world, threadHeight, i, in[i], isP, extra)
	}
}

func sendWorldFromWorker(p golParams, world [][]byte, out chan<- byte, height int) {
	for y := 1; y < height-1; y++ {
		for x := 0; x < p.imageWidth; x++ {
			out <- world[y][x]
		}
	}
}


func notifyWorkers(p golParams, keyChannels []chan int, key int) {
	for i := 0; i < p.threads; i++ {
		keyChannels[i] <- key
	}
}

func combineWorkers(p golParams, out []chan byte, threadHeight int, isP bool, extra int) [][]byte {
	world := buildWorld(p, p.imageHeight)
	for i := 0; i < p.threads; i++ {
		for y := 0; y < threadHeight; y++ {
			for x := 0; x < p.imageWidth; x++ {
				world[y+(i*(threadHeight))][x] = <-out[i]
			}
		}
	}
	if !isP {
		for e := 0; e < extra; e++ {
			for x := 0; x < p.imageWidth; x++ {
				world[e+(p.threads*(threadHeight))][x] = <-out[p.threads-1]
			}
		}
	}
	return world
}

func sendToWorker(p golParams, world [][]byte, threadHeight int, i int, in chan<- byte, isP bool, extra int) {
	yBound := threadHeight + 2
	if i == p.threads-1 && !isP {
		yBound += extra
	}
	for y := 0; y < yBound; y++ {
		proposedY := y + (i * threadHeight) - 1
		if proposedY < 0 {
			proposedY += p.imageHeight
		}
		proposedY %= p.imageHeight
		for x := 0; x < p.imageWidth; x++ {
			in <- world[proposedY][x]
		}
	}
}


func worker(haloHeight int, in <-chan byte, out chan<- byte, p golParams, sending []chan byte, receiving [2]chan byte, keyChannel chan int) {
	workerWorld := buildWorld(p, haloHeight)
	for y := 0; y < haloHeight; y++ {
		for x := 0; x < p.imageWidth; x++ {
			workerWorld[y][x] = <-in
		}
	}
	temp := buildWorld(p, haloHeight)
	running := true
	for running == true {
		select {
		case val := <-keyChannel:
			//fmt.Println(val)
			if val == 1 {
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
						xRight := x + 1
						xLeft := x - 1

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

			if val == 2 {
				go sendWorldFromWorker(p, workerWorld, out, haloHeight)
			}
			if val == 3 {
				running = false
				go sendWorldFromWorker(p, workerWorld, out, haloHeight)
			}
			if val == 4 {
				paused := true
				for paused == true {
					val := <-keyChannel
					if val == 4 {
						paused = false
					}
				}
			}
			if val == 5 {
				keyChannel <- countAliveCells(p, workerWorld, haloHeight)
			}

		}
	}
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p golParams, d distributorChans, alive chan []cell, in []chan byte, out []chan byte, keyChannels []chan int, threadHeight int) {

	// Create the 2D slice to store the world.
	world := getWorldFromPGM(p, d)
	extra := p.imageHeight % p.threads
	giveWorldToWorkers(p, world, in, threadHeight, extra, powerOfTwo(p))
	ticker := time.NewTicker(2 * time.Second)
	running := true
	isP := powerOfTwo(p)

	for turn := 0; turn < p.turns && running == true; turn++ {
		select {
		case keyValue := <-d.key:
			char := string(keyValue)
			if char == "s" {
				fmt.Println("S Pressed")
				go notifyWorkers(p, keyChannels, 2)
				w := combineWorkers(p, out, threadHeight,isP, extra)
				go sendWorldToPGM(p, w, d, turn)
			}
			if char == "q" {
				fmt.Println("Q pressed, breaking from program")
				running = false
			}
			if char == "p" {
				fmt.Println("P pressed, pausing at turn: " + strconv.Itoa(turn))
				go notifyWorkers(p, keyChannels, 4)
				paused := true
				for paused == true {
					char := string(<-d.key)
					if char == "p" {
						fmt.Println("Continuing")
						paused = false
						notifyWorkers(p, keyChannels, 4)
					}
				}
			}
		case <-ticker.C:
			totalAliveCells(p, keyChannels, out, threadHeight, extra)
		default:
			notifyWorkers(p, keyChannels, 1)
		}
	}

	go notifyWorkers(p, keyChannels, 3)
	world = combineWorkers(p, out, threadHeight,isP, extra)
	go sendWorldToPGM(p, world, d, p.turns)

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
