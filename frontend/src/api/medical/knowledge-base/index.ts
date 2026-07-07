import { get, post, put, del } from '../../../utils/request'
import {
  listFAQEntries as listFAQ,
  createFAQEntry as createFAQ,
  updateFAQEntry as updateFAQ,
  upsertFAQEntries as upsertFAQ,
  updateFAQEntryFieldsBatch as batchUpdateFields,
  deleteFAQEntries as deleteFAQ,
  getFAQImportProgress,
} from '../../../api/knowledge-base/index'

// ---------- types ----------

export interface MedicalKBConfigItem {
  key: string                       // symptom / disease / drug / lab
  name: string                      // 症状知识库 / 疾病知识库 / 药品知识库 / 检验检查知识库
  document_kb_id: string
  faq_kb_id: string
}

export interface MedicalKBConfigResponse {
  items: MedicalKBConfigItem[]
}

// Category key → display name mapping for labels
export const MEDICAL_KB_LABELS: Record<string, string> = {
  symptom: '症状知识库',
  disease: '疾病知识库',
  drug: '药品知识库',
  lab: '检验检查知识库',
}

// ---------- KB Config API ----------

/** 获取医疗知识库配置（首次自动创建底层 KB） */
export function getMedicalKBConfig() {
  return get('/api/v1/medical/knowledge-base-config')
}

// ---------- FAQ APIs (wrappers with kbId parameter) ----------
// These re-export from knowledge-base/index.ts but we provide
// convenience wrappers with the same signatures for clarity.

export function listFAQEntries(
  kbId: string,
  params?: { page?: number; page_size?: number; keyword?: string },
) {
  return listFAQ(kbId, params)
}

export function createFAQEntry(kbId: string, data: {
  standard_question: string
  answers: string[]
  similar_questions?: string[]
  negative_questions?: string[]
  is_enabled?: boolean
}) {
  return createFAQ(kbId, data)
}

export function updateFAQEntry(kbId: string, entryId: number, data: {
  standard_question: string
  answers: string[]
  similar_questions?: string[]
  negative_questions?: string[]
  is_enabled?: boolean
}) {
  return updateFAQ(kbId, entryId, data)
}

export function upsertFAQEntries(kbId: string, data: { entries: any[]; mode: 'append' | 'replace' }) {
  return upsertFAQ(kbId, data)
}

export function updateFAQEntryFieldsBatch(kbId: string, data: {
  by_id?: Record<number, { is_enabled?: boolean; is_recommended?: boolean; tag_id?: number | null }>
}) {
  return batchUpdateFields(kbId, data)
}

export function deleteFAQEntries(kbId: string, ids: number[]) {
  return deleteFAQ(kbId, ids)
}

export { getFAQImportProgress }
