<template>
  <div class="medical-department-page">
    <!-- 页面标题 + 说明 -->
    <div class="header" style="--wails-draggable: drag">
      <div class="header-title" style="--wails-draggable: drag">
        <div class="title-row" style="--wails-draggable: drag">
          <h2 style="--wails-draggable: drag">知识库管理</h2>
        </div>
        <p class="header-subtitle" style="--wails-draggable: drag">查看并管理你的专属数据源</p>
      </div>
    </div>

    <!-- 顶部知识库卡片 -->
    <div class="kb-cards">
      <div
        v-for="card in kbCards"
        :key="card.key"
        class="kb-card"
        :class="{ active: activeCard === card.key }"
        @click="handleCardClick(card.key)"
      >
        <div class="card-header">
          <span class="card-icon">{{ card.icon }}</span>
          <span class="card-title">{{ card.title }}</span>
        </div>
        <div class="card-meta">
          <span class="card-time-label">最近更新时间:</span>
          <span class="card-time">{{ card.latestTime || '--' }}</span>
        </div>
      </div>
    </div>

    <!-- 搜索 / 筛选区（仅科室知识库显示） -->
    <div v-if="activeCard === 'department'" class="search-bar">
      <div class="search-row">
        <t-input
          v-model="searchKeyword"
          placeholder="搜索科室名称 / 编号"
          clearable
          class="search-input"
          @enter="handleSearch"
          @clear="handleSearch"
        />
        <t-select
          v-model="filterEnabled"
          placeholder="启用状态"
          clearable
          class="filter-select"
          @change="handleSearch"
        >
          <t-option label="启用" value="true" />
          <t-option label="停用" value="false" />
        </t-select>
        <t-button theme="primary" @click="handleSearch">搜索</t-button>
        <t-button variant="outline" @click="handleReset">重置</t-button>
      </div>
      <div class="action-row">
        <t-button theme="primary" @click="openCreateDialog">
          <template #icon><t-icon name="add" /></template>
          新建科室
        </t-button>
      </div>
    </div>

    <!-- 非科室知识库 → 医疗知识库内容（文件 / Q&A） -->
    <div v-if="activeCard !== 'department'" class="medical-kb-section">
      <!-- 加载中 -->
      <div v-if="kbConfigLoading" class="loading-section">
        <t-loading text="正在加载知识库配置..." />
      </div>
      <!-- 非科室知识库内容 -->
      <MedicalKBContent
        v-else-if="currentKBConfig"
        ref="kbContentRef"
        :config-item="currentKBConfig"
        @latest-updated="handleKBLatestUpdated"
      />
      <!-- 配置获取失败 -->
      <div v-else class="placeholder-section">
        <t-icon name="error-circle" size="48px" style="color: var(--td-error-color)" />
        <p class="placeholder-text">知识库配置加载失败，请确认已配置默认模型</p>
      </div>
    </div>

    <!-- 科室列表表格 -->
    <div v-if="activeCard === 'department'" class="table-section">
      <t-table
        :data="tableData"
        :columns="columns"
        :loading="loading"
        :pagination="pagination"
        row-key="id"
        hover
        @page-change="handlePageChange"
      >
        <template #enabled="{ row }">
          <t-tag :theme="row.enabled ? 'success' : 'default'" variant="light">
            {{ row.enabled ? '启用' : '停用' }}
          </t-tag>
        </template>
        <template #operation="{ row }">
          <t-space size="small">
            <t-link theme="primary" hover="color" @click="openEditDialog(row)">编辑</t-link>
            <t-link
              :theme="row.enabled ? 'warning' : 'success'"
              hover="color"
              @click="toggleEnabled(row)"
            >
              {{ row.enabled ? '停用' : '启用' }}
            </t-link>
            <t-popconfirm content="确认删除该科室吗？" @confirm="handleDelete(row.id)">
              <t-link theme="danger" hover="color">删除</t-link>
            </t-popconfirm>
          </t-space>
        </template>
      </t-table>
    </div>

    <!-- 新建/编辑科室弹窗 -->
    <DepartmentFormDialog
      v-model="dialogVisible"
      :mode="dialogMode"
      :department="editingDepartment"
      @saved="handleSaved"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import {
  listDepartments,
  getDepartment,
  deleteDepartment,
  updateDepartment,
  type MedicalDepartment,
} from '@/api/medical/department/index'
import {
  getMedicalKBConfig,
  listFAQEntries,
  type MedicalKBConfigItem,
} from '@/api/medical/knowledge-base/index'
import { listKnowledgeFiles } from '@/api/knowledge-base/index'
import DepartmentFormDialog from './DepartmentFormDialog.vue'
import MedicalKBContent from '../knowledge/MedicalKBContent.vue'

// ---------- 知识库卡片 ----------
interface KBCard {
  key: string
  icon: string
  title: string
  latestTime: string
}

const activeCard = ref<string>('department')
const kbCards = ref<KBCard[]>([
  { key: 'department', icon: '🏥', title: '科室知识库', latestTime: '' },
  { key: 'symptom', icon: '💊', title: '症状知识库', latestTime: '' },
  { key: 'disease', icon: '🩺', title: '疾病知识库', latestTime: '' },
  { key: 'drug', icon: '💉', title: '药品知识库', latestTime: '' },
  { key: 'lab', icon: '🔬', title: '检验检查知识库', latestTime: '' },
])

// ---------- 医疗知识库配置 ----------
const kbConfigCache = ref<MedicalKBConfigItem[]>([])
const kbConfigLoading = ref(false)
const currentKBConfig = ref<MedicalKBConfigItem | null>(null)
const kbContentRef = ref()

function formatDisplayTime(value?: string) {
  if (!value) return ''
  if (/^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$/.test(value)) return value

  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value

  const pad = (n: number) => String(n).padStart(2, '0')
  return [
    date.getFullYear(),
    pad(date.getMonth() + 1),
    pad(date.getDate()),
  ].join('-') + ' ' + [
    pad(date.getHours()),
    pad(date.getMinutes()),
    pad(date.getSeconds()),
  ].join(':')
}

function updateKBCardLatestTimes(items: MedicalKBConfigItem[]) {
  for (const item of items) {
    const card = kbCards.value.find((c) => c.key === item.key)
    if (card && item.latest_updated_at) {
      card.latestTime = formatDisplayTime(item.latest_updated_at)
    }
  }
}

function handleKBLatestUpdated(value: string) {
  const card = kbCards.value.find((c) => c.key === activeCard.value)
  if (card) card.latestTime = formatDisplayTime(value)
}

function pickLatestTime(values: Array<string | undefined>) {
  const sorted = values
    .filter((value): value is string => Boolean(value))
    .sort((a, b) => new Date(b).getTime() - new Date(a).getTime())
  return sorted[0] || ''
}

function getResponseItems(res: any) {
  if (Array.isArray(res?.data)) return res.data
  if (Array.isArray(res?.data?.data)) return res.data.data
  if (Array.isArray(res?.data?.entries)) return res.data.entries
  return []
}

async function loadKBContentLatestTime(item: MedicalKBConfigItem) {
  const times: string[] = []

  if (item.document_kb_id) {
    try {
      const res = await listKnowledgeFiles(item.document_kb_id, {
        page: 1,
        page_size: 500,
      })
      for (const file of getResponseItems(res)) {
        times.push(file.updated_at || file.created_at)
      }
    } catch (err) {
      console.error(`加载${item.name}文件更新时间失败`, err)
    }
  }

  if (item.faq_kb_id) {
    try {
      const res = await listFAQEntries(item.faq_kb_id, {
        page: 1,
        page_size: 500,
      })
      for (const qa of getResponseItems(res)) {
        times.push(qa.updated_at || qa.created_at)
      }
    } catch (err) {
      console.error(`加载${item.name} Q&A 更新时间失败`, err)
    }
  }

  return pickLatestTime(times)
}

async function refreshAllKBCardLatestTimes(items: MedicalKBConfigItem[]) {
  await Promise.all(items.map(async (item) => {
    const latest = await loadKBContentLatestTime(item)
    if (!latest) return
    const card = kbCards.value.find((c) => c.key === item.key)
    if (card) card.latestTime = formatDisplayTime(latest)
  }))
}

async function refreshKBConfigCards() {
  try {
    const res = await getMedicalKBConfig()
    if (res.success && res.data?.items) {
      kbConfigCache.value = res.data.items
      const items = res.data.items as MedicalKBConfigItem[]
      updateKBCardLatestTimes(items)
      refreshAllKBCardLatestTimes(items)
    }
  } catch (err) {
    console.error('加载医疗知识库更新时间失败', err)
  }
}

async function handleCardClick(key: string) {
  activeCard.value = key
  if (key === 'department') {
    currentKBConfig.value = null
    fetchData()
  } else {
    await loadKBConfig(key)
  }
}

async function loadKBConfig(category: string) {
  // 优先用缓存
  if (kbConfigCache.value.length > 0) {
    const cached = kbConfigCache.value.find((i) => i.key === category)
    if (cached) {
      currentKBConfig.value = cached
      return
    }
  }

  kbConfigLoading.value = true
  currentKBConfig.value = null
  try {
    const res = await getMedicalKBConfig()
    if (res.success && res.data?.items) {
      kbConfigCache.value = res.data.items
      updateKBCardLatestTimes(res.data.items as MedicalKBConfigItem[])
      const item = (res.data.items as MedicalKBConfigItem[]).find(
        (i) => i.key === category,
      )
      if (item) {
        currentKBConfig.value = item
      }
    }
  } catch (err: any) {
    const msg = err?.response?.data?.message || err?.message || '加载知识库配置失败'
    MessagePlugin.error(msg)
  } finally {
    kbConfigLoading.value = false
  }
}

// ---------- 搜索 / 筛选 ----------
const searchKeyword = ref('')
const filterEnabled = ref('')

function handleSearch() {
  pagination.current = 1
  fetchData()
}

function handleReset() {
  searchKeyword.value = ''
  filterEnabled.value = ''
  pagination.current = 1
  fetchData()
}

// ---------- 表格 ----------
const loading = ref(false)
const tableData = ref<MedicalDepartment[]>([])

const columns = [
  { colKey: 'name', title: '科室名称', width: 140 },
  { colKey: 'code', title: '编号', width: 160 },
  { colKey: 'hospital_area', title: '所属院区', width: 120 },
  { colKey: 'enabled', title: '启用状态', width: 100, cell: 'enabled' },
  { colKey: 'created_by_name', title: '创建人', width: 120 },
  { colKey: 'created_at', title: '创建时间', width: 170 },
  { colKey: 'operation', title: '操作', width: 180, cell: 'operation' },
]

const pagination = reactive({
  current: 1,
  pageSize: 20,
  total: 0,
  showJumper: true,
  showPageSize: true,
  pageSizeOptions: [10, 20, 50],
})

function handlePageChange(pageInfo: { current: number; pageSize: number }) {
  pagination.current = pageInfo.current
  pagination.pageSize = pageInfo.pageSize
  fetchData()
}

// ---------- 弹窗 ----------
const dialogVisible = ref(false)
const dialogMode = ref<'create' | 'edit'>('create')
const editingDepartment = ref<MedicalDepartment | null>(null)

function openCreateDialog() {
  dialogMode.value = 'create'
  editingDepartment.value = null
  dialogVisible.value = true
}

function openEditDialog(row: MedicalDepartment) {
  dialogMode.value = 'edit'
  editingDepartment.value = row
  dialogVisible.value = true
}

function handleSaved() {
  fetchData()
}

// ---------- 操作 ----------
async function toggleEnabled(row: MedicalDepartment) {
  try {
    await updateDepartment(row.id, { enabled: !row.enabled })
    MessagePlugin.success(row.enabled ? '已停用' : '已启用')
    fetchData()
  } catch (err: any) {
    const msg = err?.response?.data?.message || err?.message || '操作失败'
    MessagePlugin.error(msg)
  }
}

async function handleDelete(id: string) {
  try {
    await deleteDepartment(id)
    MessagePlugin.success('删除成功')
    fetchData()
  } catch (err: any) {
    const msg = err?.response?.data?.message || err?.message || '删除失败'
    MessagePlugin.error(msg)
  }
}

// ---------- 数据加载 ----------
async function fetchData() {
  loading.value = true
  try {
    const res = await listDepartments({
      keyword: searchKeyword.value || undefined,
      enabled: filterEnabled.value || undefined,
      page: pagination.current,
      page_size: pagination.pageSize,
    })
    if (res.success) {
      const data = res.data
      tableData.value = (data.list || []).map((item: MedicalDepartment) => ({
        ...item,
        created_at: formatDisplayTime(item.created_at),
        updated_at: formatDisplayTime(item.updated_at),
      }))
      pagination.total = data.total || 0
      // Update card latest time
      if (tableData.value.length > 0) {
        const latest = tableData.value.reduce((max, d) =>
          d.updated_at > max ? d.updated_at : max,
          tableData.value[0].updated_at,
        )
        const card = kbCards.value.find((c) => c.key === 'department')
        if (card) card.latestTime = formatDisplayTime(latest)
      }
    }
  } catch (err: any) {
    const msg = err?.response?.data?.message || err?.message || '加载失败'
    MessagePlugin.error(msg)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchData()
  refreshKBConfigCards()
})
</script>

<style scoped lang="less">
.medical-department-page {
  padding: 24px;
  max-width: 1200px;
  margin: 0 auto;
  height: 100%;
  overflow-y: auto;
}

.header {
  margin-bottom: 20px;
}

.header-title h2 {
  font-size: 20px;
  font-weight: 600;
  margin: 0 0 6px 0;
  color: var(--td-text-color-primary);
}

.header-subtitle {
  font-size: 13px;
  color: var(--td-text-color-placeholder);
  margin: 0;
}

// ---- 卡片 ----
.kb-cards {
  display: grid;
  grid-template-columns: repeat(5, 1fr);
  gap: 16px;
  margin-bottom: 24px;
}

.kb-card {
  background: var(--td-bg-color-container);
  border: 2px solid var(--td-component-border);
  border-radius: 8px;
  padding: 16px 20px;
  cursor: pointer;
  transition: all 0.2s;

  &:hover {
    border-color: var(--td-brand-color);
  }

  &.active {
    border-color: var(--td-brand-color);
    background: var(--td-brand-color-light);
  }
}

.card-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.card-icon {
  font-size: 22px;
}

.card-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--td-text-color-primary);
}

.card-meta {
  font-size: 12px;
  color: var(--td-text-color-placeholder);
}

// ---- 搜索栏 ----
.search-bar {
  margin-bottom: 16px;
}

.search-row {
  display: flex;
  gap: 10px;
  margin-bottom: 12px;
  align-items: center;
}

.search-input {
  width: 240px;
  flex-shrink: 0;
}

.filter-select {
  width: 140px;
  flex-shrink: 0;
}

.action-row {
  display: flex;
  justify-content: flex-start;
}

// ---- 表格 ----
.table-section {
  background: var(--td-bg-color-container);
  border-radius: 8px;
  padding: 0;
}

// ---- 医疗知识库内容 ----
.medical-kb-section {
  min-height: 300px;
  overflow: visible;
}

.loading-section {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 80px 0;
}

// ---- 占位提示 ----
.placeholder-section {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 80px 0;
  gap: 16px;
}

.placeholder-text {
  font-size: 14px;
  color: var(--td-text-color-placeholder);
  margin: 0;
}
</style>
