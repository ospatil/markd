import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: './tests/e2e',
  timeout: 10000,
  use: {
    baseURL: 'http://localhost:3000',
    headless: true,
  },
  projects: [
    { name: 'chromium', use: { browserName: 'chromium' } },
  ],
  webServer: {
    command: 'make run',
    url: 'http://localhost:3000',
    reuseExistingServer: true,
    timeout: 30000,
  },
});
