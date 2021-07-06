<template>
  <div id="app">
   <div>
    <vs-row vs-align="flex-start" vs-w="12">
      <br><br><br><br><br>
      <vs-col style="z-index:99" vs-offset="0.25" vs-w="4.3" v-show="true">  
        <Grid/>
      </vs-col>
      <vs-col style="z-index:99" vs-offset="0.2" vs-w="7">  
        <Graph/>
      </vs-col>  
    </vs-row>
    
    <vs-row>
      <vs-col style="z-index:99" vs-offset="0.25" vs-w="7">  
        <Table/>
      </vs-col>
      <vs-col style="z-index:99" vs-offset="0.2" vs-w="4.3">  
        <Console/>
      </vs-col>
    </vs-row>
    
    <vs-row>
      <vs-col style="z-index:99" vs-offset="0.25" vs-w="11.5">  
        <Flow/>
      </vs-col>
    </vs-row>
    </div>
  </div>
  
</template>

<script>
import Console from './components/Console.vue'
import Flow from './components/Flow.vue'
import Graph from "./components/Graph.vue"
import Grid from './components/Grid.vue'
import Table from './components/Table.vue'


export default {
  name: 'app',
  components: {
    Graph,
    Table,
    Grid,
    Console,
    Flow
  }
}
</script>

<style>
html, body {
  max-height: 120%;
  
  width: 100%;
  margin: 0;
  padding: 0;
  background-color: #EBECF1;
  font-size: 1.15rem;
}

#app {
  position: relative;
  min-height: calc(100vh - 80px);
  font-family: "Noto Sans", sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  color: #2c3e50;
  margin-top: 20px;
  /* height: 100%; */
}

#content-wrap {
  padding-bottom: 50px;
  /* overflow: hidden; */
}
</style>
