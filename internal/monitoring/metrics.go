# Создаем директорию для monitoring
mkdir -p internal/monitoring

# Создаем файл metrics.go
cat > internal/monitoring/metrics.go << 'EOF'
package monitoring

import (
    "time"

    "github.com/gin-gonic/gin"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    HTTPRequestsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "http_requests_total",
            Help: "Total number of HTTP requests",
        },
        []string{"method", "endpoint", "status"},
    )

    HTTPRequestDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "http_request_duration_seconds",
            Help:    "HTTP request duration in seconds",
            Buckets: prometheus.DefBuckets,
        },
        []string{"method", "endpoint"},
    )

    RateLimiterBlocks = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "rate_limiter_blocks_total",
            Help: "Total number of requests blocked by rate limiter",
        },
        []string{"ip"},
    )

    LoginAttempts = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "login_attempts_total",
            Help: "Total number of login attempts",
        },
        []string{"status"},
    )
)

func InitMetrics() {
    // Инициализация метрик
    prometheus.Register(HTTPRequestsTotal)
    prometheus.Register(HTTPRequestDuration)
    prometheus.Register(RateLimiterBlocks)
    prometheus.Register(LoginAttempts)
}

func MetricsMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        
        c.Next()
        
        duration := time.Since(start).Seconds()
        status := c.Writer.Status()
        
        HTTPRequestsTotal.WithLabelValues(
            c.Request.Method,
            c.FullPath(),
            http.StatusText(status),
        ).Inc()
        
        HTTPRequestDuration.WithLabelValues(
            c.Request.Method,
            c.FullPath(),
        ).Observe(duration)
    }
}
EOF
