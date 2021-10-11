package main

import (
	"fmt"
	"log"
	"os"
	"sort"
)

type Node struct {
	ID              int           `json:"id"`
	Parent          int           `json:"parent"`
	Children        map[int]Child `json:"-"`
	Layer           int           `json:"layer"`        // equals to hop count
	Traffic         int           `json:"-"`            // local traffic of each node is 1
	Interface       map[int][]int `json:"interface"`    // resource interface [slots, channels]
	SubPartitionAbs map[int][]int `json:"subpartition"` // allocated sub-partition [slots start&end, channels start&end]

	AdjustingNodes []int

	receivedInterfaceCnt int

	// internal signal
	sigWaitChildrenInterfaces     chan int
	sigWaitAllocatedSubpartition  chan int
	sigWaitSubpartitionAdjustment chan int

	// external message rx
	RXCh   chan Msg    `json:"-"`
	Logger *log.Logger `json:"-"`
}
type AdjustingNode struct {
	ID           int
	OldInterface map[int][]int
	NewInterface map[int][]int
}

func NewNode(id, parent, layer int) *Node {
	var traffic = 1
	if id == 0 {
		traffic = 0
	}
	node := &Node{
		ID:                            id,
		Parent:                        parent,
		Children:                      make(map[int]Child),
		Layer:                         layer,
		Traffic:                       traffic,
		Interface:                     make(map[int][]int),
		SubPartitionAbs:               make(map[int][]int),
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
	ID              int
	Traffic         int
	Interface       map[int][]int
	SubPartitionRel map[int][]int // output of interface composition, logical location; left->right, bottom->top
	SubPartitionAbs map[int][]int // output of sub-partition allocation, physical location
}

func NewChild(id, traffic int) Child {
	return Child{
		ID:              id,
		Traffic:         traffic,
		Interface:       make(map[int][]int),
		SubPartitionRel: make(map[int][]int),
		SubPartitionAbs: make(map[int][]int),
	}
}

type skyline struct {
	start  int
	end    int
	width  int
	height int
	prev   *skyline
	next   *skyline
}

func (n *Node) Run(blocker chan bool) {
	defer func() {
		<-blocker
		// if len(n.Children) > 0 {
		// 	n.Logger.Printf("resource interface: %v\n\t sub-partition: %v", n.Interface, n.SubPartitionAbs)
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
			affectedNodes[n.ID] = true
		}
		if _, ok := affectedNodes[dst]; !ok {
			affectedNodes[dst] = true
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
	// n.Logger.Println("received interface from", msg.Src, msg.Payload.(map[int][]int))
	// n.Children[msg.Src].Interface = msg.Payload.(map[int][]int)
	for k, v := range msg.Payload.(map[int][]int) {
		n.Children[msg.Src].Interface[k] = v
		// if n.ID == 0 && msg.Src == 14 && k == 3 {
		// 	n.Children[msg.Src].Interface[k] = []int{2, 1}
		// }
	}
	n.receivedInterfaceCnt++

	if n.receivedInterfaceCnt == len(n.Children) {
		n.sigWaitChildrenInterfaces <- 1
	}
}

func (n *Node) subpartitionMsgHandler(msg Msg) {
	// n.Logger.Println("received subpartition from", msg.Src, msg.Payload)
	n.SubPartitionAbs = msg.Payload.(map[int][]int)
	n.sigWaitAllocatedSubpartition <- 1
}

func (n *Node) interfaceUpdateMsgHandler(msg Msg) {
	layer := msg.Payload.([]int)[0]
	// n.Logger.Printf("received SP_ADJ_REQ @ L%d from #%d", layer, msg.Src)
	// wsLogger <- wsLog{
	// 	WS_LOG_MSG,
	// 	fmt.Sprintf("#%d received SP_ADJ_REQ @ L%d from #%d", n.ID, layer, msg.Src),
	// 	nil,
	// }
	n.AdjustingNodes = []int{msg.Src}

	n.Children[msg.Src].Interface[layer] = msg.Payload.([]int)[1:]

	// fesibility test
	feasbility, newIface := n.compositionFeasibilityTest(layer)
	if !feasbility {
		n.Logger.Printf("SP @ L%d cannot handle, multi-hop adjustment, new iface: %v send SP_ADJ_REQ to #%d\n", layer, newIface, n.Parent)

		n.sendTo(n.Parent, MSG_IF_UPDATE, append([]int{layer}, newIface...))
		adjMsgCnt++
		wsLogger <- wsLog{
			WS_LOG_MSG,
			fmt.Sprintf("%d) #%d SP @ L%d cannot handle, send SP_ADJ_REQ to #%d", adjMsgCnt, n.ID, layer, n.Parent),
			nil,
		}

	} else {
		fmt.Println("one hop adjustment")

		for len(n.AdjustingNodes) > 0 {
			n.adaptSubpartition(layer)

		}
		n.adjustSubpartition(layer)

	}
}

func (n *Node) subpartitionUpdateMsgHandler(msg Msg) {
	var layer = msg.Payload.([]int)[0]
	// n.Logger.Printf("received SP_UPDATE @ L%d from #%d", layer, msg.Src)
	// wsLogger <- fmt.Sprintf("#%d received SP_UPDATE @ L%d from #%d", n.ID, layer, msg.Src)
	n.SubPartitionAbs[layer] = msg.Payload.([]int)[1:]
	if len(n.AdjustingNodes) > 0 {
		n.adaptSubpartition(layer)
	}
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

// whether can finish the adjustment in one hop, if not, return the new intereface
func (n *Node) compositionFeasibilityTest(layer int) (bool, []int) {
	res := true
	newIface := []int{0, 0}

	// backup
	childrenRelSpBackup := make(map[int][]int)
	for k, v := range n.Children {
		childrenRelSpBackup[k] = v.SubPartitionRel[layer]
	}

	// fmt.Println(n.Children[26].SubPartitionRel[3])
	n.packingBestFitSkyline(layer)
	// fmt.Println(n.Children[26].SubPartitionRel[3])
	// exceed allocated sub-partition
	if n.Interface[layer][0] > n.SubPartitionAbs[layer][1]-n.SubPartitionAbs[layer][0] ||
		n.Interface[layer][1] > n.SubPartitionAbs[layer][3]-n.SubPartitionAbs[layer][2] {
		res = false
		newIface = n.Interface[layer]
	}

	// recover
	for k, v := range childrenRelSpBackup {
		n.Children[k].SubPartitionRel[layer] = v
	}

	// fmt.Println(n.Children[26].SubPartitionRel[3])
	return res, newIface
}

// adapt original sub-partition layout to a feasible solution
// 1. Remove the changed node's sub-partition in the original layout, and find all idle rectangular areas, try if can fit in any of them.
// 2. Remove a neighbor with smallest weight, and repeat step 1.
func (n *Node) adaptSubpartition(layer int) {
	if len(n.AdjustingNodes) == 0 {
		return
	}

	sort.SliceStable(n.AdjustingNodes, func(i, j int) bool {
		return n.Children[n.AdjustingNodes[i]].Interface[layer][0] > n.Children[n.AdjustingNodes[j]].Interface[layer][0]
	})
	fmt.Println("adjusting", n.AdjustingNodes)
	adjNode := n.AdjustingNodes[0]
	idleRectangles := n.findIdleRectangles(layer)
	// fmt.Println(idleRectangles)
	found := false

	for _, rect := range idleRectangles {
		if rect[1]-rect[0] >= n.Children[adjNode].Interface[layer][0] &&
			rect[3]-rect[2] >= n.Children[adjNode].Interface[layer][1] {
			fmt.Printf("found a rectangle %v to place #%d's iface@l%d: %v\n",
				rect, adjNode, layer, n.Children[adjNode].Interface[layer])
			n.Children[adjNode].SubPartitionRel[layer] = []int{rect[0], rect[0] + n.Children[adjNode].Interface[layer][0],
				rect[2], rect[2] + n.Children[adjNode].Interface[layer][1]}
			found = true
			n.AdjustingNodes = append(n.AdjustingNodes[:0], n.AdjustingNodes[1:]...)
			return
		}
	}

	if !found {
		fmt.Println("need to relocate a neighbor first")

		childrenSlice := []Child{}
		for _, c := range n.Children {
			var skip = false
			for _, nn := range n.AdjustingNodes {
				if c.ID == nn || c.SubPartitionRel[layer] == nil {
					skip = true
					break
				}
			}
			if skip {
				continue
			}
			childrenSlice = append(childrenSlice, c)
		}
		sort.SliceStable(childrenSlice, func(i, j int) bool {
			if childrenSlice[i].Interface[layer][0] == childrenSlice[j].Interface[layer][0] {
				return childrenSlice[i].Interface[layer][1] < childrenSlice[j].Interface[layer][1]
			}
			return childrenSlice[i].Interface[layer][0] < childrenSlice[j].Interface[layer][0]
		})

		for _, c := range childrenSlice {
			n.AdjustingNodes = append(n.AdjustingNodes, c.ID)
			fmt.Println("trying to move", c.ID)
			idleRectangles = n.findIdleRectangles(layer)
			// fmt.Println(idleRectangles)

			for _, rect := range idleRectangles {
				if rect[1]-rect[0] >= n.Children[adjNode].Interface[layer][0] &&
					rect[3]-rect[2] >= n.Children[adjNode].Interface[layer][1] {
					fmt.Printf("found a rectangle %v to place #%d's iface@l%d: %v\n",
						rect, adjNode, layer, n.Children[adjNode].Interface[layer])
					n.Children[adjNode].SubPartitionRel[layer] = []int{rect[0], rect[0] + n.Children[adjNode].Interface[layer][0],
						rect[2], rect[2] + n.Children[adjNode].Interface[layer][1]}
					found = true
					n.AdjustingNodes = append(n.AdjustingNodes[:0], n.AdjustingNodes[1:]...)

					return
				}
			}

			// n.AdjustingNodes = append(n.AdjustingNodes[:1], n.AdjustingNodes[2:]...)

		}

	}
}

// return idle rectangles and fixed nodes' sub-partition
func (n *Node) findIdleRectangles(layer int) [][]int {
	idleRectangles := [][]int{}

	// fixed nodes' relative subpartition at this layer
	fixedNodesRelSp := [][]int{}
	for _, c := range n.Children {
		var skip = false
		for _, r := range n.AdjustingNodes {
			if c.ID == r || c.SubPartitionRel[layer] == nil {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		fixedNodesRelSp = append(fixedNodesRelSp, c.SubPartitionRel[layer])
	}

	// bitmap version
	bitmap := [15]uint32{}
	for _, sp := range fixedNodesRelSp {
		for y := sp[2]; y < sp[3]; y++ {
			for x := sp[0]; x < sp[1]; x++ {
				bitmap[y] |= 0x80000000 >> x
			}
		}
	}
	// for i := len(bitmap) - 1; i >= 0; i-- {
	// 	fmt.Printf("%032b\n", bitmap[i])
	// }

	for yCur := 0; yCur < n.Interface[layer][1]; yCur++ {
		for xCur := 0; xCur < n.Interface[layer][0]; xCur++ {
			if bitmap[yCur]<<xCur&0x80000000 == 0 {
				// endPoints = append(endPoints, []int{xCur, yCur})
				xStart := xCur
				xEnd := xCur
				yStart := yCur
				yEnd := yCur
				for yy := yCur; yy < n.Interface[layer][1]; yy++ {
					if bitmap[yy]<<xCur&0x80000000 != 0 {
						yEnd = yy
						break
					}
					if yy == n.Interface[layer][1]-1 {
						yEnd = yy + 1
					}
				}
				for xx := xCur; xx < n.Interface[layer][0]; xx++ {
					allZero := true
					for yyy := yStart; yyy < yEnd; yyy++ {
						if bitmap[yyy]<<xx&0x80000000 != 0 {
							allZero = false
						}
					}
					if allZero == true {
						xEnd++
					}
				}

				duplicated := false
				for _, rect := range idleRectangles {
					if xStart >= rect[0] && xEnd <= rect[1] && yStart >= rect[2] && yEnd <= rect[3] {
						duplicated = true
						break
					}
				}
				if !duplicated {
					idleRectangles = append(idleRectangles, []int{xStart, xEnd, yStart, yEnd})
				}
			}
		}
	}
	return idleRectangles
}

func (n *Node) packingGreedyChannel(layer int) {
	var slots = 0
	var channels = 0

	// sort children by slot range, decreasing
	var childrenSlice = []Child{}
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

		c.SubPartitionRel[layer] = []int{0, c.Interface[layer][0], channels, channels + c.Interface[layer][1]}

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
	var childrenSlice = []Child{}
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
	child.SubPartitionRel[layer] = []int{0, child.Interface[layer][0], channels, channels + child.Interface[layer][1]}
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
			c.SubPartitionRel[layer] = []int{0, c.Interface[layer][0], channels, channels + c.Interface[layer][1]}
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
					c.SubPartitionRel[layer] = []int{v.slotEdge, v.slotEdge + c.Interface[layer][0], v.channelStart, v.channelStart + c.Interface[layer][1]}
					v.slotEdge += c.Interface[layer][0]
					v.idleSlots -= c.Interface[layer][0]
					found = true
					break
				}
			}
			if !found { // create a new level
				var h = levels[len(levels)-1].channelEnd

				c.SubPartitionRel[layer] = []int{0, c.Interface[layer][0], h, h + c.Interface[layer][1]}
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

// Best-Fit skyline strip packing
// Burke, Edmund K., Graham Kendall, and Glenn Whitwell. "A new placement heuristic for the orthogonal stock-cutting problem." Operations Research 52.4 (2004): 655-671.
func (n *Node) packingBestFitSkyline(layer int) {
	var slots = 0
	var channels = 0

	// sort children by slot range, decreasing
	var childrenSlice = []Child{}
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
	child.SubPartitionRel[layer] = []int{0, child.Interface[layer][0], channels, channels + child.Interface[layer][1]}
	slots = child.Interface[layer][0]
	channels += child.Interface[layer][1]
	if len(childrenSlice) == 1 {
		n.Interface[layer] = []int{slots, channels}
		return
	}
	childrenSlice = childrenSlice[1:]

	s := &skyline{
		start:  0,
		end:    slots,
		width:  slots,
		height: channels,
	}

	head := new(skyline)
	head.next = s

	for len(childrenSlice) > 0 {
		tmp := head.next
		for s := head.next; s != nil; s = s.next {
			if s.height < tmp.height {
				tmp = s
			}
		}
		s = tmp

		var hasFit bool
		for j, c := range childrenSlice {
			if s.width >= c.Interface[layer][0] {
				hasFit = true
				c.SubPartitionRel[layer] = []int{s.start, s.start + c.Interface[layer][0], s.height, s.height + c.Interface[layer][1]}
				childrenSlice = append(childrenSlice[:j], childrenSlice[j+1:]...)

				// create a new skyline, remaining part
				if s.width > c.Interface[layer][0] {
					newS := &skyline{
						start:  s.start + c.Interface[layer][0],
						end:    s.end,
						width:  s.width - c.Interface[layer][0],
						height: s.height,
						prev:   s,
						next:   s.next,
					}

					// update the used skyline
					s.end = s.start + c.Interface[layer][0]
					s.width = c.Interface[layer][0]
					s.height += c.Interface[layer][1]
					s.next = newS
				} else {
					s.height += c.Interface[layer][1]
				}
				break
			}
		}
		if !hasFit {
			s.prev.end = s.end
			s.prev.width += s.width
			s.prev.next = s.next
			if s.next != nil {
				s.next.prev = s.prev
			}
		}
		for ss := head.next; ss != nil; ss = ss.next {
			if ss.next != nil {
				if ss.height == ss.next.height {
					ss.width += ss.next.width
					ss.end = ss.next.end
					ss.next = ss.next.next
					if ss.next != nil {
						ss.next.prev = ss
					}
				}
			}
		}

	}

	for s = head.next; s != nil; s = s.next {
		if channels < s.height {
			channels = s.height
		}
	}

	// exceed channel limit, use max channel as the width of the strip and recompute
	if channels > MAX_CHANNEL {
		slots = 0
		channels = MAX_CHANNEL

		childrenSlice = []Child{}
		for _, c := range n.Children {
			if c.Interface[layer] != nil {
				// reset subpartition offset
				c.SubPartitionRel[layer] = nil
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

		s := &skyline{
			start:  0,
			end:    channels,
			width:  channels,
			height: 0,
		}

		head := new(skyline)
		head.next = s

		for len(childrenSlice) > 0 {
			tmp := head.next
			for s := head.next; s != nil; s = s.next {
				if s.height < tmp.height {
					tmp = s
				}
			}
			s = tmp

			var hasFit bool
			for j, c := range childrenSlice {
				if s.width >= c.Interface[layer][1] {
					hasFit = true
					c.SubPartitionRel[layer] = []int{s.height, s.height + c.Interface[layer][0], s.start, s.start + c.Interface[layer][1]}
					childrenSlice = append(childrenSlice[:j], childrenSlice[j+1:]...)

					// create a new skyline, remaining part
					if s.width > c.Interface[layer][1] {
						newS := &skyline{
							start:  s.start + c.Interface[layer][1],
							end:    s.end,
							width:  s.width - c.Interface[layer][1],
							height: s.height,
							prev:   s,
							next:   s.next,
						}
						// update the used skyline
						s.end = s.start + c.Interface[layer][1]
						s.width = c.Interface[layer][1]
						s.height += c.Interface[layer][0]
						s.next = newS
					} else {
						s.height += c.Interface[layer][0]
					}

					break
				}
			}

			if !hasFit {
				s.prev.end = s.end
				s.prev.width += s.width
				s.prev.next = s.next
				if s.next != nil {
					s.next.prev = s.prev
				}
			}
			for ss := head.next; ss != nil; ss = ss.next {
				if ss.next != nil {
					if ss.height == ss.next.height {
						ss.width += ss.next.width
						ss.end = ss.next.end
						ss.next = ss.next.next
						if ss.next != nil {
							ss.next.prev = ss
						}
					}
				}
			}

		}
		for s = head.next; s != nil; s = s.next {
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
		var gap = 1
		var slotIdx = 0
		for l := MaxLayer; l > 0; l-- {
			if n.Interface[l] == nil {
				continue
			}
			// n.SubPartitionAbs[l] = []int{slotIdx, slotIdx + n.Interface[l][0], MAX_CHANNEL + 1 - n.Interface[l][1], MAX_CHANNEL + 1}
			n.SubPartitionAbs[l] = []int{slotIdx, slotIdx + n.Interface[l][0], 1, MAX_CHANNEL + 1}
			slotIdx += gap + n.Interface[l][0]
		}
	}

	for l := n.Layer + 1; l <= MaxLayer; l++ {
		if n.SubPartitionAbs[l] == nil {
			continue
		}
		for _, c := range n.Children {
			if c.Interface[l] != nil && c.SubPartitionRel[l] != nil {
				c.SubPartitionAbs[l] = []int{
					n.SubPartitionAbs[l][0] + c.SubPartitionRel[l][0],
					n.SubPartitionAbs[l][0] + c.SubPartitionRel[l][1],
					n.SubPartitionAbs[l][3] - c.SubPartitionRel[l][3],
					n.SubPartitionAbs[l][3] - c.SubPartitionRel[l][2],
				}

			}
		}
	}
	for _, c := range n.Children {
		if len(c.SubPartitionAbs) > 0 {
			n.sendTo(c.ID, MSG_SP, c.SubPartitionAbs)
		}
	}
}

// update interface, simulate dynamic network
func (n *Node) updateInterface(layer int, newIF []int) {
	adjMsgCnt = 0
	n.Interface[layer] = newIF
	if n.Interface[layer][0] > n.SubPartitionAbs[layer][1]-n.SubPartitionAbs[layer][0] ||
		n.Interface[layer][1] > n.SubPartitionAbs[layer][3]-n.SubPartitionAbs[layer][2] {
		n.sendTo(n.Parent, MSG_IF_UPDATE, append([]int{layer}, n.Interface[layer]...))
		adjMsgCnt++
		n.Logger.Printf("IF @ L%d exceeds allocated SP, send SP_ADJ_REQ to #%d\n", layer, n.Parent)
		wsLogger <- wsLog{
			WS_LOG_MSG,
			"Enter Sub-partition Adjustment phase...",
			nil,
		}
		wsLogger <- wsLog{
			WS_LOG_MSG,
			fmt.Sprintf("%d) #%d IF @ L%d exceeds allocated SP, send SP_ADJ_REQ to #%d", adjMsgCnt, n.ID, layer, n.Parent),
			nil,
		}
	}
}

// Check if children's sub-partition changed, send SP_Update
func (n *Node) adjustSubpartition(layer int) {
	relocatedCnt := 0
	for _, c := range n.Children {
		if c.Interface[layer] != nil && c.SubPartitionRel[layer] != nil && c.SubPartitionAbs[layer] != nil {
			newSubpartition := []int{
				n.SubPartitionAbs[layer][0] + c.SubPartitionRel[layer][0],
				n.SubPartitionAbs[layer][0] + c.SubPartitionRel[layer][1],
				n.SubPartitionAbs[layer][3] - c.SubPartitionRel[layer][3],
				n.SubPartitionAbs[layer][3] - c.SubPartitionRel[layer][2],
			}
			if c.SubPartitionAbs[layer][0] != newSubpartition[0] ||
				c.SubPartitionAbs[layer][1] != newSubpartition[1] ||
				c.SubPartitionAbs[layer][2] != newSubpartition[2] ||
				c.SubPartitionAbs[layer][3] != newSubpartition[3] {
				c.SubPartitionAbs[layer] = newSubpartition
				relocatedCnt++
				mutex.Lock()
				adjMsgCnt++
				mutex.Unlock()
				n.Logger.Printf("send SP_UPDATE @ L%d to #%d \n", layer, c.ID)
				wsLogger <- wsLog{
					WS_LOG_MSG,
					fmt.Sprintf("%d) #%d send SP_UPDATE @ L%d to #%d", adjMsgCnt, n.ID, layer, c.ID),
					nil,
				}
				n.sendTo(c.ID, MSG_SP_UPDATE, append([]int{layer}, c.SubPartitionAbs[layer]...))
			}
		}
	}
	if relocatedCnt > 0 {
		// n.Logger.Println("total relocated sub-partitions:", relocatedCnt)
	}
}
