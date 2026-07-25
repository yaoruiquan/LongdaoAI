<template>
  <div class="bg-gray-50/50 dark:bg-dark-700/30">
    <div v-if="loading" class="flex items-center justify-center py-3">
      <LoadingSpinner />
    </div>
    <div v-else-if="items.length === 0" class="py-2 text-center text-xs text-gray-400">
      {{ t('admin.dashboard.noDataAvailable') }}
    </div>
    <table v-else class="w-full text-xs">
      <tbody>
        <tr
          v-for="user in items"
          :key="user.user_id"
          class="border-t border-gray-100/50 dark:border-gray-700/50"
        >
          <td class="max-w-[120px] truncate py-1 pl-6 text-gray-600 dark:text-gray-300" :title="user.email">
            {{ user.email || `User #${user.user_id}` }}
          </td>
          <td class="py-1 text-right text-gray-500 dark:text-gray-400">
            {{ user.requests.toLocaleString() }}
          </td>
          <td class="py-1 text-right text-gray-500 dark:text-gray-400">
            {{ formatTokens(user.total_tokens) }}
          </td>
          <td class="py-1 text-right text-green-600 dark:text-green-400">
            {{ money(user.actual_cost) }}
          </td>
          <td v-if="showAccountCost" class="py-1 text-right text-orange-500 dark:text-orange-400">
            {{ money(user.account_cost) }}
          </td>
          <td class="py-1 pr-1 text-right text-gray-400 dark:text-gray-500">
            {{ money(user.cost) }}
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import type { UserBreakdownItem } from '@/types'

const { t } = useI18n()

const props = withDefaults(defineProps<{
  items: UserBreakdownItem[]
  loading?: boolean
  showAccountCost?: boolean
  /** 用户端展示层汇率（1 USD → CNY）。传正数则金额按 ¥ 显示，缺省/非正数按 $ 显示（管理端默认）。 */
  displayCnyRate?: number | null
}>(), {
  loading: false,
  showAccountCost: true,
  displayCnyRate: null,
})

const showAccountCost = computed(() => props.showAccountCost)

// 金额展示前缀与换算：用户端传 displayCnyRate 显示 ¥（USD×汇率），管理端不传保持 $。
const isCny = computed(
  () => typeof props.displayCnyRate === 'number' && Number.isFinite(props.displayCnyRate) && props.displayCnyRate > 0
)
const money = (value: number | undefined | null): string => {
  const usd = typeof value === 'number' && Number.isFinite(value) ? value : 0
  return isCny.value ? `¥${formatCost(usd * (props.displayCnyRate as number))}` : `$${formatCost(usd)}`
}

const formatTokens = (value: number): string => {
  if (value >= 1_000_000_000) return `${(value / 1_000_000_000).toFixed(2)}B`
  if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(2)}M`
  if (value >= 1_000) return `${(value / 1_000).toFixed(2)}K`
  return value.toLocaleString()
}

const formatCost = (value: number | undefined | null): string => {
  if (value == null) return '0.0000'
  if (value >= 1000) return (value / 1000).toFixed(2) + 'K'
  if (value >= 1) return value.toFixed(2)
  if (value >= 0.01) return value.toFixed(3)
  return value.toFixed(4)
}
</script>
