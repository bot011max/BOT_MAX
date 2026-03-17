// internal/security/rate_limiter_advanced.go
package security

import (
    "context"
    "fmt"
    "net"
    "net/http"
    "strings"
    "sync"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/go-redis/redis/v8"
    "github.com/mssola/user_agent"
    "golang.org/x/time/rate"
)

type AdvancedRateLimiter struct {
    redis      *redis.Client
    limits     map[string]*UserLimit
    mu         sync.RWMutex
    config     *AdvancedConfig
    blacklist  *sync.Map
    whitelist  *sync.Map
}

type AdvancedConfig struct {
    RequestsPerSecond float64
    Burst             int
    BlockDuration     time.Duration
    CleanupInterval   time.Duration
    EnableMachineLearning bool
    EnableGeoIP       bool
    EnableBehavioral  bool
    TorBlock          bool
    VPNBlock          bool
    DatacenterBlock   bool
}

type UserLimit struct {
    limiter     *rate.Limiter
    lastSeen    time.Time
    violations  int
    behavior    *BehaviorProfile
    geoInfo     *GeoInfo
}

type BehaviorProfile struct {
    AvgRequestInterval time.Duration
    RequestPattern     []time.Time
    SuspiciousScore    float64
    LastPath           string
    PathTransition     int
}

type GeoInfo struct {
    Country  string
    City     string
    ISP      string
    IsTor    bool
    IsVPN    bool
    IsProxy  bool
    IsDatacenter bool
}

func NewAdvancedRateLimiter(redisAddr string, config *AdvancedConfig) (*AdvancedRateLimiter, error) {
    rdb := redis.NewClient(&redis.Options{
        Addr:         redisAddr,
        Password:     "",
        DB:           0,
        PoolSize:     100,
        MinIdleConns: 10,
    })

    // Проверка подключения
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    if err := rdb.Ping(ctx).Err(); err != nil {
        return nil, fmt.Errorf("redis connection failed: %w", err)
    }

    limiter := &AdvancedRateLimiter{
        redis:      rdb,
        limits:     make(map[string]*UserLimit),
        config:     config,
        blacklist:  &sync.Map{},
        whitelist:  &sync.Map{},
    }

    go limiter.cleanupLoop()
    go limiter.syncWithRedis()
    
    return limiter, nil
}

func (rl *AdvancedRateLimiter) Middleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        clientIP := c.ClientIP()
        
        // Белый список
        if rl.isWhitelisted(clientIP) {
            c.Next()
            return
        }

        // Черный список
        if rl.isBlacklisted(clientIP) {
            rl.blockResponse(c, "IP is blacklisted")
            return
        }

        // Проверка Tor/VPN
        if rl.config.TorBlock || rl.config.VPNBlock || rl.config.DatacenterBlock {
            if geoInfo := rl.getGeoInfo(clientIP); geoInfo != nil {
                if rl.config.TorBlock && geoInfo.IsTor {
                    rl.blockResponse(c, "Tor exit nodes are not allowed")
                    return
                }
                if rl.config.VPNBlock && geoInfo.IsVPN {
                    rl.blockResponse(c, "VPN/proxy detected")
                    return
                }
                if rl.config.DatacenterBlock && geoInfo.IsDatacenter {
                    rl.blockResponse(c, "Datacenter IPs are not allowed")
                    return
                }
            }
        }

        // Получаем или создаем лимит
        limit := rl.getLimit(clientIP)
        
        // Поведенческий анализ
        if rl.config.EnableBehavioral {
            if score := rl.analyzeBehavior(c, limit); score > 0.8 {
                rl.blacklist.Store(clientIP, time.Now().Add(rl.config.BlockDuration))
                rl.blockResponse(c, "Suspicious behavior detected")
                return
            }
        }

        // Проверка лимитов
        if !limit.limiter.Allow() {
            limit.violations++
            
            // Адаптивная блокировка
            if limit.violations > rl.config.Burst*2 {
                rl.blacklist.Store(clientIP, time.Now().Add(rl.config.BlockDuration))
                SecurityAlert("RATE_LIMIT_BLOCK", map[string]interface{}{
                    "ip":         clientIP,
                    "violations": limit.violations,
                    "path":       c.FullPath(),
                })
            }
            
            rl.blockResponse(c, "Rate limit exceeded")
            return
        }

        // Обновляем статистику
        limit.lastSeen = time.Now()
        limit.violations = 0
        
        // Сохраняем в Redis для распределенных систем
        rl.updateRedisStats(clientIP, c)
        
        c.Next()
    }
}

func (rl *AdvancedRateLimiter) analyzeBehavior(c *gin.Context, limit *UserLimit) float64 {
    now := time.Now()
    path := c.FullPath()
    
    // Инициализация профиля
    if limit.behavior == nil {
        limit.behavior = &BehaviorProfile{
            RequestPattern: make([]time.Time, 0, 10),
        }
    }

    // Добавляем запрос в паттерн
    limit.behavior.RequestPattern = append(limit.behavior.RequestPattern, now)
    if len(limit.behavior.RequestPattern) > 10 {
        limit.behavior.RequestPattern = limit.behavior.RequestPattern[1:]
    }

    // Анализ интервалов
    var totalInterval time.Duration
    for i := 1; i < len(limit.behavior.RequestPattern); i++ {
        totalInterval += limit.behavior.RequestPattern[i].Sub(limit.behavior.RequestPattern[i-1])
    }
    
    if len(limit.behavior.RequestPattern) > 1 {
        avgInterval := totalInterval / time.Duration(len(limit.behavior.RequestPattern)-1)
        
        // Слишком частые запросы
        if avgInterval < 100*time.Millisecond {
            limit.behavior.SuspiciousScore += 0.3
        }
    }

    // Анализ путей
    if limit.behavior.LastPath != "" && limit.behavior.LastPath != path {
        limit.behavior.PathTransition++
        
        // Слишком быстрые переходы между разными путями
        if limit.behavior.PathTransition > 5 && len(limit.behavior.RequestPattern) < 10 {
            limit.behavior.SuspiciousScore += 0.2
        }
    }
    limit.behavior.LastPath = path

    // Проверка User-Agent
    ua := c.GetHeader("User-Agent")
    if ua == "" {
        limit.behavior.SuspiciousScore += 0.5
    } else {
        parsed := user_agent.New(ua)
        if parsed.Bot() {
            limit.behavior.SuspiciousScore += 0.7
        }
    }

    // Проверка заголовков
    if c.GetHeader("Accept") == "" || c.GetHeader("Accept-Language") == "" {
        limit.behavior.SuspiciousScore += 0.2
    }

    return limit.behavior.SuspiciousScore
}

func (rl *AdvancedRateLimiter) getGeoInfo(ip string) *GeoInfo {
    // Используем MaxMind GeoIP или аналоги
    // В production здесь должен быть реальный lookup
    
    // Пример для демо
    return &GeoInfo{
        Country:  "RU",
        IsTor:    false,
        IsVPN:    false,
        IsProxy:  false,
        IsDatacenter: false,
    }
}

func (rl *AdvancedRateLimiter) isBlacklisted(ip string) bool {
    if val, ok := rl.blacklist.Load(ip); ok {
        if until, ok := val.(time.Time); ok {
            if time.Now().Before(until) {
                return true
            }
            rl.blacklist.Delete(ip)
        }
    }
    return false
}

func (rl *AdvancedRateLimiter) isWhitelisted(ip string) bool {
    _, ok := rl.whitelist.Load(ip)
    return ok
}

func (rl *AdvancedRateLimiter) getLimit(ip string) *UserLimit {
    rl.mu.Lock()
    defer rl.mu.Unlock()

    limit, exists := rl.limits[ip]
    if !exists {
        limit = &UserLimit{
            limiter: rate.NewLimiter(rate.Limit(rl.config.RequestsPerSecond), rl.config.Burst),
            lastSeen: time.Now(),
        }
        rl.limits[ip] = limit
    }
    
    return limit
}

func (rl *AdvancedRateLimiter) updateRedisStats(ip string, c *gin.Context) {
    ctx := context.Background()
    key := fmt.Sprintf("ratelimit:%s", ip)
    
    pipe := rl.redis.Pipeline()
    pipe.HIncrBy(ctx, key, "requests", 1)
    pipe.HSet(ctx, key, "last_seen", time.Now().Unix())
    pipe.HSet(ctx, key, "last_path", c.FullPath())
    pipe.HSet(ctx, key, "user_agent", c.GetHeader("User-Agent"))
    pipe.Expire(ctx, key, 24*time.Hour)
    pipe.Exec(ctx)
}

func (rl *AdvancedRateLimiter) blockResponse(c *gin.Context, message string) {
    c.Header("Retry-After", fmt.Sprintf("%d", int(rl.config.BlockDuration.Seconds())))
    c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
        "error": message,
        "code":  "RATE_LIMITED",
    })
}

func (rl *AdvancedRateLimiter) cleanupLoop() {
    ticker := time.NewTicker(rl.config.CleanupInterval)
    for range ticker.C {
        rl.mu.Lock()
        for ip, limit := range rl.limits {
            if time.Since(limit.lastSeen) > 24*time.Hour {
                delete(rl.limits, ip)
            }
        }
        rl.mu.Unlock()
    }
}

func (rl *AdvancedRateLimiter) syncWithRedis() {
    ticker := time.NewTicker(5 * time.Minute)
    for range ticker.C {
        ctx := context.Background()
        
        // Синхронизация черного списка с Redis
        blacklistKey := "global:blacklist"
        blacklisted, err := rl.redis.SMembers(ctx, blacklistKey).Result()
        if err == nil {
            for _, ip := range blacklisted {
                rl.blacklist.Store(ip, time.Now().Add(rl.config.BlockDuration))
            }
        }
    }
}

// Логин middleware с дополнительной защитой
func (rl *AdvancedRateLimiter) LoginMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        ip := c.ClientIP()
        key := fmt.Sprintf("login:attempts:%s", ip)
        
        ctx := context.Background()
        
        // Проверка количества попыток
        attempts, err := rl.redis.Get(ctx, key).Int()
        if err == nil && attempts >= 5 {
            c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
                "error": "Too many login attempts. Try again later.",
                "code":  "LOGIN_LIMITED",
            })
            return
        }

        c.Next()

        // Если ошибка аутентификации - увеличиваем счетчик
        if c.Writer.Status() == http.StatusUnauthorized {
            pipe := rl.redis.Pipeline()
            pipe.Incr(ctx, key)
            pipe.Expire(ctx, key, 15*time.Minute)
            pipe.Exec(ctx)
        }
    }
}
