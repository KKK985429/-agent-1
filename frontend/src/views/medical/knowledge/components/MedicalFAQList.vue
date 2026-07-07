<template>
  <div class="medical-faq-list">
    <!-- 操作栏 -->
    <div class="faq-toolbar">
      <div class="search-row">
        <t-input
          v-model="searchKeyword"
          placeholder="搜索问题..."
          clearable
          style="width: 260px"
          @enter="handleSearch"
          @clear="handleSearch"
        />
        <t-select
          v-model="filterEnabled"
          placeholder="启用状态"
          clearable
          style="width: 120px"
          @change="handleSearch"
        >
          <t-option label="启用" value="true" />
          <t-option label="停用" value="false" />
        </t-select>
        <t-button theme="default" @click="handleSearch">搜索</t-button>
        <t-button variant="outline" @click="handleReset">重置</t-button>
      </div>
      <div class="action-row">
        <t-button theme="primary" @click="openCreateDialog">
          <template #icon><t-icon name="add" /></template>
          单个新建 Q&A
        </t-button>
        <t-button variant="outline" @click="openImportDialog">
          <template #icon><t-icon name="upload" /></template>
          批量上传 Q&A
        </t-button>
      </div>
    </div>

    <!-- 表格 -->
    <t-table
      :data="tableData"
      :columns="columns"
      :loading="loading"
      :pagination="pagination"
      row-key="id"
      hover
      @page-change="handlePageChange"
    >
      <!-- 处理状态 -->
      <template #status="{ row }">
        <t-tag
          :theme="statusTheme(row.status)"
          variant="light"
        >
          {{ statusLabel(row.status) }}
        </t-tag>
      </template>

      <!-- 启用状态 -->
      <template #enabled="{ row }">
        <t-tag
          :theme="statusTagTheme(row)"
          variant="light"
        >
          {{ row.is_enabled ? '启用' : '停用' }}
        </t-tag>
      </template>

      <!-- 操作列 -->
      <template #operation="{ row }">
        <t-space size="small">
          <t-link
            v-if="row.status === 'success'"
            theme="primary"
            hover="color"
            @click="openDetailDialog(row)"
          >
            详情
          </t-link>
          <t-link
            v-if="row.status === 'success'"
            :theme="row.is_enabled ? 'warning' : 'success'"
            hover="color"
            @click="toggleEnabled(row)"
          >
            {{ row.is_enabled ? '停用' : '启用' }}
          </t-link>
          <t-popconfirm content="确认删除该 Q&A 吗？" @confirm="handleDelete(row.id)">
            <t-link theme="danger" hover="color">删除</t-link>
          </t-popconfirm>
        </t-space>
      </template>
    </t-table>

    <!-- 新建/详情弹窗 -->
    <MedicalFAQFormDialog
      v-model="dialogVisible"
      :mode="dialogMode"
      :entry="currentEntry"
      :faq-kb-id="faqKbId"
      @saved="handleSaved"
    />

    <!-- 批量上传弹窗 -->
    <MedicalFAQImportDialog
      v-model="importVisible"
      :faq-kb-id="faqKbId"
      @saved="handleSaved"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import {
  listFAQEntries,
  updateFAQEntryFieldsBatch,
  deleteFAQEntries,
} from '@/api/medical/knowledge-base/index'
import MedicalFAQFormDialog from './MedicalFAQFormDialog.vue'
import MedicalFAQImportDialog from './MedicalFAQImportDialog.vue'

// ---------- 处理状态 ----------
// 前端维护的处理状态（因为 WeKnora FAQ 没有原生 parsestatus）
// - pending:  创建请求已发出，等待确认
// - success:  接口返回成功
// - failed:   接口返回失败或导入部分失败
interface FAQEntryWithStatus {
  id: number
  standard_question: string
  answers: string[]
  is_enabled: boolean
  tag_name: string
  created_at: string
  updated_at: string
  status: 'pending' | 'success' | 'failed'
  // 用户创建时选择的启用状态（成功后同步）
  _chosen_enabled?: boolean
}

const props = defineProps<{
  faqKbId: string
  category: string
}>()

// ---------- 搜筛 ----------
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
const tableData = ref<FAQEntryWithStatus[]>([])

const columns = [
  { colKey: 'standard_question', title: '问题名称', width: 220, ellipsis: true },
  { colKey: 'type', title: '类型', width: 60 },
  { colKey: 'status', title: '处理状态', width: 110, cell: 'status' },
  { colKey: 'enabled', title: '启用状态', width: 100, cell: 'enabled' },
  { colKey: 'updated_at', title: '更新时间', width: 170 },
  { colKey: 'operation', title: '操作', width: 200, cell: 'operation' },
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

// ---------- 处理状态辅助 ----------
function statusTheme(status: string) {
  switch (status) {
    case 'success': return 'success'
    case 'pending': return 'warning'
    case 'failed': return 'danger'
    default: return 'default'
  }
}

function statusLabel(status: string) {
  switch (status) {
    case 'success': return '处理成功'
    case 'pending': return '处理中'
    case 'failed': return '处理失败'
    default: return status
  }
}

function statusTagTheme(row: FAQEntryWithStatus) {
  if (row.status !== 'success') return 'default'
  return row.is_enabled ? 'success' : 'default'
}

// ---------- 弹窗 ----------
const dialogVisible = ref(false)
const dialogMode = ref<'create' | 'detail'>('create')
const currentEntry = ref<FAQEntryWithStatus | null>(null)

function openCreateDialog() {
  dialogMode.value = 'create'
  currentEntry.value = null
  dialogVisible.value = true
}

function openDetailDialog(row: FAQEntryWithStatus) {
  if (row.status !== 'success') {
    MessagePlugin.warning('处理中和处理失败的 Q&A 无法查看详情')
    return
  }
  dialogMode.value = 'detail'
  currentEntry.value = row
  dialogVisible.value = true
}

const importVisible = ref(false)

function openImportDialog() {
  importVisible.value = true
}

function handleSaved() {
  fetchData()
}

// ---------- 启停 ----------
async function toggleEnabled(row: FAQEntryWithStatus) {
  if (row.status !== 'success') return
  try {
    await updateFAQEntryFieldsBatch(props.faqKbId, {
      by_id: { [row.id]: { is_enabled: !row.is_enabled } },
    })
    MessagePlugin.success(row.is_enabled ? '已停用' : '已启用')
    fetchData()
  } catch (err: any) {
    const msg = err?.response?.data?.message || err?.message || '操作失败'
    MessagePlugin.error(msg)
  }
}

// ---------- 删除 ----------
async function handleDelete(id: number) {
  try {
    await deleteFAQEntries(props.faqKbId, [id])
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
    const res = await listFAQEntries(props.faqKbId, {
      page: pagination.current,
      page_size: pagination.pageSize,
      keyword: searchKeyword.value || undefined,
    })

    if (res.success) {
      // FAQ list returns res.data.data (entries array) and res.data.total
      const pageData = (res.data || {}) as { data: any[]; total: number }
      const rawList = (pageData.data || pageData.entries || []) as any[]
      const items: FAQEntryWithStatus[] = rawList.map((entry: any) => ({
        ...entry,
        id: entry.id ?? entry.seq_id ?? entry.ID,
        // 从后端拿到的都是已持久化的数据，标记为 success
        status: entry.status || 'success',
        _chosen_enabled: entry.is_enabled,
      }))

      // 前端过滤启用状态（后端 FAQ 接口暂不支持 is_enabled 筛选参数）
      let filtered = items
      if (filterEnabled.value === 'true') {
        filtered = items.filter((e) => e.is_enabled)
      } else if (filterEnabled.value === 'false') {
        filtered = items.filter((e) => !e.is_enabled)
      }

      // 处理中/失败的行强制显示为禁用
      filtered.forEach((e) => {
        if (e.status !== 'success') {
          e.is_enabled = false
        }
      })

      tableData.value = filtered.map((entry) => ({
        ...entry,
        type: 'Q&A',
      }))
      pagination.total = pageData.total ?? filtered.length
    }
  } catch (err: any) {
    const msg = err?.response?.data?.message || err?.message || '加载失败'
    MessagePlugin.error(msg)
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  if (props.faqKbId) {
    fetchData()
  }
})

defineExpose({ fetchData })
</script>

<style scoped lang="less">
.medical-faq-list {
  background: var(--td-bg-color-container);
  border-radius: 8px;
}

.faq-toolbar {
  padding: 16px 0 0 0;
}

.search-row {
  display: flex;
  gap: 10px;
  align-items: center;
  margin-bottom: 12px;
}

.action-row {
  display: flex;
  gap: 10px;
  margin-bottom: 16px;
}
</style>
