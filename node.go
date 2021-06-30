package main

import (
	"fmt"
	"log"
	"os"
	"sort"
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
		RXCh:         make(chan Msg, 8),
		Logger:       log.New(os.Stdout, fmt.Sprintf("[+] #%d ", id), 0),
	}
	return node
}

// Child only stores the information of child that parent needs to know
type Child struct {
	ID                 int
	Traffic            int
	Interface          map[int][]int
	SubPartitionOffset map[int][]int // output of interface composition, logical location; right->left, bottom->top
	SubPartition       map[int][]int // output of sub-partition allocation, physical location
}

func NewChild(id, traffic int) *Child {
	return &Child{
		ID:                 id,
		Traffic:            traffic,
		Interface:          make(map[int][]int),
		SubPartitionOffset: make(map[int][]int),
		SubPartition:       make(map[int][]int),
	}
}

func (n *Node) Run(blocker chan bool) {
	defer func() { <-blocker }()
	go n.Listen()

	// bottom-up collect resource interfaces
	n.abstractInterface()
	if len(n.Children) == 0 {
		n.reportInterface()
	}
	// wait all children's interfaces
	if len(n.Children) > 0 {
		<-n.sig
		n.compositeInterface()
		n.reportInterface()
		n.Logger.Println("resource interface:", n.Interface)

		// top-down allocate sub-partitions
		if n.ID == 0 {
			n.allocateSubpartition()
		}
		n.Logger.Println("sub-partition:", n.SubPartition)
	}

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
	// n.Logger.Println("received subpartition from", msg.Src, msg.Payload)

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

// Compute the composited interface size and each children's sub-partition offset
// Objective: minimize the composited size
// A strip packing problem or rectangle packing problem
// https://en.wikipedia.org/wiki/Strip_packing_problem
// https://en.wikipedia.org/wiki/Rectangle_packing#Packing_different_rectangles_in_a_minimum-area_rectangle
func (n *Node) compositeInterface() {
	n.packingGreedyChannel()
	// n.packingFFDH()
}

func (n *Node) packingGreedyChannel() {
	for l := MaxLayer; l > n.Layer+1; l-- {
		var slots = 0
		var channels = 0

		// sort children by slot range, decreasing
		var childrenSlice = []*Child{}
		for _, c := range n.Children {
			if c.Interface[l] != nil {
				if c.Interface[l][0] != 0 {
					childrenSlice = append(childrenSlice, c)
				}
			}
		}
		sort.SliceStable(childrenSlice, func(i, j int) bool {
			return childrenSlice[i].Interface[l][0] > childrenSlice[j].Interface[l][0]
		})

		for i, c := range childrenSlice {
			// slots = children's max slot
			if i == 0 {
				slots = c.Interface[l][0]
			}

			c.SubPartitionOffset[l] = []int{c.Interface[l][0], 0, channels, channels + c.Interface[l][1]}

			// channels = sum of children's channels
			channels += c.Interface[l][1]
		}

		if slots == 0 && channels == 0 {
			continue
		}
		n.Interface[l] = []int{slots, channels}
	}
}

// for level based strip packing methods
type level struct {
	idleSlots    int // remaining width
	slotEdge     int
	height       int
	channelStart int
	channelEnd   int
}

// First-Fit Decreasing Height for strip packing, with level concept
func (n *Node) packingFFDH() {
	for l := MaxLayer; l > n.Layer+1; l-- {
		var slots = 0
		var channels = 0

		// sort children by slot range, decreasing
		var childrenSlice = []*Child{}
		for _, c := range n.Children {
			if c.Interface[l] != nil {
				if c.Interface[l][0] != 0 {
					childrenSlice = append(childrenSlice, c)
				}
			}
		}
		if len(childrenSlice) == 0 {
			continue
		}
		sort.SliceStable(childrenSlice, func(i, j int) bool {
			return childrenSlice[i].Interface[l][0] > childrenSlice[j].Interface[l][0]
		})

		// find the children with longest slot range, and place it at the bottom, as the width bound
		var child = childrenSlice[0]
		child.SubPartitionOffset[l] = []int{child.Interface[l][0], 0, channels, channels + child.Interface[l][1]}
		slots = child.Interface[l][0]
		channels += child.Interface[l][1]
		if len(childrenSlice) == 1 {
			n.Interface[l] = []int{slots, channels}
			continue
		}

		// sort other children by height (channel range), then run FFDH
		childrenSlice = childrenSlice[1:]
		sort.SliceStable(childrenSlice, func(i, j int) bool {
			return childrenSlice[i].Interface[l][1] > childrenSlice[j].Interface[l][1]
		})

		// idle slots,  and channel start of each level
		levels := make(map[int]*level)
		for i, c := range childrenSlice {
			if i == 0 {
				c.SubPartitionOffset[l] = []int{c.Interface[l][0], 0, channels, channels + c.Interface[l][1]}
				levels[0] = &level{
					idleSlots:    slots - c.Interface[l][0],
					slotEdge:     c.Interface[l][0],
					height:       c.Interface[l][1],
					channelStart: channels,
					channelEnd:   channels + c.Interface[l][1],
				}
				channels += c.Interface[l][1]
			} else {
				var found = false
				for lv := 0; lv < len(levels); lv++ {
					v := levels[lv]
					if v.idleSlots >= c.Interface[l][0] {
						c.SubPartitionOffset[l] = []int{v.slotEdge + c.Interface[l][0], v.slotEdge, v.channelStart, v.channelStart + c.Interface[l][1]}
						v.slotEdge += c.Interface[l][0]
						v.idleSlots -= c.Interface[l][0]
						found = true
						break
					}
				}
				if !found { // create a new level
					var h = levels[len(levels)-1].channelEnd

					c.SubPartitionOffset[l] = []int{c.Interface[l][0], 0, h, h + c.Interface[l][1]}
					levels[len(levels)] = &level{
						idleSlots:    slots - c.Interface[l][0],
						slotEdge:     c.Interface[l][0],
						height:       c.Interface[l][1],
						channelStart: h,
						channelEnd:   h + c.Interface[l][1],
					}
					channels += c.Interface[l][1]
				}
			}
		}
		n.Interface[l] = []int{slots, channels}
	}
}

func (n *Node) allocateSubpartition() {
	if n.ID == 0 {
		var redundant = 5
		var slotIdx = 0
		for l := MaxLayer; l > 0; l-- {
			if n.Interface[l] == nil {
				continue
			}
			n.SubPartition[l] = []int{slotIdx, slotIdx + redundant + n.Interface[l][0], 1, 17}
			slotIdx += redundant + n.Interface[l][0]
		}
	}

	for l := n.Layer + 1; l <= MaxLayer; l++ {
		if n.SubPartition[l] == nil {
			continue
		}

		for _, c := range n.Children {
			if c.SubPartitionOffset[l] != nil {
				c.SubPartition[l] = []int{
					n.SubPartition[l][1] - c.SubPartitionOffset[l][0],
					n.SubPartition[l][1] - c.SubPartitionOffset[l][1],
					n.SubPartition[l][3] - c.SubPartitionOffset[l][3],
					n.SubPartition[l][3] - c.SubPartitionOffset[l][2],
				}
			}
		}
	}
	for _, c := range n.Children {
		if len(c.SubPartition) != 0 {
			n.sendTo(c.ID, MSG_SP, c.SubPartition)
		}
	}
}
