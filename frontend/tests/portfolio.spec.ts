import { test, expect } from '@playwright/test';

test.describe('Fluxo E2E de Portfólio', () => {
  // Configuração inicial comum: criar um usuário e fazer login
  const testEmail = `e2e_portfolio_${Date.now()}@test.com`;
  const testPassword = 'Password123!';
  const testName = 'E2E Portfolio User';

  test.beforeEach(async ({ page }) => {
    // Redireciona chamadas da API do frontend (localhost:8080) para o container backend (backend:8080)
    await page.route('http://localhost:8080/**/*', async (route) => {
      const url = route.request().url().replace('localhost:8080', 'backend:8080');
      const response = await route.fetch({ url });
      await route.fulfill({ response });
    });

    // Registra e logo cai no dashboard
    await page.goto('/register');
    await page.fill('input[type="text"]', testName);
    await page.fill('input[type="email"]', testEmail);
    await page.fill('input[type="password"]', testPassword);
    await page.click('button[type="submit"]');

    // Aguarda o redirecionamento
    await expect(page).toHaveURL(/\/dashboard/, { timeout: 10000 });
  });

  test('deve criar uma nova carteira, adicionar uma transação e verificar o saldo', async ({ page }) => {
    // 1. Acessa a página de Portfólios
    await page.click('text=Carteiras');
    await expect(page).toHaveURL(/\/dashboard\/portfolio/);

    // 2. Cria uma nova carteira
    await page.click('button:has-text("+ Criar Carteira")');
    // Preenche o nome buscando pelo placeholder
    await page.fill('input[placeholder="Ex: Minha Aposentadoria, Ações B3..."]', 'Carteira de Aposentadoria');
    // Obs: A aplicação não possui campo de descrição para a carteira.
    await page.click('button:has-text("Salvar")');

    // Verifica se a carteira aparece nas abas (PortfolioTabs)
    await expect(page.locator('button.tab-button:has-text("Carteira de Aposentadoria")')).toBeVisible();
    await page.click('button.tab-button:has-text("Carteira de Aposentadoria")');

    // 3. Adiciona uma transação de Compra (PETR4)
    await page.click('button:has-text("+ Lançar Operação")');
    
    // O mock provider retorna preço de R$ 50,00.
    // Ticker (Autocomplete)
    await page.fill('input[placeholder="Pesquise o ticker (Ex: PETR4, AAPL, IVV)..."]', 'PETR4');
    // Seleciona a primeira opção do dropdown de autocomplete
    await page.click('text=PETR4.SA'); // O MockProvider retorna "PETR4.SA"

    // Tipo de transação já deve ser COMPRA por padrão, mas podemos clicar pra garantir
    await page.click('button:has-text("COMPRA")');
    
    // Quantidade
    await page.fill('label:has-text("Quantidade") + input', '100');
    
    // Preço
    await page.fill('input[placeholder="0.00"]', '50');
    
    // Data de execução
    await page.fill('input[type="date"]', '2023-01-01');
    await page.click('button[type="submit"]:has-text("Lançar")');

    // 4. Verifica na Tabela de Posições Ativas
    await expect(page.locator('table >> text=PETR4').first()).toBeVisible();

    // 5. Verifica os Cards de Saldo
    // Mock Price é R$ 50. Quantidade é 100. Saldo total esperado: R$ 5.000,00
    // O backend ou frontend pode formatar como "R$ 5.000,00" ou "5,000.00"
    await expect(page.locator('text=5.000,00').first()).toBeVisible();
  });
});
