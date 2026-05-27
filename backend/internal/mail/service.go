package mail

import (
	"fmt"
	"log"
	"net/smtp"
	"os"
)

// SendMailFunc define a assinatura para enviar emails, permitindo mock nos testes.
var SendMailFunc = smtp.SendMail

// Service gerencia o disparo de e-mails do sistema via protocolo SMTP.
type Service struct {
	host string
	port string
	from string
}

// NewService inicializa o serviço de e-mail buscando as configurações SMTP do ambiente.
func NewService() *Service {
	host := os.Getenv("SMTP_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("SMTP_PORT")
	if port == "" {
		port = "1025" // Porta SMTP padrão do Mailpit/Mailhog
	}
	from := os.Getenv("SMTP_FROM")
	if from == "" {
		from = "no-reply@stock-pulse.com"
	}

	return &Service{
		host: host,
		port: port,
		from: from,
	}
}

// SendAlertEmail envia um e-mail HTML altamente estilizado notificando que um alerta de preço foi atingido.
func (s *Service) SendAlertEmail(to string, userName string, ticker string, assetName string, currentPrice float64, targetPrice float64, condition string, currency string) error {
	subject := fmt.Sprintf("🚨 ALERTA: %s atingiu o preço alvo de %s %s!", ticker, currency, fmt.Sprintf("%.2f", targetPrice))

	// Escolhe a cor temática neon com base no sentido do alerta
	accentColor := "#00e676" // Neon Green para ABOVE
	conditionText := "subiu para"
	if condition == "BELOW" {
		accentColor = "#ff3d00" // Neon Red para BELOW
		conditionText = "caiu para"
	}

	body := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Alerta stock-pulse</title>
    <style>
        body {
            margin: 0;
            padding: 0;
            background-color: #0b0f19;
            color: #e2e8f0;
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif;
        }
        .container {
            max-width: 600px;
            margin: 0 auto;
            padding: 20px;
        }
        .header {
            text-align: center;
            padding: 20px 0;
        }
        .logo {
            font-size: 24px;
            font-weight: 800;
            letter-spacing: -0.03em;
            background: linear-gradient(135deg, #00f2fe 0%%, #4facfe 100%%);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            display: inline-block;
        }
        .card {
            background-color: #111827;
            border: 1px solid rgba(255, 255, 255, 0.08);
            border-radius: 16px;
            padding: 32px;
            box-shadow: 0 10px 25px -5px rgba(0, 0, 0, 0.3);
        }
        .status-badge {
            display: inline-block;
            padding: 6px 12px;
            border-radius: 20px;
            font-size: 12px;
            font-weight: 700;
            text-transform: uppercase;
            letter-spacing: 0.05em;
            background-color: rgba(255, 255, 255, 0.05);
            margin-bottom: 20px;
        }
        h1 {
            font-size: 22px;
            font-weight: 700;
            margin: 0 0 16px 0;
            color: #ffffff;
        }
        p {
            font-size: 15px;
            line-height: 1.6;
            margin: 0 0 24px 0;
            color: #9ca3af;
        }
        .highlight-box {
            background: rgba(255, 255, 255, 0.02);
            border: 1px solid rgba(255, 255, 255, 0.05);
            border-radius: 12px;
            padding: 20px;
            margin-bottom: 28px;
            text-align: center;
        }
        .asset-ticker {
            font-size: 32px;
            font-weight: 800;
            color: #ffffff;
            margin: 0 0 4px 0;
            letter-spacing: -0.02em;
        }
        .asset-name {
            font-size: 14px;
            color: #6b7280;
            margin: 0 0 16px 0;
        }
        .price-compare {
            font-size: 18px;
            font-weight: 600;
            color: #e2e8f0;
        }
        .current-price {
            color: %s;
            font-weight: 800;
            font-size: 24px;
        }
        .btn {
            display: block;
            text-align: center;
            background: linear-gradient(135deg, #00f2fe 0%%, #4facfe 100%%);
            color: #0b0f19 !important;
            text-decoration: none;
            padding: 14px 24px;
            border-radius: 10px;
            font-weight: 700;
            font-size: 15px;
            box-shadow: 0 4px 15px rgba(0, 242, 254, 0.2);
            transition: all 0.2s ease;
        }
        .footer {
            text-align: center;
            padding: 24px 0;
            font-size: 12px;
            color: #4b5563;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <div class="logo">stock-pulse</div>
        </div>
        <div class="card">
            <div class="status-badge" style="color: %s; border: 1px solid %s;">
                Alerta Disparado
            </div>
            <h1>Olá, %s!</h1>
            <p>Seu alerta de preço personalizado foi acionado porque o ativo <strong>%s</strong> atingiu a condição configurada.</p>
            
            <div class="highlight-box">
                <div class="asset-ticker">%s</div>
                <div class="asset-name">%s</div>
                <div class="price-compare">
                    Meta: %s %s | O preço %s <br>
                    <span class="current-price">%s %s</span>
                </div>
            </div>
            
            <a href="http://localhost:3000/dashboard" class="btn">Visualizar no Painel</a>
        </div>
        <div class="footer">
            &copy; 2026 stock-pulse. Todos os direitos reservados.<br>
            Este é um e-mail automático do seu monitor de investimentos.
        </div>
    </div>
</body>
</html>`, accentColor, accentColor, accentColor, userName, ticker, ticker, assetName, currency, fmt.Sprintf("%.2f", targetPrice), conditionText, currency, fmt.Sprintf("%.2f", currentPrice))

	// Montagem do cabeçalho SMTP (RFC 822)
	message := fmt.Sprintf("From: %s\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"MIME-Version: 1.0\r\n"+
		"Content-Type: text/html; charset=UTF-8\r\n\r\n"+
		"%s", s.from, to, subject, body)

	addr := fmt.Sprintf("%s:%s", s.host, s.port)

	// Envia o e-mail (sem autenticação local de desenvolvimento, suportando SMTP anônimo do Mailpit)
	err := SendMailFunc(addr, nil, s.from, []string{to}, []byte(message))
	if err != nil {
		log.Printf("[SMTP] Falha ao enviar alerta por e-mail para %s: %v", to, err)
		return err
	}

	log.Printf("[SMTP] Alerta de preço para %s enviado com sucesso para %s", ticker, to)
	return nil
}
