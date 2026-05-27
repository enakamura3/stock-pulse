package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTPRequestsTotal conta a quantidade total de requisições HTTP processadas.
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "stock-pulse_http_requests_total",
			Help: "Quantidade total de requisições HTTP recebidas pela API",
		},
		[]string{"method", "path", "status"},
	)

	// HTTPRequestDuration calcula o tempo de resposta das requisições HTTP da API.
	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "stock-pulse_http_request_duration_seconds",
			Help:    "Latência de processamento das requisições HTTP da API em segundos",
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0},
		},
		[]string{"method", "path", "status"},
	)
)

// responseWriterDelegator envolve http.ResponseWriter para interceptar o status code HTTP.
type responseWriterDelegator struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriterDelegator) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *responseWriterDelegator) Write(b []byte) (int, error) {
	// Se WriteHeader não tiver sido chamado, por padrão o Go assume 200 OK
	if w.statusCode == 0 {
		w.statusCode = http.StatusOK
	}
	return w.ResponseWriter.Write(b)
}

// Metrics retorna o middleware de coleta de dados do Prometheus.
func Metrics() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startTime := time.Now()

			// Envolve a resposta para capturar o código HTTP final
			delegator := &responseWriterDelegator{ResponseWriter: w}

			next.ServeHTTP(delegator, r)

			// Identifica a rota lógica no roteador Chi para evitar poluição de UUIDs no Prometheus
			routePattern := ""
			if routeContext := chi.RouteContext(r.Context()); routeContext != nil {
				routePattern = routeContext.RoutePattern()
			}
			if routePattern == "" {
				routePattern = r.URL.Path // Fallback se não bater com o roteador
			}

			// Previne rotas de métricas ou saúde de sujarem os dados
			if routePattern == "/metrics" || routePattern == "/healthz" {
				return
			}

			duration := time.Since(startTime).Seconds()
			statusStr := strconv.Itoa(delegator.statusCode)
			if delegator.statusCode == 0 {
				statusStr = "200" // Fallback seguro
			}

			// Registra as métricas no Prometheus
			HTTPRequestsTotal.WithLabelValues(r.Method, routePattern, statusStr).Inc()
			HTTPRequestDuration.WithLabelValues(r.Method, routePattern, statusStr).Observe(duration)
		})
	}
}
