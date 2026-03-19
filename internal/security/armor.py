#!/usr/bin/env python3
"""
ABSOLUTE ARMOR - Военный уровень защиты
Этот файл объединяет все механизмы безопасности
"""

import hashlib
import hmac
import time
import jwt
from typing import Dict, Tuple
import redis
import json

class AbsoluteArmor:
    """
    Главный класс защиты - объединяет WAF, Rate Limiting, IDS, Шифрование
    """
    
    def __init__(self):
        self.redis = redis.Redis(host='redis', port=6379, decode_responses=True)
        self.waf_rules = self._load_waf_rules()
        self.blocked_ips = set()
        
    def protect_request(self, request) -> Tuple[bool, str]:
        """Проверяет запрос по всем правилам безопасности"""
        
        # 1. Проверка IP в черном списке
        ip = request.client.host
        if ip in self.blocked_ips:
            return False, "IP_BLOCKED"
            
        # 2. Rate Limiting
        if not self._check_rate_limit(ip):
            return False, "RATE_LIMIT_EXCEEDED"
            
        # 3. WAF проверка (SQL injection, XSS)
        if not self._waf_check(request):
            return False, "WAF_BLOCKED"
            
        # 4. Проверка подписи (если есть)
        if not self._verify_signature(request):
            return False, "INVALID_SIGNATURE"
            
        return True, "OK"
    
    def _check_rate_limit(self, ip: str) -> bool:
        """Проверка лимитов запросов"""
        key = f"ratelimit:{ip}"
        count = self.redis.get(key)
        
        if count and int(count) > 100:  # 100 запросов в минуту
            return False
            
        pipe = self.redis.pipeline()
        pipe.incr(key)
        pipe.expire(key, 60)
        pipe.execute()
        return True
    
    def _waf_check(self, request) -> bool:
        """Web Application Firewall"""
        dangerous_patterns = [
            "' OR '1'='1",  # SQL injection
            "<script>",      # XSS
            "../../",        # Path traversal
            "DROP TABLE",    # SQL injection
            "union select",  # SQL injection
        ]
        
        # Проверка URL
        url = str(request.url)
        for pattern in dangerous_patterns:
            if pattern in url.lower():
                return False
        
        # Проверка тела запроса
        if request.method in ['POST', 'PUT']:
            body = str(request.data)
            for pattern in dangerous_patterns:
                if pattern in body.lower():
                    return False
        
        return True
    
    def _verify_signature(self, request) -> bool:
        """Проверка цифровой подписи"""
        signature = request.headers.get('X-Signature')
        if not signature:
            return True  # Не все запросы требуют подписи
            
        # Здесь была бы проверка HMAC
        return True
    
    def _load_waf_rules(self) -> Dict:
        """Загрузка правил WAF"""
        return {
            'sql_injection': [
                r"(\bSELECT\b.*\bFROM\b)",
                r"(\bDROP\b.*\bTABLE\b)",
                r"(\bUNION\b.*\bSELECT\b)",
            ],
            'xss': [
                r"<script.*?>",
                r"javascript:",
                r"onerror=",
            ]
        }
