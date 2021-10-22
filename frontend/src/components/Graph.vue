<template>
  <vs-card>
    <ECharts id="graph" autoresize :options="option"/>
  </vs-card>
</template>

<script>
import ECharts from "vue-echarts/components/ECharts";
import "echarts/lib/chart/tree";
import "echarts/lib/component/legend";
import "echarts/lib/component/toolbox";
import "echarts/lib/component/tooltip";

// import t from "./topology_121_part.json"
import nodes from "./topo.json";

export default {
  components: {
    ECharts
  },
  data() {
    return {
      hp: {},
      selectedGW: "any",
      selectedRange: "day",
      selectedSensor: {},
      topo: [],
      affectedNodes:[],
      trees: {},
      layer: 10,
      option: {
        tooltip: {
          formatter: (item)=>{
            var node = this.hp[item.name]
            var s = ''
            for(var i in node.interface) {
              s+= i+" - ["+node.interface[i][0]+", "+node.interface[i][1]+"]<br>"
            }

            return s
          }
        },
        series: [
          {
            type: 'tree',
            data:[
              {name:"0",children:[]}
            ],
            top: '5%',
            left: '0%',
            bottom: '5%',
            right: '0%',
            roam: false,
            symbol:"circle",
            symbolSize: 12,
            orient: 'vertical',
            itemStyle:{
              color:"white",
              borderColor: "red",
              shadowColor:"red"
            },
            lineStyle: {
              width:1.5
            },
            label: {
              position: 'top',
              verticalAlign: 'middle',
              align: 'right',
              fontSize: 12,
              fontWeight: "bold"
            },
            initialTreeDepth: 11,
            // expandAndCollapse: false,
            animationDuration: 300,
            animationDurationUpdate: 300
          }
        ]
      }
    }
  },
  methods: {
    draw() {
      this.trees = {
        0: {
          name:"0", 
          children:[], 
          symbolSize: 13,
          itemStyle:{color:"white"},
          lineStyle:{width:5}
        } 
      }
      for(var i=3;i<Object.keys(this.topo).length+2;i++) {
        // if
        var node = i
        var parent = this.topo[node].parent

        if(this.trees[node]==null) 
          this.trees[node] = {name: node, children:[], symbolSize:12, itemStyle:{color:"white"},lineStyle:{width:1.5} }

        if(this.trees[parent]==null)
          this.trees[parent] = { name: parent, children: [ this.trees[node] ], symbolSize:12, itemStyle:{color:"white"},lineStyle:{width:1.5}}
        else
          this.trees[parent].children.push(this.trees[node])
      }
      this.option.series[0].data = [this.trees[0]]
    },

  },
  mounted() {
    this.topo = nodes
    this.draw()
    window.console.log(nodes)
    // this.$EventBus.$on("topo", (topo) => {
    //   this.topo = topo.data
    //   this.draw()
    //   window.console.log(123,this.topo)
    // });

    this.$EventBus.$on("hp_res", (res) => {
      this.hp = res
    })

    this.$EventBus.$on("current_layer", (layer) => {
      this.layer = layer
      // this.option.series[0].initialTreeDepth = this.layer
    })

    this.$EventBus.$on("adjustment", ()=>{
      for(var i=0;i<Object.keys(this.trees).length;i++) {
        this.trees[i].itemStyle.color = "white"
        this.trees[i].symbolSize = 11
      }
      this.affectedNodes = []
    })

    this.$EventBus.$on("affectedNodes", (nodes)=>{
      for(var i=0;i<nodes.length;i++) {
        this.trees[ nodes[i] ].itemStyle.color = "red"
        this.trees[ nodes[i] ].symbolSize = 13
        // this.trees[ nodes[i] ].label.fontSize = 13
      }
    })
  }
}
</script>

<style lang="stylus" scoped>
#graph
  width 100%
  height 425px
</style>

<!-- 
Edge to Tree, 1 loop

```go
package main

type node struct {
	name     int
	children []*node
}

func main() {
	var edges = [][2]int{{1, 0}, {2, 1}, {3, 1}, {4, 2}, {5, 4}, {6, 2}}
	trees := make(map[int]*node)

	for _, pair := range edges {
		c := pair[0]
		p := pair[1]
		if _, ok := trees[c]; !ok {
			trees[c] = &node{c, []*node{}}
		}
		if parent, ok := trees[p]; ok {
			parent.children = append(parent.children, trees[c])
		} else {
			trees[p] = &node{p, []*node{trees[c]}}
		}
	}
}
```
-->