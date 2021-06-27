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
      selectedGW: "any",
      selectedRange: "day",
      selectedSensor: {},
      topo: [],
      trees: {},
      // from table.sch
      topo_tree: {},
      option: {
        // tooltip: {
        //   formatter: (item)=>{
        //     var table = this.topo_tree[item.name].resource_table
        //     var s = ''
        //     for(var i in table) {
        //       if(table[i].slots==0&&table[i].channels==0) continue

        //       s+= i+" - ["+table[i].slots+", "+table[i].channels+"]<br>"
        //     }
        //     return s
        //   }
        // },
        series: [
          {
            type: 'tree',
            data:[
              {name:"0",children:[]}
            ],
            top: '5%',
            left: '3%',
            bottom: '5%',
            right: '3%',
            roam: true,
            symbolSize: 8,
            orient: 'vertical',
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
                    align: 'left'
                }
            },
            initialTreeDepth: 10,
            expandAndCollapse: true,
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
            fontSize: 14
          }
        } 
      }
      for(var i=1;i<Object.keys(this.topo).length;i++) {
        var node = i
        var parent = this.topo[node].parent

        if(this.trees[node]==null) 
          this.trees[node] = {name: node, children:[]}

        if(this.trees[parent]==null)
          this.trees[parent] = { name: parent, children: [ this.trees[node] ] }
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

    this.$EventBus.$on("topo_tree", (topo_tree) => {
      this.topo_tree = topo_tree
    })
  }
}
</script>

<style lang="stylus" scoped>
#graph
  width 100%
  height 500px
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