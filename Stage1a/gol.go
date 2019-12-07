package main

import (
	"fmt"
	"strconv"
	"strings"
)


func isAlive(p golParams, x, y int, temp [][]byte) bool{
	x+= p.imageWidth
	x %= p.imageWidth
	y+=p.imageHeight
	y%=p.imageHeight
	//fmt.Println(x, y)
	if temp[y][x] == 0{
		return false
	}else{
		return true
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
	temp:= make([][]byte, p.imageHeight)
	for i := range world{
		temp[i] = make([]byte, p.imageWidth)
		copy(temp[i], world[i])

	}

	// Calculate the new state of Game of Life after the given number of turns.
	for turns := 0; turns < p.turns; turns++ {
		for y := 0; y < p.imageHeight; y++ {
			for x := 0; x < p.imageWidth; x++ {
				xRight, xLeft := x+1, x-1
                			yUp, yDown:= y-1, y+1
                            						if xRight >= p.imageWidth {
                            							xRight %= p.imageWidth
                            						}
                            						if xLeft < 0 {
                            							xLeft += p.imageWidth
                            						}
                            						if yDown >= p.imageHeight {
                                                                							yDown %= p.imageHeight
                                                                						}
                                                                						if yUp< 0 {
                                                                							yUp+= p.imageHeight
                                                                						}
                            						count := 0
                            						count = int(temp[yUp][xLeft]) +
                            								int(temp[yUp][x]) +
                            								int(temp[yUp][xRight]) +
                            								int(temp[y][xLeft]) +
                            								int(temp[y][xRight]) +
                            								int(temp[yDown][xLeft]) +
                            								int(temp[yDown][x]) +
                            								int(temp[yDown][xRight])
                            						count /= 255
				if count == 3 || ( temp[y][x] == 0xFF && count == 2){
					world[y][x] = 0xFF
				}else{
					world[y][x] = 0
				}


				// for all neighbours, if number of 0xFF < 2 die, if number of 0xFF>3 die, if dead and number of 0xFF == 3 alive
				//neighbours := [8]{(x, y-1), (x, y+1), (x+1,y), (x-1, y), (x+1, y+1)(x+1, y-1), (x-1,y+1), (x-1, y-1)}
				// Placeholder for the actual Game of Life logic: flips alive cells to dead and dead cells to alive.

			}
		}
		for i := range world{
			copy(temp[i], world[i])
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

	// Make sure that the Io has finished any output before exiting.
	d.io.command <- ioCheckIdle
	<-d.io.idle

	// Return the coordinates of cells that are still alive.
	alive <- finalAlive
}