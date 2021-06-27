import Vue from 'vue'
import App from './App.vue'
import Vuesax from 'vuesax'
import api from './api'
import 'vuesax/dist/vuesax.css'
import 'material-icons/iconfont/material-icons.css';
import '@fortawesome/fontawesome-free/css/all.css'
import '@fortawesome/fontawesome-free/js/all.js'

Vue.prototype.$EventBus = new Vue()
Vue.prototype.$api = api
Vue.use(Vuesax)
Vue.config.productionTip = false

new Vue({
  render: h => h(App),
}).$mount('#app')

