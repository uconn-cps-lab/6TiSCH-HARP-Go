package main

import (
	"fmt"
)

var (
	Nodes    map[int]*Node
	sig1     = make(chan int)
	sig2     = make(chan int)
	MaxLayer = 0
)

func main() {
	go func() {
		for {
			signal := <-sig1
			if signal == 1 {
				buildTopo()
				blockers := make(chan bool, len(Nodes))
				for _, n := range Nodes {
					blockers <- true
					go n.Run(blockers)
				}
				for i := 0; i < cap(blockers); i++ {
					blockers <- true
				}
				fmt.Println("HP finished")
				sig2 <- 2
			}

		}
	}()
	runHTTPServer()
}

func buildTopo() {

	for _, n := range Nodes {
		if n.Layer > MaxLayer {
			MaxLayer = n.Layer
		}
	}

	fmt.Printf("%d-hop %d-nodes network starts\n", MaxLayer, len(Nodes))

	for l := MaxLayer; l > 0; l-- {
		for _, nn := range Nodes {
			if nn.Layer == l {
				Nodes[nn.Parent].Children[nn.ID] = NewChild(nn.ID, nn.Traffic)
				Nodes[nn.Parent].Traffic += nn.Traffic
			}
		}
	}
}
