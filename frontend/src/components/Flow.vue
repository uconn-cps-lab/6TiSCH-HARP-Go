<template>
  <vs-card>
    <ECharts id="flow" autoresize :options="option"/>
  </vs-card>
</template>

<script>
import ECharts from "vue-echarts/components/ECharts";
import "echarts/lib/chart/line";
import "echarts/lib/chart/scatter";
import "echarts/lib/component/legend";
import "echarts/lib/component/toolbox";
import "echarts/lib/component/tooltip";

export default {
  components: {
    ECharts
  },
  data() {
    return {
      affectedNodes:[],
      cnt: 0,
      option: {
        grid: {
          top: "8%",
          left:"2%",
          right:"1%",
          bottom: "2%",
        },
        xAxis: {
          type: 'value',
          show: false,
          min: 0,
          max: 10,
          data: []
        },
        yAxis: {
          // show: false,
          type: 'value',
          inverse: true,
          interval: 1,
          min: 0,
          max: 10,
          data: []
        },
        series: [{
          data: [],
          type: 'scatter',
          symbolSize: 12,
          color: "blue",
          markLine: {
            data:[],
            lineStyle: {
              type:"solid",
              symbolSize: 1,
              color:"black",
              width: 1.5
            },
            label: {
              show: true,
              position: "middle"
            },
            animation: false
          }
        }]
      }
    }
  },
  methods: {
    draw(node) {
      this.affectedNodes.push(node)
      this.option.xAxis.max = this.affectedNodes.length+1
      
      this.option.series[0].data.push({
        value:[this.affectedNodes.length,0],
        label: {
          show: true,
          position: "top",
          color:"black",
          formatter:()=>{
            return "#"+node
          }
        }
      })
      this.option.series[0].markLine.data.push(
        {
          xAxis:this.affectedNodes.length,
          silent: true,
          label: {
            show: false
          }
        }
      )
    }
  },
  mounted() {
    this.$EventBus.$on("adjustment", ()=>{
      this.option.series[0].data = []
      this.affectedNodes = []
      this.option.series[0].markLine.data = []
      this.cnt = 0
    })

    this.$EventBus.$on("flow", (flow)=>{
      this.cnt++
      this.option.yAxis.max = this.cnt+1
      var type = flow[0]
      var src = flow[1]
      var dst = flow[2]
      var layer = flow[3]

      if(this.affectedNodes.indexOf(src)==-1) {
        this.draw(src)
      }
      if(this.affectedNodes.indexOf(dst)==-1) {
        this.draw(dst)
      }

      if(type==0x12) {
        type = "SP_ADJ_REQ"
      } else if(type==0x14) {
        type = "SP_UPDATE"
      }
      type += " @ L"+layer
      this.option.series[0].markLine.data.push([
        {
          name: type,
          coord:[this.affectedNodes.indexOf(src)+1, this.cnt],
        },
        {
          coord:[this.affectedNodes.indexOf(dst)+1,this.cnt],
          symbolSize: 8
        }
      ])
    })
    
  }
}
</script>

<style scoped>
#flow {
  width: 100%;
  height: 400px
}
</style>