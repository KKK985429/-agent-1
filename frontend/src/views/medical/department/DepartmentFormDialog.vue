<template>
  <t-dialog
    v-model:visible="visible"
    :header="isEdit ? '编辑科室' : '新建科室'"
    :confirm-btn="isEdit ? '确定' : '新建'"
    :cancel-btn="'取消'"
    :close-on-overlay-click="false"
    width="480px"
    @confirm="handleConfirm"
    @close="handleClose"
  >
    <t-form
      ref="formRef"
      :data="form"
      :rules="rules"
      label-width="80px"
      @submit="handleConfirm"
    >
      <t-form-item label="科室名称" name="name">
        <t-input v-model="form.name" placeholder="请输入" clearable />
      </t-form-item>
      <t-form-item label="科室编号" name="code">
        <t-input v-model="form.code" placeholder="请输入" clearable />
      </t-form-item>
      <t-form-item label="院区" name="hospital_area">
        <t-select v-model="form.hospital_area" placeholder="请选择" clearable>
          <t-option
            v-for="area in hospitalAreas"
            :key="area.value"
            :label="area.label"
            :value="area.value"
          />
        </t-select>
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
import { ref, watch, computed } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import type { FormInstanceFunctions, FormRule } from 'tdesign-vue-next'
import {
  createDepartment,
  updateDepartment,
  listHospitalAreas,
  type DepartmentPayload,
  type HospitalArea,
  type MedicalDepartment,
} from '@/api/medical/department/index'

const props = defineProps<{
  modelValue: boolean
  mode: 'create' | 'edit'
  department?: MedicalDepartment | null
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', v: boolean): void
  (e: 'saved'): void
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v),
})

const isEdit = computed(() => props.mode === 'edit')

const formRef = ref<FormInstanceFunctions>()
const loading = ref(false)
const hospitalAreas = ref<HospitalArea[]>([])

const form = ref<{
  name: string
  code: string
  hospital_area: string
  enabled: boolean
}>({
  name: '',
  code: '',
  hospital_area: '',
  enabled: true,
})

const rules: Record<string, FormRule[]> = {
  name: [{ required: true, message: '请输入科室名称', trigger: 'blur' }],
  code: [{ required: true, message: '请输入科室编号', trigger: 'blur' }],
  hospital_area: [{ required: true, message: '请选择院区', trigger: 'change' }],
  enabled: [{ required: true, message: '请选择启用状态', trigger: 'change' }],
}

// Load hospital areas on mount
async function loadHospitalAreas() {
  try {
    const res = await listHospitalAreas()
    if (res.success) {
      hospitalAreas.value = res.data || []
    }
  } catch {
    // Fallback: keep empty
  }
}

loadHospitalAreas()

// When editing, populate the form with existing data
watch(
  () => props.department,
  (dept) => {
    if (dept && props.mode === 'edit') {
      form.value = {
        name: dept.name,
        code: dept.code,
        hospital_area: dept.hospital_area,
        enabled: dept.enabled,
      }
    }
  },
  { immediate: true },
)

// Reset form when opening in create mode
watch(
  () => props.modelValue,
  (open) => {
    if (open && props.mode === 'create') {
      form.value = { name: '', code: '', hospital_area: '', enabled: true }
    }
  },
)

async function handleConfirm() {
  const valid = await formRef.value?.validate()
  if (valid !== true && valid !== undefined) return

  loading.value = true
  try {
    const payload: DepartmentPayload = {
      name: form.value.name,
      code: form.value.code,
      hospital_area: form.value.hospital_area,
      enabled: form.value.enabled,
    }

    if (isEdit.value && props.department) {
      await updateDepartment(props.department.id, payload)
      MessagePlugin.success('编辑成功')
    } else {
      await createDepartment(payload)
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
