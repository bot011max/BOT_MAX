document.addEventListener('DOMContentLoaded', function() {
    console.log('Медицинский бот загружен');

    // Проверка соединения с API
    fetch('/health')
        .then(response => response.json())
        .then(data => {
            console.log('API Status:', data);
        })
        .catch(error => {
            console.error('API Error:', error);
        });

    // Обработка формы регистрации
    const registerForm = document.getElementById('register-form');
    if (registerForm) {
        registerForm.addEventListener('submit', handleRegister);
    }

    // Обработка формы входа
    const loginForm = document.getElementById('login-form');
    if (loginForm) {
        loginForm.addEventListener('submit', handleLogin);
    }

    // Обработка кнопок меню
    document.querySelectorAll('.menu-btn').forEach(btn => {
        btn.addEventListener('click', function() {
            const action = this.dataset.action;
            if (action) {
                navigateTo(action);
            }
        });
    });

    // Проверка аутентификации
    checkAuth();
});

async function handleRegister(event) {
    event.preventDefault();
    
    const formData = new FormData(event.target);
    const data = Object.fromEntries(formData);

    try {
        const response = await fetch('/api/register', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(data)
        });

        const result = await response.json();
        
        if (result.success) {
            alert('Регистрация успешна! Проверьте email для подтверждения.');
            if (result.twofa_secret) {
                show2FASetup(result.twofa_secret);
            } else {
                window.location.href = '/login';
            }
        } else {
            alert('Ошибка: ' + (result.error || 'Неизвестная ошибка'));
        }
    } catch (error) {
        console.error('Error:', error);
        alert('Произошла ошибка при регистрации');
    }
}

async function handleLogin(event) {
    event.preventDefault();
    
    const formData = new FormData(event.target);
    const data = Object.fromEntries(formData);

    try {
        const response = await fetch('/api/login', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(data)
        });

        const result = await response.json();
        
        if (result.twofa_required) {
            show2FAInput();
        } else if (result.success) {
            localStorage.setItem('token', result.token);
            localStorage.setItem('user', JSON.stringify(result.user));
            showNotification('Вход выполнен успешно!', 'success');
            window.location.href = '/profile';
        } else {
            alert('Ошибка: ' + (result.error || 'Неверные учетные данные'));
        }
    } catch (error) {
        console.error('Error:', error);
        alert('Произошла ошибка при входе');
    }
}

function show2FASetup(secret) {
    const container = document.getElementById('twofa-container');
    if (!container) return;

    container.innerHTML = `
        <div class="card">
            <h2>🔐 Настройка двухфакторной аутентификации</h2>
            <p>Отсканируйте QR-код в Google Authenticator:</p>
            <div id="qrcode"></div>
            <p>Или введите код вручную: <strong>${secret}</strong></p>
            <button class="btn" onclick="verify2FA()">Продолжить</button>
        </div>
    `;

    // Генерация QR-кода
    new QRCode(document.getElementById("qrcode"), {
        text: `otpauth://totp/MedicalBot:${localStorage.getItem('email')}?secret=${secret}&issuer=MedicalBot`,
        width: 200,
        height: 200
    });
}

function showNotification(message, type) {
    const notification = document.createElement('div');
    notification.className = `notification ${type}`;
    notification.textContent = message;
    notification.style.cssText = `
        position: fixed;
        top: 20px;
        right: 20px;
        padding: 15px 20px;
        background: ${type === 'success' ? '#28a745' : '#dc3545'};
        color: white;
        border-radius: 5px;
        z-index: 1000;
        animation: slideIn 0.3s ease;
    `;

    document.body.appendChild(notification);

    setTimeout(() => {
        notification.remove();
    }, 3000);
}

function checkAuth() {
    const token = localStorage.getItem('token');
    const user = JSON.parse(localStorage.getItem('user') || '{}');

    if (token && user.role) {
        document.querySelectorAll('.auth-only').forEach(el => {
            el.style.display = 'block';
        });

        if (user.role === 'admin') {
            document.querySelectorAll('.admin-only').forEach(el => {
                el.style.display = 'block';
            });
        }
    }
}

function navigateTo(path) {
    window.location.href = path;
}

// Добавление CSS анимации
const style = document.createElement('style');
style.textContent = `
    @keyframes slideIn {
        from {
            transform: translateX(100%);
            opacity: 0;
        }
        to {
            transform: translateX(0);
            opacity: 1;
        }
    }
`;
document.head.appendChild(style);
