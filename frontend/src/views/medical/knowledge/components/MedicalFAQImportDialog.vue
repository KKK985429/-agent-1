<template>
  <t-dialog
    v-model:visible="visible"
    header="批量上传 Q&A"
    :confirm-btn="{
      content: importing ? '导入中...' : '确定',
      loading: importing,
      disabled: !canConfirm,
    }"
    :cancel-btn="'取消'"
    :close-on-overlay-click="false"
    width="540px"
    @confirm="handleConfirm"
    @close="handleClose"
  >
    <div class="import-body">
      <!-- 说明 -->
      <p class="import-desc">
        请先下载模板，按照模板格式（Q 列 / A 列）填写后上传。
        单个文件不超过 10MB，仅支持 <code>.xlsx</code> 格式。
      </p>

      <!-- 下载模板 -->
      <t-button variant="outline" @click="downloadTemplate">
        <template #icon><t-icon name="download" /></template>
        下载上传模板
      </t-button>

      <!-- 上传区域 -->
      <div
        class="upload-area"
        :class="{ 'has-file': selectedFile }"
        @dragover.prevent
        @drop.prevent="handleDrop"
        @click="triggerFileInput"
      >
        <template v-if="!selectedFile">
          <t-icon name="upload" size="36px" style="color: var(--td-text-color-placeholder)" />
          <p>拖拽文件至此，或点击上传</p>
          <p class="upload-hint">仅支持 .xlsx，不超过 10MB</p>
        </template>
        <template v-else>
          <div class="file-info">
            <t-icon name="file-excel" size="20px" />
            <span class="file-name">{{ selectedFile.name }}</span>
            <span class="file-size">{{ formatSize(selectedFile.size) }}</span>
            <t-button
              variant="text"
              theme="danger"
              size="small"
              @click.stop="removeFile"
            >
              移除
            </t-button>
          </div>
        </template>
        <input
          ref="fileInputRef"
          type="file"
          accept=".xlsx"
          style="display: none"
          @change="handleInputChange"
        />
      </div>

      <!-- 校验结果 -->
      <div v-if="validationMessage" class="validation-result" :class="fileValid ? 'success' : 'error'">
        <t-icon :name="fileValid ? 'check-circle' : 'close-circle'" />
        <span>{{ validationMessage }}</span>
      </div>

      <!-- 导入进度 -->
      <div v-if="importing" class="import-progress">
        <t-progress :percentage="importProgress" :label="importProgressText" />
      </div>
    </div>
  </t-dialog>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { upsertFAQEntries, getFAQImportProgress } from '@/api/medical/knowledge-base/index'
import * as XLSX from 'xlsx'

const props = defineProps<{
  modelValue: boolean
  faqKbId: string
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', v: boolean): void
  (e: 'saved'): void
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v),
})

const fileInputRef = ref<HTMLInputElement>()
const selectedFile = ref<File | null>(null)
const fileValid = ref(false)
const validationMessage = ref('')
const importing = ref(false)
const importProgress = ref(0)
const importProgressText = ref('')

// ---------- 文件处理 ----------
function triggerFileInput() {
  fileInputRef.value?.click()
}

function handleInputChange(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  if (file) processFile(file)
  input.value = '' // Reset so re-selecting same file works
}

function handleDrop(e: DragEvent) {
  const file = e.dataTransfer?.files?.[0]
  if (file) processFile(file)
}

function removeFile() {
  selectedFile.value = null
  fileValid.value = false
  validationMessage.value = ''
  parsedEntries = []
}

function processFile(file: File) {
  if (file.size > 10 * 1024 * 1024) {
    fileValid.value = false
    validationMessage.value = '文件大小超过 10MB 限制'
    selectedFile.value = null
    return
  }
  selectedFile.value = file
  parseAndValidate(file)
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}

// ---------- 模板下载 ----------
function downloadTemplate() {
  const wb = XLSX.utils.book_new()
  const ws = XLSX.utils.aoa_to_sheet([['Q', 'A'], ['', '']])
  // Set column widths
  ws['!cols'] = [{ wch: 40 }, { wch: 60 }]
  XLSX.utils.book_append_sheet(wb, ws, 'Q&A模板')
  XLSX.writeFile(wb, 'Q&A导入模板.xlsx')
  MessagePlugin.success('模板已下载')
}

// ---------- 解析 + 校验 ----------
let parsedEntries: any[] = []

async function parseAndValidate(file: File) {
  // Check extension
  if (!file.name.toLowerCase().endsWith('.xlsx')) {
    fileValid.value = false
    validationMessage.value = '仅支持 .xlsx 格式文件'
    return
  }

  try {
    const data = await readXLSX(file)

    // 1. Must have exactly one sheet (or at least data in first sheet)
    if (data.length === 0) {
      fileValid.value = false
      validationMessage.value = '文件中没有数据'
      return
    }

    const rows = data as any[][]
    if (rows.length < 2) {
      fileValid.value = false
      validationMessage.value = '模板至少需要表头行和一行数据'
      return
    }

    // 2. Header must be exactly Q / A
    const header = rows[0].map((h: any) => String(h || '').trim())
    if (header.length !== 2) {
      fileValid.value = false
      validationMessage.value = `模板必须严格为 Q / A 两列，当前检测到 ${header.length} 列`
      return
    }
    if (header[0] !== 'Q' || header[1] !== 'A') {
      fileValid.value = false
      validationMessage.value = `表头必须严格为 Q / A，当前为 ${header[0]} / ${header[1]}`
      return
    }

    // 3. Validate each row: Q and A must both be non-empty
    const entries: any[] = []
    for (let i = 1; i < rows.length; i++) {
      const q = String(rows[i][0] || '').trim()
      const a = String(rows[i][1] || '').trim()

      if (q === '' || a === '') {
        fileValid.value = false
        validationMessage.value = `第 ${i + 1} 行存在空问题或空答案，请检查后重新上传`
        return
      }
      entries.push({
        standard_question: q,
        answers: [a],
        similar_questions: [],
        negative_questions: [],
        is_enabled: true,
      })
    }

    parsedEntries = entries
    fileValid.value = true
    validationMessage.value = `校验通过，共 ${entries.length} 条 Q&A 待导入`
  } catch (err: any) {
    fileValid.value = false
    validationMessage.value = `文件解析失败：${err.message || '未知错误'}`
  }
}

function readXLSX(file: File): Promise<any[][]> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = (e) => {
      try {
        const data = new Uint8Array(e.target?.result as ArrayBuffer)
        const wb = XLSX.read(data, { type: 'array' })
        const sheetName = wb.SheetNames[0]
        if (!sheetName) {
          reject(new Error('文件中没有工作表'))
          return
        }
        const ws = wb.Sheets[sheetName]
        const rows = XLSX.utils.sheet_to_json<(string | number | boolean | null)[]>(ws, { header: 1, defval: '' })
        if (!rows || rows.length === 0 || !Array.isArray(rows[0])) {
          reject(new Error('文件格式不正确，第一行应为表头'))
          return
        }
        resolve(rows as any[][])
      } catch (err) {
        reject(err)
      }
    }
    reader.onerror = () => reject(new Error('文件读取失败'))
    reader.readAsArrayBuffer(file)
  })
}

// ---------- 确定导入 ----------
const canConfirm = computed(() => fileValid.value && !importing.value)

async function handleConfirm() {
  if (!canConfirm.value || parsedEntries.length === 0) return

  if (!props.faqKbId) {
    MessagePlugin.error('知识库 ID 未配置，请刷新页面后重试')
    return
  }

  importing.value = true
  importProgress.value = 0
  importProgressText.value = '正在导入...'

  try {
    const res = await upsertFAQEntries(props.faqKbId, {
      entries: parsedEntries,
      mode: 'append',
    })

    const taskId = res?.data?.task_id

    if (taskId) {
      // Poll progress
      await pollProgress(taskId)
    } else {
      // No task_id — import completed synchronously
      importProgress.value = 100
      importProgressText.value = '导入完成'
    }

    MessagePlugin.success(`成功导入 ${parsedEntries.length} 条 Q&A`)
    visible.value = false
    emit('saved')
  } catch (err: any) {
    const msg = err?.response?.data?.message || err?.message || '导入失败'
    MessagePlugin.error(msg)
  } finally {
    importing.value = false
    parsedEntries = []
  }
}

async function pollProgress(taskId: string) {
  const maxAttempts = 120 // max 2 minutes (1s interval)
  for (let i = 0; i < maxAttempts; i++) {
    try {
      const res = await getFAQImportProgress(taskId)
      const status = res?.data?.status || res?.status || 'processing'
      const total = res?.data?.total || 0
      const completed = res?.data?.completed || 0
      const failed = res?.data?.failed || 0

      if (total > 0) {
        importProgress.value = Math.round(((completed + failed) / total) * 100)
        importProgressText.value = `已完成 ${completed}，失败 ${failed}，共 ${total}`
      }

      if (status === 'completed') {
        importProgress.value = 100
        importProgressText.value = '导入完成'
        return
      }
      if (status === 'failed') {
        throw new Error(res?.data?.error || '导入任务失败')
      }
    } catch (err: any) {
      if (err.message?.includes('导入任务失败')) throw err
      // Network errors during polling are non-fatal; keep retrying
    }

    await sleep(1000)
  }
  throw new Error('导入超时，请稍后检查列表')
}

function sleep(ms: number) {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

function handleClose() {
  fileValid.value = false
  validationMessage.value = ''
  parsedEntries = []
  importing.value = false
  selectedFile.value = null
}
</script>

<style scoped lang="less">
.import-body {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.import-desc {
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

.upload-area {
  border: 1px dashed var(--td-component-border);
  border-radius: 8px;
  padding: 30px;
  text-align: center;
  cursor: pointer;
  transition: border-color 0.2s;
  color: var(--td-text-color-placeholder);

  &:hover {
    border-color: var(--td-brand-color);
  }

  &.has-file {
    border-style: solid;
    border-color: var(--td-success-color);
    padding: 16px 20px;
    cursor: default;
  }

  p {
    margin: 4px 0;
    font-size: 13px;
  }
}

.upload-area.has-file .file-info {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 13px;
  color: var(--td-text-color-primary);
}

.file-name {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-size {
  color: var(--td-text-color-placeholder);
  font-size: 12px;
}

.upload-hint {
  font-size: 12px !important;
  color: var(--td-text-color-placeholder);
}

.validation-result {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  border-radius: 6px;
  font-size: 13px;

  &.success {
    background: var(--td-success-color-1);
    color: var(--td-success-color);
  }

  &.error {
    background: var(--td-error-color-1);
    color: var(--td-error-color);
  }
}

.import-progress {
  padding: 4px 0;
}
</style>
