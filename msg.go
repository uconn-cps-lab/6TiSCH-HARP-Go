package main

const (
	MSG_IF = 1
	MSG_SP = 2
)

type Msg struct {
	Src     int
	Dst     int
	Type    int
	Payload map[int][]int
}
