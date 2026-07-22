<template>
  <div class="medical-kb-content">
    <!-- 搜索栏 -->
    <div class="search-row">
      <t-input v-model="keyword" placeholder="搜索名称..." clearable style="width:240px" @enter="fetchData" @clear="fetchData" />
      <t-select v-model="filterType" placeholder="文件类型" clearable style="width:140px" @change="fetchData">
        <t-option value="Q&A" label="Q&A" />
        <t-option value="pdf" label="PDF" />
        <t-option value="docx" label="Word" />
        <t-option value="md" label="Markdown" />
        <t-option value="txt" label="文本" />
        <t-option value="xlsx" label="Excel" />
      </t-select>
      <t-select v-model="filterStatus" placeholder="处理状态" clearable style="width:130px" @change="fetchData">
        <t-option value="completed" label="处理成功" />
        <t-option value="processing" label="处理中" />
        <t-option value="failed" label="处理失败" />
      </t-select>
      <t-select v-model="filterEnabled" placeholder="启用状态" clearable style="width:120px" @change="fetchData">
        <t-option value="true" label="启用" />
        <t-option value="false" label="停用" />
      </t-select>
      <t-button theme="primary" @click="fetchData">搜索</t-button>
      <t-button variant="outline" @click="resetFilters">重置</t-button>
    </div>
    <div class="action-row">
      <t-button theme="primary" @click="openUpload">
        <template #icon><t-icon name="upload" /></template>上传文件
      </t-button>
      <t-button variant="outline" @click="openQACreate">
        <template #icon><t-icon name="add" /></template>单个新建 Q&A
      </t-button>
      <t-button variant="outline" @click="openQAImport">批量上传 Q&A</t-button>
    </div>

    <!-- 统一表格 -->
    <t-table
      :data="tableData"
      :columns="columns"
      :loading="loading"
      :pagination="pagination"
      row-key="uid"
      hover
      @page-change="handlePageChange"
    >
      <template #type="{ row }">
        <span class="type-tag">{{ row.typeLabel }}</span>
      </template>
      <template #hit_count="{ row }">
        <span style="color:var(--td-text-color-placeholder)">{{ row.hitCount }}</span>
      </template>
      <template #status="{ row }">
        <t-tag :theme="statusTheme(row.status)" variant="light" size="small">
          {{ statusText(row.status) }}
        </t-tag>
      </template>
      <template #operation="{ row }">
        <t-space size="small">
          <t-link v-if="!isDeleting(row.status)" theme="primary" hover="color" @click="openDetail(row)">详情</t-link>
          <t-link v-else theme="default" disabled>详情</t-link>
          <t-popconfirm v-if="!isDeleting(row.status)" content="确认删除？" @confirm="handleDelete(row)">
            <t-link theme="danger" hover="color">删除</t-link>
          </t-popconfirm>
          <t-link v-else theme="default" disabled>删除中</t-link>
        </t-space>
      </template>
    </t-table>

    <!-- 文件上传弹窗 -->
    <MedicalFileUploadDialog v-model="uploadVisible" :document-kb-id="configItem?.document_kb_id||''" @saved="fetchData" />

    <!-- 文件详情弹窗 -->
    <MedicalFileDetail v-if="detailVisible" v-model="detailVisible" :knowledge-id="detailKnowledgeId" />

    <!-- Q&A 弹窗 -->
    <MedicalFAQFormDialog v-model="qaDialogVisible" :mode="qaMode" :entry="qaEntry" :faq-kb-id="configItem?.faq_kb_id||''" @saved="fetchData" />

    <!-- Q&A 批量导入 -->
    <MedicalFAQImportDialog v-model="qaImportVisible" :faq-kb-id="configItem?.faq_kb_id||''" @saved="fetchData" />
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onBeforeUnmount, watch } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { useAuthStore } from '@/stores/auth'
import type { MedicalKBConfigItem } from '@/api/medical/knowledge-base/index'
import { listKnowledgeFiles, batchDeleteKnowledge } from '@/api/knowledge-base/index'
import { listFAQEntries, deleteFAQEntries } from '@/api/medical/knowledge-base/index'
import MedicalFileUploadDialog from './components/MedicalFileUploadDialog.vue'
import MedicalFileDetail from './components/MedicalFileDetail.vue'
import MedicalFAQFormDialog from './components/MedicalFAQFormDialog.vue'
import MedicalFAQImportDialog from './components/MedicalFAQImportDialog.vue'

const props = defineProps<{ configItem: MedicalKBConfigItem | null }>()
const emit = defineEmits<{
  (e: 'latest-updated', value: string): void
}>()
const auth = useAuthStore()
const userName = computed(() => auth.user?.username || auth.user?.name || '-')

// ── 搜索/筛选 ──
const keyword = ref('')
const filterType = ref('')
const filterStatus = ref('')
const filterEnabled = ref('')
const loading = ref(false)

function resetFilters() { keyword.value = ''; filterType.value = ''; filterStatus.value = ''; filterEnabled.value = ''; fetchData() }

// ── 统一数据 ──
interface UnifiedItem {
  uid: string
  name: string
  hitCount: number
  type: string       // Q&A / pdf / docx / md / txt / xlsx ...
  typeLabel: string  // Q&A / PDF / DOCX / MD / TXT / XLSX ...
  status: string
  updatedBy: string
  updatedAt: string
  createdBy: string
  createdAt: string
  raw: any          // 原始数据，详情/删除时用
  source: 'file' | 'qa'
}
const tableData = ref<UnifiedItem[]>([])
const pendingDeletingIds = ref(new Set<string>())
let deletePollTimer: ReturnType<typeof setTimeout> | null = null

const columns = [
  { colKey: 'name', title: '名称', width: 240, ellipsis: true },
  { colKey: 'hit_count', title: '命中次数', width: 90, cell: 'hit_count' },
  { colKey: 'type', title: '文件类型', width: 100, cell: 'type' },
  { colKey: 'status', title: '处理状态', width: 100, cell: 'status' },
  { colKey: 'updatedBy', title: '更新人', width: 100 },
  { colKey: 'updatedAt', title: '更新时间', width: 160 },
  { colKey: 'createdBy', title: '创建人', width: 100 },
  { colKey: 'createdAt', title: '创建时间', width: 160 },
  { colKey: 'operation', title: '操作', width: 130, cell: 'operation' },
]

const pagination = reactive({ current: 1, pageSize: 20, total: 0, showJumper: true, showPageSize: true, pageSizeOptions: [10, 20, 50] })
function handlePageChange(p: {current:number;pageSize:number}) { pagination.current = p.current; pagination.pageSize = p.pageSize; fetchData() }

// ── 文件弹窗 ──
const uploadVisible = ref(false)
const detailVisible = ref(false)
const detailKnowledgeId = ref('')
function openUpload() { uploadVisible.value = true }
function openDetail(row: UnifiedItem) {
  if (row.source === 'file') { detailKnowledgeId.value = row.raw.id; detailVisible.value = true }
  else { openQADialog('detail', row.raw) }
}

// ── Q&A 弹窗 ──
const qaDialogVisible = ref(false)
const qaMode = ref<'create'|'detail'>('create')
const qaEntry = ref<any>(null)
function openQACreate() { qaMode.value = 'create'; qaEntry.value = null; qaDialogVisible.value = true }
function openQADialog(mode:'create'|'detail', entry?:any) { qaMode.value = mode; qaEntry.value = entry || null; qaDialogVisible.value = true }
const qaImportVisible = ref(false)
function openQAImport() { qaImportVisible.value = true }

// ── 状态辅助 ──
function statusTheme(s: string) {
  if (s==='completed'||s==='success') return 'success'
  if (s==='processing'||s==='pending') return 'warning'
  if (s==='failed') return 'danger'
  if (s==='deleting') return 'warning'
  return 'default'
}
function statusText(s: string) {
  if (s==='completed'||s==='success') return '处理成功'
  if (s==='processing'||s==='pending'||s==='finalizing') return '处理中'
  if (s==='failed'||s==='cancelled') return '处理失败'
  if (s==='deleting') return '删除中'
  return s || '未知'
}
function isDeleting(s: string) { return s === 'deleting' }
function isProcessingStatus(s: string) {
  return s === 'pending' || s === 'processing' || s === 'finalizing' || s === 'deleting'
}
function normalizeFAQStatus(entry: any) {
  return entry?.parse_status || entry?.status || 'completed'
}

function markRowDeleting(row: UnifiedItem) {
  pendingDeletingIds.value.add(row.uid)
  tableData.value = tableData.value.map(item => item.uid === row.uid ? { ...item, status: 'deleting' } : item)
}

function scheduleDeletePoll() {
  if (deletePollTimer || pendingDeletingIds.value.size === 0) return
  deletePollTimer = setTimeout(() => {
    deletePollTimer = null
    fetchData()
  }, 2000)
}

// ── 类型标签 ──
function typeLabel(ft: string): string {
  const t = (ft||'').toLowerCase()
  return t === 'qa' ? 'Q&A' : t.toUpperCase()
}

function formatDateTime(value?: string) {
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

function emitLatestUpdated(items: UnifiedItem[]) {
  const latest = items
    .map((item) => item.updatedAt || item.createdAt)
    .filter(Boolean)
    .sort((a, b) => new Date(b).getTime() - new Date(a).getTime())[0]
  emit('latest-updated', latest || '')
}

// ── 删除 ──
async function handleDelete(row: UnifiedItem) {
  try {
    if (row.source === 'file') {
      await batchDeleteKnowledge(props.configItem!.document_kb_id, [row.raw.id])
    } else {
      await deleteFAQEntries(props.configItem!.faq_kb_id, [row.raw.id])
    }
    markRowDeleting(row)
    MessagePlugin.success('删除任务已提交')
    scheduleDeletePoll()
  } catch(e:any) { MessagePlugin.error(e?.message||'删除失败') }
}

// ── 数据加载 ──
async function fetchData() {
  loading.value = true
  const all: UnifiedItem[] = []

  try {
    // 文件
    if (props.configItem?.document_kb_id) {
      const p: any = { page: 1, page_size: 500 }
      if (keyword.value) p.keyword = keyword.value
      const r = await listKnowledgeFiles(props.configItem.document_kb_id, p)
      if (r.success) {
        const items = Array.isArray(r.data) ? r.data : (r.data?.data || [])
        for (const f of items) {
          const ft = (f.file_type||'').toLowerCase()
          all.push({
            uid: 'file-'+f.id, name: f.file_name||f.title||'', hitCount: 0,
            type: ft, typeLabel: typeLabel(ft),
            status: f.parse_status||'',
            updatedBy: userName.value||'-', updatedAt: formatDateTime(f.updated_at),
            createdBy: userName.value||'-', createdAt: formatDateTime(f.created_at),
            raw: f, source: 'file',
          })
        }
      }
    }

    // Q&A
    if (props.configItem?.faq_kb_id) {
      const p: any = { page: 1, page_size: 500 }
      if (keyword.value) p.keyword = keyword.value
      const r = await listFAQEntries(props.configItem.faq_kb_id, p)
      if (r.success) {
        const items = r.data?.data || r.data?.entries || r.data || []
        const arr = Array.isArray(items) ? items : []
        for (const q of arr) {
          all.push({
            uid: 'qa-'+q.id, name: q.standard_question||'', hitCount: 0,
            type: 'qa', typeLabel: 'Q&A',
            status: normalizeFAQStatus(q),
            updatedBy: userName.value||'-', updatedAt: formatDateTime(q.updated_at),
            createdBy: userName.value||'-', createdAt: formatDateTime(q.created_at),
            raw: q, source: 'qa',
          })
        }
      }
    }
  } catch(e) { console.error(e) }

  // 前端筛选
  const existingUIDs = new Set(all.map(item => item.uid))
  pendingDeletingIds.value.forEach(uid => {
    if (!existingUIDs.has(uid)) pendingDeletingIds.value.delete(uid)
  })

  let filtered = all.map(item => pendingDeletingIds.value.has(item.uid) ? { ...item, status: 'deleting' } : item)
  if (filterType.value) {
    const ft = filterType.value === 'Q&A' ? 'qa' : filterType.value.toLowerCase()
    filtered = filtered.filter(i => i.type === ft)
  }
  if (filterStatus.value) {
    const fs = filterStatus.value
    filtered = filtered.filter(i => {
      if (i.source === 'file') return fs === 'processing' ? isProcessingStatus(i.status) : i.status === fs
      return (fs==='completed' && (i.status==='completed' || i.status==='success')) || (fs==='failed' && i.status==='failed') || (fs==='processing' && isProcessingStatus(i.status))
    })
  }
  if (filterEnabled.value) {
    const fe = filterEnabled.value === 'true'
    filtered = filtered.filter(i => {
      if (i.source === 'qa') return i.raw.is_enabled === fe
      return true // 文件没有启用状态，保留
    })
  }

  emitLatestUpdated(all)
  pagination.total = filtered.length
  const start = (pagination.current-1)*pagination.pageSize
  tableData.value = filtered.slice(start, start+pagination.pageSize)
  loading.value = false
  scheduleDeletePoll()
}

// 卡片切换时重新加载
watch(() => props.configItem, () => {
  if (props.configItem?.document_kb_id || props.configItem?.faq_kb_id) {
    resetFilters()
    fetchData()
  }
})
onMounted(() => { fetchData() })
onBeforeUnmount(() => {
  if (deletePollTimer) clearTimeout(deletePollTimer)
})
defineExpose({ reload: fetchData })
</script>

<style scoped lang="less">
.medical-kb-content { margin-top: 4px; }
.search-row { display: flex; gap: 10px; align-items: center; margin-bottom: 12px; }
.action-row { display: flex; gap: 10px; margin-bottom: 16px; }
.type-tag { font-size: 11px; background: var(--td-bg-color-component); padding: 2px 8px; border-radius: 4px; color: var(--td-text-color-secondary); }
</style>
