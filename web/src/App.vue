<template>
  <el-config-provider :locale="locale">
    <router-view />
  </el-config-provider>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import zhCn from 'element-plus/dist/locale/zh-cn.mjs'

const locale = ref(zhCn)
</script>

<style>
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

a {
  text-decoration: none;
  color: inherit;
}

button {
  border: none;
  background: none;
  cursor: pointer;
  font-family: inherit;
}

html,
body,
#app {
  width: 100%;
  height: 100%;
  overflow: hidden;
}

#app {
  font-family: 'Inter', 'Noto Sans SC', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  color: #0f172a;
  background: #f8fafc;
}

/* Element Plus theme overrides to match demo */
:root {
  --el-color-primary: #3b82f6;
  --el-color-primary-light-3: #60a5fa;
  --el-color-primary-light-5: #93bbfd;
  --el-color-primary-light-7: #bfdbfe;
  --el-color-primary-light-9: #dbeafe;
  --el-color-primary-dark-2: #2563eb;
  --el-border-radius-base: 12px;
  --el-border-radius-small: 8px;
  --el-font-family: 'Inter', 'Noto Sans SC', sans-serif;
}

/* Card style overrides */
.el-card {
  border-radius: 16px !important;
  border: 1px solid #f1f5f9 !important;
  box-shadow: 0 1px 2px 0 rgb(0 0 0 / 0.05) !important;
}

.el-card__header {
  border-bottom: 1px solid #f1f5f9 !important;
}

/* Table style overrides */
.el-table {
  --el-table-border-color: #f1f5f9;
  --el-table-header-bg-color: #f8fafc;
  border-radius: 12px;
  overflow: hidden;
}

.el-table th.el-table__cell {
  background-color: #f8fafc !important;
  color: #64748b;
  font-size: 11px;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  font-weight: 700;
}

/* Button style overrides */
.el-button--primary {
  --el-button-bg-color: #3b82f6;
  --el-button-border-color: #3b82f6;
  --el-button-hover-bg-color: #2563eb;
  --el-button-hover-border-color: #2563eb;
  border-radius: 8px;
}

.el-button {
  border-radius: 8px;
}

/* Dialog style overrides */
.el-dialog {
  border-radius: 16px !important;
}

/* Tag style overrides */
.el-tag {
  border-radius: 6px;
}

/* Input style overrides */
.el-input__wrapper {
  border-radius: 12px !important;
}

.el-select .el-input__wrapper {
  border-radius: 12px !important;
}

/* Pagination */
.el-pagination {
  --el-pagination-button-bg-color: white;
}

/* Tabs */
.el-tabs--border-card {
  border-radius: 16px;
  border: 1px solid #f1f5f9;
  box-shadow: 0 1px 2px 0 rgb(0 0 0 / 0.05);
}
</style>
