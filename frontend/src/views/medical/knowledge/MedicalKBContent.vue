<template>
  <div class="medical-kb-content">
    <!-- Tab 切换: 文件 / Q&A -->
    <div class="content-tabs">
      <t-tabs v-model="activeTab" @change="handleTabChange">
        <t-tab-panel value="qa" label="Q&A" />
        <t-tab-panel value="file" label="文件" />
      </t-tabs>
    </div>

    <!-- Q&A Tab -->
    <div v-if="activeTab === 'qa'" class="tab-panel">
      <MedicalFAQList
        v-if="configItem?.faq_kb_id"
        :key="configItem.key"
        ref="faqListRef"
        :faq-kb-id="configItem.faq_kb_id"
        :category="configItem.key"
      />
      <div v-else class="empty-config">
        <t-icon name="info-circle" size="32px" />
        <p>该知识库尚未配置，请先配置底层知识库</p>
      </div>
    </div>

    <!-- 文件 Tab (暂未实现，显示占位) -->
    <div v-if="activeTab === 'file'" class="tab-panel placeholder-section">
      <t-icon name="build" size="48px" style="color: var(--td-text-color-placeholder)" />
      <p class="placeholder-text">文件管理功能正在开发中</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import type { MedicalKBConfigItem } from '@/api/medical/knowledge-base/index'
import MedicalFAQList from './components/MedicalFAQList.vue'

defineProps<{
  configItem: MedicalKBConfigItem | null
}>()

const activeTab = ref('qa')
const faqListRef = ref()

function handleTabChange() {
  // Tab 切换时不做特殊处理，组件内部自己管理状态
}

// Expose reload for parent
defineExpose({
  reload() {
    faqListRef.value?.fetchData()
  },
})
</script>

<style scoped lang="less">
.medical-kb-content {
  margin-top: 4px;
}

.content-tabs {
  margin-bottom: 16px;
  border-bottom: 1px solid var(--td-component-border);
}

.tab-panel {
  min-height: 200px;
}

.empty-config {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 0;
  gap: 12px;
  color: var(--td-text-color-placeholder);
}

.placeholder-section {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 80px 0;
  gap: 16px;
}

.placeholder-text {
  font-size: 14px;
  color: var(--td-text-color-placeholder);
  margin: 0;
}
</style>
