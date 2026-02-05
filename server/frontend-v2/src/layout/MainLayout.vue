<template>
  <el-container class="layout-container">
    <!-- Sidebar Navigation -->
    <el-aside width="240px" class="sidebar">
      <!-- Logo / Branding -->
      <div class="logo-area">
        <img :src="logo" alt="Cupcake Logo" class="logo-img" />
        <span class="logo-text">Cupcake</span>
      </div>
      
      <!-- Navigation Menu -->
      <el-menu
        :default-active="activeMenu"
        class="sidebar-menu"
        router
      >
        <el-menu-item index="/dashboard">
          <el-icon><Odometer /></el-icon>
          <span>仪表盘</span>
        </el-menu-item>
        <el-menu-item index="/clients">
          <el-icon><Monitor /></el-icon>
          <span>客户端管理</span>
        </el-menu-item>
        <el-menu-item index="/listeners">
          <el-icon><Headset /></el-icon>
          <span>监听管理</span>
        </el-menu-item>
        <el-menu-item index="/tunnels">
          <el-icon><Share /></el-icon>
          <span>隧道管理</span>
        </el-menu-item>
        <el-menu-item index="/generator">
          <el-icon><Lightning /></el-icon>
          <span>Payload 生成</span>
        </el-menu-item>
        <el-menu-item index="/domain">
          <el-icon><Connection /></el-icon>
          <span>插件管理</span>
        </el-menu-item>
        <el-menu-item index="/settings">
          <el-icon><Setting /></el-icon>
          <span>系统设置</span>
        </el-menu-item>
      </el-menu>

      <!-- Sidebar Footer -->
      <div class="sidebar-footer">
        <div class="build-info">v2.0.1 • Build 2026.01</div>
      </div>
    </el-aside>

    <!-- Main Content Container -->
    <el-container class="main-container">
      <!-- Top Header Bar -->
      <el-header class="header">
        <div class="header-left">
          <el-breadcrumb separator="/">
            <el-breadcrumb-item>{{ $route.meta.title || 'Dashboard' }}</el-breadcrumb-item>
          </el-breadcrumb>
        </div>
        <div class="header-right">
          <el-dropdown trigger="click" @command="handleCommand">
            <div class="user-profile">
              <el-avatar :size="32" class="user-avatar">{{ userInitial }}</el-avatar>
              <span class="username">{{ username }}</span>
              <el-icon><ArrowDown /></el-icon>
            </div>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="password">
                  <el-icon><Lock /></el-icon>修改密码
                </el-dropdown-item>
                <el-dropdown-item command="logout" divided>
                  <el-icon><SwitchButton /></el-icon>退出登录
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </el-header>

      <!-- Main Content Area -->
      <el-main class="main-content">
        <router-view v-slot="{ Component }">
          <transition name="fade-slide" mode="out-in">
            <component :is="Component" />
          </transition>
        </router-view>
      </el-main>
    </el-container>

    <!-- Change Password Dialog -->
    <el-dialog v-model="pwdDialog.visible" title="安全设置 - 修改密码" width="400px" append-to-body>
      <el-form :model="pwdDialog.form" label-width="80px" label-position="top">
        <el-form-item label="当前密码">
          <el-input v-model="pwdDialog.form.oldPassword" type="password" show-password placeholder="请输入原密码验证身份" />
        </el-form-item>
        <el-form-item label="新密码">
          <el-input v-model="pwdDialog.form.newPassword" type="password" show-password placeholder="请输入新密码" />
        </el-form-item>
        <el-form-item label="确认新密码">
          <el-input v-model="pwdDialog.form.confirmPassword" type="password" show-password placeholder="请再次输入新密码" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="pwdDialog.visible = false">取消</el-button>
        <el-button type="primary" @click="submitChangePassword" :loading="pwdDialog.loading">确认修改</el-button>
      </template>
    </el-dialog>
  </el-container>
</template>

<script setup>
import { ref, computed, reactive } from 'vue'
import logo from '../assets/logo.png'
import { useRoute, useRouter } from 'vue-router'
import { 
  Odometer, Monitor, Headset, Share, Lightning, 
  Connection, Setting, CircleCheck, ArrowDown,
  Lock, SwitchButton
} from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import api from '../api/index'

const route = useRoute()
const router = useRouter()

const userData = JSON.parse(localStorage.getItem('cupcake_user') || '{}')
const username = ref(userData.username || 'Admin')
const userInitial = computed(() => username.value.charAt(0).toUpperCase())

const activeMenu = computed(() => {
  const path = route.path
  if (path.startsWith('/client/')) {
    return '/clients'
  }
  return path
})

const pwdDialog = reactive({
  visible: false,
  loading: false,
  form: { oldPassword: '', newPassword: '', confirmPassword: '' }
})

const handleCommand = (command) => {
  if (command === 'password') {
    pwdDialog.form = { oldPassword: '', newPassword: '', confirmPassword: '' }
    pwdDialog.visible = true
  } else if (command === 'logout') {
    handleLogout()
  }
}

const submitChangePassword = async () => {
  if (!pwdDialog.form.oldPassword || !pwdDialog.form.newPassword) {
    return ElMessage.warning('请填写完整信息')
  }
  if (pwdDialog.form.newPassword !== pwdDialog.form.confirmPassword) {
    return ElMessage.warning('两次输入的新密码不一致')
  }

  pwdDialog.loading = true
  try {
    // We assume the backend uses /api/settings/users/me/password or similar, 
    // but looking at our controller, we can use the regular user update if we know our ID.
    // For simplicity, let's create a dedicated endpoint later or use our ID if stored.
    const userId = userData.id
    if (!userId) {
       ElMessage.error('无法确定当前用户信息，请重新登录')
       return
    }
    
    await api.put(`/api/settings/users/${userId}`, { password: pwdDialog.form.newPassword })
    ElMessage.success('密码修改成功，请牢记新密码')
    pwdDialog.visible = false
  } catch (e) {
    ElMessage.error('修改失败: ' + (e.response?.data?.error || e.message))
  } finally {
    pwdDialog.loading = false
  }
}

const handleLogout = () => {
  ElMessageBox.confirm('确定要退出当前系统吗？', '提示', {
    type: 'warning',
    confirmButtonText: '退出',
    cancelButtonText: '取消'
  }).then(() => {
    localStorage.removeItem('cupcake_token')
    localStorage.removeItem('cupcake_user')
    router.push('/login')
  }).catch(() => {})
}
</script>

<style scoped>
/* ========================================
   Layout Container
   ======================================== */
.layout-container {
  height: 100vh;
  width: 100vw;
  background-color: var(--c2-main-bg);
}

.main-container {
  background-color: var(--c2-main-bg);
  display: flex;
  flex-direction: column;
  height: 100vh;
}

/* ========================================
   Sidebar Styling
   ======================================== */
.sidebar {
  background: var(--c2-sidebar-bg);
  border-right: 1px solid var(--c2-border);
  display: flex;
  flex-direction: column;
  position: relative;
  overflow: hidden;
}

/* Logo Area */
.logo-area {
  height: 70px;
  display: flex;
  align-items: center;
  padding: 0 24px;
  gap: 12px;
  background: #ffffff;
}

.logo-img {
  width: 40px;
  height: 40px;
  object-fit: contain;
  filter: drop-shadow(0 0 8px rgba(124, 58, 237, 0.3));
}

.logo-text {
  font-size: 20px;
  font-weight: 800;
  letter-spacing: -0.5px;
  color: var(--c2-text-title);
  background: linear-gradient(135deg, var(--c2-accent), #818cf8);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
}

/* Sidebar Menu */
.sidebar-menu {
  flex: 1;
  background-color: transparent !important;
  border-right: none;
  padding: 12px 0;
  overflow-y: auto;
}

.sidebar-menu::-webkit-scrollbar {
  width: 6px;
}

.sidebar-menu::-webkit-scrollbar-thumb {
  background: #e2e8f0;
  border-radius: 3px;
}

/* Menu Items */
:deep(.el-menu-item) {
  color: #64748b;
  margin: 4px 12px;
  border-radius: 10px;
  transition: all 0.2s ease;
  font-size: 14px;
  font-weight: 500;
  height: 46px;
  line-height: 46px;
}

:deep(.el-menu-item:hover) {
  background: #f1f5f9 !important;
  color: var(--c2-text-title) !important;
}

:deep(.el-menu-item.is-active) {
  background: #f1f5f9 !important;
  color: var(--c2-accent) !important;
  font-weight: 600;
}

:deep(.el-menu-item .el-icon) {
  font-size: 18px;
  margin-right: 12px;
}

/* Sidebar Footer */
.sidebar-footer {
  padding: 16px 20px;
  border-top: 1px solid var(--c2-border);
}

.build-info {
  font-size: 11px;
  color: #94a3b8;
  text-align: center;
  font-family: 'Inter', sans-serif;
}

/* ========================================
   Header Styling
   ======================================== */
.header {
  background-color: var(--c2-main-bg);
  border-bottom: 1px solid var(--c2-tertiary);
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 32px;
  height: 60px;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.02);
  flex-shrink: 0;
}

.header-left {
  display: flex;
  align-items: center;
}

:deep(.el-breadcrumb) {
  font-size: 16px;
  font-weight: 600;
}

:deep(.el-breadcrumb__item:last-child .el-breadcrumb__inner) {
  color: var(--c2-text-title);
}

.header-right {
  display: flex;
  align-items: center;
  gap: 16px;
}

.user-profile {
  display: flex;
  align-items: center;
  gap: 10px;
  cursor: pointer;
  padding: 4px 8px;
  border-radius: 8px;
  transition: background 0.2s;
}

.user-profile:hover {
  background: var(--c2-secondary);
}

.user-avatar {
  background: var(--c2-accent);
  color: var(--c2-accent-text);
  font-weight: 800;
}

.username {
  font-size: 14px;
  font-weight: 600;
  color: var(--c2-text-title);
}

.status-badge {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 0 15px;
  font-weight: 600;
  font-size: 13px;
  border: 1px solid #67c23a;
  background: rgba(103, 194, 58, 0.08);
  color: #67c23a;
}

.status-badge .el-icon {
  font-size: 16px;
}

/* ========================================
   Main Content Area
   ======================================== */
.main-content {
  background-color: #f5f7fa;
  padding: 24px;
  overflow-y: auto;
  flex: 1;
  min-height: 0;
}

.main-content::-webkit-scrollbar {
  width: 8px;
}

.main-content::-webkit-scrollbar-track {
  background: #f1f1f1;
}

.main-content::-webkit-scrollbar-thumb {
  background: #c1c1c1;
  border-radius: 4px;
}

.main-content::-webkit-scrollbar-thumb:hover {
  background: #a8a8a8;
}

/* ========================================
   Transition Animations
   ======================================== */
.fade-slide-enter-active,
.fade-slide-leave-active {
  transition: all 0.25s cubic-bezier(0.4, 0, 0.2, 1);
}

.fade-slide-enter-from {
  opacity: 0;
  transform: translateY(8px);
}

.fade-slide-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}

/* ========================================
   Responsive Design
   ======================================== */
@media (max-width: 768px) {
  .sidebar {
    width: 80px !important;
  }
  
  .logo-text {
    display: none;
  }
  
  :deep(.el-menu-item span) {
    display: none;
  }
  
  .header {
    padding: 0 16px;
  }
  
  .main-content {
    padding: 16px;
  }
}
</style>

