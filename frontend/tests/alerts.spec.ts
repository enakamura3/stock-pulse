import { test, expect } from '@playwright/test';

test.describe('Fluxo E2E de Alertas de Preço', () => {
  const testEmail = `e2e_alerts_${Date.now()}@test.com`;
  const testPassword = 'Password123!';
  const testName = 'E2E Alert User';

  test.beforeEach(async ({ page }) => {
    // Redireciona chamadas da API do frontend (localhost:8080) para o container backend (backend:8080)
    await page.route('http://localhost:8080/**/*', async (route) => {
      const url = route.request().url().replace('localhost:8080', 'backend:8080');
      const response = await route.fetch({ url });
      await route.fulfill({ response });
    });

    // Registra o usuário e redireciona para o Dashboard
    await page.goto('/register');
    await page.fill('input[type="text"]', testName);
    await page.fill('input[type="email"]', testEmail);
    await page.fill('input[type="password"]', testPassword);
    await page.click('button[type="submit"]');

    await expect(page).toHaveURL(/\/dashboard/, { timeout: 10000 });
  });

  test('deve criar um alerta de preço, alternar status e excluir o alerta', async ({ page }) => {
    // 1. Navega para a página de Alertas
    await page.click('text=Alertas');
    await expect(page).toHaveURL(/\/dashboard\/alerts/);

    // 2. Preenche o ticker para busca
    const tickerInput = page.locator('input[placeholder*="Pesquise por Ticker"]');
    await tickerInput.fill('PETR4');

    // 3. Seleciona o primeiro resultado do autocomplete
    await page.click('text=PETR4.SA');

    // 4. Preenche o preço alvo
    const priceInput = page.locator('input[placeholder="Ex: 35.50"]');
    await priceInput.fill('45.00');

    // 5. Clica para criar o alerta
    await page.click('button:has-text("Criar Alerta")');

    // 6. Confirma exibição do alerta na lista de alertas ativos
    await expect(page.locator('table >> text=PETR4').first()).toBeVisible();

    // 7. Alterna o status do alerta (Pausar/Ativar)
    const toggleButton = page.locator('button:has-text("Pausar"), button:has-text("Ativar")').first();
    await toggleButton.click();

    // 8. Exclui o alerta
    const deleteButton = page.locator('button:has-text("Excluir")').first();
    await deleteButton.click();

    // 9. Confirma remoção da tabela
    await expect(page.locator('text=Nenhum alerta cadastrado')).toBeVisible({ timeout: 5000 });
  });
});
