import axios from 'axios'

var instance = axios.create()

// instance.defaults.baseURL = 'https://6tisch.amyang.xyz/'
instance.defaults.baseURL = 'http://localhost:8888/'
instance.defaults.headers.post['Content-Type'] = 'application/x-www-form-urlencoded';

instance.interceptors.response.use(function (response) {
    return response;
  }, function (error) {
    return Promise.reject(error);
  });

export default instance