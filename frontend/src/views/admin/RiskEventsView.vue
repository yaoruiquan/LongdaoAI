<template>
  <AppLayout>
    <ConstructionState
      title-key="longdao.adminConstruction.riskEvents.title"
      description-key="longdao.adminConstruction.riskEvents.description"
      :capability-keys="capabilityKeys"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import AppLayout from '@/components/layout/AppLayout.vue'
import ConstructionState from '@/components/common/ConstructionState.vue'

const capabilityKeys = [
  'longdao.adminConstruction.riskEvents.capabilities.rules',
  'longdao.adminConstruction.riskEvents.capabilities.alerts',
  'longdao.adminConstruction.riskEvents.capabilities.reviewQueue',
  'longdao.adminConstruction.riskEvents.capabilities.reporting'
]
</script>
