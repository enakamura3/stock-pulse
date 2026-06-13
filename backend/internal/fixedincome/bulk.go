package fixedincome

import (
	"context"
	"encoding/csv"
	"fmt"
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
			if a.Institution == assetName && a.Indexer == indexer && a.Rate == rate {
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
