<template>
  <div class="medical-search-page">
    <div class="header">
      <h2>检索测试</h2>
      <p class="subtitle">三路统一检索：文档 · FAQ · Wiki</p>
    </div>

    <!-- 知识库选择 -->
    <div style="margin-bottom:16px">
      <span style="font-size:13px;color:#666;margin-right:12px">选择知识库：</span>
      <t-checkbox v-for="c in kbCards" :key="c.key" v-show="c.key!=='department'"
        :checked="selectedCategories.includes(c.key)"
        @change="(v:boolean)=>toggleCat(c.key,v)"
        style="margin-right:16px">{{ c.title }}</t-checkbox>
    </div>

    <!-- 查询词 -->
    <t-input v-model="query" placeholder="输入查询内容，如：发热怎么处理" clearable @enter="doSearch" style="margin-bottom:16px;max-width:500px" />

    <!-- 可配置参数 -->
    <div class="param-grid">
      <div class="param-item">
        <label>文档匹配数 (match_count)</label>
        <input v-model.number="params.docMatchCount" type="number" min="1" max="100">
        <span class="hint">HybridSearch 最终返回条数</span>
      </div>
      <div class="param-item">
        <label>FAQ 匹配数 (match_count)</label>
        <input v-model.number="params.faqMatchCount" type="number" min="1" max="100">
        <span class="hint">FAQ Search 返回条数</span>
      </div>
      <div class="param-item">
        <label>Wiki 页数 (limit)</label>
        <input v-model.number="params.wikiLimit" type="number" min="1" max="100">
        <span class="hint">Wiki Search 返回条数</span>
      </div>
      <div class="param-item">
        <label>向量阈值 (vector_threshold)</label>
        <input v-model.number="params.vectorThreshold" type="number" step="0.05" min="0" max="1">
        <span class="hint">低于此分的向量结果丢弃，默认 0.15</span>
      </div>
      <div class="param-item">
        <label>关键词阈值 (keyword_threshold)</label>
        <input v-model.number="params.keywordThreshold" type="number" step="0.05" min="0" max="1">
        <span class="hint">低于此分的关键词结果丢弃，默认 0.3</span>
      </div>
    </div>

    <!-- 不可配置参数说明 -->
    <t-collapse style="margin-bottom:16px">
      <t-collapse-panel header="📋 未暴露的参数及原因（不可通过 API 传入，后端自动读取租户配置）" value="1">
        <t-table :data="hiddenParams" :columns="hiddenColumns" size="small" row-key="param" bordered />
      </t-collapse-panel>
    </t-collapse>

    <t-button theme="primary" :loading="searching" :disabled="!query||!selectedCategories.length" @click="doSearch" style="margin-bottom:16px">
      {{ searching ? '检索中...' : '执行检索' }}
    </t-button>

    <t-alert v-if="errorMsg" theme="error" :message="errorMsg" close @close="errorMsg=''" style="margin-bottom:12px" />

    <!-- 结果区域 -->
    <div v-if="results">
      <!-- 三路原始 JSON（和独立前端一样，三栏并排） -->
      <t-row :gutter="12" style="margin-bottom:12px">
        <t-col :span="4">
          <t-card title="文档 HybridSearch" size="small"><pre class="json-pre">{{ results.doc }}</pre></t-card>
        </t-col>
        <t-col :span="4">
          <t-card title="FAQ Search" size="small"><pre class="json-pre">{{ results.faq }}</pre></t-card>
        </t-col>
        <t-col :span="4">
          <t-card title="Wiki Search" size="small"><pre class="json-pre">{{ results.wiki }}</pre></t-card>
        </t-col>
      </t-row>

      <!-- 融合结果 JSON -->
      <t-card title="融合结果 JSON（按 Wiki > FAQ > Document 排序，同标题去重）" size="small" style="margin-bottom:12px">
        <pre class="json-pre">{{ results.unifiedJson }}</pre>
      </t-card>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { post } from '@/utils/request'
import { getMedicalKBConfig } from '@/api/medical/knowledge-base/index'

interface KBCard { key:string; icon:string; title:string; docKbId:string; faqKbId:string }

const kbCards = ref<KBCard[]>([
  { key:'symptom', icon:'💊', title:'症状知识库', docKbId:'', faqKbId:'' },
  { key:'disease', icon:'🩺', title:'疾病知识库', docKbId:'', faqKbId:'' },
  { key:'drug',    icon:'💉', title:'药品知识库', docKbId:'', faqKbId:'' },
  { key:'lab',     icon:'🔬', title:'检验检查知识库', docKbId:'', faqKbId:'' },
])

const hiddenColumns = [
  { colKey: 'param', title: '参数', width: 180 },
  { colKey: 'default', title: '默认值', width: 80 },
  { colKey: 'reason', title: '不可传入原因' },
]
const hiddenParams = [
  { param: 'embedding_top_k', default: '50', reason: 'SearchParams 无此字段，后端从 tenants.retrieval_config 读取' },
  { param: 'rrf_vector_weight', default: '0.7', reason: '同上，租户级检索配置，需调 PUT /tenants/kv/retrieval-config 修改' },
  { param: 'rrf_keyword_weight', default: '0.3', reason: '同上' },
  { param: 'rrf_k', default: '60', reason: 'RRF 平滑常数，同上' },
  { param: '过度检索倍数', default: '5x', reason: '硬编码在 knowledgebase_search.go:151，不可配置' },
  { param: '过度检索上限', default: '500', reason: '硬编码在 knowledgebase_search.go:152，不可配置' },
]

const selectedCategories = ref<string[]>([])
const query = ref('')
const searching = ref(false)
const errorMsg = ref('')
const results = ref<any>(null)

const params = reactive({
  docMatchCount: 10, faqMatchCount: 5, wikiLimit: 5,
  vectorThreshold: 0.15, keywordThreshold: 0.3,
})

function toggleCat(key: string, v: boolean) {
  if (v) selectedCategories.value.push(key)
  else selectedCategories.value = selectedCategories.value.filter(k => k !== key)
}

async function doSearch() {
  searching.value = true; errorMsg.value = ''; results.value = null
  try {
    const res: any = await post('/api/v1/medical/search', {
      categories: selectedCategories.value,
      query: query.value,
      doc_match_count: params.docMatchCount,
      faq_match_count: params.faqMatchCount,
      wiki_limit: params.wikiLimit,
      vector_threshold: params.vectorThreshold,
      keyword_threshold: params.keywordThreshold,
    })
    console.log('MedicalSearch res:', res)
    console.log('MedicalSearch res.data keys:', Object.keys(res.data||{}))
    console.log('MedicalSearch faq_raw:', res.data?.faq_raw)
    console.log('MedicalSearch wiki_raw:', res.data?.wiki_raw)
    if (res.success) {
      const docRaw = res.data?.doc_raw
      const faqRaw = res.data?.faq_raw
      const wikiRaw = res.data?.wiki_raw
      results.value = {
        unified: res.data?.results || [],
        unifiedJson: JSON.stringify(res.data?.results || [], null, 2),
        doc: JSON.stringify(Array.isArray(docRaw) ? docRaw : [], null, 2),
        faq: JSON.stringify(Array.isArray(faqRaw) ? faqRaw : [], null, 2),
        wiki: JSON.stringify(Array.isArray(wikiRaw) ? wikiRaw : [], null, 2),
      }
      console.log('MedicalSearch results:', {doc:results.value.doc?.length, faq:results.value.faq?.length, wiki:results.value.wiki?.length})
    } else {
      errorMsg.value = res.error?.message || '检索失败'
    }
  } catch(e:any) {
    errorMsg.value = e?.response?.data?.error?.message || e?.message || '检索失败'
  } finally { searching.value = false }
}

onMounted(async () => {
  try {
    const res = await getMedicalKBConfig()
    if (res.success && res.data?.items) {
      for (const item of res.data.items) {
        const card = kbCards.value.find(c => c.key === item.key)
        if (card) { card.docKbId = item.document_kb_id || ''; card.faqKbId = item.faq_kb_id || '' }
      }
    }
  } catch {}
})
</script>

<style scoped lang="less">
.medical-search-page { padding: 24px; max-width: 1200px; margin: 0 auto; height: 100%; overflow-y: auto; }
.header { margin-bottom: 20px; }
.header h2 { font-size: 20px; font-weight: 600; margin: 0 0 6px; }
.subtitle { font-size: 13px; color: var(--td-text-color-placeholder); margin: 0; }

.param-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 10px; margin-bottom: 16px; }
.param-item { display: flex; flex-direction: column; gap: 4px; }
.param-item label { font-size: 12px; color: #666; }
.param-item input { width: 100%; padding: 6px 10px; border: 1px solid var(--td-component-border); border-radius: 6px; font-size: 13px; }
.param-item .hint { font-size: 10px; color: #999; }

.result-list { max-height: 500px; overflow-y: auto; }
.result-item { border-bottom: 1px solid var(--td-component-border); padding: 8px 0; }
.result-item:last-child { border-bottom: none; }
.result-header { display: flex; align-items: center; gap: 8px; margin-bottom: 4px; }
.result-title { font-weight: 500; font-size: 13px; }
.result-score { color: #999; font-size: 11px; }
.result-content { color: #666; font-size: 12px; line-height: 1.6; }

.json-pre { max-height: 500px; overflow: auto; font-size: 10px; line-height: 1.4; white-space: pre-wrap; word-break: break-all; margin: 0; padding: 8px; background: #fafafa; border-radius: 4px; }
.empty { text-align: center; padding: 40px 0; color: var(--td-text-color-placeholder); font-size: 13px; }
</style>
