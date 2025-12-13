"""
HelixFlow API Gateway with TLS 1.3 termination
"""

from flask import Flask, request, jsonify
import ssl
import logging
from datetime import datetime, timedelta
import jwt
from hsm_service import HSMService
from rbac_service import RBACService

app = Flask(__name__)

# Initialize services
hsm = HSMService()
rbac = RBACService()

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


@app.before_request
def log_request_info():
    logger.info(f"{request.method} {request.url} - {request.remote_addr}")


@app.route("/health")
def health_check():
    """Health check endpoint"""
    return jsonify(
        {
            "status": "healthy",
            "timestamp": datetime.utcnow().isoformat(),
            "service": "api-gateway",
        }
    )


@app.route("/v1/chat/completions", methods=["POST"])
def chat_completions():
    """OpenAI-compatible chat completions endpoint"""
    try:
        # Authenticate request
        auth_header = request.headers.get("Authorization")
        if not auth_header or not auth_header.startswith("Bearer "):
            return jsonify({"error": "Missing or invalid authorization header"}), 401

        token = auth_header.split(" ")[1]

        # Validate JWT token
        payload = hsm.validate_token(token)
        if not payload:
            return jsonify({"error": "Invalid or expired token"}), 401

        user_id = payload["sub"]

        # Check permissions
        if not rbac.check_permission(user_id, rbac.Permission.MODEL_INFERENCE):
            return jsonify({"error": "Insufficient permissions"}), 403

        # Parse request
        data = request.get_json()
        if not data:
            return jsonify({"error": "Invalid JSON payload"}), 400

        model = data.get("model")
        messages = data.get("messages", [])
        max_tokens = data.get("max_tokens", 100)
        stream = data.get("stream", False)

        if not model or not messages:
            return jsonify({"error": "Missing required fields: model, messages"}), 400

        if stream:
            # Return streaming response
            return _stream_response(model, messages, max_tokens, user_id)
        else:
            # Route to inference service (placeholder)
            # In real implementation, this would forward to inference-pool
            response = {
                "id": f"chatcmpl-{datetime.utcnow().timestamp()}",
                "object": "chat.completion",
                "created": int(datetime.utcnow().timestamp()),
                "model": model,
                "choices": [
                    {
                        "index": 0,
                        "message": {
                            "role": "assistant",
                            "content": f"This is a simulated response for model {model}",
                        },
                        "finish_reason": "stop",
                    }
                ],
                "usage": {
                    "prompt_tokens": len(str(messages)),
                    "completion_tokens": 50,
                    "total_tokens": len(str(messages)) + 50,
                },
            }

            return jsonify(response)

    except Exception as e:
        logger.error(f"Error in chat completions: {e}")
        return jsonify({"error": "Internal server error"}), 500


def _stream_response(model, messages, max_tokens, user_id):
    """Generate streaming response for chat completions."""
    from flask import Response
    import json

    def generate():
        # Send initial chunk
        chunk_id = f"chatcmpl-{datetime.utcnow().timestamp()}"
        initial_chunk = {
            "id": chunk_id,
            "object": "chat.completion.chunk",
            "created": int(datetime.utcnow().timestamp()),
            "model": model,
            "choices": [
                {"index": 0, "delta": {"role": "assistant"}, "finish_reason": None}
            ],
        }
        yield f"data: {json.dumps(initial_chunk)}\n\n"

        # Simulate streaming content
        content = f"This is a streaming response for model {model}. "
        words = content.split()

        for i, word in enumerate(words):
            chunk = {
                "id": chunk_id,
                "object": "chat.completion.chunk",
                "created": int(datetime.utcnow().timestamp()),
                "model": model,
                "choices": [
                    {
                        "index": 0,
                        "delta": {"content": word + " "},
                        "finish_reason": None,
                    }
                ],
            }
            yield f"data: {json.dumps(chunk)}\n\n"

        # Final chunk
        final_chunk = {
            "id": chunk_id,
            "object": "chat.completion.chunk",
            "created": int(datetime.utcnow().timestamp()),
            "model": model,
            "choices": [{"index": 0, "delta": {}, "finish_reason": "stop"}],
        }
        yield f"data: {json.dumps(final_chunk)}\n\n"
        yield "data: [DONE]\n\n"

    return Response(generate(), mimetype="text/plain")


@app.route("/v1/models", methods=["GET"])
def list_models():
    """List available AI models"""
    try:
        # Authenticate request
        auth_header = request.headers.get("Authorization")
        if auth_header and auth_header.startswith("Bearer "):
            token = auth_header.split(" ")[1]
            payload = hsm.validate_token(token)
            if payload:
                user_id = payload["sub"]
                # Check if user can list models
                if not rbac.check_permission(user_id, rbac.Permission.MODEL_LIST):
                    return jsonify({"error": "Insufficient permissions"}), 403

        # Return available models (placeholder)
        models = [
            {
                "id": "gpt-4",
                "object": "model",
                "created": 1677649963,
                "owned_by": "openai",
            },
            {
                "id": "claude-3-sonnet",
                "object": "model",
                "created": 1677649963,
                "owned_by": "anthropic",
            },
            {
                "id": "deepseek-chat",
                "object": "model",
                "created": 1677649963,
                "owned_by": "deepseek",
            },
        ]

        return jsonify({"object": "list", "data": models})

    except Exception as e:
        logger.error(f"Error listing models: {e}")
        return jsonify({"error": "Internal server error"}), 500


if __name__ == "__main__":
    # SSL context for TLS 1.3
    context = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
    context.minimum_version = ssl.TLSVersion.TLSv1_3
    context.maximum_version = ssl.TLSVersion.TLSv1_3

    # Load certificates
    context.load_cert_chain("/certs/api-gateway.crt", "/certs/api-gateway.key")
    context.load_verify_locations("/certs/ca.crt")

    # Require client certificates for mTLS
    context.verify_mode = ssl.CERT_REQUIRED
    context.check_hostname = False

    logger.info("Starting API Gateway with TLS 1.3...")
    app.run(host="0.0.0.0", port=8443, ssl_context=context, debug=False)
