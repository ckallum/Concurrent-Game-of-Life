# Game-of-Life-in-Go
------------------

Two implementations of Conway's Game of Life in GoLang using a concurrent approach with the following paradigms.
- Memory Sharing
- Message Passing

The benchmarks/performance differences of both concurrent solutions against a baseline sequential solution can be seen in
`benchmarks.txt`.

-----------

How to Run
------------
To run the program you will need to have Go installed in version 1.12.9 or higher and correct configurations in IntelliJ/your preferred IDE.

You will also need to add the following lines to your .profile file by entering `nano ~/.profile` in the command line:
 
 
 `
  export PATH=$PATH:/us/local/go/bin`
  
  `export GOPATH=$HOME/go`
  
  `export PATH=$PATH:$GOPATH/bin`
  


Save the file and enter `source ~/.profile` in the command line.

--------------

##### `Makefile` commands:

`make gol`: runs the program. During the running of the program you can pause, save an image of the current state and quit the program using the following terminal inputs:
- `ctrl-p`
- `ctrl-s`
- `ctrl-q`


`make test`: runs the the program against the test file to check for correct logic.


`make bench`: runs all the tests before benchmarking. Then outputs the comparisson of CPU usage and runtime against the sequential solution.


You can view the input and output images in the `images` folder.
