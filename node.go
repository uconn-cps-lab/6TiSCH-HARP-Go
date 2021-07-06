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

	maxChannel           int `json:"-"`
	receivedInterfaceCnt int

	// internal signal
	sigWaitChildrenInterfaces     chan int
	sigWaitAllocatedSubpartition  chan int
	sigWaitSubpartitionAdjustment chan int

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
		ID:                            id,
		Parent:                        parent,
		Children:                      make(map[int]*Child),
		Layer:                         layer,
		Traffic:                       traffic,
		Interface:                     make(map[int][]int),
		SubPartition:                  make(map[int][]int),
		maxChannel:                    MAX_CHANNEL,
		sigWaitChildrenInterfaces:     make(chan int),
		sigWaitAllocatedSubpartition:  make(chan int),
		sigWaitSubpartitionAdjustment: make(chan int),
		RXCh:                          make(chan Msg, 64),
		Logger:                        log.New(os.Stdout, fmt.Sprintf("[#%d] ", id), 0),
	}
	return node
}

// Child only stores the information of child that parent needs to know
type Child struct {
	ID                 int
	Traffic            int
	Interface          map[int][]int
	SubPartitionOffset map[int][]int // output of interface composition, logical location; left->right, bottom->top
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
	defer func() {
		<-blocker
		// if len(n.Children) > 0 {
		// 	n.Logger.Printf("resource interface: %v\n\t sub-partition: %v", n.Interface, n.SubPartition)
		// }
	}()
	go n.listen()

	// bottom-up collect resource interfaces
	n.abstractInterface()
	if len(n.Children) == 0 {
		n.reportInterface()
	}
	// wait all children's interfaces
	if len(n.Children) > 0 {
		<-n.sigWaitChildrenInterfaces
		close(n.sigWaitChildrenInterfaces)
		n.compositeInterface(-1)
		n.reportInterface()

		// top-down allocate sub-partitions
		if n.ID == 0 {
			n.allocateSubpartition()
		} else {
			<-n.sigWaitAllocatedSubpartition
			close(n.sigWaitAllocatedSubpartition)
			// allocate sub-partitions for children
			n.allocateSubpartition()
		}
	}
}

func (n *Node) listen() {
	for {
		msg := <-n.RXCh

		switch msg.Type {
		case MSG_IF:
			go n.interfaceMsgHandler(msg)
		case MSG_SP:
			go n.subpartitionMsgHandler(msg)
		case MSG_IF_UPDATE:
			go n.interfaceUpdateMsgHandler(msg)
		case MSG_SP_UPDATE:
			go n.subpartitionUpdateMsgHandler(msg)
		}
	}
}

func (n *Node) sendTo(dst, msgType int, payload interface{}) {
	msg := Msg{n.ID, dst, msgType, payload}
	Nodes[dst].RXCh <- msg
	if msgType == MSG_SP_UPDATE || msgType == MSG_IF_UPDATE {
		if _, ok := affectedNodes[n.ID]; !ok {
			// hasn't been upload
			if !affectedNodes[n.ID] {
				affectedNodes[n.ID] = true
				wsLogger <- wsLog{
					WS_LOG_AFFECTED_NODES,
					"",
					[]int{n.ID}, // type, src, dst
				}
			}
		}

		if _, ok := affectedNodes[dst]; !ok {
			// hasn't been upload
			if !affectedNodes[dst] {
				affectedNodes[dst] = true
				wsLogger <- wsLog{
					WS_LOG_AFFECTED_NODES,
					"",
					[]int{dst}, // type, src, dst
				}
			}
		}

		wsLogger <- wsLog{
			WS_LOG_FLOW,
			"",
			[]int{msgType, n.ID, dst, payload.([]int)[0]}, // type, src, dst, layer
		}
	}
	// n.Logger.Printf("sent %d msg to %d, payload: %v\n", msgType, dst, payload)
}

func (n *Node) interfaceMsgHandler(msg Msg) {
	// n.Logger.Println("received interface from", msg.Src, msg.Payload)
	n.Children[msg.Src].Interface = msg.Payload.(map[int][]int)
	n.receivedInterfaceCnt++

	if n.receivedInterfaceCnt == len(n.Children) {
		n.sigWaitChildrenInterfaces <- 1
	}
}

func (n *Node) subpartitionMsgHandler(msg Msg) {
	// n.Logger.Println("received subpartition from", msg.Src, msg.Payload)
	n.SubPartition = msg.Payload.(map[int][]int)
	n.sigWaitAllocatedSubpartition <- 1
}

func (n *Node) interfaceUpdateMsgHandler(msg Msg) {
	layer := msg.Payload.([]int)[0]
	// n.Logger.Printf("received SP_ADJ_REQ @ L%d from #%d", layer, msg.Src)
	// wsLogger <- fmt.Sprintf("#%d received SP_ADJ_REQ @ L%d from #%d", n.ID, layer, msg.Src)
	n.Children[msg.Src].Interface[layer] = msg.Payload.([]int)[1:]

	n.compositeInterface(layer)
	// n.Logger.Printf("recomputes IF composition and SPs arrangement @ L%d\n", layer)
	// wsLogger <- fmt.Sprintf("#%d recomputes IF composition and SPs arrangement @ L%d", n.ID, layer)

	if n.Interface[layer][0] > n.SubPartition[layer][1]-n.SubPartition[layer][0] ||
		n.Interface[layer][1] > n.SubPartition[layer][3]-n.SubPartition[layer][2] {
		n.sendTo(n.Parent, MSG_IF_UPDATE, append([]int{layer}, n.Interface[layer]...))
		adjMsgCnt++
		n.Logger.Printf("SP @ L%d cannot satisfy new IF composition, send SP_ADJ_REQ to #%d\n", layer, n.Parent)
		wsLogger <- wsLog{
			WS_LOG_MSG,
			fmt.Sprintf("%d) #%d SP @ L%d cannot satisfy new IF composition, send SP_ADJ_REQ to #%d", adjMsgCnt, n.ID, layer, n.Parent),
			nil,
		}
	} else {
		n.adjustSubpartition(layer)
	}
}

func (n *Node) subpartitionUpdateMsgHandler(msg Msg) {
	var layer = msg.Payload.([]int)[0]
	// n.Logger.Printf("received SP_UPDATE @ L%d from #%d", layer, msg.Src)
	// wsLogger <- fmt.Sprintf("#%d received SP_UPDATE @ L%d from #%d", n.ID, layer, msg.Src)
	n.SubPartition[layer] = msg.Payload.([]int)[1:]

	n.adjustSubpartition(layer)
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
func (n *Node) compositeInterface(layer int) {
	if layer == -1 {
		for l := MaxLayer; l > n.Layer+1; l-- {
			// n.packingGreedyChannel(l)
			// n.packingFFDH(l)
			n.packingBestFitSkyline(l)
		}
	} else {
		n.packingBestFitSkyline(layer)
	}
}

func (n *Node) packingGreedyChannel(layer int) {

	var slots = 0
	var channels = 0

	// sort children by slot range, decreasing
	var childrenSlice = []*Child{}
	for _, c := range n.Children {
		if c.Interface[layer] != nil {
			if c.Interface[layer][0] != 0 {
				childrenSlice = append(childrenSlice, c)
			}
		}
	}
	sort.SliceStable(childrenSlice, func(i, j int) bool {
		return childrenSlice[i].Interface[layer][0] > childrenSlice[j].Interface[layer][0]
	})

	for i, c := range childrenSlice {
		// slots = children's max slot
		if i == 0 {
			slots = c.Interface[layer][0]
		}

		c.SubPartitionOffset[layer] = []int{0, c.Interface[layer][0], channels, channels + c.Interface[layer][1]}

		// channels = sum of children's channels
		channels += c.Interface[layer][1]
	}

	if slots == 0 && channels == 0 {
		return
	}
	n.Interface[layer] = []int{slots, channels}

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
// Coffman, Jr, Edward G., et al. "Performance bounds for level-oriented two-dimensional packing algorithms." SIAM Journal on Computing 9.4 (1980): 808-826.
func (n *Node) packingFFDH(layer int) {

	var slots = 0
	var channels = 0

	// sort children by slot range, decreasing
	var childrenSlice = []*Child{}
	for _, c := range n.Children {
		if c.Interface[layer] != nil {
			if c.Interface[layer][0] != 0 {
				childrenSlice = append(childrenSlice, c)
			}
		}
	}
	if len(childrenSlice) == 0 {
		return
	}
	sort.SliceStable(childrenSlice, func(i, j int) bool {
		return childrenSlice[i].Interface[layer][0] > childrenSlice[j].Interface[layer][0]
	})

	// find the children with longest slot range, and place it at the bottom, as the width bound
	var child = childrenSlice[0]
	child.SubPartitionOffset[layer] = []int{0, child.Interface[layer][0], channels, channels + child.Interface[layer][1]}
	slots = child.Interface[layer][0]
	channels += child.Interface[layer][1]
	if len(childrenSlice) == 1 {
		n.Interface[layer] = []int{slots, channels}
		return
	}

	// sort other children by height (channel range), then run FFDH
	childrenSlice = childrenSlice[1:]
	sort.SliceStable(childrenSlice, func(i, j int) bool {
		return childrenSlice[i].Interface[layer][1] > childrenSlice[j].Interface[layer][1]
	})

	// idle slots,  and channel start of each level
	levels := make(map[int]*level)
	for i, c := range childrenSlice {
		if i == 0 {
			c.SubPartitionOffset[layer] = []int{0, c.Interface[layer][0], channels, channels + c.Interface[layer][1]}
			levels[0] = &level{
				idleSlots:    slots - c.Interface[layer][0],
				slotEdge:     c.Interface[layer][0],
				height:       c.Interface[layer][1],
				channelStart: channels,
				channelEnd:   channels + c.Interface[layer][1],
			}
			channels += c.Interface[layer][1]
		} else {
			var found = false
			for lv := 0; lv < len(levels); lv++ {
				v := levels[lv]
				if v.idleSlots >= c.Interface[layer][0] {
					c.SubPartitionOffset[layer] = []int{v.slotEdge, v.slotEdge + c.Interface[layer][0], v.channelStart, v.channelStart + c.Interface[layer][1]}
					v.slotEdge += c.Interface[layer][0]
					v.idleSlots -= c.Interface[layer][0]
					found = true
					break
				}
			}
			if !found { // create a new level
				var h = levels[len(levels)-1].channelEnd

				c.SubPartitionOffset[layer] = []int{0, c.Interface[layer][0], h, h + c.Interface[layer][1]}
				levels[len(levels)] = &level{
					idleSlots:    slots - c.Interface[layer][0],
					slotEdge:     c.Interface[layer][0],
					height:       c.Interface[layer][1],
					channelStart: h,
					channelEnd:   h + c.Interface[layer][1],
				}
				channels += c.Interface[layer][1]
			}
		}
	}
	n.Interface[layer] = []int{slots, channels}

}

type skyline struct {
	start  int
	end    int
	width  int
	height int
}

// Best-Fit skyline strip packing
// Burke, Edmund K., Graham Kendall, and Glenn Whitwell. "A new placement heuristic for the orthogonal stock-cutting problem." Operations Research 52.4 (2004): 655-671.
func (n *Node) packingBestFitSkyline(layer int) {
	var slots = 0
	var channels = 0

	// sort children by slot range, decreasing
	var childrenSlice = []*Child{}
	for _, c := range n.Children {
		if c.Interface[layer] != nil {
			if c.Interface[layer][0] != 0 {
				childrenSlice = append(childrenSlice, c)
			}
		}
	}
	if len(childrenSlice) == 0 {
		return
	}
	sort.SliceStable(childrenSlice, func(i, j int) bool {
		if childrenSlice[i].Interface[layer][0] > childrenSlice[j].Interface[layer][0] {
			return true
		} else if childrenSlice[i].Interface[layer][0] == childrenSlice[j].Interface[layer][0] {
			return childrenSlice[i].Interface[layer][1] > childrenSlice[j].Interface[layer][1]
		}
		return false
	})

	// find the children with longest slot range, and place it at the bottom, as the width bound
	var child = childrenSlice[0]
	child.SubPartitionOffset[layer] = []int{0, child.Interface[layer][0], channels, channels + child.Interface[layer][1]}
	slots = child.Interface[layer][0]
	channels += child.Interface[layer][1]
	if len(childrenSlice) == 1 {
		n.Interface[layer] = []int{slots, channels}
		return
	}
	childrenSlice = childrenSlice[1:]

	skylines := []*skyline{}
	skylines = append(skylines, &skyline{
		start:  0,
		end:    slots,
		width:  slots,
		height: channels,
	})

L1:
	for len(childrenSlice) > 0 {
		// sort skylines from start to end
		sort.SliceStable(skylines, func(i, j int) bool {
			return skylines[i].start < skylines[j].start
		})
		// concat lines
		for i := 0; i < len(skylines)-1; i++ {
			if skylines[i].height == skylines[i+1].height && skylines[i].end == skylines[i+1].start {
				skylines[i].end = skylines[i+1].end
				skylines[i].width += skylines[i+1].width
				skylines = append(skylines[:i+1], skylines[i+2:]...)
			}
		}
		for i, s := range skylines {
			if s.width == 0 {
				skylines = append(skylines[:i], skylines[i+1:]...)
			}
		}
		// sort by height, increasing order
		sort.SliceStable(skylines, func(i, j int) bool {
			if skylines[i].height < skylines[j].height {
				return true
			}
			if skylines[i].height == skylines[j].height {
				return skylines[i].start < skylines[j].start
			}
			return false
		})

		// place child to the best fit skyline
		for _, s := range skylines {
			var hasFit bool
			for j, c := range childrenSlice {
				if s.width >= c.Interface[layer][0] {
					c.SubPartitionOffset[layer] = []int{s.start, s.start + c.Interface[layer][0], s.height, s.height + c.Interface[layer][1]}
					childrenSlice = append(childrenSlice[:j], childrenSlice[j+1:]...)

					// create a new skyline, remaining part
					skylines = append(skylines, &skyline{
						start:  s.start + c.Interface[layer][0],
						end:    s.end,
						width:  s.width - c.Interface[layer][0],
						height: s.height,
					})
					// update the used skyline
					s.end = s.start + c.Interface[layer][0]
					s.width = c.Interface[layer][0]
					s.height += c.Interface[layer][1]

					hasFit = true
					break
				}
			}

			if !hasFit {
				// increase height to align with its lowest neighbor
				var left, right = 16, 16
				for _, ss := range skylines {
					if ss.end == s.start {
						left = ss.height
					} else if ss.start == s.end {
						right = ss.height
					}
				}
				if left <= right {
					s.height = left
				} else if right < left {
					s.height = right
				}

			}
			goto L1
		}
	}
	for _, s := range skylines {
		if channels < s.height {
			channels = s.height
		}
	}
	// exceed channel limit, use max channel as the width of the strip and recompute
	if channels > n.maxChannel {
		slots = 0
		channels = n.maxChannel

		childrenSlice = []*Child{}
		for _, c := range n.Children {
			if c.Interface[layer] != nil {
				// reset subpartition offset
				c.SubPartitionOffset[layer] = nil
				if c.Interface[layer][0] != 0 {
					childrenSlice = append(childrenSlice, c)
				}
			}
		}
		sort.SliceStable(childrenSlice, func(i, j int) bool {
			if childrenSlice[i].Interface[layer][1] > childrenSlice[j].Interface[layer][1] {
				return true
			} else if childrenSlice[i].Interface[layer][1] == childrenSlice[j].Interface[layer][1] {
				return childrenSlice[i].Interface[layer][0] > childrenSlice[j].Interface[layer][0]
			}
			return false
		})

		skylines := []*skyline{}
		skylines = append(skylines, &skyline{
			start:  0,
			end:    channels,
			width:  channels,
			height: 0,
		})

	L2:
		for len(childrenSlice) > 0 {
			// sort skylines from start to end
			sort.SliceStable(skylines, func(i, j int) bool {
				return skylines[i].start < skylines[j].start
			})
			// concat lines
			for i := 0; i < len(skylines)-1; i++ {
				if skylines[i].height == skylines[i+1].height && skylines[i].end == skylines[i+1].start {
					skylines[i].end = skylines[i+1].end
					skylines[i].width += skylines[i+1].width
					skylines = append(skylines[:i+1], skylines[i+2:]...)
				}
			}
			for i, s := range skylines {
				if s.width == 0 {
					skylines = append(skylines[:i], skylines[i+1:]...)
				}
			}

			// sort by height, increasing order
			sort.SliceStable(skylines, func(i, j int) bool {
				if skylines[i].height < skylines[j].height {
					return true
				}
				if skylines[i].height == skylines[j].height {
					return skylines[i].start < skylines[j].start
				}
				return false
			})

			// place child to the best fit skyline
			for _, s := range skylines {
				var hasFit bool
				for j, c := range childrenSlice {
					if s.width >= c.Interface[layer][1] {
						c.SubPartitionOffset[layer] = []int{s.height, s.height + c.Interface[layer][0], s.start, s.start + c.Interface[layer][1]}
						childrenSlice = append(childrenSlice[:j], childrenSlice[j+1:]...)

						// create a new skyline, remaining part
						skylines = append(skylines, &skyline{
							start:  s.start + c.Interface[layer][1],
							end:    s.end,
							width:  s.width - c.Interface[layer][1],
							height: s.height,
						})
						// update the used skyline
						s.end = s.start + c.Interface[layer][1]
						s.width = c.Interface[layer][1]
						s.height += c.Interface[layer][0]

						hasFit = true
						break
					}
				}

				if !hasFit {
					// increase height to align with its lowest neighbor
					var left, right = 1000, 1000
					for _, ss := range skylines {
						if ss.end == s.start {
							left = ss.height
						} else if ss.start == s.end {
							right = ss.height
						}
					}
					if left <= right {
						s.height = left
					} else if right < left {
						s.height = right
					}
				}
				goto L2
			}
		}
		for _, s := range skylines {
			if slots < s.height {
				slots = s.height
			}
		}
	}
	n.Interface[layer] = []int{slots, channels}

}

// map logical sub-partition offset to physcial sub-partition locations
func (n *Node) allocateSubpartition() {
	if n.ID == 0 {
		var redundant = 2
		var slotIdx = 0
		for l := MaxLayer; l > 0; l-- {
			if n.Interface[l] == nil {
				continue
			}
			n.SubPartition[l] = []int{slotIdx, slotIdx + redundant + n.Interface[l][0], 1, 11}
			slotIdx += redundant + n.Interface[l][0]
		}
	}

	for l := n.Layer + 1; l <= MaxLayer; l++ {
		if n.SubPartition[l] == nil {
			continue
		}
		for _, c := range n.Children {
			if c.Interface[l] != nil && c.SubPartitionOffset[l] != nil {
				c.SubPartition[l] = []int{
					n.SubPartition[l][0] + c.SubPartitionOffset[l][0],
					n.SubPartition[l][0] + c.SubPartitionOffset[l][1],
					n.SubPartition[l][3] - c.SubPartitionOffset[l][3],
					n.SubPartition[l][3] - c.SubPartitionOffset[l][2],
				}

			}
		}
	}
	for _, c := range n.Children {
		if len(c.SubPartition) > 0 {
			n.sendTo(c.ID, MSG_SP, c.SubPartition)
		}
	}
}

// update interface, simulate dynamic network
func (n *Node) updateInterface(layer int, newIF []int) {
	adjMsgCnt = 0
	n.Interface[layer] = newIF
	if n.Interface[layer][0] > n.SubPartition[layer][1]-n.SubPartition[layer][0] ||
		n.Interface[layer][1] > n.SubPartition[layer][3]-n.SubPartition[layer][2] {
		n.sendTo(n.Parent, MSG_IF_UPDATE, append([]int{layer}, n.Interface[layer]...))
		adjMsgCnt++
		n.Logger.Printf("IF @ L%d changed and exceeded allocated SP, send SP_ADJ_REQ to #%d\n", layer, n.Parent)
		wsLogger <- wsLog{
			WS_LOG_MSG,
			"Enter Sub-partition Adjustment phase...",
			nil,
		}
		wsLogger <- wsLog{
			WS_LOG_MSG,
			fmt.Sprintf("%d) #%d IF @ L%d changed and exceeded allocated SP, send SP_ADJ_REQ to #%d", adjMsgCnt, n.ID, layer, n.Parent),
			nil,
		}
	}
}

// called in adjustment phase, objective is to minimize changes
func (n *Node) reCompositeInterface(layer int) {

}

func (n *Node) adjustSubpartition(layer int) {
	for _, c := range n.Children {
		if c.Interface[layer] != nil && c.SubPartitionOffset[layer] != nil && c.SubPartition[layer] != nil {
			newSubpartition := []int{
				n.SubPartition[layer][0] + c.SubPartitionOffset[layer][0],
				n.SubPartition[layer][0] + c.SubPartitionOffset[layer][1],
				n.SubPartition[layer][3] - c.SubPartitionOffset[layer][3],
				n.SubPartition[layer][3] - c.SubPartitionOffset[layer][2],
			}
			if c.SubPartition[layer][0] != newSubpartition[0] ||
				c.SubPartition[layer][1] != newSubpartition[1] ||
				c.SubPartition[layer][2] != newSubpartition[2] ||
				c.SubPartition[layer][3] != newSubpartition[3] {
				c.SubPartition[layer] = newSubpartition
				mutex.Lock()
				adjMsgCnt++
				mutex.Unlock()
				n.Logger.Printf("send SP_UPDATE @ L%d to #%d \n", layer, c.ID)
				wsLogger <- wsLog{
					WS_LOG_MSG,
					fmt.Sprintf("%d) #%d send SP_UPDATE @ L%d to #%d", adjMsgCnt, n.ID, layer, c.ID),
					nil,
				}
				n.sendTo(c.ID, MSG_SP_UPDATE, append([]int{layer}, c.SubPartition[layer]...))
			}
		}
	}
}
