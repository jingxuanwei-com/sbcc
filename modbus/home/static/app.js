const wsUrl = `ws://${location.host}/ws`;
let ws;
let history = [];

function connect() {
    const dot = document.getElementById('statusDot');
    const text = document.getElementById('statusText');

    ws = new WebSocket(wsUrl);

    ws.onopen = () => {
        dot.classList.add('connected');
        text.textContent = '已连接';
    };

    ws.onmessage = (event) => {
        const data = JSON.parse(event.data);
        updateDisplay(data);
    };

    ws.onclose = () => {
        dot.classList.remove('connected');
        text.textContent = '已断开，3秒后重连...';
        setTimeout(connect, 3000);
    };

    ws.onerror = () => {
        ws.close();
    };
}

function updateDisplay(data) {
    const val = document.getElementById('dataValue');
    const time = document.getElementById('dataTime');
    const card = document.getElementById('dataCard');

    val.textContent = data.value;
    time.textContent = `更新于 ${data.time}`;
    card.classList.add('has-data');

    val.classList.remove('pop');
    void val.offsetWidth;
    val.classList.add('pop');

    history.unshift(data);
    if (history.length > 20) history.pop();
    renderHistory();
}

function renderHistory() {
    const list = document.getElementById('historyList');
    if (history.length === 0) {
        list.innerHTML = '<div class="empty-msg">暂无数据</div>';
        return;
    }
    list.innerHTML = history.map(d =>
        `<div class="history-item">
            <span class="h-time">${d.time}</span>
            <span class="h-val">${d.value}</span>
        </div>`
    ).join('');
}

connect();
