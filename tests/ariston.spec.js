import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";
import { startSimulator, stopSimulator, simulatorUrl } from "./simulator";

const ARISTON_CONFIG = "ariston.evcc.yaml";

test.use({ baseURL: baseUrl() });

test.beforeAll(async () => {
  await startSimulator();
  await start(ARISTON_CONFIG);
});

test.afterAll(async () => {
  await stop();
  await stopSimulator();
});

test("Ariston water heater configuration", async ({ page }) => {
  await page.goto("/#/config");
  
  // Test basic configuration
  await expect(page.getByText("Ariston Water Heater")).toBeVisible();
  
  // Test authentication
  await expect(page.getByLabel("Username")).toBeVisible();
  await expect(page.getByLabel("Password")).toBeVisible();
  await expect(page.getByLabel("Device ID")).toBeVisible();
  
  // Test temperature settings
  await expect(page.getByText("Comfort Temperature")).toBeVisible();
  await expect(page.getByText("Reduced Temperature")).toBeVisible();
}); 