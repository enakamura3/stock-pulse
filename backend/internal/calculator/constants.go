package calculator

const (
	// FinancialEpsilon é a margem de tolerância para comparações de ponto flutuante em valores financeiros.
	FinancialEpsilon = 1e-6

	// BusinessDaysPerYear é a quantidade padrão de dias úteis ao ano (base 252 dias úteis - padrão B3/ANBIMA).
	BusinessDaysPerYear = 252.0

	// FuzzyMatchGrossAmountThreshold é a tolerância máxima (R$ 0,05) para considerar proventos com pequenas divergências de centavos como idênticos.
	FuzzyMatchGrossAmountThreshold = 0.05

	// USWithholdingTaxRate é a alíquota padrão de imposto retido na fonte nos EUA (30%).
	USWithholdingTaxRate = 0.30

	// USWithholdingNetFactor é o fator multiplicador líquido para proventos dos EUA (1.0 - 0.30 = 0.70).
	USWithholdingNetFactor = 1.0 - USWithholdingTaxRate

	// B3WithholdingTaxRate é a alíquota de imposto retido na fonte para JCP e dividendos de ETFs na B3 (15%).
	B3WithholdingTaxRate = 0.15

	// B3WithholdingNetFactor é o fator multiplicador líquido para proventos com retenção na B3 (1.0 - 0.15 = 0.85).
	B3WithholdingNetFactor = 1.0 - B3WithholdingTaxRate
)
