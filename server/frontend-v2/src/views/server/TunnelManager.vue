<template>
  <div class="tunnel-manager-global">
    <el-card shadow="never">
      <template #header>
        <div class="card-header">
          <span style="font-weight: bold;">
            <el-icon style="vertical-align: middle; margin-right: 8px;"><Connection /></el-icon>
            全局隧道监控
          </span>
          <el-button type="primary" :icon="Refresh" circle @click="fetchData" />
        </div>
      </template>
      
      <el-table 
        :data="tunnels" 
        v-loading="loading" 
        style="width: 100%"
        :header-cell-style="{background:'rgba(136, 192, 208, 0.05)', color:'#81A1C1'}"
      >
        <el-table-column label="监听地址" min-width="150">
          <template #default="scope">
            <span style="font-family: 'JetBrains Mono', monospace; font-weight: bold; color: #EBCB8B;">
               0.0.0.0:{{ scope.row.port }}
            </span>
          </template>
        </el-table-column>
        
        <el-table-column label="关联终端 (对方主机地址 : 主机名)" min-width="250">
          <template #default="scope">
            <div v-if="scope.row.agent_ip" class="agent-info-display">
              <router-link :to="'/client/' + scope.row.agent_id" class="agent-link-wrapper">
                <span class="agent-ip">{{ scope.row.agent_ip }}</span>
                <span class="agent-sep"> : </span>
                <span class="agent-name">{{ scope.row.agent_name }}</span>
              </router-link>
              <div class="agent-id-hint">ID: {{ scope.row.agent_id }}</div>
            </div>
            <div v-else>
               <el-tag type="info">离线或未知 ({{ scope.row.agent_id.substring(0,8) }}...)</el-tag>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="协议" width="100">
          <template #default="scope">
             <el-tag :type="scope.row.type === 'http' ? 'warning' : 'success'" effect="light">
               {{ scope.row.type ? scope.row.type.toUpperCase() : 'SOCKS5' }}
             </el-tag>
          </template>
        </el-table-column>
        
        <el-table-column label="状态" width="100">
           <template #default="scope">
             <el-tag type="success" effect="dark" v-if="scope.row.status === 'running'">运行中</el-tag>
             <el-tag type="danger" effect="dark" v-else>已停止</el-tag>
           </template>
        </el-table-column>

        <el-table-column label="操作" width="120" align="center" fixed="right">
          <template #default="scope">
            <el-dropdown trigger="click" @command="handleCommand($event, scope.row)">
              <span class="el-dropdown-link" style="cursor: pointer; color: #409EFF; display: flex; align-items: center; justify-content: center;">
                <span style="margin-right: 4px;">管理</span>
                <el-icon><ArrowDown /></el-icon>
              </span>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="start" v-if="scope.row.status !== 'running'" icon="VideoPlay" style="color: #67C23A;">
                    启动
                  </el-dropdown-item>
                  
                  <el-dropdown-item command="stop" v-if="scope.row.status === 'running'" icon="VideoPause" style="color: #E6A23C;">
                    停止
                  </el-dropdown-item>

                  <el-dropdown-item command="edit" icon="Edit" divided>
                    编辑
                  </el-dropdown-item>

                  <el-dropdown-item command="delete" icon="Delete" style="color: #F56C6C;">
                    删除
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- Restart/Edit Dialog (Shared Logic) -->
    <el-dialog 
      :title="isEdit ? '编辑隧道配置' : '启动隧道'" 
      v-model="editDialogVisible" 
      width="480px" 
      destroy-on-close
      center
    >
      <el-form label-position="top">
        <el-form-item label="监听端口 (VPS Port)">
          <el-input-number 
            v-model="editForm.port" 
            :min="1" 
            :max="65535" 
            controls-position="right"
            style="width: 100%;" 
          />
        </el-form-item>

        <el-form-item label="隧道协议 (Protocol)">
          <el-radio-group v-model="editForm.type" style="width: 100%; display: flex;">
            <el-radio-button label="socks5" style="flex: 1; text-align: center;">SOCKS5</el-radio-button>
            <el-radio-button label="http" style="flex: 1; text-align: center;">HTTP</el-radio-button>
          </el-radio-group>
        </el-form-item>

        <el-divider content-position="center">
          <el-icon style="vertical-align: middle; margin-right: 5px;"><Lock /></el-icon>
          安全设置 (Security)
        </el-divider>

        <el-form-item label="身份验证 (Authentication)">
          <el-switch 
            v-model="editForm.enableAuth" 
            active-text="启用账号密码" 
            inactive-text="无认证"
            style="--el-switch-on-color: #13ce66;"
          />
        </el-form-item>

        <transition name="el-zoom-in-top">
          <div v-if="editForm.enableAuth" class="auth-box">
            <el-row :gutter="15">
              <el-col :span="12">
                <el-form-item label="用户名">
                  <el-input 
                    v-model="editForm.username" 
                    placeholder="admin" 
                    :prefix-icon="User"
                  />
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="密码">
                  <el-input 
                    v-model="editForm.password" 
                    type="password" 
                    show-password 
                    placeholder="password" 
                    :prefix-icon="Key"
                  />
                </el-form-item>
              </el-col>
            </el-row>
          </div>
        </transition>
      </el-form>
      <template #footer>
        <el-button @click="editDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitEdit" :loading="submitting">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { getActiveTunnels, stopTunnel, startTunnel, deleteTunnel } from '@/api/socks'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Refresh, Connection, ArrowDown, VideoPlay, VideoPause, Edit, Delete, Lock, User, Key } from '@element-plus/icons-vue'

const tunnels = ref([])
const loading = ref(false)
const editDialogVisible = ref(false)
const submitting = ref(false)
const isEdit = ref(false)
const currentAgentId = ref('')
const oldPort = ref('')

const editForm = reactive({
    port: 1080,
    type: 'socks5',
    enableAuth: false,
    username: '',
    password: ''
})

const fetchData = async () => {
  loading.value = true
  try {
    const res = await getActiveTunnels()
    if (res.data && res.data.tunnels) {
        tunnels.value = res.data.tunnels
    } else {
        tunnels.value = []
    }
  } catch (error) {
    ElMessage.error('无法同步隧道数据')
  } finally {
    loading.value = false
  }
}

const handleCommand = (command, row) => {
  switch (command) {
    case 'start':
      handleRestart(row)
      break
    case 'stop':
      handleStop(row.port)
      break
    case 'edit':
      handleEdit(row)
      break
    case 'delete':
      handleDelete(row.port)
      break
  }
}

const handleRestart = async (row) => {
    try {
        await startTunnel({
            uuid: row.agent_id,
            port: String(row.port),
            type: row.type,
            username: row.username || '',
            password: row.password || ''
        })
        ElMessage.success('隧道已启动')
        fetchData()
    } catch (e) {
        ElMessage.error(e.response?.data?.message || '启动失败')
    }
}

const handleStop = async (port) => {
  try {
    await stopTunnel({ port })
    ElMessage.success('隧道已关闭')
    fetchData()
  } catch (error) {
    ElMessage.error('关闭失败')
  }
}

const handleDelete = async (port) => {
    try {
        await ElMessageBox.confirm('确定要彻底删除该隧道配置吗？', '警告', {
            confirmButtonText: '删除',
            cancelButtonText: '取消',
            type: 'warning'
        })
        await deleteTunnel({ port })
        ElMessage.success('删除成功')
        fetchData()
    } catch (e) {}
}

const handleEdit = (row) => {
    isEdit.value = true
    currentAgentId.value = row.agent_id
    oldPort.value = row.port
    editForm.port = parseInt(row.port)
    editForm.type = row.type || 'socks5'
    editForm.username = row.username || ''
    editForm.password = row.password || ''
    editForm.enableAuth = !!(editForm.username && editForm.password)
    editDialogVisible.value = true
}

const submitEdit = async () => {
    if (editForm.enableAuth && (!editForm.username || !editForm.password)) {
        ElMessage.warning('启用认证时，用户名和密码不能为空')
        submitting.value = false
        return
    }

    try {
        // Start the new configuration
        await startTunnel({
            uuid: currentAgentId.value,
            port: String(editForm.port),
            type: editForm.type,
            username: editForm.enableAuth ? editForm.username : '',
            password: editForm.enableAuth ? editForm.password : ''
        })
        
        // If the port changed, we should delete the old record
        if (String(editForm.port) !== oldPort.value) {
            await deleteTunnel({ port: oldPort.value })
        }

        ElMessage.success('配置已更新并生效')
        editDialogVisible.value = false
        fetchData()
    } catch (e) {
        ElMessage.error(e.response?.data?.message || '保存失败')
    } finally {
        submitting.value = false
    }
}

onMounted(() => {
  fetchData()
})
</script>

<style scoped>
.tunnel-manager-global {
  padding: 20px;
}
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.agent-info-display {
  display: flex;
  flex-direction: column;
}
.agent-link-wrapper {
  text-decoration: none;
  font-family: 'JetBrains Mono', monospace;
  font-size: 14px;
}
.agent-ip {
  color: #81A1C1;
  font-weight: 600;
}
.agent-sep {
  color: #4C566A;
  margin: 0 4px;
}
.agent-name {
  color: #D8DEE9;
}
.agent-id-hint {
  font-size: 11px;
  color: #4C566A;
  margin-top: 2px;
}
.agent-link-wrapper:hover .agent-ip {
  text-decoration: underline;
}

:deep(.el-radio-button) {
  flex: 1;
}
:deep(.el-radio-button__inner) {
  width: 100%;
  padding: 12px 0; 
}
:deep(.el-radio-group) {
  width: 100%;
}
.auth-box {
  background-color: rgba(0, 0, 0, 0.05);
  padding: 15px;
  border-radius: 4px;
  border: 1px dashed rgba(64, 158, 255, 0.3);
  margin-bottom: 20px;
}
</style>
