/**
 * Vitest 测试环境设置
 * 提供全局 mock 和测试工具
 */
import { config } from '@vue/test-utils'
import { vi } from 'vitest'

// Node 26 exposes an experimental global localStorage getter that returns
// undefined unless --localstorage-file is configured. Install a deterministic
// in-memory implementation before application modules are imported.
function createMemoryStorage(): Storage {
  const values = new Map<string, string>()

  return {
    get length() {
      return values.size
    },
    clear() {
      values.clear()
    },
    getItem(key: string) {
      return values.get(String(key)) ?? null
    },
    key(index: number) {
      return Array.from(values.keys())[index] ?? null
    },
    removeItem(key: string) {
      values.delete(String(key))
    },
    setItem(key: string, value: string) {
      values.set(String(key), String(value))
    }
  }
}

if (!globalThis.localStorage || typeof globalThis.localStorage.getItem !== 'function') {
  Object.defineProperty(globalThis, 'localStorage', {
    configurable: true,
    value: createMemoryStorage()
  })
}

// Mock requestIdleCallback (Safari < 15 不支持)
if (typeof globalThis.requestIdleCallback === 'undefined') {
  globalThis.requestIdleCallback = ((callback: IdleRequestCallback) => {
    return window.setTimeout(() => callback({ didTimeout: false, timeRemaining: () => 50 }), 1)
  }) as unknown as typeof requestIdleCallback
}

if (typeof globalThis.cancelIdleCallback === 'undefined') {
  globalThis.cancelIdleCallback = ((id: number) => {
    window.clearTimeout(id)
  }) as unknown as typeof cancelIdleCallback
}

// Mock IntersectionObserver
class MockIntersectionObserver {
  observe = vi.fn()
  disconnect = vi.fn()
  unobserve = vi.fn()
}

globalThis.IntersectionObserver = MockIntersectionObserver as unknown as typeof IntersectionObserver

// Mock ResizeObserver
class MockResizeObserver {
  observe = vi.fn()
  disconnect = vi.fn()
  unobserve = vi.fn()
}

globalThis.ResizeObserver = MockResizeObserver as unknown as typeof ResizeObserver

// Vue Test Utils 全局配置
config.global.stubs = {
  // 可以在这里添加全局 stub
}

// 设置全局测试超时
vi.setConfig({ testTimeout: 10000 })
