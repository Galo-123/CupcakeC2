<template>
  <div class="dashboard-container">
    <!-- Header Hero Section -->
    <div class="hero-section shadow-glow">
      <div class="hero-content">
        <h1 class="glow-text">Cupcake C2 Dashboard</h1>
        <p class="hero-subtext">实时受控端状态监控与任务调度中心</p>
      </div>
      <div class="hero-actions">
        <el-button type="primary" class="premium-btn" @click="fetchStats">
          <el-icon class="mr-1"><Refresh /></el-icon> 刷新概览
        </el-button>
      </div>
    </div>

    <!-- Top Row: 4 Stats Cards with Glassmorphism -->
    <div class="stats-row">
      <div class="stat-card glass-morphism" v-for="(item, index) in statItems" :key="index">
        <div class="icon-wrapper" :style="{ background: `linear-gradient(135deg, ${item.color}, ${item.color}88)` }">
          <el-icon class="stat-icon"><component :is="item.icon" /></el-icon>
        </div>
        <div class="stat-content">
          <div class="stat-label">{{ item.label }}</div>
          <div class="stat-number">{{ item.value }}</div>
        </div>
        <div class="stat-trend" :style="{ color: item.color }">
          <el-icon><CaretTop /></el-icon> 正常
        </div>
      </div>
    </div>

    <!-- Middle Row: Interactive Charts / Live Feed -->
    <div class="details-row">
      <!-- Left: Server Status with Detailed Visuals -->
      <div class="detail-card glass-morphism">
        <div class="card-header border-glow">
          <div class="header-title">
            <el-icon class="mr-2"><Platform /></el-icon>
            <h3>核心中继器状态</h3>
          </div>
          <el-tag type="success" size="small" class="pulse-tag">OPERATIONAL</el-tag>
        </div>
        
        <div class="server-info-grid">
          <div class="info-node">
            <div class="node-icon"><Cpu /></div>
            <div class="node-data">
              <span class="node-label">主机节点：</span>
              <span class="node-value">{{ stats.hostname }}</span>
            </div>
          </div>
          <div class="info-node">
            <div class="node-icon"><InfoFilled /></div>
            <div class="node-data">
              <span class="node-label">系统内核：</span>
              <span class="node-value">{{ stats.os }}</span>
            </div>
          </div>
          <div class="info-node">
            <div class="node-icon"><Key /></div>
            <div class="node-data">
              <span class="node-label">安全协议：</span>
              <span class="node-value">AES-256-GCM / TLS 1.3</span>
            </div>
          </div>
          <div class="info-node">
            <div class="node-icon"><Link /></div>
            <div class="node-data">
              <span class="node-label">活跃链路：</span>
              <span class="node-value">{{ stats.client_count }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Right: Resource Monitor with Animated Gauges -->
      <div class="detail-card glass-morphism">
        <div class="card-header border-glow">
          <div class="header-title">
            <el-icon class="mr-2"><Operation /></el-icon>
            <h3>资源占用监控</h3>
          </div>
        </div>
        
        <div class="resource-gauges">
          <div class="gauge-item">
            <div class="gauge-label">
              <span>CPU 负载</span>
              <span class="percent">{{ stats.cpu_usage }}%</span>
            </div>
            <div class="gauge-track">
              <div class="gauge-bar cyan-glow" :style="{ width: stats.cpu_usage + '%' }"></div>
            </div>
          </div>

          <div class="gauge-item">
            <div class="gauge-label">
              <span>内存占用</span>
              <span class="percent">{{ stats.mem_usage }}%</span>
            </div>
            <div class="gauge-track">
              <div class="gauge-bar purple-glow" :style="{ width: stats.mem_usage + '%' }"></div>
            </div>
          </div>

          <div class="gauge-item">
            <div class="gauge-label">
              <span>磁盘 I/O</span>
              <span class="percent">{{ stats.disk_usage }}%</span>
            </div>
            <div class="gauge-track">
              <div class="gauge-bar emerald-glow" :style="{ width: stats.disk_usage + '%' }"></div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, computed } from 'vue'
import api from '../api/index'
import { 
  Monitor, Headset, Timer, Connection, 
  Refresh, CaretTop, Platform, Operation,
  Cpu, InfoFilled, Key, Link
} from '@element-plus/icons-vue'

const stats = ref({
  cpu_usage: "0.0",
  mem_usage: "0.0",
  disk_usage: "0.0",
  uptime: 0,
  listener_count: 0,
  client_count: 0,
  hostname: "-",
  os: "-"
})

const formatUptime = (seconds) => {
  const d = Math.floor(seconds / 86400)
  const h = Math.floor((seconds % 86400) / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  if (d > 0) return `${d}d ${h}h`
  return `${h}h ${m}m`
}

const statItems = computed(() => [
  { 
    label: '在线主机', 
    value: stats.value.client_count, 
    icon: 'Monitor', 
    color: '#00d2ff'
  },
  { 
    label: '活跃监听', 
    value: stats.value.listener_count, 
    icon: 'Headset', 
    color: '#00f2fe'
  },
  { 
    label: '持续时间', 
    value: formatUptime(stats.value.uptime), 
    icon: 'Timer', 
    color: '#ffd000'
  },
  { 
    label: '网络心跳', 
    value: 'Stable', 
    icon: 'Connection', 
    color: '#a18cd1'
  }
])

const fetchStats = async () => {
  try {
    const res = await api.get('/api/dashboard')
    stats.value = res.data
  } catch (e) {
    console.error('Dashboard error:', e)
  }
}

let timer = null
onMounted(() => {
  fetchStats()
  timer = setInterval(fetchStats, 3000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})
</script>

<style scoped>
.dashboard-container {
  display: flex;
  flex-direction: column;
  gap: 24px;
  width: 100%;
  padding-bottom: 20px;
  animation: fadeIn 0.6s ease-out;
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}

/* --- Hero Section (Lighter & Modern) --- */
.hero-section {
  background: linear-gradient(135deg, #6366f1 0%, #818cf8 100%);
  border-radius: 16px;
  padding: 32px 40px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  position: relative;
  overflow: hidden;
  box-shadow: 0 10px 25px rgba(99, 102, 241, 0.15);
}

.hero-section::before {
  content: '';
  position: absolute;
  top: -50%;
  right: -10%;
  width: 80%;
  height: 200%;
  background: radial-gradient(circle, rgba(255, 255, 255, 0.1) 0%, transparent 70%);
  pointer-events: none;
}

.glow-text {
  font-size: 28px;
  font-weight: 800;
  color: #ffffff;
  margin: 0 0 6px 0;
  letter-spacing: -0.5px;
}

.hero-subtext {
  color: rgba(255, 255, 255, 0.9);
  font-size: 15px;
  margin: 0;
  font-weight: 500;
}

.premium-btn {
  background: rgba(255, 255, 255, 1);
  border: none;
  color: #6366f1;
  font-weight: 700;
  padding: 10px 20px;
  border-radius: 10px;
  transition: all 0.3s ease;
}

.premium-btn:hover {
  background: #f8fafc;
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

/* --- Stats Row --- */
.stats-row {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 20px;
}

.glass-morphism {
  background: #ffffff;
  border: 1px solid var(--c2-border);
  border-radius: 20px;
  box-shadow: 0 4px 6px -1px rgba(0,0,0,0.05);
}

.stat-card {
  padding: 24px;
  display: flex;
  align-items: center;
  gap: 16px;
  position: relative;
  transition: all 0.3s ease;
}

.stat-card:hover {
  transform: translateY(-5px);
  box-shadow: 0 12px 20px -5px rgba(0,0,0,0.1);
}

.icon-wrapper {
  width: 52px;
  height: 52px;
  border-radius: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
  color: #ffffff;
}

.stat-label {
  font-size: 13px;
  color: #64748b;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  margin-bottom: 2px;
}

.stat-number {
  font-size: 26px;
  font-weight: 800;
  color: #1e293b;
  line-height: 1.2;
}

.stat-trend {
  font-size: 11px;
  font-weight: 700;
  position: absolute;
  top: 16px;
  right: 16px;
  padding: 2px 8px;
  background: var(--c2-secondary);
  border-radius: 6px;
}

/* --- Details Row --- */
.details-row {
  display: grid;
  grid-template-columns: 1.25fr 1fr;
  gap: 24px;
}

.detail-card {
  padding: 28px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 28px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--c2-border);
}

.header-title {
  display: flex;
  align-items: center;
  color: var(--c2-text-title);
}

.header-title h3 {
  margin: 0;
  font-size: 18px;
  font-weight: 700;
}

.pulse-tag {
  background: #f0fdf4 !important;
  color: #16a34a !important;
  border: 1px solid #bbf7d0 !important;
  font-weight: 700 !important;
}

/* Info Nodes Grid */
.server-info-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}

.info-node {
  background: #f8fafc;
  padding: 16px 20px;
  border-radius: 14px;
  display: flex;
  align-items: center;
  gap: 14px;
  border: 1px solid var(--c2-border);
  transition: all 0.2s ease;
}

.info-node:hover {
  background: #ffffff;
  border-color: var(--c2-accent);
}

.node-icon {
  font-size: 22px;
  color: var(--c2-accent);
}

.node-label {
  font-size: 12px;
  color: #64748b;
  font-weight: 600;
}

.node-value {
  font-size: 14px;
  color: var(--c2-text-title);
  font-weight: 700;
}

/* Resource Gauges */
.resource-gauges {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.gauge-item {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.gauge-label {
  display: flex;
  justify-content: space-between;
  font-size: 14px;
  color: #64748b;
  font-weight: 600;
}

.gauge-label .percent {
  color: var(--c2-text-title);
  font-weight: 700;
}

.gauge-track {
  height: 8px;
  background: #f1f5f9;
  border-radius: 4px;
  overflow: hidden;
}

.gauge-bar {
  height: 100%;
  border-radius: 4px;
  transition: width 1s ease-in-out;
}

.cyan-glow { background: #6366f1; }
.purple-glow { background: #a855f7; }
.emerald-glow { background: #10b981; }

.mr-1 { margin-right: 4px; }
.mr-2 { margin-right: 8px; }
</style>
