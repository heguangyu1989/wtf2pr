<template>
  <div class="fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center z-50 px-4" @click.self="$emit('close')">
    <div class="bg-white dark:bg-gray-900 rounded-xl shadow-2xl w-full max-w-xl max-h-[80vh] flex flex-col border border-gray-200/60 dark:border-gray-700/60">
      <div class="px-5 py-3.5 border-b dark:border-gray-700 flex items-center justify-between">
        <div class="font-semibold text-sm">使用帮助</div>
        <button class="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 text-xl leading-none px-1" @click="$emit('close')">×</button>
      </div>
      <div class="p-5 overflow-auto text-sm space-y-3 leading-relaxed">
        <p><strong>wtf2pr</strong> 是一个本地 Git 代码 Review 工具，用于将未提交的变更或某个 Commit 的 Diff 可视化展示，并输出结构化 Review 报告给 AI 修复。</p>
        <ul class="list-disc pl-5 space-y-1.5">
          <li><strong>访问令牌</strong>：服务启动时会生成随机 Token（也可在配置文件中指定固定值），打开启动日志中带 token 的链接或手动输入即可进入。</li>
          <li><strong>工作目录</strong>：点击顶部目录名可切换工作目录；每个目录的 Review 数据独立存储、互不影响。</li>
          <li><strong>Working tree</strong>：查看当前工作目录所有未提交的变更（含已修改、已暂存、未跟踪的新文件）。</li>
          <li><strong>Commit</strong>：选择某个历史 Commit，查看该 Commit 的完整 Diff。</li>
          <li><strong>添加 Review</strong>：点击代码行即可在对应行下方添加行级评论；点击文件标题栏“添加文件评论”可对整体文件发表意见。</li>
          <li><strong>保存 Review</strong>：点击右上角“保存 Review”，数据会自动持久化到 JSON 文件。</li>
          <li><strong>导出</strong>：支持 Markdown / JSON / XML / 自定义 Template，仅导出包含 Review 的核心信息（文件、行号、对应代码、评论内容），避免冗余 Token。</li>
        </ul>
        <p class="text-xs text-gray-500 pt-2">开发模式下前端通过 Vite Proxy 访问后端 API。</p>
      </div>
    </div>
  </div>
</template>

<script setup>
defineEmits(['close'])
</script>
