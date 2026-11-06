<template>
  <li>
    <div v-if="node.type === 'directory'" class="flex min-h-8 items-center rounded" :style="indent">
      <button
        type="button"
        class="flex min-h-8 w-full min-w-0 items-center gap-1.5 rounded px-1 py-1 text-left text-xs font-medium text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-800"
        :aria-expanded="!isCollapsed"
        :aria-label="`${isCollapsed ? '展开' : '折叠'}目录 ${node.path}`"
        :title="node.path"
        @click="$emit('toggle', node.path)"
      >
        <AppIcon :name="isCollapsed ? 'chevron-right' : 'chevron-down'" class="h-3.5 w-3.5 shrink-0 text-gray-500" />
        <AppIcon name="folder" class="h-4 w-4 shrink-0 text-gray-500" />
        <span class="truncate">{{ node.name }}</span>
      </button>
    </div>
    <button
      v-else
      type="button"
      class="flex min-h-8 w-full items-center gap-2 rounded px-2 py-1 text-left text-xs hover:bg-gray-100 dark:hover:bg-gray-800"
      :class="selected ? 'bg-[var(--selection-bg)] text-[var(--text)]' : 'text-gray-700 dark:text-gray-300'"
      :style="indent"
      :aria-current="selected ? 'page' : undefined"
      :title="node.path"
      @click="$emit('select', node.path)"
    >
      <span class="w-4 shrink-0 text-center font-medium" :class="statusClass">{{ status }}</span>
      <span class="min-w-0 flex-1 truncate font-mono">{{ node.name }}</span>
      <span v-if="reviewCount" class="rounded-full bg-red-100 px-1.5 text-[11px] text-red-700 dark:bg-red-900/40 dark:text-red-300">{{ reviewCount }}</span>
    </button>
    <ul v-if="node.type === 'directory' && !isCollapsed" role="list">
      <FileTreeNode
        v-for="child in node.children"
        :key="getNodeKey(child)"
        :node="child"
        :depth="depth + 1"
        :selected-path="selectedPath"
        :collapsed="collapsed"
        :comment-counts="commentCounts"
        @select="$emit('select', $event)"
        @toggle="$emit('toggle', $event)"
      />
    </ul>
  </li>
</template>

<script setup>
import { computed } from 'vue'
import AppIcon from './AppIcon.vue'
import { getNodeKey } from '../utils/fileExplorer.js'

defineOptions({ name: 'FileTreeNode' })

const props = defineProps({
  node: { type: Object, required: true },
  depth: { type: Number, default: 0 },
  selectedPath: { type: String, default: '' },
  collapsed: { type: Set, required: true },
  commentCounts: { type: Map, required: true },
})
defineEmits(['select', 'toggle'])

const isCollapsed = computed(() => props.collapsed.has(props.node.path))
const selected = computed(() => props.node.type === 'file' && props.selectedPath === props.node.path)
const reviewCount = computed(() => props.commentCounts.get(props.node.path) || 0)
const indent = computed(() => ({ paddingLeft: `${props.depth * 12 + 8}px` }))
const status = computed(() => props.node.file?.isNew ? 'A' : props.node.file?.isDeleted ? 'D' : 'M')
const statusClass = computed(() => props.node.file?.isNew ? 'text-green-700' : props.node.file?.isDeleted ? 'text-red-700' : 'text-gray-500')
</script>
