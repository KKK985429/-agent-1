<template>
  <t-dialog
    v-model:visible="visible"
    header="文件详情"
    :confirm-btn="null"
    :cancel-btn="'关闭'"
    :close-on-overlay-click="true"
    width="680px"
    @close="handleClose"
  >
    <div v-if="loading" class="detail-loading">
      <t-loading text="加载中..." />
    </div>

    <template v-else-if="detail">
      <!-- 基本信息 -->
      <div class="detail-header">
        <div class="dh-icon">
          <t-icon :name="fileIcon(detail.file_type)" size="28px" />
        </div>
        <div class="dh-info">
          <h3 class="dh-title">{{ detail.title || detail.file_name }}</h3>
          <div class="dh-meta">
            <span>类型：{{ (detail.file_type || '').toUpperCase() }}</span>
            <span>大小：{{ formatSize(detail.file_size) }}</span>
            <span>创建时间：{{ detail.created_at }}</span>
          </div>
        </div>
        <t-tag :theme="statusTheme(detail.parse_status)" variant="light" size="medium">
          {{ statusText(detail.parse_status) }}
        </t-tag>
      </div>

      <t-divider />

      <!-- 操作按钮 -->
      <div class="detail-actions">
        <t-button variant="outline" @click="handleDownload">
          <template #icon><t-icon name="download" /></template>
          下载文件
        </t-button>
        <t-button variant="outline" @click="handlePreview">
          <template #icon><t-icon name="view-list" /></template>
          查看预览
        </t-button>
      </div>

      <!-- 内容预览区（简单展示 segments / chunks 摘要） -->
      <div v-if="previewContent" class="detail-preview">
        <t-divider />
        <h4>文本摘要</h4>
        <div class="preview-text">{{ previewContent }}</div>
      </div>
    </template>

    <div v-else class="detail-empty">
      <t-icon name="error-circle" size="32px" />
      <p>无法加载文件信息</p>
    </div>
  </t-dialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { getKnowledgeDetails } from '@/api/knowledge-base/index'

const props = defineProps<{
  modelValue: boolean
  knowledgeId: string
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', v: boolean): void
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v),
})

import { computed } from 'vue'

const loading = ref(false)
const detail = ref<any>(null)
const previewContent = ref('')

// ====== 加载详情 ======
watch(
  () => props.modelValue,
  async (open) => {
    if (open && props.knowledgeId) {
      await loadDetail()
    }
  },
)

async function loadDetail() {
  loading.value = true
  detail.value = null
  previewContent.value = ''
  try {
    const res = await getKnowledgeDetails(props.knowledgeId)
    if (res.success && res.data) {
      detail.value = res.data
      // 尝试提取文本摘要
      const content = res.data.content || res.data.description || ''
      const chunks = res.data.chunks || res.data.segments || []
      if (content) {
        previewContent.value = typeof content === 'string' ? content.slice(0, 2000) : JSON.stringify(content).slice(0, 2000)
      } else if (chunks.length > 0) {
        previewContent.value = chunks.map((c: any) => c.content || c.text || '').join('\n\n').slice(0, 2000)
      }
    }
  } catch (err: any) {
    // 静默处理
  } finally {
    loading.value = false
  }
}

// ====== 下载 ======
function handleDownload() {
  window.open(`/api/v1/knowledge/${props.knowledgeId}/download`, '_blank')
}

// ====== 预览 ======
function handlePreview() {
  window.open(`/api/v1/knowledge/${props.knowledgeId}/preview`, '_blank')
}

// ====== 工具 ======
function fileIcon(ft: string): string {
  const t = (ft || '').toLowerCase()
  if (t === 'pdf')  return 'file-pdf'
  if (t === 'xlsx' || t === 'xls' || t === 'csv') return 'file-excel'
  if (t === 'docx' || t === 'doc') return 'file-word'
  if (t === 'txt' || t === 'md')  return 'file-text'
  return 'file'
}

function formatSize(bytes: number): string {
  if (!bytes) return '--'
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}

function statusTheme(s: string) {
  if (s === 'completed') return 'success'
  if (s === 'failed' || s === 'cancelled') return 'danger'
  return 'warning'
}

function statusText(s: string) {
  if (s === 'completed') return '处理成功'
  if (s === 'failed' || s === 'cancelled') return '处理失败'
  return '处理中'
}

function handleClose() {
  detail.value = null
  previewContent.value = ''
}
</script>

<style scoped lang="less">
.detail-loading, .detail-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px 0;
  gap: 12px;
  color: var(--td-text-color-placeholder);
}

.detail-header {
  display: flex;
  align-items: flex-start;
  gap: 14px;
}

.dh-icon {
  color: var(--td-brand-color);
  flex-shrink: 0;
  margin-top: 2px;
}

.dh-info {
  flex: 1;
}

.dh-title {
  font-size: 15px;
  font-weight: 600;
  margin: 0 0 6px 0;
  color: var(--td-text-color-primary);
  word-break: break-all;
}

.dh-meta {
  display: flex;
  gap: 16px;
  font-size: 12px;
  color: var(--td-text-color-placeholder);
}

.detail-actions {
  display: flex;
  gap: 10px;
}

.detail-preview h4 {
  font-size: 13px;
  font-weight: 600;
  margin: 0 0 8px 0;
}

.preview-text {
  background: var(--td-bg-color-component);
  border-radius: 6px;
  padding: 14px;
  font-size: 13px;
  line-height: 1.7;
  white-space: pre-wrap;
  word-break: break-all;
  max-height: 300px;
  overflow-y: auto;
  color: var(--td-text-color-secondary);
}
</style>
