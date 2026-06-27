// ─── 登录状态 ──────────────────────────────────────────
let loggedIn = false;

// ─── DOM 引用 ─────────────────────────────────────────
const loginOverlay = document.getElementById('loginOverlay');
const mainContent = document.getElementById('mainContent');
const userName = document.getElementById('userName');
const btnLogout = document.getElementById('btnLogout');

const loginForm = document.getElementById('loginForm');
const loginUsername = document.getElementById('loginUsername');
const loginPassword = document.getElementById('loginPassword');
const loginError = document.getElementById('loginError');

const registerForm = document.getElementById('registerForm');
const registerUsername = document.getElementById('registerUsername');
const registerPassword = document.getElementById('registerPassword');
const registerError = document.getElementById('registerError');

// ─── Tab 切换 ────────────────────────────────────────
document.querySelectorAll('.login-tab').forEach(tab => {
    tab.addEventListener('click', () => {
        document.querySelectorAll('.login-tab').forEach(t => t.classList.remove('active'));
        tab.classList.add('active');
        const show = tab.dataset.tab;
        loginForm.style.display = show === 'login' ? '' : 'none';
        registerForm.style.display = show === 'register' ? '' : 'none';
        loginError.textContent = '';
        registerError.textContent = '';
    });
});

// ─── 检查登录状态 ────────────────────────────────────
async function checkLogin() {
    try {
        const res = await fetch('/api/user/me');
        const data = await res.json();
        if (data.logged_in) {
            loggedIn = true;
            userName.textContent = data.username;
            loginOverlay.style.display = 'none';
            mainContent.style.display = '';
            connect();
        } else {
            showLogin();
        }
    } catch {
        showLogin();
    }
}

function showLogin() {
    loginOverlay.style.display = 'flex';
    mainContent.style.display = 'none';
}

// ─── 登录 ────────────────────────────────────────────
loginForm.addEventListener('submit', async (e) => {
    e.preventDefault();
    loginError.textContent = '';
    try {
        const res = await fetch('/api/user', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                username: loginUsername.value.trim(),
                password: loginPassword.value,
            }),
        });
        const data = await res.json();
        if (data.ok) {
            checkLogin();
        } else {
            loginError.textContent = data.message;
        }
    } catch {
        loginError.textContent = '网络错误，请重试';
    }
});

// ─── 注册 ────────────────────────────────────────────
registerForm.addEventListener('submit', async (e) => {
    e.preventDefault();
    registerError.textContent = '';
    try {
        const res = await fetch('/api/user/register', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                username: registerUsername.value.trim(),
                password: registerPassword.value,
            }),
        });
        const data = await res.json();
        if (data.ok) {
            checkLogin();
        } else {
            registerError.textContent = data.message;
        }
    } catch {
        registerError.textContent = '网络错误，请重试';
    }
});

// ─── 注销 ────────────────────────────────────────────
btnLogout.addEventListener('click', async () => {
    try {
        await fetch('/api/user/logout', { method: 'POST' });
    } catch { /* ignore */ }
    loggedIn = false;
    showLogin();
    loginUsername.value = '';
    loginPassword.value = '';
});

// ─── WebSocket（登录后才连接） ──────────────────────
const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:';
const wsUrl = `${protocol}//${location.host}/websocket`;
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
        const { type, payload } = JSON.parse(event.data);

        switch (type) {
            case 'home.random':
                updateDisplay(payload);
                break;

            default:
                console.log('未知消息类型:', type, payload);
        }
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

// ─── 启动 ────────────────────────────────────────────
checkLogin();
