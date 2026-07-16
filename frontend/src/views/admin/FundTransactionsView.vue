<template>
  <AppLayout>
    <ConstructionState
      title-key="longdao.adminConstruction.fundTransactions.title"
      description-key="longdao.adminConstruction.fundTransactions.description"
      :capability-keys="capabilityKeys"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import AppLayout from '@/components/layout/AppLayout.vue'
import ConstructionState from '@/components/common/ConstructionState.vue'

const capabilityKeys = [
  'longdao.adminConstruction.fundTransactions.capabilities.ledger',
  'longdao.adminConstruction.fundTransactions.capabilities.adjustments',
  'longdao.adminConstruction.fundTransactions.capabilities.auditTrail',
  'longdao.adminConstruction.fundTransactions.capabilities.export'
]
</script>
