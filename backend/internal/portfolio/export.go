package portfolio

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/onigiri/stock-pulse/backend/internal/auth"
)

// ExportPortfolio gera e baixa um ZIP com o backup completo em CSV (RV e RF).
func (h *Handler) ExportPortfolio(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(auth.UserIDKey).(string)
	if !ok || userID == "" {
		h.respondWithError(w, http.StatusUnauthorized, "Não autorizado")
		return
	}

	portfolioID := chi.URLParam(r, "id")
	if portfolioID == "" {
		h.respondWithError(w, http.StatusBadRequest, "ID da carteira é obrigatório")
		return
	}

	ctx := r.Context()

	// Anti-IDOR: Garantir que a carteira pertence ao usuário
	p, _, err := h.service.GetPortfolioDetails(ctx, portfolioID, userID)
	if err != nil {
		h.respondWithError(w, http.StatusForbidden, "Carteira não encontrada ou sem acesso")
		return
	}

	// 1. Buscar transações de Renda Variável
	rvTxs, err := h.service.GetPortfolioTransactions(ctx, portfolioID, userID)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "Erro ao buscar transações de renda variável")
		return
	}

	// 2. Buscar transações e ativos de Renda Fixa
	fiSvc := h.service.GetFixedIncomeService()
	fiTxs, err := fiSvc.GetRawTransactions(ctx, portfolioID)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "Erro ao buscar transações de renda fixa")
		return
	}
	
	fiAssets, err := fiSvc.GetAssetsByPortfolio(ctx, portfolioID)
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "Erro ao buscar ativos de renda fixa")
		return
	}

	// Mapear AssetID -> Asset (Renda Fixa)
	type fiAssetInfo struct {
		Name     string
		Indexer  string
		Rate     string
		Maturity string
	}
	fiAssetMap := make(map[string]fiAssetInfo)
	for _, a := range fiAssets {
		maturity := ""
		if !a.MaturityDate.IsZero() {
			maturity = a.MaturityDate.Format("2006-01-02")
		}
		
		name := a.Institution + " " + a.Type
		if a.DebtType == "POS" || a.DebtType == "HIBRIDO" {
			name += fmt.Sprintf(" %.2f%% %s", a.Rate, a.Indexer)
		} else {
			name += fmt.Sprintf(" %.2f%% a.a.", a.Rate)
		}

		fiAssetMap[a.ID] = fiAssetInfo{
			Name:     name,
			Indexer:  a.Indexer,
			Rate:     fmt.Sprintf("%.2f", a.Rate),
			Maturity: maturity,
		}
	}

	// Criar buffer do ZIP
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	// --- CSV Renda Variável ---
	rvFile, err := zipWriter.Create("renda_variavel.csv")
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "Erro ao criar arquivo ZIP")
		return
	}
	rvWriter := csv.NewWriter(rvFile)
	rvWriter.Comma = ';' // Delimitador Ponto e Vírgula
	
	// Headers RV
	rvWriter.Write([]string{"Date", "Ticker", "Type", "Quantity", "UnitPrice", "ExchangeRate"})
	for _, tx := range rvTxs {
		rvWriter.Write([]string{
			tx.ExecutedAt.Format("2006-01-02"),
			tx.Ticker,
			tx.Type,
			fmt.Sprintf("%.6f", tx.Quantity),
			fmt.Sprintf("%.6f", tx.UnitPrice),
			fmt.Sprintf("%.4f", tx.ExchangeRate),
		})
	}
	rvWriter.Flush()

	// --- CSV Renda Fixa ---
	fiFile, err := zipWriter.Create("renda_fixa.csv")
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "Erro ao criar arquivo ZIP")
		return
	}
	fiWriter := csv.NewWriter(fiFile)
	fiWriter.Comma = ';' // Delimitador Ponto e Vírgula
	
	// Headers RF
	fiWriter.Write([]string{"Date", "AssetName", "Type", "Value", "Indexer", "Rate", "MaturityDate"})
	for _, tx := range fiTxs {
		info := fiAssetMap[tx.AssetID]
		fiWriter.Write([]string{
			tx.Date.Format("2006-01-02"),
			info.Name,
			tx.Type,
			fmt.Sprintf("%.2f", tx.Amount),
			info.Indexer,
			info.Rate,
			info.Maturity,
		})
	}
	fiWriter.Flush()

	// Fechar ZIP
	err = zipWriter.Close()
	if err != nil {
		h.respondWithError(w, http.StatusInternalServerError, "Erro ao finalizar ZIP")
		return
	}

	// Retornar o ZIP
	filename := fmt.Sprintf("stock-pulse-backup-%s.zip", p.Name)
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Write(buf.Bytes())
}
