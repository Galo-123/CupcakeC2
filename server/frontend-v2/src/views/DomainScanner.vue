<template>
  <div class="plugin-management-page">
    <el-card shadow="never" class="main-card">
      <template #header>
        <div class="card-header">
          <div class="header-left">
            <el-icon class="header-icon"><Collection /></el-icon>
            <span class="header-title">全局插件管理 (Payload Arsenal)</span>
          </div>
          <div class="header-right">
            <el-input
              v-model="searchQuery"
              placeholder="搜索插件..."
              class="search-input"
              prefix-icon="Search"
              clearable
            />
            <el-button type="primary" :icon="Upload" @click="showUploadDialog = true">
              上传新插件
            </el-button>
          </div>
        </div>
      </template>

      <el-table :data="filteredPlugins" v-loading="loading" border stripe class="plugin-table">
        <el-table-column label="插件名称" min-width="140">
          <template #default="{ row }">
            <span class="plugin-name">{{ row.name }}</span>
          </template>
        </el-table-column>
        
        <el-table-column prop="description" label="功能描述" min-width="200" show-overflow-tooltip />
        
        <el-table-column label="目标平台" width="140" align="center">
          <template #default="{ row }">
            <el-tag 
              :type="row.required_os === 'windows' ? 'primary' : (row.required_os === 'linux' ? 'success' : 'info')" 
              size="default"
              disable-transitions
            >
              {{ formatOS(row.required_os) }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="执行方式" width="240" align="center">
          <template #default="{ row }">
            <el-tag :type="getTypeTag(row.type)" effect="light">
              {{ translateType(row.type) }}
            </el-tag>
          </template>
        </el-table-column>

        <el-table-column label="操作" width="80" align="center" fixed="right">
          <template #default="{ row }">
            <el-button 
              type="danger" 
              size="small" 
              :icon="Delete" 
              circle 
              @click="confirmDelete(row.id)"
            />
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- Upload Dialog -->
    <el-dialog v-model="showUploadDialog" title="上传受控端插件" width="500px">
      <el-form label-position="top">
        <el-row :gutter="20">
          <el-col :span="14">
            <el-form-item label="插件名称" required>
              <el-input v-model="uploadForm.name" placeholder="如: SharpKatz, Mimikatz..." />
            </el-form-item>
          </el-col>
          <el-col :span="10">
            <el-form-item label="目标操作系统" required>
              <el-select v-model="uploadForm.required_os" class="w-full">
                <el-option label="Windows" value="windows" />
                <el-option label="Linux" value="linux" />
                <el-option label="Multi-platform" value="multi" />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>

        <el-form-item label="功能描述" required>
          <el-input v-model="uploadForm.description" type="textarea" :rows="2" placeholder="简述该插件的主要公能..." />
        </el-form-item>

        <el-form-item label="下发执行方式" required>
          <el-select v-model="uploadForm.type" class="w-full" @change="onTypeChange">
            <el-option label="C# .NET 反射执行 (execute-assembly)" value="execute-assembly" />
            <el-option label="Linux 内存执行 (memfd-exec)" value="memfd-exec" />
            <el-option label="Windows Shellcode 注入 (inject-shellcode)" value="inject-shellcode" />
            <el-option label="原生可执行文件直接运行" value="native-exec" />
          </el-select>
        </el-form-item>

        <el-form-item label="功能分类">
          <el-select v-model="uploadForm.category" class="w-full">
            <el-option label="凭据窃取 (Credentials)" value="credentials" />
            <el-option label="内网横向 (Lateral)" value="lateral" />
            <el-option label="环境探测 (Enum)" value="enum" />
            <el-option label="提权工具 (Privesc)" value="privesc" />
            <el-option label="其他 (General)" value="general" />
          </el-select>
        </el-form-item>

        <el-form-item label="插件文件 (.exe, .elf, .bin, .dll)" required>
          <el-upload 
            drag 
            action="#" 
            :auto-upload="false" 
            :limit="1" 
            :on-change="handleFileChange" 
            class="upload-box"
          >
            <el-icon class="el-icon--upload"><UploadFilled /></el-icon>
            <div class="el-upload__text">拖拽文件至此 或 <em>点击选择文件</em></div>
          </el-upload>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showUploadDialog = false">取消</el-button>
        <el-button type="primary" :loading="uploading" @click="submitUpload">立即上传注册</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { Collection, Search, Upload, Delete, UploadFilled } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import api from '@/api'

const loading = ref(false)
const plugins = ref([])
const searchQuery = ref('')
const showUploadDialog = ref(false)
const uploading = ref(false)

const uploadForm = ref({
  name: '',
  description: '',
  required_os: 'windows',
  type: 'execute-assembly',
  category: 'general',
  file: null
})

const filteredPlugins = computed(() => {
  if (!searchQuery.value) return plugins.value
  const q = searchQuery.value.toLowerCase()
  return plugins.value.filter(p => 
    p.name.toLowerCase().includes(q) || 
    (p.description && p.description.toLowerCase().includes(q)) ||
    p.type.toLowerCase().includes(q)
  )
})

const onTypeChange = (type) => {
  if (type === 'memfd-exec') uploadForm.value.required_os = 'linux'
  if (type === 'execute-assembly' || type === 'inject-shellcode') uploadForm.value.required_os = 'windows'
}

const handleFileChange = (f) => {
  uploadForm.value.file = f.raw
}

const fetchPlugins = async () => {
  loading.value = true
  try {
    const res = await api.get('/api/plugins')
    plugins.value = res.data
  } catch (e) {
    ElMessage.error('获取插件列表失败')
  } finally {
    loading.value = false
  }
}

const submitUpload = async () => {
  if (!uploadForm.value.file || !uploadForm.value.name) {
    return ElMessage.warning('请填写必要的信息并选择文件')
  }
  
  uploading.value = true
  const fd = new FormData()
  fd.append('file', uploadForm.value.file)
  fd.append('name', uploadForm.value.name)
  fd.append('description', uploadForm.value.description)
  fd.append('type', uploadForm.value.type)
  fd.append('required_os', uploadForm.value.required_os)
  fd.append('category', uploadForm.value.category)
  
  try {
    await api.post('/api/plugins/upload', fd, {
      headers: { 'Content-Type': 'multipart/form-data' }
    })
    ElMessage.success('插件注册并上传成功')
    showUploadDialog.value = false
    fetchPlugins()
    // Reset form
    uploadForm.value = { name: '', description: '', required_os: 'windows', type: 'execute-assembly', category: 'general', file: null }
  } catch (e) {
    ElMessage.error('上传失败: ' + (e.response?.data?.error || e.message))
  } finally {
    uploading.value = false
  }
}

const confirmDelete = (id) => {
  ElMessageBox.confirm('确定要从武器库中移除该插件及其文件吗？', '删除确认', {
    confirmButtonText: '确定删除',
    cancelButtonText: '取消',
    type: 'warning'
  }).then(async () => {
    try {
      await api.delete(`/api/plugins/${id}`)
      ElMessage.success('插件已移除')
      fetchPlugins()
    } catch (e) {
      ElMessage.error('删除失败')
    }
  })
}

const getTypeTag = (type) => {
  const map = {
    'execute-assembly': 'warning',
    'memfd-exec': 'success',
    'inject-shellcode': 'danger',
    'native-exec': ''
  }
  return map[type] || 'info'
}

const translateType = (type) => {
  const map = {
    'execute-assembly': 'C# 内存加载 (ExecuteAssembly)',
    'memfd-exec': 'Linux 内存执行 (Memfd)',
    'inject-shellcode': 'Shellcode 注入 (Injection)',
    'native-exec': '原生可执行运行 (Native)'
  }
  return map[type] || type
}

const translateCategory = (cat) => {
  const map = {
    'credentials': '凭据窃取',
    'lateral': '内网横向',
    'enum': '环境探测',
    'privesc': '权限提升',
    'general': '通用工具'
  }
  return map[cat] || cat
}

const formatOS = (os) => {
  if (!os || os === 'multi' || os === 'any') return '全平台'
  if (os === 'windows') return 'Windows'
  if (os === 'linux') return 'Linux'
  return os.charAt(0).toUpperCase() + os.slice(1)
}

onMounted(fetchPlugins)
</script>

<style scoped>
.plugin-management-page {
  padding: 24px;
  background-color: #f5f7fa;
  min-height: calc(100vh - 60px);
}

.main-card {
  border-radius: 8px;
  border: none;
  box-shadow: 0 2px 12px 0 rgba(0,0,0,0.05) !important;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 10px;
}

.header-icon {
  font-size: 20px;
  color: #409eff;
}

.header-title {
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

.header-right {
  display: flex;
  gap: 15px;
}

.search-input {
  width: 280px;
}

.plugin-table {
  margin-top: 10px;
}

.name-col {
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.plugin-name {
  font-weight: 600;
  color: #333;
}

.category-text {
  font-size: 13px;
  color: #64748b;
  font-weight: 500;
}

.upload-box {
  width: 100%;
}

.w-full {
  width: 100%;
}
</style>
