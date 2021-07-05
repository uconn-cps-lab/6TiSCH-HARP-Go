package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/AmyangXYZ/SweetyGo/middlewares"
	"github.com/AmyangXYZ/sweetygo"
	"github.com/gorilla/websocket"
)

var upgrader websocket.Upgrader

func runHTTPServer() {
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	app := sweetygo.New()
	// app.USE(middlewares.Logger(os.Stdout, middlewares.DefaultSkipper))
	app.USE(middlewares.CORS(middlewares.CORSOpt{}))

	app.GET("/", home)
	app.GET("/static/*files", static)
	app.POST("/api/topo", postTopo)
	app.GET("/api/nodes", getNodes)
	app.GET("/api/node/:id", adjustInterface)
	app.GET("/api/ws", wsLog)
	app.Run(":8888")
}

func home(ctx *sweetygo.Context) error {
	return ctx.Render(200, "index")
}

func static(ctx *sweetygo.Context) error {
	staticHandle := http.StripPrefix("/static",
		http.FileServer(http.Dir("./static")))
	staticHandle.ServeHTTP(ctx.Resp, ctx.Req)
	return nil
}

type topoData struct {
	Topo map[int]topoData2 `json:"topo"`
}

type topoData2 struct {
	Parent   int    `json:"parent"`
	Position [2]int `json:"position"`
	Layer    int    `json:"layer"`
	Path     []int  `json:"Path"`
}

func postTopo(ctx *sweetygo.Context) error {
	Nodes = map[int]*Node{}
	MaxLayer = 0
	var topo topoData
	for k := range ctx.Params() {
		json.Unmarshal([]byte(k), &topo)
	}

	for i, n := range topo.Topo {
		Nodes[i] = NewNode(i, n.Parent, n.Layer+1)
	}

	// fmt.Println(len(Nodes))
	sig1 <- 1
	<-sig2
	return ctx.Text(200, "123")
}

func getNodes(ctx *sweetygo.Context) error {
	if len(Nodes) == 0 {
		return ctx.JSON(200, 0, "no nodes", nil)
	}
	return ctx.JSON(200, 1, "success", Nodes)
}

func adjustInterface(ctx *sweetygo.Context) error {
	id, _ := strconv.Atoi(ctx.Param("id"))
	layer, _ := strconv.Atoi(ctx.Param("layer"))
	newIF := strings.Split(ctx.Param("iface"), ",")

	newIFts, _ := strconv.Atoi(newIF[0])
	newIFch, _ := strconv.Atoi(newIF[1])

	Nodes[id].updateInterface(layer, []int{newIFts, newIFch})
	return ctx.Text(200, "123")
}

func wsLog(ctx *sweetygo.Context) error {
	ws, err := upgrader.Upgrade(ctx.Resp, ctx.Req, nil)
	if err != nil {
		fmt.Println(err)
		ctx.Resp.Write([]byte(err.Error()))
		return err
	}
	defer func() {
		ws.Close()
		// wsLogger = make(chan string, 64)
	}()

	for {
		ws.WriteJSON(<-wsLogger)
	}
}
