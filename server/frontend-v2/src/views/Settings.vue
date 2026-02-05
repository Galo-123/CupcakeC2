<template>
  <div class="settings-page">
    <el-card shadow="never" class="settings-card">
      <el-tabs v-model="activeTab" class="settings-tabs">
        <!-- 1. User & Security Management -->
        <el-tab-pane name="users">
          <template #label>
            <el-icon><User /></el-icon> 用户与安全
          </template>
          
          <div class="tab-content">
            <div class="section-header">
              <h3>操作员管理</h3>
              <el-button type="primary" :icon="Plus" @click="openUserDialog()">新增操员</el-button>
            </div>
            
            <el-table :data="users" style="width: 100%" v-loading="loading">
              <el-table-column prop="username" label="用户名" width="180" />
              <el-table-column prop="role" label="角色" width="120">
                <template #default="scope">
                  <el-tag :type="scope.row.role === 'admin' ? 'danger' : 'success'">
                    {{ scope.row.role === 'admin' ? '管理员' : '操作员' }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="is_active" label="状态" width="100">
                <template #default="scope">
                  <el-switch 
                    v-model="scope.row.is_active" 
                    @change="toggleUserStatus(scope.row)"
                    active-color="#13ce66"
                  />
                </template>
              </el-table-column>
              <el-table-column prop="created_at" label="创建时间" width="200">
                <template #default="scope">
                  {{ formatDate(scope.row.created_at) }}
                </template>
              </el-table-column>
              <el-table-column label="操作" align="center">
                <template #default="scope">
                  <el-button link type="primary" @click="openUserDialog(scope.row)">修改密码</el-button>
                  <el-button 
                    link 
                    type="danger" 
                    @click="deleteUser(scope.row)" 
                    :disabled="scope.row.username === 'admin'"
                  >删除</el-button>
                </template>
              </el-table-column>
            </el-table>

            <el-divider />
            
            <div class="section-header">
              <h3>登录审计日志</h3>
            </div>
            <el-table :data="loginLogs" style="width: 100%" size="small" stripe>
              <el-table-column prop="created_at" label="时间" width="180">
                <template #default="scope">{{ formatDate(scope.row.created_at) }}</template>
              </el-table-column>
              <el-table-column prop="username" label="用户" width="120" />
              <el-table-column prop="ip" label="登录 IP" width="140" />
              <el-table-column prop="status" label="状态" width="100">
                <template #default="scope">
                  <el-tag :type="scope.row.status === 'success' ? 'success' : 'danger'" size="small">
                    {{ scope.row.status === 'success' ? '成功' : '失败' }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column prop="user_agent" label="浏览器/UA" show-overflow-tooltip />
            </el-table>
          </div>
        </el-tab-pane>

        <!-- 2. Notifications & Webhooks -->
        <el-tab-pane name="notifications">
          <template #label>
            <el-icon><Bell /></el-icon> 通知配置
          </template>
          
          <div class="tab-content">
            <div class="section-header">
              <h3>Webhook 推送</h3>
              <el-button type="primary" :icon="Plus" @click="openWebhookDialog()">添加 Webhook</el-button>
            </div>
            <div class="notify-tips">
              系统支持通过 Webhook 将 Agent 上线/断开事件推送到你的即时通讯软件。
            </div>

            <el-row :gutter="20">
              <el-col :span="12" v-for="hook in webhooks" :key="hook.id">
                <el-card shadow="hover" class="webhook-card">
                  <div class="webhook-header">
                    <div class="hook-type">
                      <img :src="getWebhookIcon(hook.type)" class="hook-icon" />
                      <span>{{ hook.name }}</span>
                    </div>
                    <el-switch v-model="hook.is_enabled" @change="saveWebhook(hook)" />
                  </div>
                  <div class="webhook-body">
                    <div class="hook-url">{{ hook.url }}</div>
                    <div class="hook-events">
                      订阅事件: 
                      <el-tag v-for="ev in hook.events.split(',')" :key="ev" size="small" effect="plain" style="margin-right: 5px;">
                        {{ ev === 'agent_online' ? '上线' : (ev === 'agent_offline' ? '掉线' : ev) }}
                      </el-tag>
                    </div>
                  </div>
                  <div class="webhook-footer">
                    <el-button link type="primary" @click="openWebhookDialog(hook)">编辑</el-button>
                    <el-button link type="danger" @click="deleteWebhook(hook.id)">删除</el-button>
                  </div>
                </el-card>
              </el-col>
            </el-row>
          </div>
        </el-tab-pane>

        <!-- 3. Global Policies -->
        <el-tab-pane name="policies">
          <template #label>
            <el-icon><Setting /></el-icon> 全局策
          </template>
          
          <div class="tab-content narrow-content">
            <h3>回连策略</h3>
            <el-form label-position="left" label-width="150px">
              <el-form-item label="默认心跳 (s)">
                <el-input-number v-model="globalConfig.default_sleep" :min="1" />
                <div class="form-tip">新生成的 Payload 默认使用的心跳间隔。</div>
              </el-form-item>
              <el-form-item label="默认抖动 (%)">
                <el-input-number v-model="globalConfig.default_jitter" :min="0" :max="100" />
                <div class="form-tip">心跳时间的随机浮动比例。</div>
              </el-form-item>
              
              <el-divider />
              <h3>OpSec 自身隐藏</h3>
              <el-form-item label="默认反连地址">
                <el-input v-model="globalConfig.system_c2_host" placeholder="例如: 1.2.3.4 或 c2.domain.com" />
                <div class="form-tip">当监听器绑定 0.0.0.0 时，迁移或生成 Payload 默认使用的反连地址。</div>
              </el-form-item>
              
              <el-form-item label="404 伪装目标">
                <el-input v-model="globalConfig.opsec_cloak_url" placeholder="例如: https://www.google.com" />
                <div class="form-tip">非 API 路径访问时，服务端会自动重定向到该地址。</div>
              </el-form-item>
              
              <el-divider />
              <h3>API 与访问安全</h3>
              <el-form-item label="Master API Token">
                <el-input v-model="globalConfig.system_api_token" placeholder="系统自动生成" show-password>
                  <template #append>
                    <el-button @click="copyToken">复制</el-button>
                  </template>
                </el-input>
                <div class="form-tip">
                  用于自动化脚本 (MCP) 远程调用 API 的唯一凭证。
                  <el-button link type="primary" size="small" @click="regenerateToken">重置令牌</el-button>
                </div>
              </el-form-item>

              <el-form-item label="MCP 服务状态">
                <el-switch 
                  v-model="globalConfig.system_mcp_enabled" 
                  active-value="true" 
                  inactive-value="false"
                  active-text="开启"
                  inactive-text="关闭"
                />
                <div class="form-tip">关闭后，所有外部自动化脚本 (MCP) 将无法通过 Token 访问 API。</div>
              </el-form-item>

              <el-form-item label="IP 白名单">
                <el-input type="textarea" v-model="globalConfig.allowed_ips" placeholder="每行一个 IP 或网段" />
                <div class="form-tip">允许访问管理面板的 IP (暂未启用)。</div>
              </el-form-item>

              <el-form-item>
                <el-button type="primary" size="large" @click="saveGlobalSettings">保存全局配置</el-button>
              </el-form-item>
            </el-form>
          </div>
        </el-tab-pane>

        <!-- 4. Data Maintenance -->
        <el-tab-pane name="maintenance">
          <template #label>
            <el-icon><DataLine /></el-icon> 数据维护
          </template>
          
          <div class="tab-content">
             <el-alert 
              title="危险操作区域" 
              type="warning" 
              description="此页面内的操作涉及敏感数据修改或删除，请谨慎操作。" 
              show-icon 
              :closable="false"
              style="margin-bottom: 30px;"
            />

            <el-row :gutter="20">
              <el-col :span="12">
                <el-card header="数据备份与导出" shadow="never">
                  <div class="maintenance-item">
                    <p>将当前数据库中的所有主机信息、操作日志导出为 JSON 格式报告。</p>
                    <el-button type="primary" :icon="Download" @click="exportData">立即导出</el-button>
                  </div>
                </el-card>
              </el-col>
              <el-col :span="12">
                <el-card header="系统重置" shadow="never">
                  <div class="maintenance-item">
                    <p>清空所有 Agent 记录、命令日志。此操作不会影响操作员账号。</p>
                    <el-button type="danger" :icon="Delete" @click="resetDatabase">一键重置环境</el-button>
                  </div>
                </el-card>
              </el-col>
            </el-row>
          </div>
        </el-tab-pane>
      </el-tabs>
    </el-card>

    <!-- User Dialog -->
    <el-dialog v-model="userDialog.visible" :title="userDialog.isEdit ? '修改用户' : '新增用户'" width="450px">
      <el-form :model="userDialog.form" label-width="100px">
        <el-form-item label="用户名">
          <el-input v-model="userDialog.form.username" :disabled="userDialog.isEdit" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input v-model="userDialog.form.password" type="password" show-password placeholder="若不修改请留空" />
        </el-form-item>
        <el-form-item label="角色">
          <el-select v-model="userDialog.form.role" style="width: 100%">
            <el-option label="管理员" value="admin" />
            <el-option label="操作员" value="operator" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="userDialog.visible = false">取消</el-button>
        <el-button type="primary" @click="saveUser">确认</el-button>
      </template>
    </el-dialog>

    <!-- Webhook Dialog -->
    <el-dialog v-model="webhookDialog.visible" title="Webhook 配置" width="550px">
      <el-form :model="webhookDialog.form" label-width="100px" label-position="left">
        <el-form-item label="名称">
          <el-input v-model="webhookDialog.form.name" placeholder="例如: 我的钉钉推送" />
        </el-form-item>
        <el-form-item label="类型">
          <el-radio-group v-model="webhookDialog.form.type">
            <el-radio-button label="dingtalk">钉钉</el-radio-button>
            <el-radio-button label="feishu">飞书</el-radio-button>
            <el-radio-button label="slack">Slack</el-radio-button>
            <el-radio-button label="telegram">Telegram</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="Webhook URL">
          <el-input v-model="webhookDialog.form.url" type="textarea" :rows="3" />
        </el-form-item>
        <el-form-item label="订阅事件">
          <el-checkbox-group v-model="webhookDialog.selectedEvents">
            <el-checkbox label="agent_online">Agent 上线</el-checkbox>
            <el-checkbox label="agent_offline">Agent 掉线</el-checkbox>
          </el-checkbox-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="webhookDialog.visible = false">取消</el-button>
        <el-button type="primary" @click="submitWebhook">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { 
  User, Bell, Setting, Plus, Download, Delete, DataLine 
} from '@element-plus/icons-vue'
import api from '../api/index'
import { ElMessage, ElMessageBox } from 'element-plus'

const activeTab = ref('users')
const loading = ref(false)

// Data State
const users = ref([])
const loginLogs = ref([])
const webhooks = ref([])
const globalConfig = reactive({
  default_sleep: 60,
  default_jitter: 10,
  system_c2_host: '',
  system_api_token: '',
  system_mcp_enabled: 'true',
  opsec_cloak_url: '',
  allowed_ips: ''
})

// Dialog States
const userDialog = reactive({
  visible: false,
  isEdit: false,
  form: { id: null, username: '', password: '', role: 'operator' }
})
const webhookDialog = reactive({
  visible: false,
  isEdit: false,
  form: { id: null, name: '', type: 'dingtalk', url: '', events: '' },
  selectedEvents: ['agent_online']
})

// --- Data Fetching ---

const fetchAll = async () => {
    loading.value = true
    try {
        const [u, logs, hooks, conf] = await Promise.all([
            api.get('/api/settings/users'),
            api.get('/api/settings/logs/login'),
            api.get('/api/settings/webhooks'),
            api.get('/api/settings/config')
        ])
        users.value = u.data
        loginLogs.value = logs.data
        webhooks.value = hooks.data
        
        // Parse config
        conf.data.forEach(item => {
            if (globalConfig.hasOwnProperty(item.key)) {
                if (['default_sleep', 'default_jitter'].includes(item.key)) {
                    globalConfig[item.key] = parseInt(item.value)
                } else {
                    globalConfig[item.key] = item.value
                }
            }
        })
    } catch (e) {
        ElMessage.error('无法加载配置数据')
    } finally {
        loading.value = false
    }
}

// --- User Logic ---

const openUserDialog = (row = null) => {
    userDialog.isEdit = !!row
    if (row) {
        userDialog.form = { ...row, password: '' }
    } else {
        userDialog.form = { id: null, username: '', password: '', role: 'operator' }
    }
    userDialog.visible = true
}

const saveUser = async () => {
    try {
        if (userDialog.isEdit) {
            await api.put(`/api/settings/users/${userDialog.form.id}`, userDialog.form)
            ElMessage.success('用户更新成功')
        } else {
            await api.post('/api/settings/users', userDialog.form)
            ElMessage.success('用户创建成功')
        }
        userDialog.visible = false
        fetchAll()
    } catch (e) { ElMessage.error('操作失败') }
}

const toggleUserStatus = async (user) => {
    try {
        await api.put(`/api/settings/users/${user.id}`, { is_active: user.is_active })
        ElMessage.success('状态已更新')
    } catch (e) {
        user.is_active = !user.is_active
        ElMessage.error('更新失败')
    }
}

const deleteUser = (user) => {
    ElMessageBox.confirm(`确定删除用户 ${user.username} 吗？`, '警告', { type: 'warning' })
    .then(async () => {
        await api.delete(`/api/settings/users/${user.id}`)
        ElMessage.success('用户已删除')
        fetchAll()
    }).catch(() => {})
}

// --- Webhook Logic ---

const openWebhookDialog = (row = null) => {
    if (row) {
        webhookDialog.form = { ...row }
        webhookDialog.selectedEvents = row.events.split(',')
    } else {
        webhookDialog.form = { id: null, name: '', type: 'dingtalk', url: '', events: '' }
        webhookDialog.selectedEvents = ['agent_online']
    }
    webhookDialog.visible = true
}

const submitWebhook = async () => {
    webhookDialog.form.events = webhookDialog.selectedEvents.join(',')
    await saveWebhook(webhookDialog.form)
    webhookDialog.visible = false
}

const saveWebhook = async (hook) => {
    try {
        await api.post('/api/settings/webhooks', hook)
        ElMessage.success('Webhook 已保存')
        fetchAll()
    } catch (e) { ElMessage.error('保存失败') }
}

const deleteWebhook = (id) => {
    api.delete(`/api/settings/webhooks/${id}`).then(() => {
        ElMessage.success('已删除')
        fetchAll()
    })
}

const getWebhookIcon = (type) => {
    const icons = {
        dingtalk: 'https://img.icons8.com/color/48/000000/dingtalk.png',
        feishu: 'https://img.icons8.com/color/48/000000/lark.png',
        slack: 'https://img.icons8.com/color/48/000000/slack-new.png',
        telegram: 'https://img.icons8.com/color/48/000000/telegram-app.png'
    }
    return icons[type] || ''
}

// --- Global Settings ---

const saveGlobalSettings = async () => {
    const payload = Object.entries(globalConfig).map(([key, value]) => {
        let group = 'access'
        if (key.startsWith('opsec')) group = 'opsec'
        else if (key.startsWith('default')) group = 'general'
        else if (key.includes('token')) group = 'security'
        
        return { key, value: String(value), group }
    })
    try {
        await api.post('/api/settings/config', payload)
        ElMessage.success('配置已保存')
    } catch (e) {
        ElMessage.error('保存失败')
    }
}

const copyToken = () => {
    navigator.clipboard.writeText(globalConfig.system_api_token)
    ElMessage.success('Token 已复制到剪贴板')
}

const regenerateToken = () => {
    const charset = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()-_=+'
    let token = ''
    for (let i = 0; i < 32; i++) {
        token += charset.charAt(Math.floor(Math.random() * charset.length))
    }
    globalConfig.system_api_token = token
    ElMessage.warning('Token 已重置，请点击保存以生效')
}

// --- Maintenance ---

const exportData = () => {
    window.open('/api/maintenance/export', '_blank')
}

const resetDatabase = () => {
    ElMessageBox.confirm('这清空所有主机记录和命令历史，确定继续吗？', '极度危险', {
        type: 'error',
        confirmButtonText: '确定重置',
        confirmButtonClass: 'el-button--danger'
    }).then(async () => {
        await api.post('/api/maintenance/reset')
        ElMessage.success('系统已重置')
        fetchAll()
    }).catch(() => {})
}

// --- Helpers ---

const formatDate = (ts) => {
    if (!ts) return '-'
    const d = new Date(ts)
    return d.toLocaleString()
}

onMounted(fetchAll)
</script>

<style scoped>
.settings-page {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.settings-card {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.settings-tabs {
  height: 100%;
}

:deep(.el-tabs__content) {
  height: calc(100% - 55px);
  overflow-y: auto;
  padding: 20px;
}

.tab-content {
  max-width: 1000px;
  margin: 0 auto;
}

.narrow-content {
  max-width: 600px;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.section-header h3 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: #303133;
}

.notify-tips {
  margin-bottom: 20px;
  padding: 12px;
  background: rgba(64, 158, 255, 0.05);
  border-left: 4px solid #409eff;
  font-size: 14px;
  color: #606266;
}

.webhook-card {
  margin-bottom: 20px;
}

.webhook-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 15px;
}

.hook-type {
  display: flex;
  align-items: center;
  gap: 10px;
  font-weight: 600;
}

.hook-icon {
  width: 24px;
  height: 24px;
}

.hook-url {
  font-family: 'JetBrains Mono', monospace;
  font-size: 12px;
  color: #909399;
  background: #f8f9fa;
  padding: 8px;
  border-radius: 4px;
  word-break: break-all;
  margin-bottom: 10px;
}

.webhook-footer {
  margin-top: 15px;
  padding-top: 10px;
  border-top: 1px solid #f2f6fc;
  text-align: right;
}

.form-tip {
  font-size: 12px;
  color: #909399;
  line-height: 1.4;
  margin-top: 4px;
}

.maintenance-item {
  text-align: center;
  padding: 20px 0;
}

.maintenance-item p {
  color: #606266;
  margin-bottom: 20px;
  font-size: 14px;
}
</style>
