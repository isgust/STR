const ws = new WebSocket('ws://' + window.location.host + '/ws');

const startScreen = document.getElementById('start-screen');
const gameScreen = document.getElementById('game-screen');
const lagCompToggle = document.getElementById('lag-comp-toggle');
const showPingToggle = document.getElementById('show-ping-toggle');

const bomb = document.getElementById('bomb');
const timerEl = document.getElementById('timer');
const statusText = document.getElementById('status-text');
const rankingContainer = document.getElementById('ranking-container');
const rankingList = document.getElementById('ranking-list');
const restartBtn = document.getElementById('restart-btn');

let gameState = 'waiting';
let explosionTime = 0;
let renderTimer = null;

const audioCtx = new (window.AudioContext || window.webkitAudioContext)();
function playBeep(freq, type, duration) {
    if (audioCtx.state === 'suspended') audioCtx.resume();
    const osc = audioCtx.createOscillator();
    osc.type = type;
    osc.frequency.setValueAtTime(freq, audioCtx.currentTime);
    osc.connect(audioCtx.destination);
    osc.start();
    osc.stop(audioCtx.currentTime + duration);
}

ws.onopen = () => {
    console.log("Conectado ao servidor");
};

ws.onmessage = (event) => {
    const data = JSON.parse(event.data);
    if (data.type === 'state') {
        updateGameState(data);
    }
};

function updateGameState(data) {
    gameState = data.state;
    explosionTime = data.explosionTime;
    
    if (gameState === 'waiting') {
        startScreen.classList.add('active');
        gameScreen.classList.remove('active');
        resetUI();
    } else {
        startScreen.classList.remove('active');
        gameScreen.classList.add('active');
    }

    if (gameState === 'countdown') {
        bomb.className = 'bomb countdown';
        bomb.innerText = '💣';
        statusText.innerText = "GET READY!";
        statusText.style.color = "white";
        if (!renderTimer) {
            renderTimer = requestAnimationFrame(updateTimerUI);
        }
    } else if (gameState === 'exploded') {
        bomb.className = 'bomb exploded';
        bomb.innerText = '💥';
        timerEl.innerText = "00:00";
        statusText.innerText = "BOOM!";
        statusText.style.color = "red";
        playBeep(100, 'square', 0.5);
        if (renderTimer) {
            cancelAnimationFrame(renderTimer);
            renderTimer = null;
        }
    } else if (gameState === 'results') {
        statusText.innerText = "RESULTS!";
        statusText.style.color = "yellow";
        showResults(data.results, data.explosionTime);
    }
}

function updateTimerUI() {
    if (gameState !== 'countdown') return;
    
    const now = Date.now();
    let timeLeft = explosionTime - now;
    
    if (timeLeft < 0) timeLeft = 0;
    
    const seconds = Math.floor(timeLeft / 1000);
    const ms = Math.floor((timeLeft % 1000) / 10);
    
    timerEl.innerText = `0${seconds}:${ms < 10 ? '0' : ''}${ms}`;
    
    if (timeLeft > 0) {
        renderTimer = requestAnimationFrame(updateTimerUI);
    }
}

function formatTime(timestamp) {
    const date = new Date(timestamp);
    const h = date.getHours().toString().padStart(2, '0');
    const m = date.getMinutes().toString().padStart(2, '0');
    const s = date.getSeconds().toString().padStart(2, '0');
    const ms = date.getMilliseconds().toString().padStart(3, '0');
    return `${h}:${m}:${s}.${ms}`;
}

function showResults(results, expTime) {
    rankingContainer.style.display = 'block';
    rankingList.innerHTML = '';

    const players = [
        { id: 1, name: "P1 (A)", result: results["1"] },
        { id: 2, name: "P2 (G)", result: results["2"] },
        { id: 3, name: "P3 (L)", result: results["3"] }
    ];

    players.forEach(p => {
        updatePlayerResult(p.id, p.result);
    });

    // Filtra só quem apertou
    const validPlayers = players.filter(p => p.result !== undefined);
    
    // Ordena: VIVOS primeiro (por menor erro abs), depois MORTOS (por menor erro abs)
    validPlayers.sort((a, b) => {
        if (a.result.status === "ALIVE" && b.result.status === "DEAD") return -1;
        if (a.result.status === "DEAD" && b.result.status === "ALIVE") return 1;
        return Math.abs(a.result.diff) - Math.abs(b.result.diff);
    });

    let html = `<div style="margin-bottom: 10px; color: var(--highlight);">Objetivo: Apertar exatamente no 00:00.000!</div>`;
    
    validPlayers.forEach((p, index) => {
        let sign = p.result.diff > 0 ? "+" : "";
        
        if (p.result.status === "DEAD") {
            html += `<div style="color: red;">💀 ${p.name} - Clicou depois do zero (Atraso: ${sign}${p.result.diff}ms) -> EXPLODIU!</div>`;
        } else {
            let medal = index === 0 ? "🥇" : (index === 1 ? "🥈" : "🥉");
            
            // diff é negativo se foi antes da explosão.
            let timeLeft = -p.result.diff;
            const seconds = Math.floor(timeLeft / 1000);
            const ms = Math.floor(timeLeft % 1000);
            const clickTimeStr = `0${seconds}:${ms.toString().padStart(3, '0')}`;

            html += `<div style="color: var(--green);">${medal} ${p.name} - Clicou no timer: ${clickTimeStr} (Antecipou: ${timeLeft}ms) -> SOBREVIVEU!</div>`;
        }
    });

    if (validPlayers.length === 0) {
        html += `<div>Ninguém apertou!</div>`;
    }

    rankingList.innerHTML = html;
}

function updatePlayerResult(playerId, result) {
    const resEl = document.getElementById(`p${playerId}-result`);
    if (!result) {
        resEl.innerText = "MISS";
        resEl.className = "result bad";
        return;
    }

    let diff = result.diff;
    let sign = diff > 0 ? "+" : "";
    resEl.innerText = `${sign}${diff}ms`;

    if (Math.abs(diff) < 50) {
        resEl.className = "result good"; // PERFECT
    } else if (Math.abs(diff) < 200) {
        resEl.className = "result late"; // OK
    } else {
        resEl.className = "result bad"; // BAD
    }
}

function resetUI() {
    timerEl.innerText = "--:--";
    bomb.className = 'bomb idle';
    bomb.innerText = '💣';
    rankingContainer.style.display = 'none';
    [1, 2, 3].forEach(i => {
        const el = document.getElementById(`p${i}-result`);
        el.innerText = "--";
        el.className = "result";
    });
}

// Checkbox Latências
showPingToggle.addEventListener('change', (e) => {
    if (e.target.checked) {
        document.body.classList.add('show-pings');
    } else {
        document.body.classList.remove('show-pings');
    }
});

// Botão Reiniciar
restartBtn.addEventListener('click', () => {
    startGame();
});

// Input Handling
window.addEventListener('keydown', (e) => {
    if (e.code === 'Space') {
        if (gameState === 'waiting' || gameState === 'results') {
            e.preventDefault();
            startGame();
        }
    }
    
    if (gameState === 'countdown' || gameState === 'exploded') {
        let player = 0;
        if (e.code === 'KeyA') player = 1;
        if (e.code === 'KeyG') player = 2;
        if (e.code === 'KeyL') player = 3;
        
        if (player > 0) {
            sendPress(player);
        }
    }
});

function startGame() {
    ws.send(JSON.stringify({
        type: 'start',
        lagComp: lagCompToggle.checked
    }));
    playBeep(440, 'sine', 0.1);
}

function sendPress(player) {
    const ping = parseInt(document.getElementById(`p${player}-ping`).value) || 0;
    
    ws.send(JSON.stringify({
        type: 'press',
        player: player,
        clientTime: Date.now(),
        latency: ping
    }));
}
