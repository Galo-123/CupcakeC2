<template>
  <div class="login-container">
    <div class="login-box animate__animated animate__fadeIn">
      <div class="login-header">
        <span class="logo-emoji">🧁</span>
        <h1>Cupcake C2</h1>
        <p>Advanced Command & Control Platform</p>
      </div>
      
      <el-form :model="form" class="login-form" @keyup.enter="handleLogin">
        <el-form-item>
          <el-input 
            v-model="form.username" 
            placeholder="Username" 
            :prefix-icon="User"
            size="large"
          />
        </el-form-item>
        <el-form-item>
          <el-input 
            v-model="form.password" 
            type="password" 
            placeholder="Password" 
            :prefix-icon="Lock" 
            show-password
            size="large"
          />
        </el-form-item>
        <el-form-item>
          <el-button 
            type="primary" 
            class="login-btn" 
            :loading="loading" 
            @click="handleLogin"
            size="large"
          >
            Sign In
          </el-button>
        </el-form-item>
      </el-form>
      
      <div class="login-footer">
        <p>&copy; 2026 Cupcake Team • Security Audit Mode</p>
      </div>
    </div>
    
    <!-- Background Decoration -->
    <div class="bg-circles">
      <div class="circle circle-1"></div>
      <div class="circle circle-2"></div>
      <div class="circle circle-3"></div>
    </div>
  </div>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { User, Lock } from '@element-plus/icons-vue'
import api from '../api/index'
import { ElMessage } from 'element-plus'

const router = useRouter()
const loading = ref(false)

const form = reactive({
  username: '',
  password: ''
})

const handleLogin = async () => {
  if (!form.username || !form.password) {
    ElMessage.warning('Please enter credentials')
    return
  }
  
  loading.value = true
  try {
    const res = await api.post('/api/auth/login', form)
    const { token, user } = res.data
    
    // Save token and user info
    localStorage.setItem('cupcake_token', token)
    localStorage.setItem('cupcake_user', JSON.stringify(user))
    
    ElMessage.success('Login successful')
    router.push('/dashboard')
  } catch (e) {
    ElMessage.error(e.response?.data?.error || 'Login failed')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-container {
  height: 100vh;
  width: 100vw;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #0f172a;
  position: relative;
  overflow: hidden;
}

.login-box {
  width: 400px;
  padding: 40px;
  background: rgba(30, 41, 59, 0.7);
  backdrop-filter: blur(12px);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 20px;
  z-index: 10;
  box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.5);
}

.login-header {
  text-align: center;
  margin-bottom: 40px;
}

.logo-emoji {
  font-size: 48px;
  display: block;
  margin-bottom: 10px;
}

.login-header h1 {
  color: #fff;
  font-size: 28px;
  font-weight: 700;
  margin: 0;
}

.login-header p {
  color: #94a3b8;
  font-size: 14px;
  margin-top: 5px;
}

.login-btn {
  width: 100%;
  margin-top: 10px;
  background: linear-gradient(135deg, #3b82f6 0%, #2563eb 100%);
  border: none;
  font-weight: 600;
  letter-spacing: 0.5px;
}

.login-footer {
  text-align: center;
  margin-top: 30px;
}

.login-footer p {
  color: #475569;
  font-size: 12px;
}

/* Background Circles Animation */
.bg-circles {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  z-index: 1;
}

.circle {
  position: absolute;
  border-radius: 50%;
  filter: blur(80px);
  opacity: 0.4;
}

.circle-1 {
  width: 400px;
  height: 400px;
  background: #3b82f6;
  top: -100px;
  right: -100px;
}

.circle-2 {
  width: 300px;
  height: 300px;
  background: #8b5cf6;
  bottom: -50px;
  left: -50px;
}

.circle-3 {
  width: 250px;
  height: 250px;
  background: #ec4899;
  top: 50%;
  left: 10%;
  transform: translateY(-50%);
}

:deep(.el-input__wrapper) {
  background-color: rgba(15, 23, 42, 0.6) !important;
  box-shadow: none !important;
  border: 1px solid rgba(255, 255, 255, 0.1);
}

:deep(.el-input__inner) {
  color: #fff !important;
}

:deep(.el-form-item) {
  margin-bottom: 20px;
}
</style>
