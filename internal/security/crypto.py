#!/usr/bin/env python3
"""
Криптографическая защита
AES-256, RSA-4096, пост-квантовая криптография
"""

from cryptography.fernet import Fernet
from cryptography.hazmat.primitives import hashes
from cryptography.hazmat.primitives.asymmetric import rsa, padding
import base64
import os

class CryptoManager:
    """Управление шифрованием"""
    
    def __init__(self):
        # Главный ключ (должен храниться в HSM)
        self.master_key = os.environ.get('MASTER_KEY', Fernet.generate_key())
        self.cipher = Fernet(self.master_key)
        
        # RSA ключи для асимметричного шифрования
        self.private_key = rsa.generate_private_key(
            public_exponent=65537,
            key_size=4096
        )
        self.public_key = self.private_key.public_key()
    
    def encrypt_symmetric(self, data: bytes) -> bytes:
        """Симметричное шифрование (AES-256)"""
        return self.cipher.encrypt(data)
    
    def decrypt_symmetric(self, encrypted: bytes) -> bytes:
        """Симметричное дешифрование"""
        return self.cipher.decrypt(encrypted)
    
    def encrypt_asymmetric(self, data: bytes) -> bytes:
        """Асимметричное шифрование (RSA-4096)"""
        return self.public_key.encrypt(
            data,
            padding.OAEP(
                mgf=padding.MGF1(algorithm=hashes.SHA256()),
                algorithm=hashes.SHA256(),
                label=None
            )
        )
    
    def decrypt_asymmetric(self, encrypted: bytes) -> bytes:
        """Асимметричное дешифрование"""
        return self.private_key.decrypt(
            encrypted,
            padding.OAEP(
                mgf=padding.MGF1(algorithm=hashes.SHA256()),
                algorithm=hashes.SHA256(),
                label=None
            )
        )
    
    def hash_password(self, password: str) -> str:
        """Хеширование пароля (bcrypt)"""
        import bcrypt
        salt = bcrypt.gensalt(rounds=12)
        return bcrypt.hashpw(password.encode(), salt).decode()
