<template>
  <vs-card id="console">
    <div slot="header">
      <h4>Console</h4>
    </div>
    <textarea
      autofocus
      id="logs"
      ref="logs"
      :value="log"
      @change="(v) => (log = v)"
      disabled
    />
  </vs-card>
</template>

<script>
export default {
  data() {
    return {
      ws: {},
      cnt: 0,
      log: "",
    };
  },
  methods: {
    startWS() {
      var loc = window.location;
      "ws://" + loc.host + "/api/ws";
      this.ws = new WebSocket("ws://localhost:8888/api/ws");
      // this.ws = new WebSocket("ws://"+loc.host+"/ws")
      this.ws.onopen = () => {
        this.log += "[*] Console connected\n";
        // console.log("ws connected", evt)
      };
      this.ws.onclose = () => {
        window.console.log("ws closed, trying to reconnect...");
        this.log += "[!] Connection dropped, trying to reconnect...\n";
        setTimeout(this.startWS, 3000);
      };
      this.ws.onmessage = (evt) => {
        this.cnt++;

        this.log += "[+] " + eval(evt.data) + "\n";
        this.$nextTick(() => {
          this.$refs.logs.scrollTop = this.$refs.logs.scrollHeight;
        });
      };
    },
  },
  created() {
    this.startWS();
  },
};
</script>

<style scoped>
#console {
  height: 435px;
}
#logs {
  color: "blue";
  width: 100%;
  height: 370px;
  font-size: 0.70rem;
  line-height: 1.2;
  resize: none;
  outline: none;
  text-transform: none;
  text-decoration: none;
  
}

textarea {
  border: none;
}
textarea:disabled {
  background: white;
}
</style>