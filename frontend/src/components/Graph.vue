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
            roam: true,
            symbol:"circle",
            symbolSize: 11,
            orient: 'vertical',
            itemStyle:{
              color:"white",
              borderColor: "red",
              shadowColor:"red"
            },
            label: {
              position: 'top',
              verticalAlign: 'middle',
              align: 'right',
              fontSize: 13
            },

            leaves: {
                label: {
                    position: 'top',
                    verticalAlign: 'middle',
                    align: 'right'
                }
            },
            initialTreeDepth: 10,
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
          symbolSize: 12,
          label: {
            fontSize: 12
          },
          itemStyle:{color:"white"},
          lineStyle:{width:8}
        } 
      }
      for(var i=1;i<Object.keys(this.topo).length;i++) {
        var node = i
        var parent = this.topo[node].parent

        if(this.trees[node]==null) 
          this.trees[node] = {name: node, children:[], itemStyle:{color:"white"}}

        if(this.trees[parent]==null)
          this.trees[parent] = { name: parent, children: [ this.trees[node] ], itemStyle:{color:"white"}}
        else
          this.trees[parent].children.push(this.trees[node])
      }
      this.option.series[0].data = [this.trees[0]]
    },

  },
  mounted() {
    this.$EventBus.$on("topo", (topo) => {
      this.topo = topo.data
      this.draw()
    });

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
        this.trees[i].lineStyle.width = 8
      }
      this.affectedNodes = []
    })

    this.$EventBus.$on("affectedNodes", (node)=>{
      this.trees[ node ].itemStyle.color = "red"
      this.trees[ node ].lineStyle.width = 4
      this.trees[ node ].lineStyle.color = "red"
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