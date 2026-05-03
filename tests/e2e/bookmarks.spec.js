import { expect, test } from '@playwright/test'

test.describe('Keyboard shortcuts', () => {
	test('Ctrl+K focuses search', async ({ page }) => {
		await page.goto('/')
		await page.keyboard.press('Control+k')
		await expect(page.locator('input[name="q"]')).toBeFocused()
	})

	test('Escape blurs search', async ({ page }) => {
		await page.goto('/')
		await page.locator('input[name="q"]').focus()
		await expect(page.locator('input[name="q"]')).toBeFocused()
		await page.keyboard.press('Escape')
		await expect(page.locator('input[name="q"]')).not.toBeFocused()
	})
})

test.describe('Add bookmark dialog', () => {
	test('opens on button click', async ({ page }) => {
		await page.goto('/')
		await page.click('text=+ Add Bookmark')
		await expect(page.locator('#add-bookmark-dialog')).toHaveAttribute(
			'open',
			'',
		)
	})

	test('closes on Cancel', async ({ page }) => {
		await page.goto('/')
		await page.click('text=+ Add Bookmark')
		await expect(page.locator('#add-bookmark-dialog')).toHaveAttribute(
			'open',
			'',
		)
		await page
			.locator('#add-bookmark-dialog')
			.locator('button:has-text("Cancel")')
			.click({ force: true })
		await expect(page.locator('#add-bookmark-dialog')).not.toHaveAttribute(
			'open',
			'',
		)
	})

	test('resets form on close', async ({ page }) => {
		await page.goto('/')
		await page.click('text=+ Add Bookmark')
		await page.fill('#bookmark-title', 'test title')
		await page.fill('#bookmark-url', 'https://test.com')
		await page
			.locator('#add-bookmark-dialog')
			.locator('button:has-text("Cancel")')
			.click({ force: true })
		await page.click('text=+ Add Bookmark')
		await expect(page.locator('#bookmark-title')).toHaveValue('')
		await expect(page.locator('#bookmark-url')).toHaveValue('')
	})

	test('creates bookmark and closes dialog', async ({ page }) => {
		await page.goto('/')
		await page.click('text=+ Add Bookmark')
		await page.fill('#bookmark-title', 'Playwright Test')
		await page.fill('#bookmark-url', 'https://playwright.dev')
		await page.fill('#bookmark-tags', 'test, e2e')
		await page
			.locator('#add-bookmark-dialog')
			.locator('button:has-text("Save")')
			.click({ force: true })
		await expect(page.locator('#add-bookmark-dialog')).not.toHaveAttribute(
			'open',
			'',
		)
		await expect(page.locator('#bookmark-list')).toContainText(
			'Playwright Test',
		)
	})
})

test.describe('Theme toggle', () => {
	test('toggles between dark and light', async ({ page }) => {
		await page.goto('/')
		// Get initial state (may be light or dark depending on prefers-color-scheme)
		const initialTheme = await page.locator('html').getAttribute('data-theme')
		await page.click('button[aria-label="Toggle dark mode"]')
		const afterToggle = await page.locator('html').getAttribute('data-theme')
		expect(afterToggle).not.toBe(initialTheme)
		await page.click('button[aria-label="Toggle dark mode"]')
		const afterSecondToggle = await page
			.locator('html')
			.getAttribute('data-theme')
		expect(afterSecondToggle).toBe(initialTheme)
	})
})
