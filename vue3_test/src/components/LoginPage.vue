<template>
  <div class="auth-container" :class="{ 'show-register': activeView === 'register' }">
    <div class="form-box login-box" v-if="!user">
      <h2>登录</h2>
      <form @submit.prevent="handleLogin">
        <input type="text" v-model="loginForm.username" placeholder="用户名" required />
        <input type="password" v-model="loginForm.password" placeholder="密码" required />
        <button type="submit">登录</button>
      </form>
    </div>

    <div class="form-box register-box" v-if="!user">
      <h2>注册</h2>
      <form @submit.prevent="handleRegister">
        <input type="text" v-model="registerForm.username" placeholder="用户名" required />
        <input type="password" v-model="registerForm.password" placeholder="密码" required />
        <button type="submit">注册</button>
      </form>
    </div>


    <div class="logged-in-container" v-if="user">
     <h2>欢迎回来，{{ user.username }}</h2>
     <button @click="logout">退出登录</button>
    </div>

    <div class="sliding-panel" v-if="!user">
      <div class="panel-content login-panel" v-if="activeView === 'login'">
        <h3>新用户？</h3>
        <p>还没有账号？快去注册一个吧！</p>
        <button @click.prevent="activeView = 'register'">去注册</button>
      </div>
      <div class="panel-content register-panel" v-else>
        <h3>已有账号？</h3>
        <p>请直接登录吧！</p>
        <button @click.prevent="activeView = 'login'">去登录</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from "vue";
import request from "@/utils/request";

const activeView = ref("login");
const user = ref(null);

const loginForm = ref({
  username: "",
  password: "",
});

const registerForm = ref({
  username: "",
  password: "",
});

// 登录方法
const handleLogin = async () => {
  try {
    const res = await request.post("/api/login", loginForm.value);

    // 检查 res 是否有数据，并处理两种可能的后端结构
    let token, userData;

    // 尝试从 res.data.data 中解构，这是之前假设的结构
    if (res.data && res.data.data) {
        token = res.data.data.token;
        userData = res.data.data.user;
    } 
    // 否则，尝试从 res.data 中直接解构
    else if (res.data) {
        token = res.data.token;
        userData = res.data.user;
    } else {
        // 如果 res.data 都不存在，则返回错误
        throw new Error("后端响应数据格式不正确");
    }

    if (token && userData) {
      localStorage.setItem("token", token);
      user.value = userData;
      alert("登录成功！");
    } else {
      alert("登录失败: " + (res.message || '凭证无效'));
    }
  } catch (err) {
    console.error("登录请求失败:", err.response ? err.response.data : err.message);
    alert("登录请求失败：" + (err.response ? err.response.data.message : err.message));
  }
};

// 注册方法
const handleRegister = async () => {
  try {
    const { username, password } = registerForm.value;
    const res = await request.post("/api/register", { username, password });
    
    // 检查后端返回的状态码和消息
    if (res.code === 200) {
      alert("注册成功！");
      activeView.value = "login"; // 注册成功后切换到登录界面
    } else {
      // 注册失败，显示后端返回的错误信息
      alert("注册失败：" + res.message);
    }
  } catch (err) {
    console.error("注册请求失败:", err.response ? err.response.data : err.message);
    alert("注册请求失败：" + (err.response ? err.response.data.message : err.message));
  }
};

// 获取用户资料
const fetchUserProfile = async () => {
  try {
    const profileRes = await request.get("/api/user/profile");
    // 后端 /api/user/profile 接口返回的 username 字段也在 data.data 里
    user.value = {
      username: profileRes.data.data.username
    };
  } catch (err) {
    console.error("获取用户资料失败，请重新登录", err);
    logout();
  }
};

// 退出登录
const logout = () => {
  localStorage.removeItem("token");
  user.value = null;
  activeView.value = "login";
};

// 页面加载时如果已有 token，尝试获取用户资料
onMounted(() => {
  const token = localStorage.getItem("token");
  if (token) {
    fetchUserProfile();
  }
});
</script>


<style scoped>
/* 外层容器 */
.auth-container {
  position: relative;
  width: 900px;
  height: 500px;
  margin: 100px auto;
  background: #fff;
  border-radius: 15px;
  box-shadow: 0 10px 25px linear-gradient(pink, blue);
  overflow: hidden;
  display: flex;
}

/* 表单公共样式 */
.form-box {
  width: 50%;
  padding: 40px;
  display: flex;
  flex-direction: column;
  justify-content: center;
  transition: all 0.6s ease;
  z-index: 1;
}

.form-box h2 {
  text-align: center;
  margin-bottom: 20px;
  color: #333;
}

form {
  display: flex;
  flex-direction: column;
  gap: 15px;
}

form input {
  padding: 12px;
  border: 1px solid #ddd;
  border-radius: 8px;
  outline: none;
  transition: all 0.3s;
}

form input:focus {
  border-color: #4e9af1;
  box-shadow: 0 0 5px rgba(78, 154, 241, 0.3);
}

form button {
  padding: 12px;
  background:linear-gradient(to right, pink, rgb(166, 204, 227));
  color: white;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  transition: background 0.3s;
}

form button:hover {
  background: linear-gradient(to right, pink, rgb(166, 204, 227));
}

/* 登录框和注册框位置 */
.login-box {
  left: 0;
}

.register-box {
  right: 0;
}

/* 滑动面板 */
.sliding-panel {
  position: absolute;
  top: 0;
  left: 50%;
  width: 50%;
  height: 100%;
  background: linear-gradient(to right, pink, rgb(166, 204, 227));
  color: white;
  display: flex;
  justify-content: center;
  align-items: center;
  text-align: center;
  transition: transform 0.6s ease-in-out;
  z-index: 2;
}

/* 滑动状态：显示注册时，面板移到左边 */
.auth-container.show-register .sliding-panel {
  transform: translateX(-100%);
}

.panel-content {
  max-width: 80%;
}

.panel-content h3 {
  font-size: 24px;
  margin-bottom: 10px;
}

.panel-content p {
  margin-bottom: 20px;
}

.panel-content button {
  padding: 10px 20px;
  background: white;
  color:linear-gradient(to right, pink, rgb(166, 204, 227));
  border: none;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.3s;
}

.panel-content button:hover {
  background: #f0f0f0;
  color: #3b82e0;
}

/*
 * 登录成功界面的样式
 * 它会替代 .form-box 的默认样式，让界面更友好
 */
.logged-in-container {
  width: 100%; /* 让它占满整个容器，和 form-box 的 50% 不同 */
  padding: 40px;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center; /* 居中显示内容 */
  text-align: center;
}

.logged-in-container h2 {
  font-size: 2.5em; /* 增大标题字体 */
  color: #4e9af1; /* 使用主题色 */
  margin-bottom: 10px;
}

.logged-in-container p {
  color: #666;
  font-size: 1.1em;
  margin-bottom: 25px;
}

.logged-in-container button {
  padding: 12px 30px;
  background: #ff5252; /* 退出登录按钮使用醒目的红色 */
  color: white;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  font-size: 1em;
  transition: background 0.3s;
}

.logged-in-container button:hover {
  background: #e04b4b;
}
</style>
