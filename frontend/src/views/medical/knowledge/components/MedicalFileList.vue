<template>
  <div class="medical-file-list">
    <!-- 操作栏 -->
    <div class="file-toolbar">
      <div class="search-row">
        <t-input
          v-model="searchKeyword"
          placeholder="搜索文件名..."
          clearable
          style="width: 220px"
          @enter="handleSearch"
          @clear="handleSearch"
        />
        <t-select
          v-model="filterParseStatus"
          placeholder="处理状态"
          clearable
          style="width: 130px"
          @change="handleSearch"
        >
          <t-option label="处理成功" value="completed" />
          <t-option label="处理中" value="processing" />
          <t-option label="处理失败" value="failed" />
        </t-select>
        <t-button theme="default" @click="handleSearch">搜索</t-button>
        <t-button variant="outline" @click="handleReset">重置</t-button>
      </div>
      <div class="action-row">
        <t-button theme="primary" @click="openUploadDialog">
          <template #icon><t-icon name="upload" /></template>
          上传文件
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
      <!-- 文件类型图标 -->
      <template #type="{ row }">
        <span class="file-type-tag">
          <t-icon :name="fileTypeIcon(row.file_type)" size="14px" />
          {{ (row.file_type || '').toUpperCase() }}
        </span>
      </template>

      <!-- 文件大小 -->
      <template #size="{ row }">
        {{ formatFileSize(row.file_size) }}
      </template>

      <!-- 处理状态 -->
      <template #status="{ row }">
        <t-tag :theme="statusTheme(row.parse_status)" variant="light">
          {{ statusText(row.parse_status) }}
        </t-tag>
      </template>

      <!-- 操作列 -->
      <template #operation="{ row }">
        <t-space size="small">
          <t-link
            v-if="isParsed(row.parse_status)"
            theme="primary"
            hover="color"
            @click="openDetail(row)"
          >
            详情
          </t-link>
          <t-popconfirm
            content="确认删除该文件吗？对应解析数据也将被删除"
            @confirm="handleDelete(row.id)"
          >
            <t-link theme="danger" hover="color">删除</t-link>
          </t-popconfirm>
        </t-space>
      </template>
    </t-table>

    <!-- 上传弹窗 -->
    <MedicalFileUploadDialog
      v-model="uploadVisible"
      :document-kb-id="documentKbId"
      @saved="handleSaved"
    />

    <!-- 文件详情（点击详情打开） -->
    <MedicalFileDetail
      v-if="detailVisible"
      v-model="detailVisible"
      :knowledge-id="detailKnowledgeId"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { listKnowledgeFiles, batchDeleteKnowledge } from '@/api/knowledge-base/index'
import MedicalFileUploadDialog from './MedicalFileUploadDialog.vue'
import MedicalFileDetail from './MedicalFileDetail.vue'

const props = defineProps<{
  documentKbId: string
  category: string
}>()

// ====== 搜筛 ======
const searchKeyword = ref('')
const filterParseStatus = ref('')

function handleSearch() {
  pagination.current = 1
  fetchData()
}

function handleReset() {
  searchKeyword.value = ''
  filterParseStatus.value = ''
  pagination.current = 1
  fetchData()
}

// ====== 表格 ======
const loading = ref(false)
const tableData = ref<any[]>([])

const columns = [
  { colKey: 'file_name', title: '文件名称', width: 200, ellipsis: true },
  { colKey: 'type',      title: '类型', width: 80, cell: 'type' },
  { colKey: 'size',      title: '大小', width: 80, cell: 'size' },
  { colKey: 'status',    title: '处理状态', width: 110, cell: 'status' },
  { colKey: 'created_at',title: '创建时间', width: 160 },
  { colKey: 'operation', title: '操作', width: 120, cell: 'operation' },
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

// ====== 文件类型图标 ======
function fileTypeIcon(ft: string): string {
  const t = (ft || '').toLowerCase()
  if (t === 'pdf')  return 'file-pdf'
  if (t === 'xlsx' || t === 'xls' || t === 'csv') return 'file-excel'
  if (t === 'docx' || t === 'doc') return 'file-word'
  if (t === 'txt')  return 'file-text'
  return 'file'
}

function formatFileSize(bytes: number): string {
  if (!bytes) return '--'
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}

// ====== 处理状态 ======
const inFlight = new Set(['pending', 'processing', 'finalizing'])
function isParsed(s: string) { return s === 'completed' }
function isFailed(s: string) { return s === 'failed' || s === 'cancelled' }

function statusTheme(s: string) {
  if (isParsed(s))  return 'success'
  if (isFailed(s))  return 'danger'
  return 'warning'  // pending / processing / finalizing
}

function statusText(s: string) {
  if (isParsed(s))        return '处理成功'
  if (isFailed(s))        return '处理失败'
  if (s === 'finalizing') return '处理中'
  if (s === 'deleting')   return '删除中'
  return '处理中'
}

// ====== 弹窗 ======
const uploadVisible = ref(false)

function openUploadDialog() {
  uploadVisible.value = true
}

const detailVisible = ref(false)
const detailKnowledgeId = ref('')

function openDetail(row: any) {
  detailKnowledgeId.value = row.id
  detailVisible.value = true
}

// ====== 删除 ======
async function handleDelete(id: string) {
  try {
    await batchDeleteKnowledge(props.documentKbId, [id])
    MessagePlugin.success('删除成功')
    fetchData()
  } catch (err: any) {
    MessagePlugin.error(err?.response?.data?.message || err?.message || '删除失败')
  }
}

function handleSaved() {
  uploadVisible.value = false
  fetchData()
}

// ====== 数据加载 ======
async function fetchData() {
  loading.value = true
  try {
    const res = await listKnowledgeFiles(props.documentKbId, {
      page: pagination.current,
      page_size: pagination.pageSize,
      keyword: searchKeyword.value || undefined,
      parse_status: filterParseStatus.value || undefined,
    } as any)

    if (res.success) {
      // API 返回: { data: [...], total: N, page: 1, page_size: 20 }
      const list = Array.isArray(res.data) ? res.data : (res.data?.data || [])
      tableData.value = list
      pagination.total = res.total || list.length
    }
  } catch (err: any) {
    MessagePlugin.error(err?.response?.data?.message || err?.message || '加载失败')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchData()
})

defineExpose({ fetchData })
</script>

<style scoped lang="less">
.medical-file-list {
  background: var(--td-bg-color-container);
  border-radius: 8px;
}

.file-toolbar {
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

.file-type-tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  color: var(--td-text-color-secondary);
}
</style>
