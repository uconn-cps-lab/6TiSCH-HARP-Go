package main

const (
	MSG_IF        = 1
	MSG_IF_UPDATE = 2
	MSG_SP        = 3
	MSG_SP_UPDATE = 4
)

type Msg struct {
	Src     int
	Dst     int
	Type    int
	Payload interface{}
}
