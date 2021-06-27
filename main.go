package main

import (
	"fmt"
)

var (
	Nodes    = map[int]*Node{}
	sig      = make(chan int)
	MaxLayer = 0
)

func main() {
	go func() {
		for {
			signal := <-sig
			if signal == 1 {
				buildTopo()

				for _, n := range Nodes {
					go n.Run()
				}
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
