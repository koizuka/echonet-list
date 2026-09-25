/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly DEV: boolean
  readonly PROD: boolean
  readonly MODE: string
  readonly VITE_WS_URL?: string
  BUILD_DATE: string // defined by vite.config.ts; overwritten in tests
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}