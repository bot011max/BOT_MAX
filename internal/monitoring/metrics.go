package monitoring

import (
    "time"

    "github.com/gin-gonic/gin"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
    // HTTP метрики
    HTTPRequestsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "http_requests_total",
            Help: "Total number of HTTP requests",
        },
        []string{"method", "endpoint", "status", "role"},
    )

    HTTPRequestDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "http_request_duration_seconds",
            Help:    "HTTP request duration in seconds",
            Buckets: prometheus.DefBuckets,
        },
        []string{"method", "endpoint"},
    )

    // Бизнес-метрики
    ActiveUsers = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "active_users_total",
            Help: "Total number of active users by role",
        },
        []string{"role"},
    )

    SubscriptionsByTier = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "subscriptions_total",
            Help: "Total number of subscriptions by tier",
        },
        []string{"tier"},
    )

    PrescriptionsCreated = promauto.NewCounter(
        prometheus.CounterOpts{
            Name: "prescriptions_created_total",
            Help: "Total number of prescriptions created",
        },
    )

    RemindersSent = promauto.NewCounter(
        prometheus.CounterOpts{
            Name: "reminders_sent_total",
            Help: "Total number of reminders sent",
        },
    )

    // Метрики безопасности
    FailedLogins = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "failed_logins_total",
            Help: "Total number of failed login attempts",
        },
        []string{"reason"},
    )

    WAFBlocks = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "waf_blocks_total",
            Help: "Total number of requests blocked by WAF",
        },
        []string{"rule"},
    )

    RateLimiterBlocks = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "rate_limiter_blocks_total",
            Help: "Total number of requests blocked by rate limiter",
        },
        []string{"ip"},
    )

    // Метрики базы данных
    DatabaseQueries = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "database_queries_total",
            Help: "Total number of database queries",
        },
        []string{"operation", "table"},
    )

    DatabaseQueryDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "database_query_duration_seconds",
            Help:    "Database query duration in seconds",
            Buckets: prometheus.DefBuckets,
        },
        []string{"operation", "table"},
    )

    // Метрики очередей
    QueueSize = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "queue_size",
            Help: "Current size of the queue",
        },
        []string{"queue_name"},
    )

    // Метрики Telegram бота
    TelegramMessages = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "telegram_messages_total",
            Help: "Total number of Telegram messages",
        },
        []string{"type", "command"},
    )

    TelegramActiveChats = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "telegram_active_chats_total",
            Help: "Total number of active Telegram chats",
        },
    )

    // Бизнес-метрики по лимитам
    PatientsByDoctor = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "patients_by_doctor",
            Help: "Number of patients per doctor",
        },
        []string{"doctor_id"},
    )

    UsagePercentBySubscription = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "usage_percent_by_subscription",
            Help: "Usage percentage by subscription tier",
        },
        []string{"tier", "metric"},
    )
)

// MetricsMiddleware - middleware для сбора метрик
func MetricsMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        
        c.Next()
        
        duration := time.Since(start).Seconds()
        status := c.Writer.Status()
        role := c.GetString("user_role")
        if role == "" {
            role = "anonymous"
        }
        
        HTTPRequestsTotal.WithLabelValues(
            c.Request.Method,
            c.FullPath(),
            http.StatusText(status),
            role,
        ).Inc()
        
        HTTPRequestDuration.WithLabelValues(
            c.Request.Method,
            c.FullPath(),
        ).Observe(duration)
    }
}

// MetricsHandler - handler для Prometheus
func MetricsHandler() gin.HandlerFunc {
    h := promhttp.Handler()
    return func(c *gin.Context) {
        h.ServeHTTP(c.Writer, c.Request)
    }
}

// RecordDatabaseQuery - запись метрики запроса к БД
func RecordDatabaseQuery(operation, table string, duration time.Duration) {
    DatabaseQueries.WithLabelValues(operation, table).Inc()
    DatabaseQueryDuration.WithLabelValues(operation, table).Observe(duration.Seconds())
}

// UpdateSubscriptionMetrics - обновление метрик подписок
func UpdateSubscriptionMetrics(tierCounts map[string]int) {
    for tier, count := range tierCounts {
        SubscriptionsByTier.WithLabelValues(tier).Set(float64(count))
    }
}

// UpdateActiveUsers - обновление метрик активных пользователей
func UpdateActiveUsers(roleCounts map[string]int) {
    for role, count := range roleCounts {
        ActiveUsers.WithLabelValues(role).Set(float64(count))
    }
}
