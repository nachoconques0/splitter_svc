<template>
  <q-layout view="hHh lpR fFf">
    <q-header elevated>
      <q-toolbar>
        <q-toolbar-title>Bill splitter</q-toolbar-title>
      </q-toolbar>
    </q-header>

    <q-page-container>
      <BillPage />
    </q-page-container>
  </q-layout>
</template>

<script setup lang="ts">
import BillPage from '@/pages/BillPage.vue'
</script>
