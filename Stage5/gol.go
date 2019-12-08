package main

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
)

func buildWorld(p golParams, height int) [][]byte {
	workerWorld := make([][]byte, height)
	for i := range workerWorld {
		workerWorld[i] = make([]byte, p.imageWidth)
	}
	return workerWorld
}

func countAliveCells(p golParams, world [][]byte) {
	alive := 0
	for y := 0; y < p.imageWidth; y++ {
		for x := 0; x < p.imageWidth; x++ {
			if world[y][x] == 0xFF {
				alive++
			}
		}
	}
	fmt.Println(alive)
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

func worker(p golParams, world [][]byte, tempWorld [][]byte, threadNum int, threadHeight int, extra int, isEven bool, wg *sync.WaitGroup) {
	//GOL Logic
	//visualiseMatrix(world, p.imageWidth, p.imageHeight)
	yBound := (threadNum + 1) * threadHeight
	if isEven {
		yBound += extra
	}
	for y := threadNum * threadHeight; y < yBound; y++ {
		for x := 0; x < p.imageWidth; x++ {
			xRight, xLeft := x+1, x-1
			yUp, yDown := y-1, y+1
			if xRight >= p.imageWidth {
				xRight %= p.imageWidth
			}
			if xLeft < 0 {
				xLeft += p.imageWidth
			}
			if yDown >= p.imageHeight {
				yDown %= p.imageHeight
			}
			if yUp < 0 {
				yUp += p.imageHeight
			}
			count := 0
			count = int(world[yUp][xLeft]) +
				int(world[yUp][x]) +
				int(world[yUp][xRight]) +
				int(world[y][xLeft]) +
				int(world[y][xRight]) +
				int(world[yDown][xLeft]) +
				int(world[yDown][x]) +
				int(world[yDown][xRight])
			count /= 255
			if count == 3 || (world[y][x] == 0xFF && count == 2) {
				tempWorld[y][x] = 0xFF
			} else {
				tempWorld[y][x] = 0
			}
		}
	}

	wg.Done()
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p golParams, d distributorChans, alive chan []cell, threadHeight int) {

	// Create the 2D slice to store the world.
	world := getWorldFromPGM(p, d)
	tempWorld := buildWorld(p, p.imageHeight)
	isEven := !powerOfTwo(p)
	extra := p.imageHeight % p.threads

	for turn := 0; turn < p.turns; turn++ {

		var waitGroup = &sync.WaitGroup{}
		waitGroup.Add(p.threads)
		for i := 0; i < p.threads; i++ {
			if i == p.threads-1 && isEven {
				go worker(p, world, tempWorld, i, threadHeight, extra, true, waitGroup)
			} else {
				go worker(p, world, tempWorld, i, threadHeight, extra, false, waitGroup)
			}
		}
		waitGroup.Wait()
		tmp := world
		world = tempWorld
		tempWorld = tmp

	}

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
