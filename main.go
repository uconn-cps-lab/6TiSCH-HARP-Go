package main

import (
	"fmt"
	"sync"
)

const (
	MAX_CHANNEL = 10
)

var (
	Nodes         map[int]*Node
	sig1          = make(chan int)
	sig2          = make(chan int)
	MaxLayer      = 0
	wsLogger      chan wsLog
	mutex         sync.Mutex
	adjMsgCnt     = 0
	affectedNodes = map[int]bool{}
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

				wsLogger <- wsLog{
					WS_LOG_MSG,
					"HP finished",
					nil,
				}

				sig2 <- 2
				// time.Sleep(3 * time.Second)
				// Nodes[52].updateInterface(4, []int{3, 1})
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

	fmt.Printf("%d-hop %d-nodes network up\n", MaxLayer, len(Nodes))
	wsLogger <- wsLog{
		WS_LOG_MSG,
		fmt.Sprintf("%d-hop %d-nodes network up", MaxLayer, len(Nodes)),
		nil,
	}
	for l := MaxLayer; l > 0; l-- {
		for _, nn := range Nodes {
			if nn.Layer == l {
				Nodes[nn.Parent].Children[nn.ID] = NewChild(nn.ID, nn.Traffic)
				Nodes[nn.Parent].Traffic += nn.Traffic
			}
		}
	}
}
