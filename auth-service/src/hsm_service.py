"""
HelixFlow HSM Integration for Key Management
Hardware Security Module integration for encryption keys
"""

import os
import base64
from typing import Optional, Dict, Any
from cryptography.hazmat.primitives import serialization, hashes
from cryptography.hazmat.primitives.asymmetric import rsa, padding
from cryptography.hazmat.backends import default_backend
import logging


class HSMService:
    def __init__(self):
        self.hsm_endpoint = os.getenv("HSM_ENDPOINT", "http://hsm-service:8080")
        self.logger = logging.getLogger(__name__)

        # Initialize with software fallback if HSM not available
        self.private_key = None
        self.public_key = None
        self._load_or_generate_keys()

    def _load_or_generate_keys(self):
        """Load keys from HSM or generate new ones"""
        try:
            # Try to load from HSM first
            self._load_from_hsm()
        except Exception as e:
            self.logger.warning(f"HSM not available, using software fallback: {e}")
            self._generate_software_keys()

    def _load_from_hsm(self):
        """Load keys from Hardware Security Module"""
        # This would integrate with actual HSM API
        # For now, simulate HSM key loading
        key_id = "helixflow-jwt-key"

        # In real implementation, this would call HSM API:
        # response = requests.get(f"{self.hsm_endpoint}/keys/{key_id}")
        # self.private_key = response.json()['private_key']
        # self.public_key = response.json()['public_key']

        raise Exception("HSM not configured - using software fallback")

    def _generate_software_keys(self):
        """Generate RSA key pair for software fallback"""
        self.logger.info("Generating RSA key pair for JWT signing")

        # Generate RSA private key
        private_key = rsa.generate_private_key(
            public_exponent=65537, key_size=2048, backend=default_backend()
        )

        # Get public key
        public_key = private_key.public_key()

        # Serialize private key
        self.private_key = private_key.private_bytes(
            encoding=serialization.Encoding.PEM,
            format=serialization.PrivateFormat.PKCS8,
            encryption_algorithm=serialization.NoEncryption(),
        ).decode()

        # Serialize public key
        self.public_key = public_key.public_bytes(
            encoding=serialization.Encoding.PEM,
            format=serialization.PublicFormat.SubjectPublicKeyInfo,
        ).decode()

    def get_private_key(self) -> str:
        """Get private key for JWT signing"""
        return self.private_key

    def get_public_key(self) -> str:
        """Get public key for JWT verification"""
        return self.public_key

    def encrypt_data(self, data: str, key_id: str = "default") -> str:
        """Encrypt data using HSM"""
        try:
            # In real implementation, send to HSM for encryption
            # For now, use software encryption
            return self._software_encrypt(data)
        except Exception as e:
            self.logger.error(f"HSM encryption failed: {e}")
            return self._software_encrypt(data)

    def decrypt_data(self, encrypted_data: str, key_id: str = "default") -> str:
        """Decrypt data using HSM"""
        try:
            # In real implementation, send to HSM for decryption
            # For now, use software decryption
            return self._software_decrypt(encrypted_data)
        except Exception as e:
            self.logger.error(f"HSM decryption failed: {e}")
            return self._software_decrypt(encrypted_data)

    def _software_encrypt(self, data: str) -> str:
        """Software-based encryption fallback"""
        # Load encryption key (in real implementation, this would be from HSM)
        key = b"helixflow-encryption-key-32bytes!"  # 32 bytes for AES-256

        from cryptography.hazmat.primitives.ciphers import Cipher, algorithms, modes
        import os

        # Generate random IV
        iv = os.urandom(16)

        # Create cipher
        cipher = Cipher(algorithms.AES(key), modes.CBC(iv), backend=default_backend())
        encryptor = cipher.encryptor()

        # Pad data to block size
        block_size = 16
        padded_data = data.encode() + b"\0" * (
            block_size - len(data.encode()) % block_size
        )

        # Encrypt
        encrypted = encryptor.update(padded_data) + encryptor.finalize()

        # Return base64 encoded iv + encrypted data
        return base64.b64encode(iv + encrypted).decode()

    def _software_decrypt(self, encrypted_data: str) -> str:
        """Software-based decryption fallback"""
        key = b"helixflow-encryption-key-32bytes!"  # 32 bytes for AES-256

        from cryptography.hazmat.primitives.ciphers import Cipher, algorithms, modes

        # Decode from base64
        decoded = base64.b64decode(encrypted_data)
        iv = decoded[:16]
        encrypted = decoded[16:]

        # Create cipher
        cipher = Cipher(algorithms.AES(key), modes.CBC(iv), backend=default_backend())
        decryptor = cipher.decryptor()

        # Decrypt
        decrypted_padded = decryptor.update(encrypted) + decryptor.finalize()

        # Remove padding
        return decrypted_padded.rstrip(b"\0").decode()

    def sign_data(self, data: str) -> str:
        """Sign data using HSM private key"""
        try:
            # In real implementation, send to HSM for signing
            # For now, use software signing
            return self._software_sign(data)
        except Exception as e:
            self.logger.error(f"HSM signing failed: {e}")
            return self._software_sign(data)

    def _software_sign(self, data: str) -> str:
        """Software-based signing fallback"""
        from cryptography.hazmat.primitives import serialization

        # Load private key
        private_key = serialization.load_pem_private_key(
            self.private_key.encode(), password=None, backend=default_backend()
        )

        # Sign data
        signature = private_key.sign(
            data.encode(),
            padding.PSS(
                mgf=padding.MGF1(hashes.SHA256()), salt_length=padding.PSS.MAX_LENGTH
            ),
            hashes.SHA256(),
        )

        return base64.b64encode(signature).decode()

    def verify_signature(self, data: str, signature: str) -> bool:
        """Verify signature using HSM public key"""
        try:
            # In real implementation, send to HSM for verification
            # For now, use software verification
            return self._software_verify(data, signature)
        except Exception as e:
            self.logger.error(f"HSM verification failed: {e}")
            return self._software_verify(data, signature)

    def _software_verify(self, data: str, signature: str) -> bool:
        """Software-based signature verification fallback"""
        from cryptography.hazmat.primitives import serialization

        try:
            # Load public key
            public_key = serialization.load_pem_public_key(
                self.public_key.encode(), backend=default_backend()
            )

            # Decode signature
            signature_bytes = base64.b64decode(signature)

            # Verify signature
            public_key.verify(
                signature_bytes,
                data.encode(),
                padding.PSS(
                    mgf=padding.MGF1(hashes.SHA256()),
                    salt_length=padding.PSS.MAX_LENGTH,
                ),
                hashes.SHA256(),
            )
            return True
        except Exception:
            return False
