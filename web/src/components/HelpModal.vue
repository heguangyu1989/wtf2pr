<template>
  <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 px-4">
    <div class="bg-white dark:bg-gray-900 rounded-lg shadow-xl w-full max-w-xl max-h-[80vh] flex flex-col">
      <div class="px-4 py-3 border-b flex items-center justify-between">
        <div class="font-medium">使用帮助</div>
        <button class="text-gray-500 hover:text-gray-700" @click="$emit('close')">关闭</button>
      </div>
      <div class="p-4 overflow-auto text-sm space-y-3 leading-relaxed">
        <p><strong>wtf2pr</strong> 是一个本地 Git 代码 Review 工具，用于将未提交的变更或某个 Commit 的 Diff 可视化展示，并输出结构化 Review 报告给 AI 修复。</p>
        <ul class="list-disc pl-5 space-y-1">
          <li><strong>Working tree</strong>：查看当前工作目录所有未提交的变更（含已修改、已暂存、未跟踪的新文件）。</li>
          <li><strong>Commit</strong>：选择某个历史 Commit，查看该 Commit 的完整 Diff。</li>
          <li><strong>添加 Review</strong>：点击代码行即可在对应行下方添加行级评论；点击文件标题栏“添加文件评论”可对整体文件发表意见。</li>
          <li><strong>保存 Review</strong>：点击右上角“保存 Review”，数据会自动持久化到 JSON 文件。</li>
          <li><strong>导出</strong>：支持 Markdown / JSON / XML，仅导出包含 Review 的核心信息（文件、行号、对应代码、评论内容），避免冗余 Token。</li>
        </ul>
        <p class="text-xs text-gray-500 pt-2">快捷键：无 | 开发模式下前端通过 Vite Proxy 访问后端 API。</p>
      </div>
    </div>
  </div>
</template>

<script setup>
defineEmits(['close'])
</script>
