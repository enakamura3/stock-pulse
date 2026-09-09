package fixedincome

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"math"
	"mime/multipart"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type BulkImportResult struct {
	Success int      `json:"success"`
	Errors  []string `json:"errors"`
}

func (s *service) BulkAddTransactions(ctx context.Context, portfolioID string, file multipart.File) (*BulkImportResult, error) {
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

	// Remove cabeçalho
	if strings.ToLower(strings.TrimSpace(records[0][0])) == "date" || strings.ToLower(strings.TrimSpace(records[0][0])) == "data" {
		records = records[1:]
	}

	result := &BulkImportResult{}

	// Carrega ativos existentes para tentar mapear
	existingAssets, err := s.repo.GetAssetsByPortfolio(ctx, portfolioID)
	if err != nil {
		return nil, fmt.Errorf("erro ao carregar ativos existentes: %w", err)
	}

	for i, row := range records {
		lineNum := i + 2
		if len(row) < 7 {
			result.Errors = append(result.Errors, fmt.Sprintf("Linha %d: formato inválido, esperado 7 colunas", lineNum))
			continue
		}

		dateStr := strings.TrimSpace(row[0])
		assetName := strings.TrimSpace(row[1])
		txType := strings.ToUpper(strings.TrimSpace(row[2]))
		amountStr := strings.TrimSpace(row[3])
		indexer := strings.ToUpper(strings.TrimSpace(row[4]))
		rateStr := strings.TrimSpace(row[5])
		maturityStr := strings.TrimSpace(row[6])

		if assetName == "" || txType == "" || dateStr == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("Linha %d: ativo, tipo e data são obrigatórios", lineNum))
			continue
		}

		// Valida valores numéricos
		amount, err := strconv.ParseFloat(amountStr, 64)
		if err != nil || amount <= 0 {
			result.Errors = append(result.Errors, fmt.Sprintf("Linha %d: valor '%s' inválido", lineNum, amountStr))
			continue
		}

		rate := 0.0
		if rateStr != "" {
			if r, err := strconv.ParseFloat(rateStr, 64); err == nil {
				rate = r
			}
		}

		// Valida datas
		execTime, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			execTime, err = time.Parse("02/01/2006", dateStr)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("Linha %d: data '%s' inválida", lineNum, dateStr))
				continue
			}
		}

		var maturityDate time.Time
		if maturityStr != "" && maturityStr != "--" {
			maturityDate, _ = time.Parse("2006-01-02", maturityStr)
		}

		// Busca ou cria o ativo baseado nos dados
		var targetAsset *Asset
		for i, a := range existingAssets {
			if a.Institution == assetName && a.Indexer == indexer && math.Abs(a.Rate-rate) < 1e-6 {
				targetAsset = &existingAssets[i]
				break
			}
		}

		if targetAsset == nil {
			// Inferir tipo de dívida (Simplificado)
			debtType := "POS"
			if indexer == "PRE" {
				debtType = "PRE"
			} else if indexer == "IPCA" || indexer == "IGPM" {
				debtType = "HIBRIDO"
			}

			newAsset := &Asset{
				ID:           uuid.New().String(),
				PortfolioID:  portfolioID,
				Institution:  assetName,
				Type:         "CDB", // Default assumido
				DebtType:     debtType,
				Indexer:      indexer,
				Rate:         rate,
				MaturityDate: maturityDate,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			}

			targetAsset, err = s.repo.CreateAsset(ctx, newAsset)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("Linha %d: erro ao criar ativo: %v", lineNum, err))
				continue
			}
			existingAssets = append(existingAssets, *targetAsset)
		}

		tx := &Transaction{
			ID:        uuid.New().String(),
			AssetID:   targetAsset.ID,
			Type:      txType,
			Amount:    amount,
			Date:      execTime.UTC(),
			CreatedAt: time.Now(),
		}

		_, err = s.repo.CreateTransaction(ctx, tx)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Linha %d: erro ao processar transação: %v", lineNum, err))
			continue
		}

		// Dispara o worker de backfill para a aplicação
		if tx.Type == "APLICACAO" || tx.Type == "SUBSCRIPTION" {
			if targetAsset.DebtType == "POS" || targetAsset.DebtType == "HIBRIDO" {
				go s.TriggerBackfill(context.Background(), targetAsset.Indexer, tx.Date)
			}
		}

		result.Success++
	}

	return result, nil
}

func extractYearFromTicker(ticker string) int {
	words := strings.Fields(ticker)
	for _, w := range words {
		if len(w) == 4 {
			if y, err := strconv.Atoi(w); err == nil && y >= 2000 && y <= 2100 {
				return y
			}
		}
	}
	return 0
}

func (s *service) BulkAddTreasuryTransactions(ctx context.Context, portfolioID string, file multipart.File) (*BulkImportResult, error) {
	reader := csv.NewReader(file)
	reader.Comma = ';'
	reader.FieldsPerRecord = -1
	records, err := reader.ReadAll()
	if err != nil {
		// Se falhar com delimitador ponto e vírgula, tenta ler com vírgula
		if _, seekErr := file.Seek(0, 0); seekErr == nil {
			reader = csv.NewReader(file)
			reader.Comma = ','
			reader.FieldsPerRecord = -1
			records, err = reader.ReadAll()
		}
		if err != nil {
			return nil, fmt.Errorf("erro ao ler arquivo CSV: %w", err)
		}
	} else if len(records) > 0 && len(records[0]) == 1 && strings.Contains(records[0][0], ",") {
		// Se leu com sucesso mas veio apenas 1 coluna contendo vírgula, tenta ler com vírgula
		if _, seekErr := file.Seek(0, 0); seekErr == nil {
			reader = csv.NewReader(file)
			reader.Comma = ','
			reader.FieldsPerRecord = -1
			if r2, err2 := reader.ReadAll(); err2 == nil {
				records = r2
			}
		}
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("arquivo CSV vazio")
	}

	// Remove cabeçalho se houver
	firstCol := strings.ToLower(strings.TrimSpace(records[0][0]))
	if firstCol == "date" || firstCol == "data" || firstCol == "ticker" {
		records = records[1:]
	}

	result := &BulkImportResult{}

	parseNum := func(s string) (float64, error) {
		clean := strings.TrimSpace(s)
		clean = strings.ReplaceAll(clean, "R$", "")
		clean = strings.ReplaceAll(clean, " ", "")
		clean = strings.ReplaceAll(clean, "%", "")
		clean = strings.ReplaceAll(clean, ",", ".")
		return strconv.ParseFloat(clean, 64)
	}

	for i, row := range records {
		lineNum := i + 2
		if len(row) < 5 {
			result.Errors = append(result.Errors, fmt.Sprintf("Linha %d: formato inválido, esperado pelo menos 5 colunas", lineNum))
			continue
		}

		dateStr := strings.TrimSpace(row[0])
		ticker := strings.TrimSpace(row[1])
		rawType := strings.ToUpper(strings.TrimSpace(row[2]))
		qtyStr := strings.TrimSpace(row[3])
		priceStr := strings.TrimSpace(row[4])

		if dateStr == "" || ticker == "" || rawType == "" || qtyStr == "" || priceStr == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("Linha %d: data, ticker, tipo, quantidade e preço unitário são obrigatórios", lineNum))
			continue
		}

		// Normaliza datas
		txTime, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			txTime, err = time.Parse("02/01/2006", dateStr)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("Linha %d: data '%s' inválida", lineNum, dateStr))
				continue
			}
		}
		formattedTxDate := txTime.Format("2006-01-02")

		// Normaliza tipo de transação
		var txType string
		switch rawType {
		case "SUBSCRIPTION", "APLICACAO", "APLICAÇÃO", "COMPRA", "BUY":
			txType = "SUBSCRIPTION"
		case "REDEMPTION", "RESGATE", "VENDA", "SELL":
			txType = "REDEMPTION"
		default:
			result.Errors = append(result.Errors, fmt.Sprintf("Linha %d: tipo de transação '%s' inválido", lineNum, rawType))
			continue
		}

		quantity, err := parseNum(qtyStr)
		if err != nil || quantity <= 1e-6 {
			result.Errors = append(result.Errors, fmt.Sprintf("Linha %d: quantidade '%s' inválida", lineNum, qtyStr))
			continue
		}

		unitPrice, err := parseNum(priceStr)
		if err != nil || unitPrice <= 1e-6 {
			result.Errors = append(result.Errors, fmt.Sprintf("Linha %d: preço unitário '%s' inválido", lineNum, priceStr))
			continue
		}

		contractedRate := 0.0
		if len(row) > 5 && row[5] != "" {
			if r, err := parseNum(row[5]); err == nil {
				contractedRate = r
			}
		}

		treasuryType := ""
		if len(row) > 6 && row[6] != "" {
			treasuryType = strings.ToUpper(strings.TrimSpace(row[6]))
		}
		if treasuryType == "" {
			upperTicker := strings.ToUpper(ticker)
			if strings.Contains(upperTicker, "SELIC") || strings.Contains(upperTicker, "LFT") {
				treasuryType = "SELIC"
			} else if strings.Contains(upperTicker, "PREFIXADO") || strings.Contains(upperTicker, "LTN") || strings.Contains(upperTicker, "NTN-F") {
				treasuryType = "PREFIXADO"
			} else if strings.Contains(upperTicker, "IPCA") || strings.Contains(upperTicker, "NTN-B") {
				treasuryType = "IPCA+"
			} else {
				treasuryType = "SELIC"
			}
		}

		maturityDateStr := ""
		if len(row) > 7 && row[7] != "" {
			matTime, err := time.Parse("2006-01-02", strings.TrimSpace(row[7]))
			if err != nil {
				matTime, err = time.Parse("02/01/2006", strings.TrimSpace(row[7]))
			}
			if err == nil {
				maturityDateStr = matTime.Format("2006-01-02")
			}
		}
		if maturityDateStr == "" {
			reYear := extractYearFromTicker(ticker)
			if reYear > 0 {
				maturityDateStr = fmt.Sprintf("%04d-01-01", reYear)
			} else {
				maturityDateStr = txTime.AddDate(2, 0, 0).Format("2006-01-02")
			}
		}

		hasCoupons := false
		if len(row) > 8 && row[8] != "" {
			v := strings.ToLower(strings.TrimSpace(row[8]))
			hasCoupons = v == "true" || v == "sim" || v == "s" || v == "1"
		} else {
			upperTicker := strings.ToUpper(ticker)
			hasCoupons = strings.Contains(upperTicker, "COM JUROS") || strings.Contains(upperTicker, "JUROS SEMESTRAIS") || strings.Contains(upperTicker, "NTN-F")
		}

		txReq := &TreasuryTxRequest{
			Ticker:          ticker,
			TreasuryType:    treasuryType,
			MaturityDate:    maturityDateStr,
			HasCoupons:      hasCoupons,
			Type:            txType,
			Quantity:        quantity,
			UnitPrice:       unitPrice,
			ContractedRate:  contractedRate,
			TransactionDate: formattedTxDate,
		}

		_, err = s.CreateTreasuryTransaction(ctx, portfolioID, txReq)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Linha %d: erro ao processar operação: %v", lineNum, err))
			continue
		}

		result.Success++
	}

	return result, nil
}

func (s *service) ExportTreasuryTransactions(ctx context.Context, portfolioID string) ([]byte, error) {
	txs, err := s.GetTreasuryTransactions(ctx, portfolioID)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)
	writer.Comma = ';'

	err = writer.Write([]string{"Date", "Ticker", "Type", "Quantity", "UnitPrice", "ContractedRate", "TreasuryType", "MaturityDate", "HasCoupons"})
	if err != nil {
		return nil, fmt.Errorf("erro ao escrever cabeçalho CSV: %w", err)
	}

	for _, tx := range txs {
		record := []string{
			tx.TransactionDate,
			tx.Ticker,
			tx.Type,
			fmt.Sprintf("%.6f", tx.Quantity),
			fmt.Sprintf("%.4f", tx.UnitPrice),
			fmt.Sprintf("%.4f", tx.ContractedRate),
			tx.TreasuryType,
			tx.MaturityDate,
			strconv.FormatBool(tx.HasCoupons),
		}
		if err := writer.Write(record); err != nil {
			return nil, fmt.Errorf("erro ao escrever registro CSV: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("erro ao finalizar CSV: %w", err)
	}

	return buf.Bytes(), nil
}

