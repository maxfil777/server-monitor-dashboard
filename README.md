# Server Monitor Dashboard v1.0

Лёгкий дашборд мониторинга сервера. Собирает метрики CPU, RAM, PHP-FPM active processes и FPM status errors с внешнего HTTP JSON-эндпоинта, сохраняет в SQLite и отображает на тёмном веб-дашборде с Chart.js.

Спроектирован для VDS с 1 ГБ RAM. Не требует Docker, Node.js, React, PostgreSQL, Prometheus или Grafana.

## Метрики

| Метрика | Описание |
|---|---|
| `cpu` | Загрузка CPU, % |
| `mem` | Использование RAM, % |
| `fpm_active` | Активные PHP-FPM процессы |
| `fpm_status_errors` | Ошибки FPM за интервал |

## Требования

- Go 1.21+
- Не нужны CGO, gcc или libsqlite3-dev
- 64-битный Linux (для продакшена), но собирается кросс-платформенно

## Сборка

```bash
go mod tidy
go build -o server-monitor-dashboard .
```

Кросс-компиляция для Linux с любой платформы (Windows/macOS/Linux):

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o server-monitor-dashboard .
```

## Конфигурация

Отредактируйте `config.json`:

| Поле | Описание | По умолчанию |
|---|---|---|
| `listen_addr` | Адрес HTTP-сервера | `127.0.0.1:8090` |
| `metrics_url` | Внешний JSON-эндпоинт с метриками | `http://127.0.0.1:8080/metrics.json` |
| `poll_interval_seconds` | Интервал опроса метрик (сек) | `60` |
| `retention_days` | Дней хранить историю | `30` |
| `database_path` | Путь к файлу SQLite | `./metrics.db` |

Формат внешнего JSON-эндпоинта:

```json
{"cpu": 14.7, "mem": 39.9, "fpm_active": 23, "fpm_status_errors": 0}
```

## Запуск

```bash
./server-monitor-dashboard
```

Откройте http://127.0.0.1:8090 в браузере.

## Systemd Service

```bash
sudo cp server-monitor-dashboard.service /etc/systemd/system/
sudo mkdir -p /opt/server-monitor-dashboard
sudo cp server-monitor-dashboard config.json /opt/server-monitor-dashboard/
sudo cp -r static /opt/server-monitor-dashboard/
sudo systemctl daemon-reload
sudo systemctl enable server-monitor-dashboard
sudo systemctl start server-monitor-dashboard
```

Проверка статуса:

```bash
sudo systemctl status server-monitor-dashboard
```

## Nginx Reverse Proxy

Создайте конфиг `/etc/nginx/sites-available/monitor`:

```nginx
server {
    listen 80;
    server_name monitor.example.com;

    location / {
        proxy_pass http://127.0.0.1:8090/;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

Если нужен HTTPS (рекомендуется):

```bash
sudo apt install certbot python3-certbot-nginx
sudo certbot --nginx -d monitor.example.com
```

Активируйте сайт:

```bash
sudo ln -s /etc/nginx/sites-available/monitor /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

Для размещения на подпапке (например `domain.com/monitor/`):

```nginx
location /monitor/ {
    proxy_pass http://127.0.0.1:8090/;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
}
```

Все статические файлы и API-запросы используют относительные пути и работают из-под подпапки.

## API

### `GET /api/latest`

Последние записанные метрики:

```json
{
  "created_at": 1779183783,
  "cpu_percent": 18.6,
  "memory_percent": 40.0,
  "fpm_active": 23,
  "fpm_status_errors": 0
}
```

### `GET /api/metrics?from=1716000000&to=1716086400`

Массив метрик за период (Unix timestamp). По умолчанию — последние 24 часа.

Максимум 1500 точек — если данных больше, сервер прореживает их равномерно.

### `GET /`

HTML-дашборд.

## Дашборд

- 4 карточки с текущими значениями (CPU, RAM, FPM Active, FPM Errors)
- Время последнего обновления
- Кнопки периода: 1 ч / 6 ч / 24 ч / 7 д / 30 д
- 4 графика Chart.js
- Автообновление: карточки — каждые 10 сек, графики — каждую минуту
- Тёмная тема
- Адаптивная вёрстка (4 колонки на десктопе, 2 на мобильном)

## Структура проекта

```
server-monitor-dashboard/
├── main.go                  # входная точка, конфиг, запуск HTTP
├── storage.go               # SQLite: схема, CRUD, дисамплинг
├── collector.go             # фоновый сбор метрик с очисткой
├── handlers.go              # API /api/latest, /api/metrics, /
├── config.json              # конфигурация
├── go.mod / go.sum          # Go-модуль (modernc.org/sqlite)
├── static/
│   ├── index.html           # дашборд
│   ├── app.js               # Chart.js + автообновление
│   └── style.css            # тёмная тема
├── server-monitor-dashboard.service  # systemd unit
└── README.md
```

## Стек технологий

- Go (стандартная библиотека + `modernc.org/sqlite`)
- SQLite (WAL-режим, busy_timeout)
- Chart.js 4.x (CDN)
- Чистый HTML/CSS/JS (без UI-фреймворков)

## История версий

- **v1.0** — Первый релиз: сбор 4 метрик, SQLite с ротацией 30 дней, 4 графика, тёмная тема, systemd, nginx reverse proxy
