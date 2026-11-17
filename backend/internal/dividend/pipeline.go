package dividend

import (
	"context"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/onigiri/stock-pulse/backend/internal/calculator"
)

// Event representa um provento coletado de uma fonte externa.
type Event struct {
	Date        time.Time `json:"date"`         // Data Com (cum_date)
	PaymentDate time.Time `json:"payment_date"` // Data de Pagamento
	Amount      float64   `json:"amount"`       // Valor bruto por cota/ação
	Type        string    `json:"type"`         // Dividendo | JCP | Rendimento | Amortização
}

// AssetEvent representa um registro de provento persistido na tabela asset_event.
type AssetEvent struct {
	ID          string    `json:"id"`
	AssetID     string    `json:"asset_id"`
	Type        string    `json:"type"`
	GrossAmount float64   `json:"gross_amount"`
	NetAmount   float64   `json:"net_amount"`
	CumDate     time.Time `json:"cum_date"`
	PaymentDate time.Time `json:"payment_date"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Source define a interface que toda fonte externa de proventos deve implementar.
type Source interface {
	GetDividends(ctx context.Context, ticker string, assetType string) ([]Event, error)
	Name() string
	SupportedAssetTypes() []string
}

// Plan descreve as operações de banco resultantes da etapa Resolve.
type Plan struct {
	Inserts []AssetEvent
	Updates []AssetEvent
	Skips   int
}

func isFiiType(assetType string) bool {
	upper := strings.ToUpper(strings.TrimSpace(assetType))
	return upper == "FII" || upper == "FIAGRO"
}

// NormalizeType converte a descrição crua da fonte externa em um dos 4 tipos canônicos:
// "JCP", "Rendimento", "Amortização" ou "Dividendo".
func NormalizeType(rawType, assetType string) string {
	tipoUpper := strings.ToUpper(strings.TrimSpace(rawType))
	if strings.Contains(tipoUpper, "JRS") || strings.Contains(tipoUpper, "JUROS") || strings.Contains(tipoUpper, "JCP") {
		return "JCP"
	}
	if strings.Contains(tipoUpper, "RENDIMENTO") {
		return "Rendimento"
	}
	if strings.Contains(tipoUpper, "AMORTIZA") {
		return "Amortização"
	}
	if isFiiType(assetType) {
		return "Rendimento"
	}
	return "Dividendo"
}

// NormalizeDate trunca uma data para meia-noite UTC (00:00:00 UTC).
func NormalizeDate(t time.Time) time.Time {
	if t.IsZero() {
		return time.Time{}
	}
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// PreviousBusinessDay retorna o dia útil imediatamente anterior à data fornecida
// (recuando sábado e domingo para sexta-feira). Usado para converter Ex-Dividend Date em Cum-Date.
func PreviousBusinessDay(exDate time.Time) time.Time {
	if exDate.IsZero() {
		return time.Time{}
	}
	d := NormalizeDate(exDate).AddDate(0, 0, -1)
	for d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
		d = d.AddDate(0, 0, -1)
	}
	return d
}

// SameCalendarDate compara se duas datas pertencem ao mesmo dia de calendário (YYYY-MM-DD).
func SameCalendarDate(a, b time.Time) bool {
	if a.IsZero() || b.IsZero() {
		return a.IsZero() && b.IsZero()
	}
	return a.Format("2006-01-02") == b.Format("2006-01-02")
}

// ResolveEventType preserva o tipo mais específico quando uma fonte informa "Dividendo" genérico
// e outra fonte (ou o banco) já possui "JCP", "Rendimento" ou "Amortização".
func ResolveEventType(existingType, incomingType string) string {
	if existingType != "Dividendo" && incomingType == "Dividendo" {
		return existingType
	}
	return incomingType
}

// Normalize padroniza os tipos e trunca as datas para UTC em todos os eventos.
func Normalize(events []Event, assetType string) []Event {
	out := make([]Event, 0, len(events))
	for _, ev := range events {
		out = append(out, Event{
			Date:        NormalizeDate(ev.Date),
			PaymentDate: NormalizeDate(ev.PaymentDate),
			Amount:      ev.Amount,
			Type:        NormalizeType(ev.Type, assetType),
		})
	}
	return out
}

// Deduplicate remove eventos duplicados dentro da própria lista entregue pelo provedor
// (mesma CumDate, mesma PaymentDate e mesmo Amount dentro de FinancialEpsilon),
// preservando o tipo mais específico caso um seja "Dividendo" e outro seja "JCP".
func Deduplicate(events []Event) []Event {
	deduped := make([]Event, 0, len(events))
	for _, ev := range events {
		dupIdx := -1
		for idx, existing := range deduped {
			if SameCalendarDate(ev.Date, existing.Date) &&
				SameCalendarDate(ev.PaymentDate, existing.PaymentDate) &&
				math.Abs(ev.Amount-existing.Amount) < calculator.FinancialEpsilon {
				dupIdx = idx
				break
			}
		}
		if dupIdx >= 0 {
			deduped[dupIdx].Type = ResolveEventType(deduped[dupIdx].Type, ev.Type)
			continue
		}
		deduped = append(deduped, ev)
	}
	return deduped
}

// Validate filtra apenas eventos válidos: exige CumDate preenchida, PaymentDate preenchida
// e Amount estritamente positivo (> FinancialEpsilon).
func Validate(events []Event) []Event {
	valid := make([]Event, 0, len(events))
	for _, ev := range events {
		if ev.Date.IsZero() || ev.PaymentDate.IsZero() {
			continue
		}
		if ev.Amount <= calculator.FinancialEpsilon {
			continue
		}
		valid = append(valid, ev)
	}

	sort.SliceStable(valid, func(i, j int) bool {
		return valid[i].Date.After(valid[j].Date)
	})

	return valid
}

// ProcessPipeline executa as 3 primeiras etapas do pipeline sobre os dados entregues por uma fonte:
// 1. Normalize -> 2. Deduplicate -> 3. Validate.
func ProcessPipeline(raw []Event, assetType string) []Event {
	normalized := Normalize(raw, assetType)
	deduped := Deduplicate(normalized)
	return Validate(deduped)
}

// Resolve executa a 4ª etapa do pipeline (Fuzzy Match contra os registros existentes no banco),
// retornando o plano de Inserts, Updates e Skips.
func Resolve(assetID string, incoming []Event, existing []AssetEvent) Plan {
	var plan Plan
	matchedIDs := make(map[string]bool)

	for _, ev := range incoming {
		bestMatch := findBestMatch(ev, existing, matchedIDs)
		if bestMatch == nil {
			plan.Inserts = append(plan.Inserts, AssetEvent{
				AssetID:     assetID,
				Type:        ev.Type,
				GrossAmount: ev.Amount,
				NetAmount:   ev.Amount,
				CumDate:     ev.Date,
				PaymentDate: ev.PaymentDate,
			})
			continue
		}

		matchedIDs[bestMatch.ID] = true
		resolvedType := ResolveEventType(bestMatch.Type, ev.Type)
		diff := math.Abs(bestMatch.GrossAmount - ev.Amount)

		if diff < calculator.FinancialEpsilon &&
			SameCalendarDate(bestMatch.PaymentDate, ev.PaymentDate) &&
			bestMatch.Type == resolvedType {
			plan.Skips++
			continue
		}

		plan.Updates = append(plan.Updates, AssetEvent{
			ID:          bestMatch.ID,
			AssetID:     assetID,
			Type:        resolvedType,
			GrossAmount: ev.Amount,
			NetAmount:   ev.Amount,
			CumDate:     bestMatch.CumDate,
			PaymentDate: ev.PaymentDate,
		})
	}

	return plan
}

func findBestMatch(ev Event, existingEvents []AssetEvent, matchedIDs map[string]bool) *AssetEvent {
	var bestMatch *AssetEvent
	var minDiff float64 = -1

	// 1. Mesmo CumDate e mesmo Type com diferença <= R$ 0,05
	for i := range existingEvents {
		existing := &existingEvents[i]
		if matchedIDs[existing.ID] || !SameCalendarDate(existing.CumDate, ev.Date) {
			continue
		}
		if existing.Type != ev.Type {
			continue
		}
		diff := math.Abs(existing.GrossAmount - ev.Amount)
		if diff <= calculator.FuzzyMatchGrossAmountThreshold {
			if minDiff == -1 || diff < minDiff {
				minDiff = diff
				bestMatch = existing
			}
		}
	}

	if bestMatch != nil {
		return bestMatch
	}

	// 2. Conciliação entre tipo genérico ("Dividendo") e específico ("JCP", etc.)
	// quando possuem mesma CumDate, mesma PaymentDate e diferença <= R$ 0,05
	for i := range existingEvents {
		existing := &existingEvents[i]
		if matchedIDs[existing.ID] || !SameCalendarDate(existing.CumDate, ev.Date) {
			continue
		}
		if (existing.Type == "Dividendo" || ev.Type == "Dividendo") &&
			SameCalendarDate(existing.PaymentDate, ev.PaymentDate) {
			diff := math.Abs(existing.GrossAmount - ev.Amount)
			if diff <= calculator.FuzzyMatchGrossAmountThreshold {
				if minDiff == -1 || diff < minDiff {
					minDiff = diff
					bestMatch = existing
				}
			}
		}
	}

	return bestMatch
}
