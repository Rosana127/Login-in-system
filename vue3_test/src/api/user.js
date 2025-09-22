import request from '@/utils/request'

// 健康检查
export function getHealth() {
  return request.get('/api/health')
}

// 注册
export function register(data) {
  return request.post('/api/register', data)
}

// 登录
export function login(data) {
  return request.post('/api/login', data)
}

// 获取受保护内容
export function getProtected() {
  return request.get('/api/protected')
}

// 获取用户信息
export function getProfile() {
  return request.get('/api/user/profile')
}
