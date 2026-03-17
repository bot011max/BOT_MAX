// internal/security/rate_limiter.go
package security

import (
    "fmt"
    "net/http"
    "sync"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/go-redis/redis/v8"
    "golang.org/x/time/rate"
)

type AdaptiveRateLimiter struct {
    redis        *redis.Client
    limits       map[string]*RateLimit
    mu           sync.RWMutex
    blocklist    *sync.Map
    config       *RateLimiterConfig
}

type RateLimit struct {
    limiter     *rate.Limiter
    lastSeen    time.Time
    burstCount  int
    blockedUntil time.Time
}

type RateLimiterConfig struct {
    RequestsPerSecond float64
    Burst             int
    BlockDuration     time.Duration
    CleanupInterval   time.Duration
    EnableAdaptive    bool
}

func NewAdaptiveRateLimiter(redisAddr string, config *RateLimiterConfig) (*AdaptiveRateLimiter, error) {
    rdb := redis.NewClient(&redis.Options{
        Addr:     redisAddr,
        Password: "", // из secrets
        DB:       0,
    })

    limiter := &AdaptiveRateLimiter{
        redis:     rdb,
        limits:    make(map[string]*RateLimit),
        blocklist: &sync.Map{},
        config:    config,
    }

    go limiter.cleanupLoop()
    return limiter, nil
}

func (rl *AdaptiveRateLimiter) Middleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        key := rl.getKey(c)
        
        // Проверка блокировки
        if rl.isBlocked(key) {
            c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
                "error": "Rate limit exceeded. Try again later.",
                "code":  "RATE_LIMITED",
                "retry_after": rl.getBlockTime(key).String(),
            })
            return
        }

        // Получаем или создаем лимитер
        limit := rl.getLimiter(key)
        
        if !limit.limiter.Allow() {
            limit.burstCount++
            
            // Адаптивная блокировка при превышении
            if rl.config.EnableAdaptive && limit.burstCount > rl.config.Burst*2 {
                rl.block(key, rl.config.BlockDuration*2)
            }
            
            c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
                "error": "Too many requests",
                "code":  "RATE_LIMIT_EXCEEDED",
            })
            return
        }

        // Сбрасываем burst счетчик при успешном запросе
        limit.burstCount = 0
        limit.lastSeen = time.Now()

        c.Next()
    }
}

func (rl *AdaptiveRateLimiter) getKey(c *gin.Context) string {
    // Комбинация IP + User-Agent + Path для точной идентификации
    ip := c.ClientIP()
    ua := c.GetHeader("User-Agent")
    path := c.FullPath()
    
    return fmt.Sprintf("%s:%s:%s", ip, ua, path)
}

func (rl *AdaptiveRateLimiter) getLimiter(key string) *RateLimit {
    rl.mu.Lock()
    defer rl.mu.Unlock()

    limit, exists := rl.limits[key]
    if !exists {
        limit = &RateLimit{
            limiter: rate.NewLimiter(rate.Limit(rl.config.RequestsPerSecond), rl.config.Burst),
            lastSeen: time.Now(),
        }
        rl.limits[key] = limit
    }
    
    return limit
}

func (rl *AdaptiveRateLimiter) isBlocked(key string) bool {
    if val, ok := rl.blocklist.Load(key); ok {
        if blockTime, ok := val.(time.Time); ok {
            if time.Now().Before(blockTime) {
                return true
            }
            rl.blocklist.Delete(key)
        }
    }
    return false
}

func (rl *AdaptiveRateLimiter) block(key string, duration time.Duration) {
    rl.blocklist.Store(key, time.Now().Add(duration))
    
    // Логируем блокировку
    AuditLog("IP_BLOCKED", "rate_limiter", map[string]interface{}{
        "key":      key,
        "duration": duration.String(),
    })
}

func (rl *AdaptiveRateLimiter) getBlockTime(key string) time.Duration {
    if val, ok := rl.blocklist.Load(key); ok {
        if blockTime, ok := val.(time.Time); ok {
            return time.Until(blockTime)
        }
    }
    return 0
}

func (rl *AdaptiveRateLimiter) cleanupLoop() {
    ticker := time.NewTicker(rl.config.CleanupInterval)
    for range ticker.C {
        rl.mu.Lock()
        for key, limit := range rl.limits {
            if time.Since(limit.lastSeen) > time.Hour {
                delete(rl.limits, key)
            }
        }
        rl.mu.Unlock()
    }
}

// Метод для белых списков (доверенные IP)
func (rl *AdaptiveRateLimiter) AddToWhitelist(ip string) {
    rl.blocklist.Store("whitelist:"+ip, true)
}

func (rl *AdaptiveRateLimiter) IsWhitelisted(ip string) bool {
    _, ok := rl.blocklist.Load("whitelist:" + ip)
    return ok
}
