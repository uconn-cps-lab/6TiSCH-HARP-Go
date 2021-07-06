package main

const (
	MSG_IF                = 0x11
	MSG_IF_UPDATE         = 0x12
	MSG_SP                = 0x13
	MSG_SP_UPDATE         = 0x14
	WS_LOG_MSG            = 0x21
	WS_LOG_AFFECTED_NODES = 0x22
)

type Msg struct {
	Src     int
	Dst     int
	Type    int
	Payload interface{}
}

type wsLog struct {
	Type int    `json:"type"`
	Msg  string `json:"msg"`
	Data []int  `json:"data"`
}
