"""HSM Service emulation for key management."""

import base64
import os
from cryptography.hazmat.primitives.ciphers.aead import AESGCM


class HSMService:
    """Provides hardware security module emulation for key management."""

    def __init__(self):
        self.master_key = AESGCM.generate_key(bit_length=256)

    def encrypt_data(self, plaintext: str) -> str:
        """Encrypt plaintext using AES-GCM."""
        aesgcm = AESGCM(self.master_key)
        nonce = os.urandom(12)
        ciphertext = aesgcm.encrypt(nonce, plaintext.encode(), None)
        return base64.b64encode(nonce + ciphertext).decode()

    def decrypt_data(self, ciphertext: str) -> str:
        """Decrypt ciphertext using AES-GCM."""
        aesgcm = AESGCM(self.master_key)
        data = base64.b64decode(ciphertext.encode())
        nonce = data[:12]
        ct = data[12:]
        plaintext = aesgcm.decrypt(nonce, ct, None)
        return plaintext.decode()
