package main

import (
	"fmt"
	"log"
	"os"
)

type Node struct {
	ID           int            `json:"id"`
	Parent       int            `json:"parent"`
	Children     map[int]*Child `json:"-"`
	Layer        int            `json:"layer"`        // equals to hop count
	Traffic      int            `json:"-"`            // local traffic of each node is 1
	Interface    map[int][]int  `json:"interface"`    // resource interface [slots, channels]
	SubPartition map[int][]int  `json:"subpartition"` // allocated sub-partition [slots start&end, channels start&end]

	receivedInterfaceCnt int

	// internal signal
	sig chan int

	// external message rx
	RXCh   chan Msg    `json:"-"`
	Logger *log.Logger `json:"-"`
}

func NewNode(id, parent, layer int) *Node {
	var traffic = 1
	if id == 0 {
		traffic = 0
	}
	node := &Node{
		ID:           id,
		Parent:       parent,
		Children:     make(map[int]*Child),
		Layer:        layer,
		Traffic:      traffic,
		Interface:    make(map[int][]int),
		SubPartition: make(map[int][]int),
		sig:          make(chan int),
		RXCh:         make(chan Msg, 16),
		Logger:       log.New(os.Stdout, fmt.Sprintf("[+] #%d ", id), 0),
	}
	return node
}

// Child only stores the information of child that parent needs to know
type Child struct {
	ID           int
	Traffic      int
	Interface    map[int][]int
	SubPartition map[int][]int
}

func NewChild(id, traffic int) *Child {
	return &Child{
		ID:           id,
		Traffic:      traffic,
		Interface:    make(map[int][]int),
		SubPartition: make(map[int][]int),
	}
}

func (n *Node) Run() {
	go n.Listen()

	// bottom-up collect resource interfaces
	n.abstractInterface()
	if len(n.Children) == 0 {
		n.reportInterface()
	}
	// wait all children's interfaces
	<-n.sig
	n.compositeInterface()
	n.reportInterface()
	n.Logger.Println("resource interface:", n.Interface)

	// top-down allocate sub-partitions
	if n.ID == 0 {
		n.allocateSubpartition()
	}
	n.Logger.Println(n.SubPartition)
}

func (n *Node) Listen() {
	for {
		msg := <-n.RXCh

		switch msg.Type {
		case MSG_IF:
			n.interfaceMsgHandler(msg)
		case MSG_SP:
			n.subpartitionMsgHandler(msg)
		}
	}
}

func (n *Node) sendTo(dst, msgType int, payload map[int][]int) {
	msg := Msg{n.ID, dst, msgType, payload}

	Nodes[dst].RXCh <- msg
}

func (n *Node) interfaceMsgHandler(msg Msg) {
	// n.Logger.Println("received interface from", msg.Src, msg.Payload)
	n.Children[msg.Src].Interface = msg.Payload
	n.receivedInterfaceCnt++

	if n.receivedInterfaceCnt == len(n.Children) {
		n.sig <- 1
	}
}

func (n *Node) subpartitionMsgHandler(msg Msg) {
	n.Logger.Println("received subpartition from", msg.Src, msg.Payload)

	n.SubPartition = msg.Payload
	n.allocateSubpartition()
}

func (n *Node) abstractInterface() {
	var slots, channels int

	var childrenTraffic = 0
	for _, c := range n.Children {
		childrenTraffic += c.Traffic
	}
	slots = childrenTraffic
	if slots > 0 {
		channels = 1
	}
	n.Interface[n.Layer+1] = []int{slots, channels}
}

func (n *Node) reportInterface() {
	if (n.ID) != 0 {
		n.sendTo(n.Parent, MSG_IF, n.Interface)
	}
}

func (n *Node) compositeInterface() {
	for l := MaxLayer; l > n.Layer+1; l-- {
		var slots, channels int
		for _, c := range n.Children {
			if c.Interface[l] != nil {
				// slot = children's max slot
				if slots < c.Interface[l][0] {
					slots = c.Interface[l][0]
				}
				// channels = sum of children's channels
				channels += c.Interface[l][1]
			}
		}
		if slots == 0 && channels == 0 {
			continue
		}
		n.Interface[l] = []int{slots, channels}
	}
}

func (n *Node) allocateSubpartition() {
	if n.ID == 0 {
		var redundant = 2
		var slotIdx = 0
		for l := MaxLayer; l > 0; l-- {
			n.SubPartition[l] = []int{slotIdx, slotIdx + redundant + n.Interface[l][0], 1, 9}
			slotIdx += redundant + n.Interface[l][0]
		}
	}

	for l := n.Layer + 1; l <= MaxLayer; l++ {
		if n.SubPartition[l] == nil {
			continue
		}
		var chIdx = n.SubPartition[l][2]
		for _, c := range n.Children {
			if c.Interface[l] != nil {
				if c.Interface[l][0] == 0 && c.Interface[l][1] == 0 {
					continue
				}
				c.SubPartition[l] = []int{n.SubPartition[l][1] - c.Interface[l][0], n.SubPartition[l][1],
					chIdx, chIdx + c.Interface[l][1]}
				chIdx += c.Interface[l][1]
			}
		}
	}
	for _, c := range n.Children {
		n.sendTo(c.ID, MSG_SP, c.SubPartition)
	}
}
