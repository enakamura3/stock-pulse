# 📦 Importação em Lote e Consistência (Bulk Import)

Muitas ferramentas de portfólio amadoras falham ao gerenciar a importação de transações via CSV: se uma das 500 linhas contiver um erro de digitação, a plataforma salva as primeiras 400 corretas, falha na 401ª e as 99 restantes são perdidas, deixando o banco de dados em um estado irrecuperável (corrompido).

O **stock-pulse** adota uma abordagem de **Atomicidade Estrita** (Transacional - Tudo ou Nada) com parse reverso para o processo de Importação Massiva (Bulk Import).

## Fluxograma da Máquina de Importação

```mermaid
flowchart TD
    Start[Upload do Arquivo CSV/TXT] --> Parser[Leitura e Limpeza (Parser Go)]
    
    Parser --> ValidateFormat{Formato Válido?<br>(Data, Preço, Tipo)}
    ValidateFormat -- Não --> ErrorValidation[Retorna Erro 400<br>Informando a Linha Específica]
    ValidateFormat -- Sim --> AssetMap[Monta Mapa Único de Tickers]
    
    AssetMap --> CheckDB[Busca Tickers no PostgreSQL]
    CheckDB --> Unregistered{Existem Tickers<br>Não Cadastrados?}
    
    Unregistered -- Sim --> CallYahoo[Dispara Requisições Concorrentes<br>ao Yahoo Finance API]
    CallYahoo --> WaitGroup[Sincroniza Goroutines (sync.WaitGroup)]
    WaitGroup --> CheckYahoo{Todos Foram<br>Encontrados?}
    CheckYahoo -- Não --> ErrorMarket[Retorna Erro 400<br>Ativo 'XYZ' Não Existe na Bolsa]
    CheckYahoo -- Sim --> CreateAssets[Cadastra Novos Ativos no DB] --> StartTransaction
    
    Unregistered -- Não --> StartTransaction[Inicia Transação SQL (`BEGIN`)]
    
    StartTransaction --> Loop[Insere Transações<br>Linha por Linha no DB]
    
    Loop --> InsertCheck{Erro Crítico<br>no Insert?}
    InsertCheck -- Sim --> Rollback[SQL `ROLLBACK`] --> ErrorValidation
    InsertCheck -- Não --> AllDone{Fim do Arquivo?}
    
    AllDone -- Não --> Loop
    AllDone -- Sim --> Commit[SQL `COMMIT`]
    
    Commit --> BackgroundJob[Dispara Job Assíncrono<br>de Auto-Cura (BackfillGap) para as Ações]
    BackgroundJob --> Success[Retorna 200 OK<br>Transações Efetuadas com Sucesso!]
```

## Tratamento Assíncrono Pós-Importação

Uma grande importação geraria dezenas de gatilhos tentando baixar dados históricos de múltiplos ativos e moedas simultaneamente, o que com certeza faria o Yahoo Finance acionar proteções contra DDoS (`HTTP 429 Too Many Requests`).

Para mitigar isso, o sistema bloqueia e realiza o download histórico apenas para taxas de câmbio ausentes de forma unificada e síncrona. Os recortes históricos profundos das ações inseridas caem em uma fila de BackfillGap assíncrona, sendo devorados por goroutines de baixa prioridade em _background_, permitindo que a interface web libere e renderize o Portfólio instantaneamente, enquanto os gráficos vão se consolidando nos segundos subsequentes.
