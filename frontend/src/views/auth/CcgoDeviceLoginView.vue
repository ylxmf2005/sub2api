<template>
  <AuthLayout>
    <div class="space-y-6">
      <div class="text-center">
        <div class="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-lg bg-primary-50 text-primary-600 dark:bg-primary-500/10 dark:text-primary-400">
          <Icon name="terminal" size="lg" />
        </div>
        <h2 class="text-2xl font-bold text-gray-900 dark:text-white">Authorize ccgo</h2>
        <p class="mt-2 text-sm text-gray-500 dark:text-dark-400">
          Confirm the code shown in your terminal before continuing.
        </p>
      </div>

      <div class="space-y-4">
        <label for="ccgo-device-code" class="input-label text-center">Device code</label>
        <input
          id="ccgo-device-code"
          v-model="userCode"
          type="text"
          autocomplete="one-time-code"
          autocapitalize="characters"
          spellcheck="false"
          maxlength="9"
          class="input py-3 text-center font-mono text-xl uppercase tracking-[0.24em]"
          :class="{ 'input-error': errorMessage }"
          :disabled="status === 'approved' || isLoading"
          placeholder="ABCD-EFGH"
          @input="formatCode"
        />

        <div
          v-if="status === 'approved'"
          class="rounded-lg border border-green-200 bg-green-50 p-4 dark:border-green-800/50 dark:bg-green-900/20"
        >
          <div class="flex items-start gap-3">
            <Icon name="checkCircle" size="md" class="mt-0.5 text-green-500" />
            <p class="text-sm text-green-700 dark:text-green-400">
              ccgo login approved. You can return to your terminal.
            </p>
          </div>
        </div>

        <div
          v-else-if="errorMessage"
          class="rounded-lg border border-red-200 bg-red-50 p-4 dark:border-red-800/50 dark:bg-red-900/20"
        >
          <div class="flex items-start gap-3">
            <Icon name="exclamationCircle" size="md" class="mt-0.5 text-red-500" />
            <p class="text-sm text-red-700 dark:text-red-400">{{ errorMessage }}</p>
          </div>
        </div>

        <button
          type="button"
          :disabled="isLoading || status === 'approved' || normalizedCode.length !== 8"
          class="btn btn-primary w-full"
          @click="approve"
        >
          <svg
            v-if="isLoading"
            class="-ml-1 mr-2 h-4 w-4 animate-spin text-white"
            fill="none"
            viewBox="0 0 24 24"
          >
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
          </svg>
          <Icon v-else name="shield" size="md" class="mr-2" />
          {{ isLoading ? 'Authorizing...' : 'Authorize' }}
        </button>
      </div>
    </div>
  </AuthLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { AuthLayout } from '@/components/layout'
import Icon from '@/components/icons/Icon.vue'
import { apiClient } from '@/api/client'

const route = useRoute()
const userCode = ref('')
const status = ref<'idle' | 'approved'>('idle')
const isLoading = ref(false)
const errorMessage = ref('')

const normalizedCode = computed(() => userCode.value.replace(/[^A-Z0-9]/gi, '').toUpperCase())

function formatCode() {
  const code = normalizedCode.value.slice(0, 8)
  userCode.value = code.length > 4 ? `${code.slice(0, 4)}-${code.slice(4)}` : code
  errorMessage.value = ''
}

async function approve() {
  formatCode()
  if (normalizedCode.value.length !== 8) {
    errorMessage.value = 'Enter the 8-character code from your terminal.'
    return
  }
  isLoading.value = true
  errorMessage.value = ''
  try {
    await apiClient.post('/ccgo/device-login/approve', { user_code: userCode.value })
    status.value = 'approved'
  } catch (error: any) {
    errorMessage.value = error?.message || 'Unable to authorize this ccgo login.'
  } finally {
    isLoading.value = false
  }
}

onMounted(() => {
  const code = route.query.code
  if (typeof code === 'string') {
    userCode.value = code
    formatCode()
  }
})
</script>
