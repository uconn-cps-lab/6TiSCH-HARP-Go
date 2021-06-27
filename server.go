package main

import (
	"encoding/json"
	"net/http"

	"github.com/AmyangXYZ/SweetyGo/middlewares"
	"github.com/AmyangXYZ/sweetygo"
)

func runHTTPServer() {
	app := sweetygo.New()
	// app.USE(middlewares.Logger(os.Stdout, middlewares.DefaultSkipper))
	app.USE(middlewares.CORS(middlewares.CORSOpt{}))

	app.GET("/", home)
	app.GET("/static/*files", static)
	app.POST("/api/topo", postTopo)
	app.GET("/api/nodes", getNodes)
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
	sig <- 1
	return ctx.Text(200, "123")
}

func getNodes(ctx *sweetygo.Context) error {
	if len(Nodes) == 0 {
		return ctx.JSON(200, 0, "no nodes", nil)
	}
	return ctx.JSON(200, 1, "success", Nodes)
}
