<template>
  <t-dialog v-model:visible="visible" header="文件详情" :confirm-btn="null" :cancel-btn="'关闭'"
    :close-on-overlay-click="true" width="720px">
    <div v-if="loading" class="detail-loading"><t-loading text="加载中..." /></div>

    <template v-else-if="detail">
      <!-- 文件信息 -->
      <div class="detail-header">
        <t-icon :name="fileIcon(detail.file_type||detail.file_name)" size="28px" />
        <div class="dh-info">
          <h3>{{ detail.title || detail.file_name }}</h3>
          <div class="dh-meta">
            <span>类型：{{ (detail.file_type||'').toUpperCase() }}</span>
            <span>大小：{{ formatSize(detail.file_size) }}</span>
            <span>{{ detail.created_at }}</span>
          </div>
        </div>
        <t-tag :theme="statusTheme(detail.parse_status)" variant="light">
          {{ statusText(detail.parse_status) }}
        </t-tag>
      </div>

      <t-divider />

      <!-- 内容摘要 -->
      <div class="preview-block">
        <h4>内容摘要</h4>
        <div class="preview-text">{{ summary || '(暂无摘要)' }}</div>
      </div>

      <div class="detail-actions">
        <t-button variant="outline" @click="handleDownload">
          <template #icon><t-icon name="download" /></template>下载文件
        </t-button>
        <t-button variant="outline" :loading="previewLoading" @click="handlePreview">
          <template #icon><t-icon name="browse" /></template>
          {{ fullMode ? '已加载原文件' : '查看原文件' }}
        </t-button>
      </div>

      <!-- 查看原文件：PDF → iframe，图片 → img，文字 → 展示原文 -->
      <div v-if="fullMode" class="preview-block">
        <t-divider />
        <h4>原文件</h4>
        <!-- PDF -->
        <iframe v-if="isPdf" :src="originalBlobUrl" width="100%" height="550px" frameborder="0" />
        <!-- 图片 -->
        <img v-else-if="isImage" :src="originalBlobUrl" style="max-width:100%;max-height:550px" />
        <!-- 文字类 -->
        <div v-else class="preview-text full">{{ originalText }}</div>
      </div>
    </template>

    <div v-else class="detail-empty"><p>无法加载文件信息</p></div>
  </t-dialog>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { getKnowledgeDetails } from '@/api/knowledge-base/index'

const props = defineProps<{ modelValue: boolean; knowledgeId: string; kbId?: string }>()
const emit = defineEmits<{ (e: 'update:modelValue', v: boolean): void }>()
const visible = computed({ get: () => props.modelValue, set: (v) => emit('update:modelValue', v) })

const loading = ref(true)
const detail = ref<any>(null)
const summary = ref('')
const fullMode = ref(false)
const originalBlobUrl = ref('')
const originalText = ref('')
const isPdf = ref(false)
const isImage = ref(false)
const previewLoading = ref(false)

onMounted(async () => {
  if (!props.knowledgeId) { loading.value = false; return }
  try {
    const res = await getKnowledgeDetails(props.knowledgeId)
    if (res.success && res.data) {
      detail.value = res.data
      summary.value = (res.data.description || res.data.content || '').slice(0, 2000)
      const ft = (res.data.file_type || res.data.file_name || '').toLowerCase()
      isPdf.value = ft.includes('pdf')
      isImage.value = ['jpg','jpeg','png','gif','webp','svg','bmp'].some(e => ft.includes(e))
    }
  } catch { /* ignore */ }
  loading.value = false
})

async function handlePreview() {
  if (fullMode.value || previewLoading.value) return
  previewLoading.value = true
  try {
    const token = localStorage.getItem('weknora_token') || ''
    const key = localStorage.getItem('weknora_api_key') || ''
    // 直接拿原文件
    const resp = await fetch(`/api/v1/knowledge/${props.knowledgeId}/download`, {
      headers: { 'Authorization': `Bearer ${token}`, 'X-API-Key': key },
    })
    if (!resp.ok) throw new Error('fetch failed')
    if (isPdf.value || isImage.value) {
      const buf = await resp.arrayBuffer()
      const mime = isPdf.value ? 'application/pdf' : 'image/*'
      const blob = new Blob([buf], { type: mime })
      originalBlobUrl.value = URL.createObjectURL(blob)
    } else {
      originalText.value = await resp.text()
    }
    fullMode.value = true
  } catch { originalText.value = '(加载失败，请下载查看)'; fullMode.value = true }
  previewLoading.value = false
}

function handleDownload() {
  const token = localStorage.getItem('weknora_token') || ''
  const key = localStorage.getItem('weknora_api_key') || ''
  fetch(`/api/v1/knowledge/${props.knowledgeId}/download`, {
    headers: { 'Authorization': `Bearer ${token}`, 'X-API-Key': key },
  }).then(r => r.blob()).then(blob => {
    const u = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = u; a.download = detail.value?.file_name || 'file'; a.click()
    setTimeout(() => URL.revokeObjectURL(u), 60000)
  })
}

function fileIcon(n: string) { const t = (n||'').toLowerCase(); if(t.includes('pdf'))return'file-pdf'; if(t.includes('xlsx')||t.includes('xls')||t.includes('csv'))return'file-excel'; if(t.includes('docx')||t.includes('doc'))return'file-word'; if(t.includes('md')||t.includes('txt'))return'file-text'; return'file' }
function formatSize(b: number) { if(!b)return'--'; if(b<1024)return b+' B'; if(b<1048576)return(b/1024).toFixed(1)+' KB'; return(b/1048576).toFixed(1)+' MB' }
function statusTheme(s: string) { if(s==='completed')return'success'; if(s==='failed'||s==='cancelled')return'danger'; return'warning' }
function statusText(s: string) { if(s==='completed')return'处理成功'; if(s==='failed'||s==='cancelled')return'处理失败'; return'处理中' }
</script>

<style scoped lang="less">
.detail-loading,.detail-empty{display:flex;flex-direction:column;align-items:center;justify-content:center;padding:40px 0;gap:12px;color:var(--td-text-color-placeholder)}
.detail-header{display:flex;align-items:flex-start;gap:14px}
.dh-info{flex:1}
.dh-info h3{font-size:15px;font-weight:600;margin:0 0 6px 0;word-break:break-all}
.dh-meta{display:flex;gap:16px;font-size:12px;color:var(--td-text-color-placeholder);flex-wrap:wrap}
.preview-block{margin:16px 0}
.preview-block h4{font-size:13px;font-weight:600;margin:0 0 8px 0}
.detail-actions{display:flex;gap:10px;margin-top:12px}
.preview-text{background:var(--td-bg-color-component);border-radius:6px;padding:14px;font-size:13px;line-height:1.7;white-space:pre-wrap;word-break:break-all;max-height:200px;overflow-y:auto;color:var(--td-text-color-secondary)}
.preview-text.full{max-height:500px}
</style>
