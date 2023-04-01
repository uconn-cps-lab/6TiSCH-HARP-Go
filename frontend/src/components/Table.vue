<template>
  <vs-card>
    <vs-row
      vs-w="12"
      vs-type="flex"
      vs-align="flex-start"
      vs-justify="space-between"
    >
      <vs-col vs-w="2">
        <!-- <div slot="header" > -->
        <h2>Scheduler: HP</h2>
        <!-- </div> -->
      </vs-col>
      <vs-col vs-w="5">
        <vs-row vs-w="12" vs-type="flex" vs-justify="flex-end">
          <vs-col vs-offset="0" vs-w="9">
            <vs-row
              class="panel"
              vs-w="10"
              vs-type="flex"
              vs-align="flex-end"
              vs-justify="space-around"
            >
              <vs-col vs-w="2">
                <vs-input
                  type="number"
                  size="small"
                  label="nodeID"
                  class="inputx"
                  v-model="adjustedNode"
                />
              </vs-col>
              <vs-col vs-w="2">
                <vs-input
                  type="number"
                  size="small"
                  label="layer"
                  class="inputx"
                  v-model="adjustedLayer"
                />
              </vs-col>
              <vs-col vs-w="2">
                <vs-input
                  type="string"
                  size="small"
                  label="interface"
                  class="inputx"
                  v-model="adjustedInterface"
                />
              </vs-col>
              <vs-col vs-w="2">
                <vs-button
                  color="primary"
                  type="filled"
                  @click="adjustInterface"
                  >Adjust</vs-button
                >
              </vs-col>
            </vs-row>
          </vs-col>
          <vs-col vs-w="2" vs-type="flex" vs-justify="flex-end">
            <vs-button color="danger" type="filled" @click="handleHPBt"
              >HP</vs-button
            >
          </vs-col>
        </vs-row>
      </vs-col>
    </vs-row>

    <ECharts id="sch-table" autoresize :options="option" />
  </vs-card>
</template>

<script>
import ECharts from "vue-echarts/components/ECharts";
import "echarts/lib/chart/line";
import "echarts/lib/chart/heatmap";
import "echarts/lib/component/visualMap";
import "echarts/lib/component/legend";
import "echarts/lib/component/toolbox";
import "echarts/lib/component/tooltip";
import "echarts/lib/component/title";
import "echarts/lib/component/markArea";
import "echarts/lib/component/markLine";
import "echarts/lib/component/dataZoom";
import "echarts/lib/chart/graph";

const SLOTFRAME = 400;
const CHANNELS = [1, 2, 3, 4, 5, 6, 7, 8,9,10,11,12,13,14,15,16];
const zoomEnd = 100
// const zoomEnd = 16.25;
// const zoomEnd = 9
// const zoomEnd = 6
// const zoomEnd = 4
// var displayedSlotNum = 40;

var LABEL = 4

export default {
  components: {
    ECharts,
  },
  data() {
    return {
      i: 0,
      nodes: {},
      layer: 0,
      selectedCell: { slot: [] },
      sch: {},
      slots: [],
      links: {},
      topo: [],
      seq: [],
      hp_res: {},

      adjustedNode: 9,
      adjustedLayer: 2,
      adjustedInterface: "3,1",

      option: {
        toolbox: {
          feature: {
            // saveAsImage:{}
          },
        },
        tooltip: {
          formatter: (item) => {
            for (var i = 0; i < this.slots.length; i++) {
              // if(this.slots[i].slot[0]==(item.data[0]-0.5) && this.slots[i].slot[1]==(item.data[1]*2+1)) {
              if (
                this.slots[i].slot[0] == item.data[0] - 0.5 &&
                this.slots[i].slot[1] == item.data[1] - 0.5
              ) {
                if (this.slots[i].type == "beacon") {
                  var res = `[${item.data[0] - 0.5}, ${item.data[1] - 0.5}]<br/>
                            Beacon<br/>
                            Subslots<br/>`;
                  for (var sub in this.bcnSubslots[this.slots[i].slot[0]]) {
                    var sub_text = sub.toString();
                    sub_text =
                      sub_text.length < 2 ? "\xa0\xa0" + sub_text : sub_text;
                    res += `${sub_text}\xa0\xa0-\xa0\xa0${
                      this.bcnSubslots[this.slots[i].slot[0]][sub]
                    }<br/>`;
                  }
                  return res;
                }
                // return `[${item.data[0]-0.5}, ${item.data[1]*2+1}]<br/>
                return `[${item.data[0] - 0.5}, ${item.data[1] - 0.5}]<br/>
                        ${this.slots[i].type.replace(/^\S/, (s) =>
                          s.toUpperCase()
                        )}<br/>
                        Layer ${this.slots[i].layer}<br/>
                        ${this.slots[i].sender} -> ${this.slots[i].receiver}`;
              }
            }
            return item.data;
          },
        },
        grid: {
          top: "11%",
          // height: '78%',
          left: "4%",
          right: "1%",
          bottom: "7.5%",
        },
        xAxis: {
          min: 0,
          max: SLOTFRAME,
          // splitNumber: SLOTFRAME,
          // minInterval: 1,
          axisLabel: {
            // formatter: (item) => {
            //   // if(item==32) return 76
            //   // if (item % 5 == 0) return item;
              
            // },
            fontSize: 15,
          },
          name: "Slot Offset",
          type: "value",
          position: "top",
          nameLocation: "middle",
          nameTextStyle: {
            fontWeight: "bold",
            padding: 15,
            fontSize: 12,
          },
          data: [],
          splitArea: {
            show: false,
          },
        },
        yAxis: {
          name: "Channel Offset",
          type: "value",
          min: 1,
          max: CHANNELS.length + 1,
          interval: 1,
          inverse: true,
          nameLocation: "middle",
          nameTextStyle: {
            fontWeight: "bold",
            padding: 10,
            fontSize: 12,
          },
          data: [],
          splitArea: {
            show: false,
          },
          axisLabel: {
            fontSize: 15,
          },
        },
        dataZoom: [
          {
            type: "inside",
            start: 0,
            end: zoomEnd,
          },
          {
            bottom: -2,
            start: 0,
            end: zoomEnd,
            handleIcon:
              "M10.7,11.9v-1.3H9.3v1.3c-4.9,0.3-8.8,4.4-8.8,9.4c0,5,3.9,9.1,8.8,9.4v1.3h1.3v-1.3c4.9-0.3,8.8-4.4,8.8-9.4C19.5,16.3,15.6,12.2,10.7,11.9z M13.3,24.4H6.7V23h6.6V24.4z M13.3,19.6H6.7v-1.4h6.6V19.6z",
            handleSize: "80%",
            handleStyle: {
              color: "#fff",
              shadowBlur: 3,
              shadowColor: "rgba(0, 0, 0, 0.6)",
              shadowOffsetX: 2,
              shadowOffsetY: 2,
            },
          },
        ],
        visualMap: {
          min: 0,
          max: 1,
          show: false,
          type: "piecewise",
          inRange: {
            color: ["green", "#4575b4", "#d73027"],
          },
          pieces: [
            { min: -1, max: -1, label: "Beacon" },
            { min: 0, max: 0, label: "Uplink" },
            { min: 1, max: 1, label: "Downlink" },
          ],
          textStyle: {
            fontSize: 12,
          },
          position: "top",
          orient: "horizontal",
          top: 0,
          right: "1%",
        },
        series: [
          {
            type: "heatmap",
            data: [],
            label: {
              show: false,
              color: "white",
              fontWeight: "bold",
              fontSize: 14.5,
              formatter: (item) => {
                for (var i = 0; i < this.slots.length; i++) {
                  // if(this.slots[i].slot[0]==(item.data[0]-0.5) && this.slots[i].slot[1]==(item.data[1]*2+1)) {
                  if (
                    this.slots[i].slot[0] == item.data[0] - 0.5 &&
                    this.slots[i].slot[1] == item.data[1] - 0.5
                  ) {
                    if (this.slots[i].type != "beacon") {
                      return `${this.slots[i].sender}\n${this.slots[i].receiver}`;
                    }
                  }
                }
                return "";
              },
            },
            itemStyle: {
              borderWidth: 1.1,
              borderType: "solid",
              borderColor: "white",
            },
            markLine: {
              data: [],
              symbolSize: 8,
              lineStyle: {
                color: "yellow",
                width: 3,
                type: "solid",
              },
              animationDuration: 300,
            },
            markArea: {
              silent: true,
              label: {
                position: "bottom",
              },
              data: [],
            },
          },
          {
            type: "line",
            markLine: {
              data: [],
              symbol: "pin",
              symbolSize: 8,
              lineStyle: {
                color: "red",
                width: 3,
                type: "solid",
              },
              label: {
                formatter: (item) => {
                  return "Slot " + (item.data.coord[0] - 0.5).toString();
                },
                fontSize: 13,
              },
              animationDuration: 300,
              animationDurationUpdate: 500,
            },
          },
          // subpartitions, idx=2
          {
            type: "heatmap",
            data: [],
            markArea: {
              silent: true,
              emphasis: {
                label: {
                  fontSize: 20,
                },
              },
              data: [],
            },
          },
        ],
      },
    };
  },
  methods: {
    getHPRes() {
      this.layer = 0;
      this.option.series[2].markArea.data = [
        [
          {
            name: "1",
            xAxis: 0,
            yAxis: 1,
          },
          {
            xAxis: 199,
            yAxis: 17,
            itemStyle: {
              color: "gray",
              opacity: 0.0,
              borderColor: "black",
              borderWidth: 0.1,
            },
          },
        ],
      ];
      this.$api.partition.getNodes().then((res) => {
        if (res.data.flag != 1) return -1;

        // var tmp = 0;
        // for(var iface = 2;iface<5;iface++) {
        //   tmp += res.data.data[0].interface[iface][0]
        // }
        // displayedSlotNum = tmp+15

        // this.option.dataZoom[0].end = displayedSlotNum;
        this.$EventBus.$emit("hp_res", res.data.data);
        this.hp_res = res.data.data;

        for (var i = 0; i < 1; i++) {
          this.drawSubPartition();
          this.layer++;
        }
        // this.sortByLayer();
      });
    },
    drawSubPartition() {
      this.$EventBus.$emit("current_layer", this.layer);
      var pos = "inside";
      var fontSize = 16;
      var color = "black";
      var showLabelFlag = false;
      if(this.layer == LABEL) showLabelFlag = true
      // if (this.layer == 0) {
      //   fontSize = 15;
      //   pos = "insideTop";
      //   color = "white";
      // }
      var colors = [
        "smokewhite",
        "#DBFFF3",
        "#7DE6FF",
        "#CBF2B8",
        "#FFFAB0",
        "#F5C1C1",
        "black",
      ];

      // var gap = 5;

      for (var i in this.hp_res) {
        var node = this.hp_res[i];
        if (node.layer != this.layer) continue;
        // if( node.id!=10 && node.id!=8 && node.id!=0) continue
        
        for (var l in node.subpartition) {
          var name = node.id.toString();
          // if(this.layer==0)
          name = "P(" + node.id + "," + l + ")";

          // var l = node.layer+1
          if (node.subpartition[l] == null) continue;
          this.option.series[2].markArea.data.push([
            {
              // name: "SP("+node.id+", "+l+")",
              name: name,
              itemStyle: {
                color: colors[node.layer + 1],
                opacity: 1,
                borderColor: "black",
                borderWidth: 2,
              },

              label: {
                show: showLabelFlag,
                color: color,
                fontWeight: "bold",
                fontSize: fontSize-1*this.layer,
                position: pos,
              },
              xAxis: node.subpartition[l][0],
              yAxis: node.subpartition[l][2]
            },
            {
              xAxis: node.subpartition[l][1],
              yAxis: node.subpartition[l][3]
            },
          ]);
          
        }
      }
    },
    handleHPBt() {
      this.drawSubPartition();
      this.layer++;
    },
    adjustInterface() {
      this.$EventBus.$emit("adjustment", true);
      this.$api.partition
        .adjustInterface(
          this.adjustedNode,
          this.adjustedLayer,
          this.adjustedInterface
        )
        .then(() => {
          setTimeout(() => {
            this.getHPRes();
          }, 1000);
        });
    },
    findInterface(id, layer) {
      // if(layer<this.hp_res[id].layer) layer = this.hp_res[id].layer
      return this.hp_res[id].interface[layer];
    },
    sortByLayer() {
      this.nodes = {};
      for (var l = 2; l < 10; l++) {
        for (var i = 0; i < Object.keys(this.hp_res).length; i++) {
          if (
            this.hp_res[i].layer == l - 1 &&
            this.hp_res[i].interface[l][0] > 0
          ) {
            if (this.nodes[l] == null) this.nodes[l] = [this.hp_res[i].id];
            else this.nodes[l].push(this.hp_res[i].id);
          }
        }
      }
    },
    adjustInterfaceBatch(layer) {
      var idx = -1;
      var timer = setInterval(() => {
        idx++;
        if (idx >= this.nodes[layer].length) {
          window.console.log("layer " + layer + " finished");
          clearInterval(timer);
          return;
        }
        this.adjustedNode = this.nodes[layer][idx];
        this.adjustedLayer = layer;
        window.console.log("adjusting", this.adjustedNode, this.adjustedLayer);
        var iface = this.findInterface(this.adjustedNode, this.adjustedLayer);
        this.adjustedInterface = iface[0] + 2 + "," + iface[1];

        this.adjustInterface();
      }, 2000);
    },
  },

  mounted() {
    window.table = this;
    this.$EventBus.$on("postTopo", () => {
      window.console.log("topo posted");
      this.getHPRes();
      this.layer = 0;
    });
  },
};
</script>

<style lang="stylus" scoped>
.bts {
  float: right;

  .vs-button {
    margin-bottom: 10px;
    font-size: 0.7rem;
    font-weight: 600;
  }
}

#topo {
  height: 480px;
  width: 100%;
}

.non-optimal {
  font-weight: 600;
  color: red;
}

.partition-usage {
  font-size: 0.9rem;
  text-align: center;

  #part {
    margin-top: 4px;
  }
}

#sch-table {
  width: 100%;
  height: 580px;
}

.panel {
  margin-top: -5px;

  .vs-input {
    width: 55px;
  }
}
</style>