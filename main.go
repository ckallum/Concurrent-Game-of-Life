package main

import (
	"flag"
	"time"
)

// golParams provides the details of how to run the Game of Life and which image to load.
type golParams struct {
	turns       int
	threads     int
	imageWidth  int
	imageHeight int
}

// ioCommand allows requesting behaviour from the io (pgm) goroutine.
type ioCommand uint8

// This is a way of creating enums in Go.
// It will evaluate to:
//		ioOutput 	= 0
//		ioInput 	= 1
//		ioCheckIdle = 2
const (
	ioOutput ioCommand = iota
	ioInput
	ioCheckIdle
)

// cell is used as the return type for the testing framework.
type cell struct {
	x, y int
}

// distributorToIo defines all chans that the distributor goroutine will have to communicate with the io goroutine.
// Note the restrictions on chans being send-only or receive-only to prevent bugs.
type distributorToIo struct {
	command chan<- ioCommand
	idle    <-chan bool

	filename  chan<- string
	inputVal  <-chan uint8
	outputVal chan<- uint8
}

// ioToDistributor defines all chans that the io goroutine will have to communicate with the distributor goroutine.
// Note the restrictions on chans being send-only or receive-only to prevent bugs.
type ioToDistributor struct {
	command <-chan ioCommand
	idle    chan<- bool

	filename  <-chan string
	inputVal  chan<- uint8
	outputVal <-chan uint8
}

// distributorChans stores all the chans that the distributor goroutine will use.
type distributorChans struct {
	io distributorToIo
	key <- chan rune
}

// ioChans stores all the chans that the io goroutine will use.
type ioChans struct {
	distributor ioToDistributor
}

func powerOfTwo(p golParams) bool{
	return (p.threads & (p.threads-1)) == 0
}

// gameOfLife is the function called by the testing framework.
// It makes some channels and starts relevant goroutines.
// It places the created channels in the relevant structs.
// It returns an array of alive cells returned by the distributor.
func gameOfLife(p golParams, keyChan <-chan rune) []cell {
	var dChans distributorChans
	var ioChans ioChans
	dChans.key = keyChan

	ioCommand := make(chan ioCommand)
	dChans.io.command = ioCommand
	ioChans.distributor.command = ioCommand

	ioIdle := make(chan bool)
	dChans.io.idle = ioIdle
	ioChans.distributor.idle = ioIdle

	ioFilename := make(chan string)
	dChans.io.filename = ioFilename
	ioChans.distributor.filename = ioFilename

	inputVal := make(chan uint8)
	dChans.io.inputVal = inputVal
	ioChans.distributor.inputVal = inputVal

	outputVal := make(chan uint8)
	dChans.io.outputVal = outputVal
	ioChans.distributor.outputVal = outputVal
	ticker := time.NewTicker(2 * time.Second)


	//creating worker channels and running them concurrently -> keeping them PERSISTENT
	threadHeight := p.imageHeight/p.threads
	in := make([]chan byte, p.threads)
	out := make([] chan byte, p.threads)
	haloChannels:= make([][]chan byte, p.threads)
	for i := 0; i<p.threads; i++{
		haloChannels[i] = make([]chan byte, 2)
		for j:= 0; j<2 ;j++{
			haloChannels[i][j] = make(chan byte, p.imageHeight)
		}
		in[i] = make(chan byte, p.imageHeight)
		out[i] = make(chan byte, p.imageHeight)
	}

	if powerOfTwo(p) {
		for i := 0; i < p.threads; i++ {
			receiving := [2]chan byte{haloChannels[(i-1+p.threads)%p.threads][1], haloChannels[(i+1)%p.threads][0]}
			go worker(threadHeight+2, in[i], out[i], p, haloChannels[i], receiving)
		}

	}else{
		extra := p.imageHeight % p.threads
		for i := 0; i< p.threads-1; i++{
			receiving := [2]chan byte{haloChannels[(i-1+p.threads) % p.threads][1], haloChannels[(i+1) % p.threads][0]}
			go worker(threadHeight+2, in[i], out[i], p, haloChannels[i], receiving)
		}
		receiving := [2]chan byte{haloChannels[p.threads-2][1], haloChannels[0][0]}
		go worker(threadHeight+2+extra, in[p.threads-1], out[p.threads-1], p, haloChannels[p.threads-1], receiving)
	}

	aliveCells := make(chan []cell)

	go distributor(p, dChans, aliveCells, in, out, ticker.C)
	go pgmIo(p, ioChans)

	alive := <-aliveCells
	return alive
}

// main is the function called when starting Game of Life with 'make gol'
// Do not edit until Stage 2.
func main() {
	var params golParams

	flag.IntVar(
		&params.threads,
		"t",
		8,
		"Specify the number of worker threads to use. Defaults to 8.")

	flag.IntVar(
		&params.imageWidth,
		"w",
		512,
		"Specify the width of the image. Defaults to 512.")

	flag.IntVar(
		&params.imageHeight,
		"h",
		512,
		"Specify the height of the image. Defaults to 512.")

	flag.Parse()

	params.turns = 100

	startControlServer(params)
	keyChan := make(chan rune)
	go getKeyboardCommand(keyChan)
	gameOfLife(params, keyChan)
	StopControlServer()
}
