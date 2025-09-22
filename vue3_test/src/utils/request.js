import axios from 'axios'

const request = axios.create({
  // 将 baseURL 设置为你的后端地址，这里是 Vite 代理的地址
  baseURL: 'http://localhost:8080', 
  timeout: 5000
})

// 请求拦截器：自动带 token
request.interceptors.request.use(config => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers['Authorization'] = `Bearer ${token}`
  }
  return config
}, error => Promise.reject(error))

// 响应拦截器：直接返回数据
request.interceptors.response.use(
  response => response.data,
  error => Promise.reject(error)
)

export default request