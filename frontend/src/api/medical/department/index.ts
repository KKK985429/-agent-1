import { get, post, put, del } from '../../../utils/request'

// ---------- types ----------

export interface MedicalDepartment {
  id: string
  name: string
  code: string
  hospital_area: string
  enabled: boolean
  created_by: string
  created_by_name: string
  created_at: string
  updated_at: string
}

export interface DepartmentListResponse {
  list: MedicalDepartment[]
  total: number
  page: number
  page_size: number
}

export interface DepartmentPayload {
  name: string
  code: string
  hospital_area: string
  enabled: boolean
}

export interface HospitalArea {
  label: string
  value: string
}

// ---------- API ----------

/** 科室列表 (支持 keyword / enabled / page / page_size) */
export function listDepartments(params?: {
  keyword?: string
  enabled?: string
  page?: number
  page_size?: number
}) {
  const query = new URLSearchParams()
  if (params?.keyword) query.set('keyword', params.keyword)
  if (params?.enabled !== undefined && params.enabled !== '') query.set('enabled', params.enabled)
  if (params?.page) query.set('page', String(params.page))
  if (params?.page_size) query.set('page_size', String(params.page_size))
  const qs = query.toString()
  return get(qs ? `/api/v1/medical/departments?${qs}` : '/api/v1/medical/departments')
}

/** 科室详情 */
export function getDepartment(id: string) {
  return get(`/api/v1/medical/departments/${id}`)
}

/** 新建科室 */
export function createDepartment(data: DepartmentPayload) {
  return post('/api/v1/medical/departments', data)
}

/** 编辑科室 */
export function updateDepartment(id: string, data: Partial<DepartmentPayload>) {
  return put(`/api/v1/medical/departments/${id}`, data)
}

/** 删除科室 (软删除) */
export function deleteDepartment(id: string) {
  return del(`/api/v1/medical/departments/${id}`)
}

/** 院区下拉列表 */
export function listHospitalAreas() {
  return get('/api/v1/medical/hospital-areas')
}
