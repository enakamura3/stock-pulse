package portfolio

import (
	"context"
	"encoding/csv"
	"fmt"
	"mime/multipart"
	"strconv"
	"strings"
	"time"
)

type BulkImportResult struct {
	Success int      `json:"success"`
	Errors  []string `json:"errors"`
}

func (s *Service) BulkAddTransactions(ctx context.Context, userID, portfolioID string, file multipart.File) (*BulkImportResult, error) {
	// Anti-IDOR: Valida se a carteira pertence ao usuário logado
	_, err := s.repo.GetPortfolioByID(ctx, portfolioID, userID)
	if err != nil {
		return nil, fmt.Errorf("carteira não encontrada ou acesso não autorizado")
	}

	reader := csv.NewReader(file)
	reader.Comma = ';'
	reader.FieldsPerRecord = -1
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("erro ao ler arquivo CSV: %w", err)
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("arquivo CSV vazio")
	}

	// Remove cabeçalho se existir
	if strings.ToLower(strings.TrimSpace(records[0][0])) == "date" || strings.ToLower(strings.TrimSpace(records[0][0])) == "data" {
		records = records[1:]
	}

	result := &BulkImportResult{}

	for i, row := range records {
		lineNum := i + 2 // +1 for 0-index, +1 because we skipped header (usually)
		if len(row) < 5 {
			result.Errors = append(result.Errors, fmt.Sprintf("Linha %d: formato inválido, esperado ao menos 5 colunas", lineNum))
			continue
		}

		dateStr := strings.TrimSpace(row[0])
		ticker := strings.ToUpper(strings.TrimSpace(row[1]))
		txType := strings.ToUpper(strings.TrimSpace(row[2]))
		qtyStr := strings.TrimSpace(row[3])
		priceStr := strings.TrimSpace(row[4])

		if ticker == "" || txType == "" || dateStr == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("Linha %d: ticker, tipo e data são obrigatórios", lineNum))
			continue
		}

		if txType != "BUY" && txType != "SELL" && txType != "SPLIT" && txType != "REVERSE_SPLIT" && txType != "BONUS" {
			result.Errors = append(result.Errors, fmt.Sprintf("Linha %d: tipo '%s' inválido", lineNum, txType))
			continue
		}

		qty, err := strconv.ParseFloat(qtyStr, 64)
		if err != nil || qty <= 0 {
			result.Errors = append(result.Errors, fmt.Sprintf("Linha %d: quantidade '%s' inválida", lineNum, qtyStr))
			continue
		}

		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil || (price <= 0 && txType != "SPLIT" && txType != "REVERSE_SPLIT" && txType != "BONUS") {
			result.Errors = append(result.Errors, fmt.Sprintf("Linha %d: preço '%s' inválido", lineNum, priceStr))
			continue
		}

		execTime, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			execTime, err = time.Parse("02/01/2006", dateStr)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("Linha %d: data '%s' inválida (use AAAA-MM-DD)", lineNum, dateStr))
				continue
			}
		}

		exchangeRate := 0.0
		if len(row) >= 6 {
			erStr := strings.TrimSpace(row[5])
			if erStr != "" {
				if parsedER, err := strconv.ParseFloat(erStr, 64); err == nil {
					exchangeRate = parsedER
				}
			}
		}

		tx := &Transaction{
			PortfolioID:  portfolioID,
			Ticker:       ticker,
			Type:         txType,
			Quantity:     qty,
			UnitPrice:    price,
			ExchangeRate: exchangeRate,
			ExecutedAt:   execTime.UTC(),
		}

		_, err = s.AddTransaction(ctx, userID, tx)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Linha %d: erro ao processar (%s)", lineNum, err.Error()))
			continue
		}

		result.Success++
	}

	return result, nil
}
