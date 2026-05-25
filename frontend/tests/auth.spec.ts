import { test, expect } from '@playwright/test';

test.describe('Fluxo de Autenticação e Route Guard', () => {
  const timestamp = Date.now();
  const testName = `Teste Playwright ${timestamp}`;
  const testEmail = `playwright_${timestamp}@stock-pulse.com`;
  const testPassword = 'securepassword123';

  test.beforeEach(async ({ page }) => {
    // Redireciona chamadas da API do frontend (localhost:8080) para o container backend (backend:8080)
    await page.route('http://localhost:8080/**/*', async (route) => {
      const url = route.request().url().replace('localhost:8080', 'backend:8080');
      const response = await route.fetch({ url });
      await route.fulfill({ response });
    });
  });

  test('deve registrar um novo usuário, acessar o dashboard, deslogar e bloquear acesso não autenticado', async ({ page }) => {
    // 1. Acessa a página de Login
    await page.goto('/login');
    await expect(page).toHaveTitle(/stock-pulse/);

    // 2. Navega para a página de Registro
    await page.click('text=Cadastre-se grátis');
    await expect(page).toHaveURL(/\/register/);

    // 3. Preenche o formulário de Registro
    await page.fill('#name', testName);
    await page.fill('#email', testEmail);
    await page.fill('#password', testPassword);

    // 4. Submete o registro
    await page.click('button[type="submit"]');

    // 5. Verifica redirecionamento para o Dashboard
    await expect(page).toHaveURL(/\/dashboard/, { timeout: 10000 });
    
    // 6. Confirma que o nome do usuário é exibido
    await expect(page.locator(`text=${testName}`)).toBeVisible();
    await expect(page.locator('text=Sessão Segura')).toBeVisible();

    // 7. Efetua o Logout
    await page.click('button:has-text("Sair")');

    // 8. Verifica redirecionamento de volta para o Login
    await expect(page).toHaveURL(/\/login/, { timeout: 10000 });

    // 9. Teste do Route Guard: Tenta acessar o dashboard diretamente após deslogar
    await page.goto('/dashboard');

    // 10. Verifica se foi redirecionado para o Login
    await expect(page).toHaveURL(/\/login/);
  });
});
