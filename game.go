package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	conn *websocket.Conn
	game *Game
	send chan []byte
}

type Msg struct {
	Type       string `json:"type"` // "start", "press", "state"
	Player     int    `json:"player,omitempty"`
	ClientTime int64  `json:"clientTime,omitempty"`
	Latency    int64  `json:"latency,omitempty"`
	LagComp    bool   `json:"lagComp,omitempty"`
}

type GameState string

const (
	StateWaiting   GameState = "waiting"
	StateCountdown GameState = "countdown"
	StateExploded  GameState = "exploded"
	StateResults   GameState = "results"
)

type ResultDetail struct {
	Diff       int64  `json:"diff"`
	EventTime  int64  `json:"eventTime"`
	ClientDiff int64  `json:"clientDiff"`
	Status     string `json:"status"`
}

type Game struct {
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte

	mu            sync.Mutex
	state         GameState
	explosionTime int64 // timestamp em ms
	timer         *time.Timer
	results       map[int]ResultDetail // player ID -> ResultDetail
	lagComp       bool
}

func NewGame() *Game {
	return &Game{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte),
		state:      StateWaiting,
		results:    make(map[int]ResultDetail),
	}
}

func (g *Game) Run() {
	for {
		select {
		case client := <-g.register:
			g.mu.Lock()
			g.clients[client] = true
			g.mu.Unlock()
			g.broadcastState()
		case client := <-g.unregister:
			g.mu.Lock()
			if _, ok := g.clients[client]; ok {
				delete(g.clients, client)
				close(client.send)
			}
			g.mu.Unlock()
		case message := <-g.broadcast:
			g.mu.Lock()
			for client := range g.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(g.clients, client)
				}
			}
			g.mu.Unlock()
		}
	}
}

func (g *Game) broadcastState() {
	g.mu.Lock()
	defer g.mu.Unlock()

	data := map[string]interface{}{
		"type":          "state",
		"state":         g.state,
		"results":       g.results,
		"lagComp":       g.lagComp,
		"explosionTime": g.explosionTime,
	}
	msg, _ := json.Marshal(data)
	
	for client := range g.clients {
		select {
		case client.send <- msg:
		default:
			close(client.send)
			delete(g.clients, client)
		}
	}
}

func (g *Game) handleStart(lagComp bool) {
	g.mu.Lock()
	if g.state == StateCountdown {
		g.mu.Unlock()
		return
	}
	if g.timer != nil {
		g.timer.Stop()
	}
	g.state = StateCountdown
	g.lagComp = lagComp
	g.results = make(map[int]ResultDetail)

	// Tempo aleatório entre 3 e 7 segundos
	delay := time.Duration(rand.Intn(4000)+3000) * time.Millisecond
	g.explosionTime = time.Now().Add(delay).UnixMilli()
	
	g.timer = time.AfterFunc(delay, func() {
		g.mu.Lock()
		if g.state != StateCountdown {
			g.mu.Unlock()
			return
		}
		g.state = StateExploded
		g.mu.Unlock()
		g.broadcastState()

		// Vai para a tela de resultados após 3 segundos no máximo,
		// caso alguém não aperte.
		time.AfterFunc(3*time.Second, func() {
			g.mu.Lock()
			if g.state == StateExploded {
				g.state = StateResults
				g.mu.Unlock()
				g.broadcastState()
			} else {
				g.mu.Unlock()
			}
		})
	})
	g.mu.Unlock()
	g.broadcastState()
}

func (g *Game) handlePress(player int, simulatedLatency int64, clientTime int64) {
	// Atraso artificial para simular a rede (Latência)
	time.Sleep(time.Duration(simulatedLatency) * time.Millisecond)

	g.mu.Lock()
	defer g.mu.Unlock()

	if g.state != StateCountdown && g.state != StateExploded {
		return
	}

	// Já apertou
	if _, exists := g.results[player]; exists {
		return
	}

	serverReceiveTime := time.Now().UnixMilli()
	
	var eventTime int64
	if g.lagComp {
		// Compensação de Lag ATIVADA
		// O servidor "retrocede" o tempo usando o RTT (que aqui chamamos de simulatedLatency)
		// Isso é a essência do "Server-side Lag Compensation"
		eventTime = serverReceiveTime - simulatedLatency
	} else {
		// Compensação de Lag DESATIVADA
		// O servidor usa o tempo que a mensagem chegou. Jogadores com ping alto se prejudicam muito.
		eventTime = serverReceiveTime
	}

	status := "ALIVE"
	if eventTime >= g.explosionTime {
		status = "DEAD"
	}

	diff := eventTime - g.explosionTime
	clientDiff := clientTime - g.explosionTime
	g.results[player] = ResultDetail{
		Diff:       diff,
		EventTime:  eventTime,
		ClientDiff: clientDiff,
		Status:     status,
	}
	
	// Se todos apertaram, encerra imediatamente
	if len(g.results) == 3 {
		g.state = StateResults
	}

	// Atualiza os clientes
	data := map[string]interface{}{
		"type":          "state",
		"state":         g.state,
		"results":       g.results,
		"lagComp":       g.lagComp,
		"explosionTime": g.explosionTime, // Corrige o NaN
	}
	msg, _ := json.Marshal(data)
	for client := range g.clients {
		select {
		case client.send <- msg:
		default:
			close(client.send)
			delete(g.clients, client)
		}
	}
}

// Client pump functions
func (c *Client) readPump() {
	defer func() {
		c.game.unregister <- c
		c.conn.Close()
	}()
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		
		var msg Msg
		if err := json.Unmarshal(message, &msg); err == nil {
			switch msg.Type {
			case "start":
				c.game.handleStart(msg.LagComp)
			case "press":
				go c.game.handlePress(msg.Player, msg.Latency, msg.ClientTime)
			}
		}
	}
}

func (c *Client) writePump() {
	defer func() {
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)
			if err := w.Close(); err != nil {
				return
			}
		}
	}
}
