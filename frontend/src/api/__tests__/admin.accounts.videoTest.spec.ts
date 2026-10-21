import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get } = vi.hoisted(() => ({ get: vi.fn() }))
vi.mock('@/api/client', () => ({ apiClient: { get } }))
import { getVideoTestTask, getVideoTestContent } from '@/api/admin/accounts'

describe('admin account video test API', () => {
  beforeEach(() => get.mockReset())

  it('queries the original task through the admin endpoint and supports cancellation', async () => {
    const signal = new AbortController().signal
    get.mockResolvedValue({ data: { id: 'task_123', status: 'queued' } })
    await expect(getVideoTestTask(38, 'task_123', signal)).resolves.toEqual({ id: 'task_123', status: 'queued' })
    expect(get).toHaveBeenCalledWith('/admin/accounts/38/video-tasks/task_123', { signal })
  })

  it('gets authenticated video blobs with a media transfer timeout', async () => {
    const signal = new AbortController().signal
    const video = new Blob(['video'], { type: 'video/mp4' })
    get.mockResolvedValue({ data: video })
    await expect(getVideoTestContent(38, 'task_123', signal)).resolves.toBe(video)
    expect(get).toHaveBeenCalledWith('/admin/accounts/38/video-tasks/task_123/content', {
      signal, responseType: 'blob', timeout: 180000
    })
  })
})
