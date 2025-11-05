import { mount } from '@vue/test-utils'
import { describe, it, expect, vi } from 'vitest'
import ModuleUpload from '@/components/ModuleUpload.vue'
import TaskScheduler from '@/components/TaskScheduler.vue'
import MetricsDashboard from '@/components/MetricsDashboard.vue'

describe('ModuleUpload', () => {
  it('uploads WASM module', async () => {
    const mockUpload = vi.fn().mockResolvedValue({ moduleId: 'test-123' })
    const wrapper = mount(ModuleUpload, {
      global: {
        mocks: { $api: { uploadModule: mockUpload } }
      }
    })

    const file = new File(['wasm'], 'test.wasm', { type: 'application/wasm' })
    await wrapper.vm.handleFileUpload({ target: { files: [file] } })

    expect(mockUpload).toHaveBeenCalledWith(file)
    expect(wrapper.emitted('uploaded')).toBeTruthy()
  })

  it('validates file type', () => {
    const wrapper = mount(ModuleUpload)
    const invalidFile = new File(['text'], 'test.txt', { type: 'text/plain' })
    
    expect(wrapper.vm.validateFile(invalidFile)).toBe(false)
  })
})

describe('TaskScheduler', () => {
  it('creates scheduled task', async () => {
    const mockCreate = vi.fn().mockResolvedValue({ taskId: 'task-123' })
    const wrapper = mount(TaskScheduler, {
      global: {
        mocks: { $api: { createTask: mockCreate } }
      }
    })

    await wrapper.vm.createTask({
      moduleId: 'mod-123',
      priority: 5,
      schedulerType: 'round_robin'
    })

    expect(mockCreate).toHaveBeenCalled()
  })

  it('displays task status', () => {
    const wrapper = mount(TaskScheduler, {
      props: {
        tasks: [{ id: '1', status: 'running', progress: 50 }]
      }
    })

    expect(wrapper.text()).toContain('running')
    expect(wrapper.text()).toContain('50%')
  })
})

describe('MetricsDashboard', () => {
  it('renders metrics charts', () => {
    const wrapper = mount(MetricsDashboard, {
      props: {
        metrics: {
          operations: [{ timestamp: Date.now(), value: 100 }],
          memory: [{ timestamp: Date.now(), value: 1024 }]
        }
      }
    })

    expect(wrapper.find('.chart-container')).toBeTruthy()
  })

  it('handles WebSocket updates', async () => {
    const wrapper = mount(MetricsDashboard)
    
    wrapper.vm.handleMetricsUpdate({
      timestamp: Date.now(),
      metrics: { operations: 150 }
    })

    await wrapper.vm.$nextTick()
    expect(wrapper.vm.liveMetrics.operations).toBe(150)
  })
})