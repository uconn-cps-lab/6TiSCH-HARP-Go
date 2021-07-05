import axios from './base'

axios.defaults.headers = {
	"Content-Type": "application/x-www-form-urlencoded;charset=utf-8"
};



const sch = {
  getGateway(range) {
      return axios.get(`/api/gateway?range=${range}`)
  },
  postTopo(topo) {
    return axios.post(`/api/topo`, {
      topo:topo
    })
  },
  getNodes() {
    return axios.get(`/api/nodes`)
  },
  adjustInterface(id, layer, iface) {
    return axios.get(`/api/node/${id}?layer=${layer}&iface=${iface}`)
  },
}

export default sch
            