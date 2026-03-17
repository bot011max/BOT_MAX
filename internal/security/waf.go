// internal/security/waf.go
package security

import (
    "encoding/json"
    "net/http"
    "regexp"
    "strings"
    "sync"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/hashicorp/golang-lru"
)

type WAFConfig struct {
    EnableSQLInjection   bool
    EnableXSS           bool
    EnablePathTraversal bool
    EnableCommandInjection bool
    EnableScannerDetection bool
    BlockThreshold      int
    RateLimit           int
}

type WAFMiddleware struct {
    config     WAFConfig
    patterns   map[string]*regexp.Regexp
    blacklist  *lru.Cache
    attacks    sync.Map
}

func NewWAFMiddleware(config WAFConfig) (*WAFMiddleware, error) {
    cache, _ := lru.New(10000)
    
    waf := &WAFMiddleware{
        config:    config,
        patterns:  make(map[string]*regexp.Regexp),
        blacklist: cache,
    }

    // Компилируем паттерны атак
    if config.EnableSQLInjection {
        waf.patterns["sql"] = regexp.MustCompile(`(?i)(union.*select|select.*from|insert.*into|update.*set|delete.*from|drop.*table|exec.*xp_cmdshell|;.*--|/\*.*\*/)`)
    }
    
    if config.EnableXSS {
        waf.patterns["xss"] = regexp.MustCompile(`(?i)(<script|javascript:|onerror=|onload=|alert\(|prompt\(|confirm\(|eval\(|document\.cookie)`)
    }
    
    if config.EnablePathTraversal {
        waf.patterns["path"] = regexp.MustCompile(`(\.\./|\.\.\\|/etc/passwd|/windows/win.ini)`)
    }
    
    if config.EnableCommandInjection {
        waf.patterns["cmd"] = regexp.MustCompile(`(?i)(;|\||\` + "`" + `|\$\{|\(\)\s*\{|:\s*\)\s*\{|\&\&|\|\||wget|curl|nc|bash|sh|python|perl|php|ruby)`)
    }

    return waf, nil
}

func (w *WAFMiddleware) Handler() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Проверка IP в черном списке
        ip := c.ClientIP()
        if w.isBlacklisted(ip) {
            c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
                "error": "Access denied",
                "code":  "BLACKLISTED",
            })
            return
        }

        // Проверка на сканеры
        if w.config.EnableScannerDetection {
            if w.isScanner(c.Request) {
                w.blacklist.Add(ip, time.Now())
                c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
                    "error": "Scanner detected",
                    "code":  "SCANNER",
                })
                return
            }
        }

        // Проверка всех входных данных
        if err := w.inspectRequest(c); err != nil {
            w.recordAttack(ip, err.Error())
            
            if w.getAttackCount(ip) > w.config.BlockThreshold {
                w.blacklist.Add(ip, time.Now())
            }
            
            c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
                "error": "Malicious request detected",
                "code":  "WAF_BLOCK",
            })
            return
        }

        c.Next()
    }
}

func (w *WAFMiddleware) inspectRequest(c *gin.Context) error {
    // Проверка URL и параметров
    if err := w.inspectString(c.Request.URL.String()); err != nil {
        return err
    }

    // Проверка query параметров
    for key, values := range c.Request.URL.Query() {
        for _, value := range values {
            if err := w.inspectString(key); err != nil {
                return err
            }
            if err := w.inspectString(value); err != nil {
                return err
            }
        }
    }

    // Проверка заголовков
    for key, values := range c.Request.Header {
        for _, value := range values {
            if err := w.inspectString(key); err != nil {
                return err
            }
            if err := w.inspectString(value); err != nil {
                return err
            }
        }
    }

    // Проверка тела запроса
    if c.Request.Body != nil {
        body := make(map[string]interface{})
        if err := c.ShouldBindJSON(&body); err == nil {
            if err := w.inspectJSON(body); err != nil {
                return err
            }
        }
    }

    return nil
}

func (w *WAFMiddleware) inspectJSON(data interface{}) error {
    switch v := data.(type) {
    case string:
        return w.inspectString(v)
    case map[string]interface{}:
        for _, val := range v {
            if err := w.inspectJSON(val); err != nil {
                return err
            }
        }
    case []interface{}:
        for _, item := range v {
            if err := w.inspectJSON(item); err != nil {
                return err
            }
        }
    }
    return nil
}

func (w *WAFMiddleware) inspectString(s string) error {
    for name, pattern := range w.patterns {
        if pattern.MatchString(s) {
            return &SecurityError{
                Type:    name,
                Message: "Malicious pattern detected",
            }
        }
    }
    return nil
}

func (w *WAFMiddleware) isScanner(r *http.Request) bool {
    // Проверка User-Agent
    ua := r.UserAgent()
    scannerPatterns := []string{
        "nmap", "sqlmap", "nikto", "gobuster", "dirb", 
        "wfuzz", "burp", "zap", "nessus", "openvas",
    }
    
    for _, pattern := range scannerPatterns {
        if strings.Contains(strings.ToLower(ua), pattern) {
            return true
        }
    }

    // Проверка необычных методов
    if r.Method == "OPTIONS" || r.Method == "TRACE" || r.Method == "CONNECT" {
        return true
    }

    return false
}

func (w *WAFMiddleware) isBlacklisted(ip string) bool {
    if val, ok := w.blacklist.Get(ip); ok {
        if t, ok := val.(time.Time); ok {
            if time.Since(t) < 24*time.Hour {
                return true
            }
            w.blacklist.Remove(ip)
        }
    }
    return false
}

func (w *WAFMiddleware) recordAttack(ip, attackType string) {
    key := ip + ":" + attackType
    count, _ := w.attacks.LoadOrStore(key, 0)
    w.attacks.Store(key, count.(int)+1)
}

func (w *WAFMiddleware) getAttackCount(ip string) int {
    total := 0
    w.attacks.Range(func(key, value interface{}) bool {
        if strings.HasPrefix(key.(string), ip) {
            total += value.(int)
        }
        return true
    })
    return total
}

type SecurityError struct {
    Type    string
    Message string
}

func (e *SecurityError) Error() string {
    return e.Message
}
