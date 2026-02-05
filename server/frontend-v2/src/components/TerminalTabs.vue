<template>
  <div class="terminal-tabs-container">
    <div class="tabs-header">
      <el-tabs
        v-model="activeTabName"
        type="border-card"
        closable
        @tab-remove="handleTabRemove"
        class="terminal-tabs"
      >
        <el-tab-pane
          v-for="tab in tabs"
          :key="tab.name"
          :label="tab.title"
          :name="tab.name"
        >
          <template #label>
            <span class="tab-label">
              <el-icon><Monitor /></el-icon>
              {{ tab.title }}
            </span>
          </template>
        </el-tab-pane>
      </el-tabs>
      <el-button
        type="primary"
        :icon="Plus"
        circle
        size="small"
        @click="addNewTab"
        class="add-tab-btn"
        title="鏂板缓缁堢"
      />
    </div>

    <!-- Terminal Content Area -->
    <div class="terminal-content">
      <!-- Debug: Show tab count -->
      <div v-if="tabs.length === 0" style="padding: 40px; text-align: center; color: #666;">
        Loading terminal... If you see this, onMounted hasn't run yet.
      </div>
      
      <div
        v-for="tab in tabs"
        :key="tab.name"
        v-show="activeTabName === tab.name"
        class="terminal-instance"
      >
        <div class="terminal-box">
          <!-- WebTerminal acts as the Output Display -->
          <div class="terminal-display-wrapper">
             <!-- PTY is only enabled for "Live Shell" tabs if we choose to add a button for it.
                  For now, let's keep the default shell as non-PTY (Legacy) unless explicitly requested.
                  OR, upgraded to PTY if available. 
                  Let's assume Tab 1 is always Legacy (Command/Response), 
                  and user can click "New PTY Tab" to open a real shell. -->
             <WebTerminal 
               :ref="el => setTerminalRef(tab.name, el)"
               :socket="socket"
               :client-id="clientId"
               :allow-p-t-y="tab.isPTY" 
             />
          </div>
          
          <!-- Input Area (Only needed for Legacy Tabs. PTY has direct xterm input) -->
          <el-input
            v-if="!tab.isPTY"
            v-model="tab.input"
            placeholder="杈撳叆 Shell 鍛戒护骞跺洖杞?.."
            @keyup.enter="sendCommand(tab)"
            :disabled="tab.submitting"
            class="terminal-input-bar"
          >
            <template #prefix>
              <el-icon><Right /></el-icon>
            </template>
          </el-input>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, defineProps, defineExpose } from 'vue'
import { Monitor, Plus, Right } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import api from '../api/index'
import WebTerminal from './WebTerminal.vue'

const props = defineProps({
  clientId: {
    type: String,
    required: true
  },
  clientInfo: {
    type: Object,
    default: null
  },
  socket: {
    type: Object,
    default: null
  }
})

// Tab management
const tabs = ref([])
const activeTabName = ref('')
let tabCounter = 0

// Terminal refs
const terminalRefs = reactive({})

const setTerminalRef = (name, el) => {
  if (el) {
    terminalRefs[name] = el
  }
}

// Global Message Handler (Called by ClientDetail)
const handleSocketMessage = (event) => {
    // Broadcast to all active terminals for now
    // Since we don't have session routing in the WebTerminal logic yet.
    Object.values(terminalRefs).forEach(termComp => {
        if (termComp && termComp.handleSocketMessage) {
            termComp.handleSocketMessage(event)
        }
    })
}

// Create initial tab
const createTab = (isPTY = false) => {
  tabCounter++
  const sessionId = `session-${Date.now()}-${tabCounter}`
  return {
    name: sessionId,
    title: isPTY ? `Interactive PTY ${tabCounter}` : `Shell ${tabCounter}`,
    sessionId: sessionId,
    isPTY: isPTY,
    input: '',
    submitting: false
  }
}

const addNewTab = () => {
    // Default to Legacy for now, or add a Dropdown to choose. 
    // Let's make "New Tab" button default to PTY if user holds Shift? 
    // Or just alternating? 
    // Let's default to PTY for better UX if backend supports it.
    const newTab = createTab(true) // Default to PTY
    tabs.value.push(newTab)
    activeTabName.value = newTab.name
}

const handleTabRemove = (targetName) => {
  if (tabs.value.length === 1) {
    ElMessage.warning('至少保留一个终端')
    return
  }
  
  const index = tabs.value.findIndex(tab => tab.name === targetName)
  if (index !== -1) {
    tabs.value.splice(index, 1)
    delete terminalRefs[targetName]
    if (activeTabName.value === targetName) {
      activeTabName.value = tabs.value[Math.max(0, index - 1)].name
    }
  }
}

const sendCommand = async (tab) => {
  if (!tab.input.trim()) return
  
  const cmd = tab.input
  tab.input = ''
  tab.submitting = true
  
  // Local echo is confusing if using xterm and we don't control the cursor perfectly.
  // But let's assume valid output comes from server.
  // Maybe valid shell output includes the command echoing? 
  // If not, we might want to manually write it:
  // if (terminalRefs[tab.name]) terminalRefs[tab.name].term.writeln(`> ${cmd}`) 
  // (We can't access term directly easily unless we expose it or use handleSocketMessage to fake it)

  try {
    await api.post('/api/cmd', {
      uuid: props.clientId,
      cmd: cmd,
      session_id: tab.sessionId 
    })
  } catch (e) {
    ElMessage.error('鍛戒护涓嬪彂澶辫触')
  } finally {
    tab.submitting = false
  }
}

onMounted(() => {
  console.log('[TerminalTabs] onMounted called, creating first tab...')
  // First tab defaults to PTY interactive shell
  const ptyTab = createTab(true)
  ptyTab.title = "Interactive Shell"
  tabs.value.push(ptyTab)
  activeTabName.value = ptyTab.name
  console.log('[TerminalTabs] First tab created, tabs:', tabs.value)
})

// Expose for parent
defineExpose({ handleSocketMessage })
</script>

<style scoped>
.terminal-tabs-container {
  height: 100%;
  display: flex;
  flex-direction: column;
  background-color: var(--bg-color);
}

.tabs-header {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 20px;
  background-color: var(--card-bg);
  border-bottom: 1px solid var(--border-color);
}

.terminal-tabs {
  flex: 1;
}

:deep(.el-tabs__header) {
  margin: 0;
  border-bottom: none;
}

:deep(.el-tabs__content) {
  display: none;
}

.tab-label {
  display: flex;
  align-items: center;
  gap: 5px;
}

.terminal-content {
  flex: 1;
  overflow: hidden;
  position: relative;
}

.terminal-instance {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  padding: 0;
}

.terminal-box {
  height: 100%;
  background-color: #000000;
  padding: 0 !important;
  border-radius: 0;
  border: none;
  display: flex;
  flex-direction: column;
  gap: 0;
}

.terminal-display-wrapper {
  flex: 1;
  overflow: hidden; /* xterm needs this */
  position: relative; /* xterm fit addon usually needs explicit size */
}

/* Ensure WebTerminal fills the wrapper */
:deep(.terminal-wrapper) {
    height: 100%;
}

.terminal-input-bar {
  /* Style overrides for the input bar to match terminal theme */
  --el-input-bg-color: #161b22;
  --el-input-text-color: #c9d1d9;
  --el-input-border-color: #30363d;
}
</style>


