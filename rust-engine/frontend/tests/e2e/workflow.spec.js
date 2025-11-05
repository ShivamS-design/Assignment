import { test, expect } from '@playwright/test'

test.describe('WASM-as-OS E2E Tests', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('http://localhost:3000')
    await page.fill('[data-testid="username"]', 'admin')
    await page.fill('[data-testid="password"]', 'admin123')
    await page.click('[data-testid="login-btn"]')
    await expect(page.locator('[data-testid="dashboard"]')).toBeVisible()
  })

  test('complete module workflow', async ({ page }) => {
    // Upload module
    await page.click('[data-testid="upload-module"]')
    await page.setInputFiles('[data-testid="file-input"]', 'tests/fixtures/test.wasm')
    await page.click('[data-testid="upload-btn"]')
    await expect(page.locator('.success-message')).toBeVisible()

    // Execute function
    await page.click('[data-testid="execute-function"]')
    await page.fill('[data-testid="function-name"]', 'main')
    await page.click('[data-testid="execute-btn"]')
    await expect(page.locator('.execution-result')).toBeVisible()

    // Check metrics
    await page.click('[data-testid="metrics-tab"]')
    await expect(page.locator('.metrics-chart')).toBeVisible()
  })

  test('scheduler operations', async ({ page }) => {
    await page.click('[data-testid="scheduler-tab"]')
    
    // Create task
    await page.click('[data-testid="create-task"]')
    await page.selectOption('[data-testid="module-select"]', 'test-module')
    await page.selectOption('[data-testid="scheduler-type"]', 'priority')
    await page.fill('[data-testid="priority"]', '5')
    await page.click('[data-testid="create-task-btn"]')
    
    await expect(page.locator('.task-item')).toBeVisible()
  })

  test('real-time metrics', async ({ page }) => {
    await page.click('[data-testid="metrics-tab"]')
    
    // Enable real-time updates
    await page.click('[data-testid="realtime-toggle"]')
    
    // Verify WebSocket connection
    await expect(page.locator('.connection-status.connected')).toBeVisible()
    
    // Check for live updates
    const initialValue = await page.textContent('[data-testid="operations-count"]')
    
    // Trigger some operations
    await page.click('[data-testid="modules-tab"]')
    await page.click('[data-testid="execute-function"]')
    
    await page.click('[data-testid="metrics-tab"]')
    const updatedValue = await page.textContent('[data-testid="operations-count"]')
    
    expect(updatedValue).not.toBe(initialValue)
  })
})