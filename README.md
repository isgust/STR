# Super Latency Bomber

Bem-vindo ao **Super Latency Bomber**! Este projeto é um jogo multiplayer simples, baseado em WebSockets e Go, projetado para demonstrar os conceitos de **Tempo Real**, **Latência (Ping)** e **Compensação de Lag (Lag Compensation)** em jogos multiplayer.

## 🚀 Como Rodar o Projeto

### Pré-requisitos
- Ter a linguagem **Go** instalada na máquina (versão 1.18+ recomendada).

### Passos para Execução
1. Abra o terminal na pasta raiz do projeto (`STR`).
2. Execute o servidor Go com o comando:
   ```bash
   go run .
   ```
   *(Ou `go run main.go game.go` se preferir).*
3. O terminal exibirá `Servidor rodando na porta :8080...`.
4. Abra o seu navegador e acesse: [http://localhost:8080](http://localhost:8080)
5. Você verá a tela inicial do jogo. Aperte **ESPAÇO** para iniciar a partida!

---

## ⏱️ O Conceito de Tempo Real e Latência

Neste jogo, você tem 3 personagens (P1, P2 e P3) controlados pelas teclas **A**, **G** e **L**. Cada jogador possui um **Ping (Latência)** simulado que pode ser alterado na interface. 

O objetivo do jogo é pressionar o seu respectivo botão **exatamente no momento em que o cronômetro chegar a 0** (quando a bomba explode).

Em um cenário de rede real (Tempo Real), a mensagem que um cliente envia pela internet leva um tempo para chegar ao servidor (Latência). Se o servidor apenas considerar o momento em que ele *recebe* a mensagem, jogadores com Ping alto (ex: 400ms) estarão em enorme desvantagem, pois suas ações sempre chegarão atrasadas.

### Compensação de Lag (Lag Compensation)

Para equilibrar a partida, ativamos a **Compensação de Lag**. Com ela, o servidor leva em consideração a latência de cada jogador para calcular o momento exato em que a ação *realmente ocorreu no cliente*.

Veja como isso é feito no nosso código Go (`game.go`):

```go
func (g *Game) handlePress(player int, simulatedLatency int64) {
	// Atraso artificial para simular a rede (Latência)
	time.Sleep(time.Duration(simulatedLatency) * time.Millisecond)

	serverReceiveTime := time.Now().UnixMilli()
	
	var eventTime int64
	if g.lagComp {
		// Compensação de Lag ATIVADA
		// O servidor "retrocede" o tempo usando a latência do jogador.
		eventTime = serverReceiveTime - simulatedLatency
	} else {
		// Compensação de Lag DESATIVADA
		// O servidor usa o tempo em que a mensagem chegou. Jogadores com ping alto se prejudicam muito.
		eventTime = serverReceiveTime
	}

	// ... verifica se o eventTime passou do tempo da explosão
}
```

No código acima, ao receber a ação do jogador, o servidor "volta no tempo" subtraindo a latência da requisição (`serverReceiveTime - simulatedLatency`). Isso garante que o servidor saiba exatamente quando o jogador apertou o botão na sua própria tela, proporcionando um ambiente justo para jogadores de diferentes partes do mundo.

---

## 💻 Estrutura do Código

- `main.go`: Ponto de entrada da aplicação. Inicia o servidor HTTP, serve os arquivos da pasta `static` e gerencia o upgrade das conexões HTTP para **WebSockets**.
- `game.go`: Contém toda a lógica do jogo (Estado da partida, cronômetro, processamento de cliques e cálculo de latência).
- `static/`: Contém o Front-end do jogo.
  - `index.html`: Interface visual retro (inspirada em 8-bits).
  - `style.css`: Estilização e animações do jogo.
  - `main.js`: Lógica do cliente, conexão via WebSocket e cálculos de renderização do timer.
