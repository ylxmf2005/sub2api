import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { resourceSupplyAPI } from '@/api/resourceSupply'
import { extractApiErrorMessage } from '@/utils/apiError'
import type { ResourceSupplyOwnedAccount } from '@/types'

export function useResourceSupplyOAuth() {
  const appStore = useAppStore()
  const { t } = useI18n()

  // State
  const authUrl = ref('')
  const sessionId = ref('')
  const oauthState = ref('')
  const loading = ref(false)
  const error = ref('')

  // Reset state
  const resetState = () => {
    authUrl.value = ''
    sessionId.value = ''
    oauthState.value = ''
    loading.value = false
    error.value = ''
  }

  // Generate auth URL for OpenAI OAuth
  const generateAuthUrl = async (
    proxyId?: number | null,
    redirectUri?: string
  ): Promise<boolean> => {
    loading.value = true
    authUrl.value = ''
    sessionId.value = ''
    oauthState.value = ''
    error.value = ''

    try {
      const payload: { redirect_uri?: string; proxy_id?: number | null } = {}
      if (proxyId) {
        payload.proxy_id = proxyId
      }
      if (redirectUri) {
        payload.redirect_uri = redirectUri
      }

      const response = await resourceSupplyAPI.generateOpenAIAuthUrl(payload)
      authUrl.value = response.auth_url
      sessionId.value = response.session_id
      try {
        const parsed = new URL(response.auth_url)
        oauthState.value = parsed.searchParams.get('state') || ''
      } catch {
        oauthState.value = ''
      }
      return true
    } catch (err: any) {
      error.value = extractApiErrorMessage(err, t('admin.accounts.oauth.openai.failedToGenerateUrl'))
      appStore.showError(error.value)
      return false
    } finally {
      loading.value = false
    }
  }

  // Exchange auth code for a resource supply owned account
  const exchangeAuthCode = async (params: {
    group_id: number
    name?: string
    code: string
    state: string
    session_id: string
    redirect_uri?: string
    proxy_id?: number | null
    model_id?: string
  }): Promise<ResourceSupplyOwnedAccount | null> => {
    if (!params.code.trim() || !params.session_id || !params.state.trim()) {
      error.value = 'Missing auth code, session ID, or state'
      return null
    }

    loading.value = true
    error.value = ''

    try {
      const account = await resourceSupplyAPI.exchangeOpenAICode({
        group_id: params.group_id,
        name: params.name,
        session_id: params.session_id,
        code: params.code.trim(),
        state: params.state.trim(),
        redirect_uri: params.redirect_uri,
        proxy_id: params.proxy_id,
        model_id: params.model_id,
      })
      return account
    } catch (err: any) {
      error.value = extractApiErrorMessage(err, t('admin.accounts.oauth.openai.failedToExchangeCode'))
      appStore.showError(error.value)
      return null
    } finally {
      loading.value = false
    }
  }

  return {
    // State
    authUrl,
    sessionId,
    oauthState,
    loading,
    error,
    // Methods
    resetState,
    generateAuthUrl,
    exchangeAuthCode,
  }
}
