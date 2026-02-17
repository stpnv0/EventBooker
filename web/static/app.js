let currentUser = null;
let eventsCache = {};

const API = '/api';

async function api(method, path, body = null) {
    const opts = {
        method,
        headers: { 'Content-Type': 'application/json' },
    };
    if (body) opts.body = JSON.stringify(body);

    const res = await fetch(API + path, opts);
    const data = await res.json();

    if (!res.ok) {
        throw new Error(data.error || `HTTP ${res.status}`);
    }
    return data;
}

// ── Toast ──
function showToast(message, type = 'success') {
    const toast = document.getElementById('toast');
    toast.textContent = message;
    toast.className = `toast ${type}`;
    setTimeout(() => toast.classList.add('hidden'), 3000);
}

// ── Tabs ──
document.querySelectorAll('.tab').forEach(tab => {
    tab.addEventListener('click', () => {
        document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
        document.querySelectorAll('.panel').forEach(p => p.classList.remove('active'));
        tab.classList.add('active');
        document.getElementById(tab.dataset.tab).classList.add('active');
    });
});

// ── Format helpers ──
function formatDate(iso) {
    return new Date(iso).toLocaleString('ru-RU', {
        day: '2-digit', month: '2-digit', year: 'numeric',
        hour: '2-digit', minute: '2-digit'
    });
}

function timeAgo(iso) {
    const diff = Date.now() - new Date(iso).getTime();
    const minutes = Math.floor(diff / 60000);
    if (minutes < 1) return 'только что';
    if (minutes < 60) return `${minutes} мин. назад`;
    const hours = Math.floor(minutes / 60);
    if (hours < 24) return `${hours} ч. назад`;
    return `${Math.floor(hours / 24)} дн. назад`;
}

function statusBadge(status) {
    const labels = {
        pending: '⏳ Ожидает оплаты',
        confirmed: '✅ Подтверждено',
        cancelled: '❌ Отменено'
    };
    return `<span class="badge badge-${status}">${labels[status] || status}</span>`;
}

function getEventName(eventId) {
    const cached = eventsCache[eventId];
    return cached ? cached.event.title : eventId.slice(0, 8) + '...';
}

function getEventDate(eventId) {
    const cached = eventsCache[eventId];
    return cached ? formatDate(cached.event.event_date) : '';
}

function getEventRequiresPayment(eventId) {
    const cached = eventsCache[eventId];
    return cached ? cached.event.requires_payment : true;
}

// ── User Panel: Auth ──
async function handleRegisterUser() {
    const username = document.getElementById('username').value.trim();
    if (!username) {
        showToast('Введите имя пользователя', 'error');
        return;
    }

    const chatIdStr = document.getElementById('telegram-chat-id').value.trim();
    const body = { username };
    if (chatIdStr) body.telegram_chat_id = parseInt(chatIdStr, 10);

    try {
        const user = await api('POST', '/users', body);
        setCurrentUser(user);
        showToast(`Пользователь ${user.username} зарегистрирован`);
    } catch (e) {
        try {
            const users = await api('GET', '/users');
            const found = users.find(u => u.username === username);
            if (found) {
                setCurrentUser(found);
                showToast(`Вход как ${found.username}`);
            } else {
                showToast(e.message, 'error');
            }
        } catch {
            showToast(e.message, 'error');
        }
    }
}

function setCurrentUser(user) {
    currentUser = user;
    const box = document.getElementById('current-user');
    box.classList.remove('hidden');
    box.innerHTML = `
        <strong>👤 ${user.username}</strong>
        <span class="user-id">ID: ${user.id.slice(0, 8)}...</span>
        ${user.telegram_chat_id ? `<span class="user-tg">📱 ${user.telegram_chat_id}</span>` : ''}
    `;

    document.getElementById('my-bookings-card').style.display = 'block';

    loadEvents();
    loadMyBookings();
}

// ── User Panel: Events ──
async function loadEvents() {
    try {
        const events = await api('GET', '/events');
        const list = document.getElementById('events-list');

        if (!events.length) {
            list.innerHTML = '<div class="empty-state">Нет мероприятий</div>';
            return;
        }

        const details = await Promise.all(
            events.map(e => api('GET', `/events/${e.id}`))
        );

        details.forEach(d => { eventsCache[d.event.id] = d; });

        list.innerHTML = details.map(d => {
            const spotsClass = d.available_spots === 0 ? 'no-spots' : '';

            // Кнопка зависит от типа мероприятия
            let bookBtn = '';
            if (currentUser && d.available_spots > 0) {
                const btnLabel = d.event.requires_payment ? 'Забронировать' : 'Записаться';
                bookBtn = `<button class="btn-small btn-book" onclick="handleBookEvent('${d.event.id}')">
                               ${btnLabel}
                           </button>`;
            } else if (d.available_spots === 0) {
                bookBtn = '<span class="badge badge-full">Мест нет</span>';
            }

            // TTL показываем только если требуется подтверждение
            const ttlInfo = d.event.requires_payment
                ? `<span>⏰ ${d.event.booking_ttl}</span>`
                : '<span class="badge badge-confirmed">Без подтверждения</span>';

            return `
                <div class="list-item event-item ${spotsClass}">
                    <div class="event-header">
                        <h3>${d.event.title}</h3>
                        ${bookBtn}
                    </div>
                    <div class="meta">
                        <span>📅 ${formatDate(d.event.event_date)}</span>
                        <span class="badge badge-spots">
                            🪑 ${d.available_spots} / ${d.event.total_spots}
                        </span>
                        ${ttlInfo}
                    </div>
                </div>
            `;
        }).join('');

    } catch (e) {
        showToast(e.message, 'error');
    }
}

function handleLoadEvents() { loadEvents(); }

async function handleBookEvent(eventId) {
    if (!currentUser) {
        showToast('Сначала войдите', 'error');
        return;
    }

    try {
        const booking = await api('POST', `/events/${eventId}/book`, { user_id: currentUser.id });

        if (booking.status === 'confirmed') {
            showToast('Вы записаны на мероприятие! ✅');
        } else {
            showToast('Место забронировано! Не забудьте подтвердить оплату.');
        }

        loadEvents();
        loadMyBookings();
    } catch (e) {
        showToast(e.message, 'error');
    }
}

// ── User Panel: My Bookings ──
async function loadMyBookings() {
    if (!currentUser) return;

    try {
        const bookings = await api('GET', `/users/${currentUser.id}/bookings`);

        const pending = bookings.filter(b => b.status === 'pending');
        const confirmed = bookings.filter(b => b.status === 'confirmed');
        const cancelled = bookings.filter(b => b.status === 'cancelled');

        renderBookingSection('pending', pending, true);
        renderBookingSection('confirmed', confirmed, false);
        renderBookingSection('cancelled', cancelled, false);

        const noBookings = document.getElementById('no-bookings');
        if (bookings.length === 0) {
            noBookings.classList.remove('hidden');
        } else {
            noBookings.classList.add('hidden');
        }

    } catch (e) {
        showToast(e.message, 'error');
    }
}

function renderBookingSection(status, bookings, showConfirmBtn) {
    const section = document.getElementById(`${status}-bookings-section`);
    const container = document.getElementById(`${status}-bookings`);

    if (bookings.length === 0) {
        section.classList.add('hidden');
        return;
    }

    section.classList.remove('hidden');

    container.innerHTML = bookings.map(b => {
        const eventName = getEventName(b.event_id);
        const eventDate = getEventDate(b.event_id);
        const requiresPayment = getEventRequiresPayment(b.event_id);

        // Кнопка подтверждения — только для pending + мероприятие требует подтверждения
        let confirmBtn = '';
        if (showConfirmBtn && requiresPayment) {
            confirmBtn = `<button class="btn-small btn-confirm" onclick="handleConfirmBooking('${b.event_id}')">
                              ✅ Подтвердить оплату
                          </button>`;
        }

        const timeInfo = status === 'pending'
            ? `<span class="time-warning">⏰ Создано ${timeAgo(b.created_at)}</span>`
            : `<span class="time-info">${formatDate(b.created_at)}</span>`;

        return `
            <div class="list-item booking-card booking-${status}">
                <div class="booking-header">
                    <div>
                        <h3>${eventName}</h3>
                        ${eventDate ? `<span class="booking-event-date">📅 ${eventDate}</span>` : ''}
                    </div>
                    ${confirmBtn}
                </div>
                <div class="meta">
                    ${statusBadge(b.status)}
                    ${timeInfo}
                </div>
            </div>
        `;
    }).join('');
}

function handleLoadMyBookings() { loadMyBookings(); }

async function handleConfirmBooking(eventId) {
    if (!currentUser) {
        showToast('Сначала войдите', 'error');
        return;
    }

    try {
        await api('POST', `/events/${eventId}/confirm`, { user_id: currentUser.id });
        showToast('Бронь подтверждена! ✅');
        loadEvents();
        loadMyBookings();
    } catch (e) {
        showToast(e.message, 'error');
    }
}

// ── Admin Panel ──
async function handleCreateEvent() {
    const title = document.getElementById('event-title').value.trim();
    const description = document.getElementById('event-description').value.trim();
    const dateStr = document.getElementById('event-date').value;
    const spots = parseInt(document.getElementById('event-spots').value, 10);
    const ttl = parseInt(document.getElementById('event-ttl').value, 10) || 0;
    const requiresPayment = document.getElementById('event-requires-payment').checked;

    if (!title || !description || !dateStr || !spots) {
        showToast('Заполните все обязательные поля', 'error');
        return;
    }

    try {
        const event = await api('POST', '/events', {
            title,
            description,
            event_date: new Date(dateStr).toISOString(),
            total_spots: spots,
            booking_ttl_minutes: ttl,
            requires_payment: requiresPayment
        });
        showToast(`Мероприятие "${event.title}" создано`);

        document.getElementById('event-title').value = '';
        document.getElementById('event-description').value = '';
        document.getElementById('event-date').value = '';
        document.getElementById('event-spots').value = '50';
        document.getElementById('event-ttl').value = '20';
        document.getElementById('event-requires-payment').checked = true;

        handleLoadAdminEvents();
    } catch (e) {
        showToast(e.message, 'error');
    }
}

async function loadAdminEvents() {
    try {
        const events = await api('GET', '/events');
        const list = document.getElementById('admin-events-list');

        if (!events.length) {
            list.innerHTML = '<div class="empty-state">Нет мероприятий</div>';
            return;
        }

        const details = await Promise.all(
            events.map(e => api('GET', `/events/${e.id}`))
        );

        list.innerHTML = details.map(d => {
            const bookingsHtml = d.bookings && d.bookings.length
                ? `<div class="bookings-list">
                       <strong>Бронирования (${d.bookings.length}):</strong>
                       ${d.bookings.map(b => `
                           <div class="booking-item">
                               👤 ${b.user_id.slice(0, 8)}...
                               ${statusBadge(b.status)}
                               <small>${formatDate(b.created_at)}</small>
                           </div>
                       `).join('')}
                   </div>`
                : '<div class="bookings-list"><em>Нет бронирований</em></div>';

            const confirmInfo = d.event.requires_payment
                ? `<span>⏰ TTL: ${d.event.booking_ttl}</span>`
                : '<span class="badge badge-confirmed">Без подтверждения</span>';

            return `
                <div class="list-item">
                    <h3>${d.event.title}</h3>
                    <div class="meta">
                        <span>📅 ${formatDate(d.event.event_date)}</span>
                        <span class="badge badge-spots">
                            🪑 ${d.available_spots} / ${d.event.total_spots}
                        </span>
                        ${confirmInfo}
                    </div>
                    <p style="margin-top:0.5rem;font-size:0.9rem;color:#555">
                        ${d.event.description}
                    </p>
                    ${bookingsHtml}
                </div>
            `;
        }).join('');

    } catch (e) {
        showToast(e.message, 'error');
    }
}

function handleLoadAdminEvents() { loadAdminEvents(); }

async function handleLoadUsers() {
    try {
        const users = await api('GET', '/users');
        const list = document.getElementById('users-list');

        if (!users.length) {
            list.innerHTML = '<div class="empty-state">Нет пользователей</div>';
            return;
        }

        list.innerHTML = users.map(u => `
            <div class="list-item">
                <h3>👤 ${u.username}</h3>
                <div class="meta">
                    <span>ID: ${u.id.slice(0, 8)}...</span>
                    ${u.telegram_chat_id
            ? `<span>📱 ${u.telegram_chat_id}</span>`
            : '<span style="color:#999">Telegram не привязан</span>'
        }
                    <span>📅 ${formatDate(u.created_at)}</span>
                </div>
            </div>
        `).join('');

    } catch (e) {
        showToast(e.message, 'error');
    }
}

setInterval(() => {
    const activePanel = document.querySelector('.panel.active');
    if (activePanel.id === 'user') {
        loadEvents();
        if (currentUser) loadMyBookings();
    } else {
        loadAdminEvents();
    }
}, 10000);