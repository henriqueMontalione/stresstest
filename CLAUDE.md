# CLAUDE.md

## Papel e contexto

Você está implementando uma CLI de load testing HTTP em Go. Siga estas instruções à risca em toda interação com este projeto.

---

## Regras de código — NUNCA faça

- Nunca use `panic` em código de aplicação
- Nunca ignore erros — trate todos explicitamente
- Nunca hardcode valores de configuração
- Nunca adicione comentários que apenas repetem o que o código já diz
- Nunca adicione dependências externas sem necessidade clara — este projeto usa apenas stdlib
- Nunca escreva logs desnecessários — o output é o relatório final, não logs de debug
- Nunca implemente lógica de concorrência dentro de `internal/report`
- Nunca implemente formatação de output dentro de `internal/runner`
- Nunca use `context.Background()` diretamente nos workers — propague o context do runner

---

## Convenções obrigatórias

### Nomenclatura
- Interfaces: nome descritivo simples — `Runner`, não `IRunner`
- Construtores: sempre `New{Type}` — `NewRunner`, `NewReport`
- Métodos: verbos claros — `Run`, `Print`, `Collect`

### Go idiomático
- Use `context.Context` em todas as operações de I/O (requisições HTTP)
- Prefira composição
- Interfaces pequenas — no máximo 2–3 métodos
- Nomes que dispensam comentários
- Siga Effective Go, Go Code Review Comments e Google Go Style Guide

### Arquitetura

```
cmd/stresstest/
  └── main.go              → entry point: parse de flags, validação, wiring e execução

internal/runner/
  └── runner.go            → CORE: worker pool, distribuição de jobs, coleta de resultados

internal/report/
  └── report.go            → OUTPUT: agregação de status codes, formatação e impressão
```

Respeite os limites de cada camada:
- `runner/` não importa nada de `report/` — apenas retorna `Summary`
- `report/` não conhece HTTP nem goroutines — apenas formata e imprime
- `main.go` faz o wiring: chama `runner.Run()`, passa o resultado para `report.Print()`

### Fluxo de dados

```
main
  └─ runner.Run(ctx, url, requests, concurrency) → Summary
        ├─ jobs channel (buffered, cap = requests) preenchido com N tickets
        ├─ C goroutines workers consomem tickets → HTTP GET → enviam Result ao results channel
        ├─ goroutine de progresso: lê contador atômico → imprime via \r
        ├─ WaitGroup aguarda todos os workers
        └─ retorna Summary{Duration, StatusCodes map[int]int, Total int}

report.Print(summary)
  └─ imprime tabela formatada no stdout
```

### Regra crítica de concorrência
O número total de requisições (`--requests`) deve ser cumprido **exatamente**. O channel de jobs com capacidade N e N tickets garante isso sem contador adicional — quando o channel estrá vazio, os workers param naturalmente.

### Barra de progresso
Usar `\r` (carriage return) para sobrescrever a linha atual no terminal. Sem dependência externa. Formato:

```
Progress: 450/1000 requests completed
```

A linha final deve ser encerrada com `\n` após o último request.

---

## Checklist antes de cada commit

- [ ] Todos os erros estão sendo tratados
- [ ] Nenhum valor hardcoded — tudo vem das flags da CLI
- [ ] `context.Context` propagado até as requisições HTTP
- [ ] Limites de pacote respeitados (`runner` sem output, `report` sem concorrência)
- [ ] Testes adicionados ou atualizados para a feature
- [ ] `go vet ./...` e `go build ./...` sem erros
- [ ] `go mod tidy` rodado
- [ ] `docker build` funciona e `docker run` executa corretamente

---

## Git workflow

### Branches
- Crie uma branch por feature a partir da `main`
- Nomenclatura: `feat/`, `fix/`, `test/`, `chore/`, `docs/`
- Após aprovação do usuário, faça merge na `main`
- A próxima branch sempre parte da `main` atualizada

### Fluxo
1. `git checkout main`
2. `git checkout -b feat/nome-da-feature`
3. Implemente em commits atômicos
4. Adicione ou atualize os testes da feature antes de commitar
5. Apresente os arquivos ao usuário para aprovação
6. `git add <arquivos específicos>` — nunca `git add .`
7. Commit após aprovação explícita do usuário
8. `git push -u origin feat/nome-da-feature` — suba a branch para o remoto após o commit
9. Merge na `main`
10. `git push origin main` — suba a `main` atualizada para o remoto após o merge

### Commits
- Mensagens em inglês, Conventional Commits
- `feat:` `fix:` `test:` `chore:` `docs:`
- Um commit = uma mudança lógica
- Nunca mencionar Claude ou IA na mensagem

---

## Notas críticas de implementação

### Worker pool com jobs channel
Preencha o channel de jobs **antes** de iniciar os workers. Isso elimina qualquer necessidade de sincronização adicional para controle de quantidade:

```go
jobs := make(chan struct{}, requests)
for i := 0; i < requests; i++ {
    jobs <- struct{}{}
}
close(jobs)
// inicie os workers após fechar o channel
```

### HTTP client com timeout
Sempre instancie o `http.Client` com timeout explícito. Nunca use o client padrão (`http.Get`) em testes de carga — ele não tem timeout e pode vazar goroutines:

```go
client := &http.Client{Timeout: 30 * time.Second}
```

### Coleta de resultados sem mutex
Use um channel de resultados lido por uma goroutine coletora. Não use slice compartilhada com mutex — o pattern de channel é mais idiomático e elimina a possibilidade de race condition:

```go
results := make(chan int, requests) // recebe status codes
```

### Progresso com contador atômico
Use `sync/atomic` para o contador de progresso — é acessado por múltiplas goroutines workers. A goroutine de progresso lê o valor atômico periodicamente via `time.Ticker`:

```go
var completed atomic.Int64
```

### Dockerfile multi-stage
- Stage `builder`: `golang:1.26-alpine` — compila com `CGO_ENABLED=0` para binário estático
- Stage `runner`: `scratch` — apenas o binário e os certificados TLS (`ca-certificates`)
- O `ENTRYPOINT` deve ser o binário, não `CMD`, para que os flags passados ao `docker run` sejam recebidos diretamente

---

## Dependências do projeto

```
stdlib apenas — net/http, flag, fmt, sync, sync/atomic, time, os
```

Zero dependências externas. Finalize sempre com `go mod tidy`.
