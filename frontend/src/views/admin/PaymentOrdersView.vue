<template>
  <AppLayout>
    <ConstructionState
      title-key="longdao.adminConstruction.paymentOrders.title"
      description-key="longdao.adminConstruction.paymentOrders.description"
      :capability-keys="capabilityKeys"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import AppLayout from '@/components/layout/AppLayout.vue'
import ConstructionState from '@/components/common/ConstructionState.vue'

const capabilityKeys = [
  'longdao.adminConstruction.paymentOrders.capabilities.search',
  'longdao.adminConstruction.paymentOrders.capabilities.reconciliation',
  'longdao.adminConstruction.paymentOrders.capabilities.refunds',
  'longdao.adminConstruction.paymentOrders.capabilities.export'
]
</script>
