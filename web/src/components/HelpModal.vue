<template>
  <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 px-4">
    <div class="bg-white dark:bg-gray-900 rounded-lg shadow-xl w-full max-w-xl max-h-[80vh] flex flex-col">
      <div class="px-4 py-3 border-b flex items-center justify-between">
        <div class="font-medium">使用帮助</div>
        <button class="text-gray-500 hover:text-gray-700" @click="$emit('close')">关闭</button>
      </div>
      <div class="p-4 overflow-auto text-sm space-y-3 leading-relaxed">
        <p><strong>wtf2pr</strong> 是一个本地 Git 代码 Review 工具，用于查看变更、记录意见并输出结构化报告给 AI 修复。</p>
        <ul class="list-disc pl-5 space-y-1">
          <li><strong>未提交</strong>：查看当前工作目录所有未提交的变更（含已修改、已暂存、未跟踪的新文件）。</li>
          <li><strong>Commit</strong>：选择某个历史 Commit，查看该 Commit 的完整 Diff。</li>
          <li><strong>分支对比</strong>：选择一个目标本地分支，按“目标分支...HEAD”查看当前分支自共同祖先以来的提交变更，不含未提交修改。</li>
          <li><strong>切换工作目录</strong>：点击审查工具栏中的当前路径，在弹窗中输入 Git 仓库绝对路径或子目录。</li>
          <li><strong>文件浏览</strong>：左侧支持树形、平铺、文件夹折叠、路径/类型/测试/文档排序和连续滚动；文件夹仅控制展开状态。包含/排除规则支持 VS Code 风格 glob，仅过滤左侧文件，偏好保存在当前浏览器。</li>
          <li><strong>查看完整文件</strong>：文件标题栏可打开新旧版本，支持行号、复制、Esc 和遮罩关闭。</li>
          <li><strong>添加 Review</strong>：在指定 Commit 模式中，点击代码行或“添加文件评论”记录意见。</li>
          <li><strong>保存 Review</strong>：点击右上角“保存 Review”，数据会自动持久化到 JSON 文件。</li>
          <li><strong>导出</strong>：支持 Markdown / JSON / XML / 自定义 Template，导出当前评论快照及关键代码，避免冗余 Token。</li>
        </ul>
        <p class="text-xs text-gray-500 pt-2">快捷键：Esc 关闭完整文件弹窗 | 开发模式下前端通过 Vite Proxy 访问后端 API。</p>
      </div>
    </div>
  </div>
</template>

<script setup>
defineEmits(['close'])
</script>
