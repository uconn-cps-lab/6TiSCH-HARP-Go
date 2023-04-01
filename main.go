package main

import (
	"fmt"
	"sync"
)

const (
	MAX_CHANNEL  = 2
	TRAFFIC_RATE = 3
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
	// var totalCellUtilization float64
	var cnt float64
	// neededCellNumbers := []int{}
	var sum float64 = 0
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
				// var totalNeededCell = 0
				// for _, v := range Nodes {
				// 	if v.ID != 0 {
				// 		// fmt.Println(v.ID, v.Traffic)
				// 		totalNeededCell += v.Traffic
				// 	}
				// }
				// neededCellNumbers = append(neededCellNumbers, totalNeededCell)
				// var totalRequestedCell = 0
				// for _, v := range Nodes[0].SubPartitionAbs {
				// 	area := (v[1] - v[0]) * (v[3] - v[2])
				// 	totalRequestedCell += area
				// 	// fmt.Println(v, area)
				// }
				cnt++
				// cu := float64(totalNeededCell) / float64(totalRequestedCell)
				// fmt.Printf("#%d ", int(cnt))
				// fmt.Printf("Traffic rate: %d", 1)
				// fmt.Printf(", needed cell: %d", totalNeededCell)
				// fmt.Printf(", requested cell: %d", totalRequestedCell)
				var collision = float64(Nodes[0].SubPartitionAbs[1][1]-200) / float64(Nodes[0].SubPartitionAbs[1][0]*MAX_CHANNEL)
				fmt.Println(collision)
				sum += collision
				fmt.Println("avg:", sum/cnt)
				// fmt.Println(Nodes[0].SubPartitionAbs[1][1]-200, Nodes[0].SubPartitionAbs[1][0])
				// fmt.Println(", cell utilization:", cu)
				// totalCellUtilization += cu
				// fmt.Println("total cu:", totalCellUtilization/cnt)
				// fmt.Println(neededCellNumbers)
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
