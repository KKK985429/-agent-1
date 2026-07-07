<template>
  <t-dialog
    v-model:visible="visible"
    :header="dialogTitle"
    :confirm-btn="isDetail ? '确定' : '新建'"
    :cancel-btn="'取消'"
    :close-on-overlay-click="false"
    width="520px"
    @confirm="handleConfirm"
    @close="handleClose"
  >
    <t-form
      ref="formRef"
      :data="form"
      :rules="rules"
      label-width="80px"
    >
      <t-form-item label="问题名称" name="question">
        <t-input
          v-model="form.question"
          placeholder="请输入"
          clearable
          :maxlength="500"
        />
      </t-form-item>
      <t-form-item label="问题答案" name="answer">
        <t-textarea
          v-model="form.answer"
          placeholder="请输入"
          :autosize="{ minRows: 4, maxRows: 10 }"
          :maxlength="5000"
        />
      </t-form-item>
      <t-form-item label="启用状态" name="enabled">
        <t-select v-model="form.enabled" placeholder="请选择">
          <t-option label="启用" :value="true" />
          <t-option label="停用" :value="false" />
        </t-select>
      </t-form-item>
    </t-form>
  </t-dialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import type { FormInstanceFunctions, FormRule } from 'tdesign-vue-next'
import { createFAQEntry, updateFAQEntry } from '@/api/medical/knowledge-base/index'

interface Entry {
  id: number
  standard_question: string
  answers: string[]
  is_enabled: boolean
  _status?: string
}

const props = defineProps<{
  modelValue: boolean
  mode: 'create' | 'detail'
  entry?: Entry | null
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

const isDetail = computed(() => props.mode === 'detail')
const isCreate = computed(() => props.mode === 'create')

const dialogTitle = computed(() => {
  if (isCreate.value) return '新建 Q&A'
  return '详情'
})

const formRef = ref<FormInstanceFunctions>()
const loading = ref(false)

const form = ref({
  question: '',
  answer: '',
  enabled: true,
})

const rules: Record<string, FormRule[]> = {
  question: [{ required: true, message: '请输入问题名称', trigger: 'blur' }],
  answer: [{ required: true, message: '请输入问题答案', trigger: 'blur' }],
  enabled: [{ required: true, message: '请选择启用状态', trigger: 'change' }],
}

// Populate form when editing
watch(
  () => props.entry,
  (entry) => {
    if (entry && isDetail.value) {
      form.value = {
        question: entry.standard_question || '',
        answer: (entry.answers && entry.answers.length > 0) ? entry.answers[0] : '',
        enabled: entry.is_enabled,
      }
    }
  },
  { immediate: true },
)

// Reset form when opening in create mode
watch(
  () => props.modelValue,
  (open) => {
    if (open && isCreate.value) {
      form.value = { question: '', answer: '', enabled: true }
    }
  },
)

async function handleConfirm() {
  const valid = await formRef.value?.validate()
  if (valid !== true && valid !== undefined) return

  loading.value = true
  try {
    const payload = {
      standard_question: form.value.question,
      answers: [form.value.answer],
      similar_questions: [],
      negative_questions: [],
      is_enabled: form.value.enabled,
    }

    if (isDetail.value && props.entry) {
      await updateFAQEntry(props.faqKbId, props.entry.id, payload)
      MessagePlugin.success('更新成功')
    } else {
      await createFAQEntry(props.faqKbId, payload)
      MessagePlugin.success('创建成功')
    }

    visible.value = false
    emit('saved')
  } catch (err: any) {
    const msg = err?.response?.data?.message || err?.message || '操作失败'
    MessagePlugin.error(msg)
  } finally {
    loading.value = false
  }
}

function handleClose() {
  formRef.value?.reset()
}
</script>
