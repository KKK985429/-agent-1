<template>
  <t-dialog
    v-model:visible="visible"
    header="上传文件"
    :confirm-btn="{
      content: hasUploading ? '上传中...' : allDone ? '关闭' : '确定上传',
      disabled: (!allDone && fileItems.length === 0) || hasUploading,
      loading: hasUploading,
    }"
    :cancel-btn="allDone || fileItems.length === 0 ? '取消' : null"
    :close-on-overlay-click="false"
    width="540px"
    @confirm="handleConfirm"
    @close="handleClose"
  >
    <div class="upload-body">
      <p class="upload-desc">
        支持 <code>.txt</code> <code>.docx</code> <code>.pdf</code> <code>.xlsx</code> <code>.md</code>
        等格式，单文件 ≤ {{ MAX_FILE_SIZE_MB }}MB，可多选。选择后点击"确定上传"开始上传。
      </p>

      <!-- 选文件 / 拖拽区：整块可点击 -->
      <div
        class="upload-drop-zone"
        @click="triggerFileInput"
        @dragenter.prevent
        @dragover.prevent
        @dragleave.prevent
        @drop.prevent.stop="handleDrop"
      >
        <t-icon name="upload" size="36px" style="color: var(--td-text-color-placeholder)" />
        <p>拖拽文件至此，或点击上传</p>
        <p class="upload-hint">支持 .txt .docx .pdf .xlsx .md 等，单文件 ≤ {{ MAX_FILE_SIZE_MB }}MB，可多选</p>
      </div>

      <!-- 已选文件列表 -->
      <div v-if="fileItems.length > 0" class="file-list">
        <div
          v-for="(item, idx) in fileItems"
          :key="idx"
          class="file-item"
          :class="{ 'is-success': item.status === 'success', 'is-error': item.status === 'error' }"
        >
          <t-icon :name="fileIcon(item.file.name)" size="20px" />
          <span class="fi-name">{{ item.file.name }}</span>
          <span class="fi-size">{{ formatSize(item.file.size) }}</span>

          <!-- 状态展示 -->
          <template v-if="item.status === 'pending'">
            <t-tag size="small" variant="light" theme="default">待上传</t-tag>
          </template>
          <template v-else-if="item.status === 'uploading'">
            <t-tag size="small" variant="light" theme="warning">上传中</t-tag>
            <div class="fi-progress">
              <t-progress :percentage="item.progress" size="small" />
            </div>
          </template>
          <template v-else-if="item.status === 'success'">
            <t-tag size="small" variant="light" theme="success">已上传</t-tag>
            <t-icon name="check-circle" size="16px" style="color: var(--td-success-color)" />
          </template>
          <template v-else-if="item.status === 'error'">
            <t-tag size="small" variant="light" theme="danger">失败</t-tag>
            <span class="fi-error">{{ item.errorMsg }}</span>
          </template>

          <!-- 待上传或失败的可移除 -->
          <t-button
            v-if="item.status === 'pending' || item.status === 'error'"
            variant="text" theme="danger" size="small"
            @click.stop="removeFile(idx)"
          >移除</t-button>
        </div>
      </div>
    </div>

    <input
      ref="fileInputRef"
      type="file"
      accept=".txt,.docx,.pdf,.xlsx,.doc,.xls,.csv,.ppt,.pptx,.md"
      multiple
      style="display: none"
      @change="handleInputChange"
    />
  </t-dialog>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { uploadKnowledgeFile } from '@/api/knowledge-base/index'
import { MAX_FILE_SIZE_MB } from '@/utils/index'

const props = defineProps<{
  modelValue: boolean
  documentKbId: string
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', v: boolean): void
  (e: 'saved'): void
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v),
})

interface FileItem {
  file: File
  status: 'pending' | 'uploading' | 'success' | 'error'
  progress: number
  errorMsg: string
}

const fileInputRef = ref<HTMLInputElement>()
const fileItems = ref<FileItem[]>([])

const allDone = computed(() =>
  fileItems.value.length > 0 && fileItems.value.every(f => f.status === 'success' || f.status === 'error')
)
const hasUploading = computed(() => fileItems.value.some(f => f.status === 'uploading'))

// ====== 选择文件（仅添加到列表，不上传） ======
function triggerFileInput() {
  fileInputRef.value?.click()
}

function handleInputChange(e: Event) {
  const input = e.target as HTMLInputElement
  const files = input.files
  if (!files || files.length === 0) return

  for (let i = 0; i < files.length; i++) {
    const file = files[i]
    // 检查是否已添加同名文件
    if (fileItems.value.some(f => f.file.name === file.name && f.file.size === file.size)) continue

    fileItems.value.push({
      file,
      status: 'pending',
      progress: 0,
      errorMsg: '',
    })
  }
  input.value = ''
}

function handleDrop(e: DragEvent) {
  const files = e.dataTransfer?.files
  if (!files || files.length === 0) return
  for (let i = 0; i < files.length; i++) {
    if (fileItems.value.some(f => f.file.name === files[i].name && f.file.size === files[i].size)) continue
    fileItems.value.push({
      file: files[i],
      status: 'pending',
      progress: 0,
      errorMsg: '',
    })
  }
}

function removeFile(idx: number) {
  fileItems.value.splice(idx, 1)
}

// ====== 点击"确定上传" → 批量上传 ======
async function handleConfirm() {
  // 全部完成 → 关闭弹窗
  if (allDone.value) {
    visible.value = false
    emit('saved')
    return
  }

  if (!props.documentKbId) {
    MessagePlugin.error('知识库 ID 未配置')
    return
  }

  // 逐个上传 pending 的文件
  const pending = fileItems.value.filter(f => f.status === 'pending')
  for (const item of pending) {
    item.status = 'uploading'
    try {
      await uploadKnowledgeFile(
        props.documentKbId,
        { file: item.file },
        (e: any) => {
          if (e?.total) item.progress = Math.round((e.loaded / e.total) * 100)
        },
      )
      item.status = 'success'
      item.progress = 100
    } catch (err: any) {
      item.status = 'error'
      item.errorMsg = err?.response?.data?.error?.message || err?.message || '上传失败'
    }
  }
}

// ====== 关闭/取消 ======
function handleClose() {
  fileItems.value = []
}

// ====== 工具 ======
function fileIcon(name: string): string {
  const ext = name.split('.').pop()?.toLowerCase() || ''
  if (ext === 'pdf') return 'file-pdf'
  if (ext === 'xlsx' || ext === 'xls' || ext === 'csv') return 'file-excel'
  if (ext === 'docx' || ext === 'doc') return 'file-word'
  if (ext === 'txt' || ext === 'md') return 'file-text'
  return 'file'
}

function formatSize(bytes: number): string {
  if (!bytes) return '--'
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}
</script>

<style scoped lang="less">
.upload-body {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.upload-desc {
  font-size: 13px;
  color: var(--td-text-color-secondary);
  margin: 0;
  line-height: 1.6;
  code {
    background: var(--td-bg-color-component);
    padding: 1px 6px;
    border-radius: 3px;
    font-size: 12px;
  }
}

.upload-drop-zone {
  border: 1px dashed var(--td-component-border);
  border-radius: 8px;
  padding: 32px 20px;
  text-align: center;
  cursor: pointer;
  color: var(--td-text-color-placeholder);
  transition: border-color 0.2s;
  &:hover { border-color: var(--td-brand-color); }
  p { margin: 4px 0; font-size: 13px; }
}

.upload-hint { font-size: 12px !important; }

.file-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-height: 280px;
  overflow-y: auto;
}

.file-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  border: 1px solid var(--td-component-border);
  border-radius: 6px;
  font-size: 13px;
  &.is-success { border-color: var(--td-success-color); background: var(--td-success-color-1); }
  &.is-error   { border-color: var(--td-error-color);   background: var(--td-error-color-1); }
}

.fi-name { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; min-width: 0; }
.fi-size { color: var(--td-text-color-placeholder); font-size: 12px; flex-shrink: 0; }
.fi-progress { width: 100px; flex-shrink: 0; }
.fi-error { color: var(--td-error-color); font-size: 12px; flex-shrink: 0; }
</style>
