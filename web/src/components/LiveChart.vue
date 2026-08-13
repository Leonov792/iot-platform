<template>
  <div ref="el" class="h-72 w-full"></div>
</template>

<script setup>
import { ref, watch, onMounted, onBeforeUnmount } from 'vue'
import * as echarts from 'echarts'

const props = defineProps({ points: { type: Array, default: () => [] } })

const el = ref(null)
let chart = null

function render() {
  if (!chart) return
  const times = props.points.map((p) => new Date(p.ts).toLocaleTimeString())
  chart.setOption({
    tooltip: { trigger: 'axis' },
    legend: { data: ['температура', 'влажность'] },
    grid: { left: 40, right: 20, top: 40, bottom: 30 },
    xAxis: { type: 'category', data: times },
    yAxis: { type: 'value' },
    series: [
      { name: 'температура', type: 'line', smooth: true, data: props.points.map((p) => p.temp) },
      { name: 'влажность', type: 'line', smooth: true, data: props.points.map((p) => p.humidity) }
    ]
  })
}

function resize() {
  chart?.resize()
}

onMounted(() => {
  chart = echarts.init(el.value)
  render()
  window.addEventListener('resize', resize)
})

watch(() => props.points.length, render)

onBeforeUnmount(() => {
  window.removeEventListener('resize', resize)
  chart?.dispose()
})
</script>
