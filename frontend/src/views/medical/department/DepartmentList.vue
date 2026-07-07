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
          <span class="card-time-label">最近更新：</span>
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

    <!-- 占位提示：其他知识库卡片（症状/疾病/药品）暂未实现 -->
    <div v-if="activeCard !== 'department'" class="placeholder-section">
      <t-icon name="build" size="48px" style="color: var(--td-text-color-placeholder)" />
      <p class="placeholder-text">该知识库模块正在开发中</p>
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
import DepartmentFormDialog from './DepartmentFormDialog.vue'

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
])

function handleCardClick(key: string) {
  activeCard.value = key
  if (key === 'department') {
    fetchData()
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
      tableData.value = data.list || []
      pagination.total = data.total || 0
      // Update card latest time
      if (tableData.value.length > 0) {
        const latest = tableData.value.reduce((max, d) =>
          d.updated_at > max ? d.updated_at : max,
          tableData.value[0].updated_at,
        )
        const card = kbCards.value.find((c) => c.key === 'department')
        if (card) card.latestTime = latest
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
})
</script>

<style scoped lang="less">
.medical-department-page {
  padding: 24px;
  max-width: 1200px;
  margin: 0 auto;
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
  grid-template-columns: repeat(4, 1fr);
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
