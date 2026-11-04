<template>
  <svg
    xmlns="http://www.w3.org/2000/svg"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    stroke-width="1.8"
    stroke-linecap="round"
    stroke-linejoin="round"
    aria-hidden="true"
    class="shrink-0"
  >
    <path v-for="path in paths" :key="path" :d="path" />
  </svg>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  name: { type: String, required: true },
})

const icons = {
  tree: ['M3 5h7l2 2h9v12H3z', 'M8 11v6', 'M8 14h4'],
  list: ['M8 6h13', 'M8 12h13', 'M8 18h13', 'M3 6h.01', 'M3 12h.01', 'M3 18h.01'],
  'chevron-right': ['m9 18 6-6-6-6'],
  'chevron-down': ['m6 9 6 6 6-6'],
  'sort-asc': ['M3 6h12', 'M3 12h9', 'M3 18h6', 'm16 15 3 3 3-3', 'M19 6v12'],
  'sort-desc': ['M3 6h6', 'M3 12h9', 'M3 18h12', 'm16 9 3-3 3 3', 'M19 18V6'],
  folder: ['M3 5h7l2 2h9v12H3z'],
}

const paths = computed(() => icons[props.name] || icons.list)
</script>
