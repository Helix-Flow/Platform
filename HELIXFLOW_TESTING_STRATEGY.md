# HELIXFLOW COMPREHENSIVE TESTING STRATEGY

## EXECUTIVE SUMMARY

This document outlines the complete testing strategy for the HelixFlow AI inference platform, covering all 6 test types with 100% coverage targets, automated testing frameworks, and continuous quality assurance processes.

**Test Types Covered:**
1. Unit Tests - 100% code coverage
2. Integration Tests - All service interactions
3. Contract Tests - API specification validation
4. Security Tests - Vulnerability and penetration testing
5. Performance Tests - Load, stress, and scalability testing
6. Chaos Tests - Resilience and failure recovery testing

**Coverage Target:** 100% for all critical paths  
**Automation Level:** 95%+ automated  
**Execution Frequency:** Continuous integration  

---

## TEST ARCHITECTURE OVERVIEW

### Test Pyramid Structure
```
                    /\
                   /  \  E2E Tests (5%)
                  /----\
                 /      \  Integration Tests (15%)
                /--------\
               /          \  Unit Tests (80%)
              /____________\
```

### Test Environment Architecture
```
Development → Unit Tests → Integration Tests → Staging → Production
     ↓              ↓              ↓              ↓         ↓
   Local         CI Pipeline     Contract        Security  Monitoring
   Testing       (Automated)     Validation      Testing   & Alerts
```

---

## 1. UNIT TESTS (100% COVERAGE)

### Go Services Testing
**Framework:** Go's built-in testing + Testify
**Coverage Target:** 100% line coverage

#### API Gateway Unit Tests
```go
// api-gateway/src/handlers/chat_test.go
package handlers

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

type MockAuthService struct {
    mock.Mock
}

func (m *MockAuthService) ValidateToken(token string) (*User, error) {
    args := m.Called(token)
    return args.Get(0).(*User), args.Error(1)
}

func TestChatCompletionHandler(t *testing.T) {
    tests := []struct {
        name           string
        requestBody    interface{}
        authToken      string
        expectedStatus int
        expectedError  string
    }{
        {
            name: "Valid request with authentication",
            requestBody: ChatCompletionRequest{
                Model: "gpt-3.5-turbo",
                Messages: []Message{
                    {Role: "user", Content: "Hello"},
                },
            },
            authToken:      "valid-token",
            expectedStatus: http.StatusOK,
            expectedError:  "",
        },
        {
            name: "Missing authentication token",
            requestBody: ChatCompletionRequest{
                Model: "gpt-3.5-turbo",
                Messages: []Message{
                    {Role: "user", Content: "Hello"},
                },
            },
            authToken:      "",
            expectedStatus: http.StatusUnauthorized,
            expectedError:  "Missing authentication token",
        },
        {
            name: "Invalid model specified",
            requestBody: ChatCompletionRequest{
                Model: "invalid-model",
                Messages: []Message{
                    {Role: "user", Content: "Hello"},
                },
            },
            authToken:      "valid-token",
            expectedStatus: http.StatusBadRequest,
            expectedError:  "Invalid model specified",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Setup
            mockAuth := new(MockAuthService)
            if tt.authToken != "" {
                mockAuth.On("ValidateToken", tt.authToken).Return(&User{ID: "123", Email: "test@example.com"}, nil)
            }
            
            handler := NewChatHandler(mockAuth)
            
            // Create request
            body, _ := json.Marshal(tt.requestBody)
            req := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewBuffer(body))
            if tt.authToken != "" {
                req.Header.Set("Authorization", "Bearer "+tt.authToken)
            }
            
            // Execute
            rr := httptest.NewRecorder()
            handler.ServeHTTP(rr, req)
            
            // Assert
            assert.Equal(t, tt.expectedStatus, rr.Code)
            if tt.expectedError != "" {
                var errorResp ErrorResponse
                json.NewDecoder(rr.Body).Decode(&errorResp)
                assert.Contains(t, errorResp.Error.Message, tt.expectedError)
            }
        })
    }
}

// Test edge cases and error conditions
func TestChatCompletionHandler_EdgeCases(t *testing.T) {
    tests := []struct {
        name        string
        requestBody string
        expectedError string
    }{
        {
            name:        "Empty request body",
            requestBody: "",
            expectedError: "Invalid request body",
        },
        {
            name:        "Malformed JSON",
            requestBody: "{invalid json",
            expectedError: "Invalid JSON",
        },
        {
            name:        "Empty messages array",
            requestBody: `{"model": "gpt-3.5-turbo", "messages": []}`,
            expectedError: "Messages array cannot be empty",
        },
        {
            name:        "Missing model field",
            requestBody: `{"messages": [{"role": "user", "content": "Hello"}]}`,
            expectedError: "Model field is required",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            handler := NewChatHandler(new(MockAuthService))
            
            req := httptest.NewRequest("POST", "/v1/chat/completions", bytes.NewBufferString(tt.requestBody))
            req.Header.Set("Authorization", "Bearer valid-token")
            
            rr := httptest.NewRecorder()
            handler.ServeHTTP(rr, req)
            
            assert.Equal(t, http.StatusBadRequest, rr.Code)
            var errorResp ErrorResponse
            json.NewDecoder(rr.Body).Decode(&errorResp)
            assert.Contains(t, errorResp.Error.Message, tt.expectedError)
        })
    }
}
```

#### Auth Service Unit Tests
```go
// auth-service/src/auth_test.go
package auth

import (
    "testing"
    "time"
    
    "github.com/golang-jwt/jwt/v5"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

type MockUserRepository struct {
    mock.Mock
}

func (m *MockUserRepository) FindByEmail(email string) (*User, error) {
    args := m.Called(email)
    return args.Get(0).(*User), args.Error(1)
}

func (m *MockUserRepository) Create(user *User) error {
    args := m.Called(user)
    return args.Error(0)
}

func TestAuthService_GenerateJWT(t *testing.T) {
    authService := NewAuthService("test-secret-key")
    
    user := &User{
        ID:    "123",
        Email: "test@example.com",
        Role:  "user",
    }
    
    token, err := authService.GenerateJWT(user)
    
    assert.NoError(t, err)
    assert.NotEmpty(t, token)
    
    // Verify token can be parsed
    parsedToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
        return []byte("test-secret-key"), nil
    })
    
    assert.NoError(t, err)
    assert.True(t, parsedToken.Valid)
}

func TestAuthService_ValidateJWT(t *testing.T) {
    tests := []struct {
        name          string
        token         string
        expectedValid bool
        expectedUser  *User
    }{
        {
            name:          "Valid token",
            token:         generateValidJWT(),
            expectedValid: true,
            expectedUser:  &User{ID: "123", Email: "test@example.com", Role: "user"},
        },
        {
            name:          "Invalid token",
            token:         "invalid-token",
            expectedValid: false,
            expectedUser:  nil,
        },
        {
            name:          "Expired token",
            token:         generateExpiredJWT(),
            expectedValid: false,
            expectedUser:  nil,
        },
        {
            name:          "Token with invalid signature",
            token:         generateJWTWithInvalidSignature(),
            expectedValid: false,
            expectedUser:  nil,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            authService := NewAuthService("test-secret-key")
            
            user, err := authService.ValidateJWT(tt.token)
            
            if tt.expectedValid {
                assert.NoError(t, err)
                assert.Equal(t, tt.expectedUser.ID, user.ID)
                assert.Equal(t, tt.expectedUser.Email, user.Email)
                assert.Equal(t, tt.expectedUser.Role, user.Role)
            } else {
                assert.Error(t, err)
                assert.Nil(t, user)
            }
        })
    }
}
```

#### Inference Pool Unit Tests
```go
// inference-pool/src/inference_test.go
package inference

import (
    "context"
    "testing"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

type MockModelProvider struct {
    mock.Mock
}

func (m *MockModelProvider) GenerateCompletion(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
    args := m.Called(ctx, req)
    return args.Get(0).(*CompletionResponse), args.Error(1)
}

func TestInferenceService_ProcessRequest(t *testing.T) {
    tests := []struct {
        name             string
        request          *InferenceRequest
        mockResponse     *CompletionResponse
        mockError        error
        expectedResponse *InferenceResponse
        expectedError    string
    }{
        {
            name: "Successful inference request",
            request: &InferenceRequest{
                Model: "gpt-3.5-turbo",
                Input: "Hello, world!",
            },
            mockResponse: &CompletionResponse{
                Text: "Hello! How can I help you today?",
                Usage: Usage{
                    PromptTokens:     5,
                    CompletionTokens: 8,
                    TotalTokens:      13,
                },
            },
            mockError: nil,
            expectedResponse: &InferenceResponse{
                Output: "Hello! How can I help you today?",
                Model:  "gpt-3.5-turbo",
                Usage: Usage{
                    PromptTokens:     5,
                    CompletionTokens: 8,
                    TotalTokens:      13,
                },
                Latency: 0, // Will be measured
            },
            expectedError: "",
        },
        {
            name: "Model not available",
            request: &InferenceRequest{
                Model: "invalid-model",
                Input: "Hello",
            },
            mockResponse:     nil,
            mockError:        ErrModelNotAvailable,
            expectedResponse: nil,
            expectedError:    "Model not available",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mockProvider := new(MockModelProvider)
            if tt.mockResponse != nil || tt.mockError != nil {
                mockProvider.On("GenerateCompletion", mock.Anything, mock.Anything).Return(tt.mockResponse, tt.mockError)
            }
            
            service := NewInferenceService(mockProvider)
            
            response, err := service.ProcessRequest(context.Background(), tt.request)
            
            if tt.expectedError == "" {
                assert.NoError(t, err)
                assert.NotNil(t, response)
                assert.Equal(t, tt.expectedResponse.Output, response.Output)
                assert.Equal(t, tt.expectedResponse.Model, response.Model)
                assert.Equal(t, tt.expectedResponse.Usage.TotalTokens, response.Usage.TotalTokens)
            } else {
                assert.Error(t, err)
                assert.Contains(t, err.Error(), tt.expectedError)
                assert.Nil(t, response)
            }
        })
    }
}
```

### Python SDK Testing
**Framework:** pytest + pytest-cov + pytest-mock
**Coverage Target:** 100% statement coverage

```python
# sdks/python/tests/test_client.py
import pytest
import responses
from unittest.mock import Mock, patch
from helixflow import Client, AuthenticationError, RateLimitError

class TestHelixFlowClient:
    
    @pytest.fixture
    def client(self):
        return Client(api_key="test-api-key")
    
    @pytest.fixture
    def mock_response(self):
        return {
            "choices": [{
                "message": {
                    "content": "Hello! How can I help you?",
                    "role": "assistant"
                }
            }],
            "usage": {
                "prompt_tokens": 5,
                "completion_tokens": 8,
                "total_tokens": 13
            }
        }
    
    @responses.activate
    def test_chat_completion_success(self, client, mock_response):
        responses.add(
            responses.POST,
            'https://api.helixflow.com/v1/chat/completions',
            json=mock_response,
            status=200
        )
        
        response = client.chat.completions.create(
            model="gpt-3.5-turbo",
            messages=[{"role": "user", "content": "Hello"}]
        )
        
        assert response.choices[0].message.content == "Hello! How can I help you?"
        assert response.usage.total_tokens == 13
        assert len(responses.calls) == 1
    
    @responses.activate
    def test_chat_completion_authentication_error(self, client):
        responses.add(
            responses.POST,
            'https://api.helixflow.com/v1/chat/completions',
            json={"error": {"message": "Invalid API key"}},
            status=401
        )
        
        with pytest.raises(AuthenticationError) as exc_info:
            client.chat.completions.create(
                model="gpt-3.5-turbo",
                messages=[{"role": "user", "content": "Hello"}]
            )
        
        assert "Invalid API key" in str(exc_info.value)
    
    @responses.activate
    def test_chat_completion_rate_limit_error(self, client):
        responses.add(
            responses.POST,
            'https://api.helixflow.com/v1/chat/completions',
            json={"error": {"message": "Rate limit exceeded"}},
            status=429
        )
        
        with pytest.raises(RateLimitError) as exc_info:
            client.chat.completions.create(
                model="gpt-3.5-turbo",
                messages=[{"role": "user", "content": "Hello"}]
            )
        
        assert "Rate limit exceeded" in str(exc_info.value)
    
    def test_client_initialization_with_invalid_api_key(self):
        with pytest.raises(ValueError):
            Client(api_key="")
    
    def test_client_initialization_without_api_key(self):
        with pytest.raises(ValueError):
            Client()
    
    @patch('helixflow.client.requests.post')
    def test_retry_mechanism(self, mock_post, client):
        # First call fails, second succeeds
        mock_post.side_effect = [
            Exception("Connection error"),
            Mock(status_code=200, json=lambda: {"choices": [{"message": {"content": "Success"}}]})
        ]
        
        response = client.chat.completions.create(
            model="gpt-3.5-turbo",
            messages=[{"role": "user", "content": "Hello"}]
        )
        
        assert response.choices[0].message.content == "Success"
        assert mock_post.call_count == 2
```

---

## 2. INTEGRATION TESTS

### Service-to-Service Integration Testing
```go
// tests/integration/service_integration_test.go
package integration

import (
    "context"
    "testing"
    "time"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/suite"
)

type ServiceIntegrationSuite struct {
    suite.Suite
    apiGatewayURL   string
    authServiceURL  string
    inferenceURL    string
    testUser        *TestUser
    testAPIKey      string
}

func (s *ServiceIntegrationSuite) SetupSuite() {
    s.apiGatewayURL = getEnv("API_GATEWAY_URL", "http://localhost:8080")
    s.authServiceURL = getEnv("AUTH_SERVICE_URL", "http://localhost:8081")
    s.inferenceURL = getEnv("INFERENCE_SERVICE_URL", "http://localhost:8082")
    
    // Create test user
    s.testUser = s.createTestUser()
    s.testAPIKey = s.generateTestAPIKey()
}

func (s *ServiceIntegrationSuite) TestEndToEndChatCompletion() {
    // Step 1: Authenticate and get JWT token
    token, err := s.authenticateUser(s.testUser.Email, s.testUser.Password)
    s.NoError(err)
    s.NotEmpty(token)
    
    // Step 2: Make chat completion request through API Gateway
    request := ChatCompletionRequest{
        Model: "gpt-3.5-turbo",
        Messages: []Message{
            {Role: "user", Content: "Hello, this is a test message"},
        },
    }
    
    response, err := s.makeAuthenticatedRequest("POST", "/v1/chat/completions", token, request)
    s.NoError(err)
    s.Equal(200, response.StatusCode)
    
    // Step 3: Verify response structure
    var completionResponse ChatCompletionResponse
    err = json.NewDecoder(response.Body).Decode(&completionResponse)
    s.NoError(err)
    
    s.NotEmpty(completionResponse.Choices)
    s.Equal("assistant", completionResponse.Choices[0].Message.Role)
    s.NotEmpty(completionResponse.Choices[0].Message.Content)
    s.Greater(completionResponse.Usage.TotalTokens, 0)
    
    // Step 4: Verify request was logged in monitoring service
    time.Sleep(100 * time.Millisecond) // Allow for async logging
    logs, err := s.getRecentLogs(s.testUser.ID)
    s.NoError(err)
    s.Greater(len(logs), 0)
}

func (s *ServiceIntegrationSuite) TestServiceHealthChecks() {
    services := []struct {
        name string
        url  string
        path string
    }{
        {"API Gateway", s.apiGatewayURL, "/health"},
        {"Auth Service", s.authServiceURL, "/health"},
        {"Inference Service", s.inferenceURL, "/health"},
    }
    
    for _, service := range services {
        s.Run(service.name, func() {
            resp, err := http.Get(service.url + service.path)
            s.NoError(err)
            s.Equal(200, resp.StatusCode)
            
            var health HealthResponse
            err = json.NewDecoder(resp.Body).Decode(&health)
            s.NoError(err)
            s.Equal("healthy", health.Status)
        })
    }
}

func (s *ServiceIntegrationSuite) TestRateLimitingAcrossServices() {
    token, _ := s.authenticateUser(s.testUser.Email, s.testUser.Password)
    
    // Make requests up to rate limit
    for i := 0; i < 100; i++ { // Assuming 100 req/min limit
        resp, err := s.makeAuthenticatedRequest("POST", "/v1/chat/completions", token, 
            ChatCompletionRequest{
                Model: "gpt-3.5-turbo",
                Messages: []Message{{Role: "user", Content: "Test"}},
            })
        
        if i < 100 {
            s.NoError(err)
            s.Equal(200, resp.StatusCode)
        } else {
            s.Equal(429, resp.StatusCode) // Rate limit exceeded
        }
    }
}
```

### Database Integration Testing
```go
// tests/integration/database_test.go
package integration

import (
    "context"
    "testing"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/suite"
)

type DatabaseIntegrationSuite struct {
    suite.Suite
    db *sql.DB
}

func (s *DatabaseIntegrationSuite) TestUserCRUDOperations() {
    ctx := context.Background()
    
    // Create user
    user := &User{
        Email:    "test@example.com",
        Password: "hashed-password",
        Role:     "user",
    }
    
    err := s.db.CreateUser(ctx, user)
    s.NoError(err)
    s.NotEmpty(user.ID)
    
    // Read user
    foundUser, err := s.db.GetUserByEmail(ctx, user.Email)
    s.NoError(err)
    s.Equal(user.Email, foundUser.Email)
    s.Equal(user.Role, foundUser.Role)
    
    // Update user
    user.Role = "admin"
    err = s.db.UpdateUser(ctx, user)
    s.NoError(err)
    
    updatedUser, err := s.db.GetUserByID(ctx, user.ID)
    s.NoError(err)
    s.Equal("admin", updatedUser.Role)
    
    // Delete user
    err = s.db.DeleteUser(ctx, user.ID)
    s.NoError(err)
    
    _, err = s.db.GetUserByID(ctx, user.ID)
    s.Error(err) // User should not exist
}

func (s *DatabaseIntegrationSuite) TestAPILogging() {
    ctx := context.Background()
    
    log := &APILog{
        UserID:    "user123",
        Method:    "POST",
        Path:      "/v1/chat/completions",
        Status:    200,
        Latency:   150,
        IPAddress: "192.168.1.1",
    }
    
    err := s.db.CreateAPILog(ctx, log)
    s.NoError(err)
    s.NotEmpty(log.ID)
    
    // Query logs
    logs, err := s.db.GetAPILogsByUserID(ctx, log.UserID, time.Now().Add(-24*time.Hour), time.Now())
    s.NoError(err)
    s.Greater(len(logs), 0)
    s.Equal(log.Path, logs[0].Path)
}
```

---

## 3. CONTRACT TESTS

### API Contract Testing with Pact
```javascript
// tests/contract/consumer_tests.js
const { Pact } = require('@pact-foundation/pact');
const axios = require('axios');

const provider = new Pact({
    consumer: 'HelixFlow-WebClient',
    provider: 'HelixFlow-API',
    port: 1234,
});

describe('HelixFlow API Contract Tests', () => {
    beforeAll(() => provider.setup());
    afterAll(() => provider.finalize());
    afterEach(() => provider.verify());

    describe('Chat Completion API', () => {
        it('should handle successful chat completion', async () => {
            const expectedRequest = {
                method: 'POST',
                path: '/v1/chat/completions',
                headers: {
                    'Authorization': 'Bearer valid-token',
                    'Content-Type': 'application/json',
                },
                body: {
                    model: 'gpt-3.5-turbo',
                    messages: [
                        { role: 'user', content: 'Hello' }
                    ]
                }
            };

            const expectedResponse = {
                status: 200,
                headers: {
                    'Content-Type': 'application/json',
                },
                body: {
                    choices: [
                        {
                            message: {
                                role: 'assistant',
                                content: like('Hello! How can I help you?')
                            }
                        }
                    ],
                    usage: {
                        prompt_tokens: integer(1, 1000),
                        completion_tokens: integer(1, 1000),
                        total_tokens: integer(1, 2000)
                    },
                    model: 'gpt-3.5-turbo'
                }
            };

            await provider.addInteraction({
                state: 'user is authenticated',
                uponReceiving: 'a chat completion request',
                withRequest: expectedRequest,
                willRespondWith: expectedResponse
            });

            const client = axios.create({
                baseURL: 'http://localhost:1234',
                headers: {
                    'Authorization': 'Bearer valid-token'
                }
            });

            const response = await client.post('/v1/chat/completions', {
                model: 'gpt-3.5-turbo',
                messages: [
                    { role: 'user', content: 'Hello' }
                ]
            });

            expect(response.status).toBe(200);
            expect(response.data.choices).toHaveLength(1);
            expect(response.data.choices[0].message.role).toBe('assistant');
            expect(response.data.usage.total_tokens).toBeGreaterThan(0);
        });

        it('should handle authentication errors', async () => {
            const expectedRequest = {
                method: 'POST',
                path: '/v1/chat/completions',
                headers: {
                    'Authorization': 'Bearer invalid-token',
                    'Content-Type': 'application/json',
                },
                body: {
                    model: 'gpt-3.5-turbo',
                    messages: [
                        { role: 'user', content: 'Hello' }
                    ]
                }
            };

            const expectedResponse = {
                status: 401,
                headers: {
                    'Content-Type': 'application/json',
                },
                body: {
                    error: {
                        message: 'Invalid authentication token',
                        type: 'authentication_error',
                        code: 'invalid_token'
                    }
                }
            };

            await provider.addInteraction({
                state: 'user has invalid authentication',
                uponReceiving: 'a chat completion request with invalid token',
                withRequest: expectedRequest,
                willRespondWith: expectedResponse
            });

            const client = axios.create({
                baseURL: 'http://localhost:1234',
                headers: {
                    'Authorization': 'Bearer invalid-token'
                }
            });

            try {
                await client.post('/v1/chat/completions', {
                    model: 'gpt-3.5-turbo',
                    messages: [
                        { role: 'user', content: 'Hello' }
                    ]
                });
                fail('Expected 401 error');
            } catch (error) {
                expect(error.response.status).toBe(401);
                expect(error.response.data.error.message).toBe('Invalid authentication token');
            }
        });
    });

    describe('Streaming API', () => {
        it('should handle streaming responses', async () => {
            const expectedRequest = {
                method: 'POST',
                path: '/v1/chat/completions',
                headers: {
                    'Authorization': 'Bearer valid-token',
                    'Content-Type': 'application/json',
                    'Accept': 'text/event-stream'
                },
                body: {
                    model: 'gpt-3.5-turbo',
                    messages: [
                        { role: 'user', content: 'Tell me a story' }
                    ],
                    stream: true
                }
            };

            const expectedResponse = {
                status: 200,
                headers: {
                    'Content-Type': 'text/event-stream',
                },
                body: eachLike({
                    data: {
                        choices: [{
                            delta: {
                                content: 'Once'
                            }
                        }]
                    }
                })
            };

            await provider.addInteraction({
                state: 'streaming is enabled',
                uponReceiving: 'a streaming chat completion request',
                withRequest: expectedRequest,
                willRespondWith: expectedResponse
            });

            // Test streaming implementation
            const eventSource = new EventSource('http://localhost:1234/v1/chat/completions');
            
            return new Promise((resolve, reject) => {
                const chunks = [];
                
                eventSource.onmessage = (event) => {
                    chunks.push(JSON.parse(event.data));
                    
                    if (chunks.length >= 3) {
                        eventSource.close();
                        resolve(chunks);
                    }
                };
                
                eventSource.onerror = reject;
            });
        });
    });
});
```

### Provider Verification Tests
```go
// tests/contract/provider_test.go
package contract

import (
    "testing"
    "net/http"
    "net/http/httptest"
    
    "github.com/pact-foundation/pact-go/provider"
    "github.com/stretchr/testify/suite"
)

type ProviderContractSuite struct {
    suite.Suite
    provider *provider.Provider
    server   *httptest.Server
}

func (s *ProviderContractSuite) SetupSuite() {
    // Setup test server with real handlers
    router := setupTestRouter()
    s.server = httptest.NewServer(router)
    
    s.provider = &provider.Provider{
        Provider:           "HelixFlow-API",
        PactBrokerURL:      "http://localhost:9292",
        ProviderBaseURL:    s.server.URL,
        Verbose:            true,
    }
}

func (s *ProviderContractSuite) TestProviderContracts() {
    // Verify all consumer contracts
    err := s.provider.Verify()
    s.NoError(err)
}

func (s *ProviderContractSuite) TestProviderStates() {
    states := map[string]func() error{
        "user is authenticated": func() error {
            // Setup authenticated user state
            createTestUser()
            return nil
        },
        "user has invalid authentication": func() error {
            // Setup invalid authentication state
            return nil
        },
        "streaming is enabled": func() error {
            // Setup streaming configuration
            enableStreaming()
            return nil
        },
    }
    
    s.provider.States = states
}
```

---

## 4. SECURITY TESTS

### OWASP ZAP Security Testing
```python
# tests/security/security_test.py
import pytest
import requests
import time
from zapv2 import ZAPv2

class TestSecurity:
    
    @pytest.fixture(scope="class")
    def zap(self):
        zap = ZAPv2(proxies={'http': 'http://127.0.0.1:8080', 'https': 'http://127.0.0.1:8080'})
        
        # Wait for ZAP to start
        for _ in range(20):
            try:
                zap.core.number_of_alerts()
                break
            except:
                time.sleep(1)
        
        return zap
    
    @pytest.fixture(scope="class")
    def target_url(self):
        return "https://api.helixflow.com"
    
    def test_authentication_security(self, zap, target_url):
        """Test authentication vulnerabilities"""
        
        # Test SQL injection in login
        login_payloads = [
            {"email": "admin' OR '1'='1", "password": "password"},
            {"email": "admin'/*", "password": "password"},
            {"email": "admin' OR '1'='1'--", "password": "password"},
            {"email": "admin'; DROP TABLE users;--", "password": "password"},
        ]
        
        for payload in login_payloads:
            response = requests.post(f"{target_url}/v1/auth/login", json=payload)
            
            # Should not return SQL errors or successful login
            assert "SQL" not in response.text
            assert response.status_code != 200
    
    def test_jwt_security(self, zap, target_url):
        """Test JWT token security"""
        
        # Get a valid token
        response = requests.post(f"{target_url}/v1/auth/login", json={
            "email": "test@example.com",
            "password": "testpassword"
        })
        
        token = response.json()['token']
        
        # Test token manipulation
        manipulated_tokens = [
            token[:-10] + "manipulated",  # Alter signature
            token.split('.')[0] + "." + token.split('.')[1] + ".fake",  # Fake signature
            "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0." + token.split('.')[1] + ".",  # None algorithm
        ]
        
        for manipulated_token in manipulated_tokens:
            headers = {"Authorization": f"Bearer {manipulated_token}"}
            response = requests.get(f"{target_url}/v1/user/profile", headers=headers)
            
            # Should reject manipulated tokens
            assert response.status_code == 401
    
    def test_injection_attacks(self, zap, target_url):
        """Test various injection attacks"""
        
        # Command injection test
        malicious_inputs = [
            "; cat /etc/passwd",
            "| whoami",
            "`whoami`",
            "$(whoami)",
        ]
        
        for malicious_input in malicious_inputs:
            response = requests.post(f"{target_url}/v1/chat/completions", json={
                "model": "gpt-3.5-turbo",
                "messages": [{"role": "user", "content": malicious_input}]
            }, headers={"Authorization": "Bearer valid-token"})
            
            # Should sanitize input and not execute commands
            assert "root" not in response.text
            assert "etc/passwd" not in response.text
    
    def test_rate_limiting_bypass(self, zap, target_url):
        """Test rate limiting bypass attempts"""
        
        # Make rapid requests
        for i in range(200):
            response = requests.post(f"{target_url}/v1/chat/completions", json={
                "model": "gpt-3.5-turbo",
                "messages": [{"role": "user", "content": f"Message {i}"}]
            }, headers={"Authorization": "Bearer valid-token"})
            
            if i >= 100:  # Assuming 100 req/min limit
                assert response.status_code == 429
    
    def test_automated_scan(self, zap, target_url):
        """Run automated security scan"""
        
        # Spider the target
        zap.spider.scan(target_url)
        
        # Wait for spider to complete
        while int(zap.spider.status()) < 100:
            time.sleep(1)
        
        # Run active scan
        zap.ascan.scan(target_url)
        
        # Wait for scan to complete
        while int(zap.ascan.status()) < 100:
            time.sleep(1)
        
        # Get alerts
        alerts = zap.core.alerts()
        
        # Filter high and medium risk alerts
        high_risk_alerts = [alert for alert in alerts if alert['risk'] in ['High', 'Medium']]
        
        # Should have no high-risk vulnerabilities
        assert len(high_risk_alerts) == 0, f"Found {len(high_risk_alerts)} high-risk vulnerabilities"
```

### Penetration Testing
```bash
#!/bin/bash
# tests/security/penetration_test.sh

TARGET_URL="https://api.helixflow.com"
OUTPUT_DIR="security_reports"

echo "Starting penetration testing for $TARGET_URL"

# Create output directory
mkdir -p $OUTPUT_DIR

# Run nmap scan
echo "Running network scan..."
nmap -sV -sC -oN $OUTPUT_DIR/nmap_scan.txt $TARGET_URL

# Run Nikto scan
echo "Running web vulnerability scan..."
nikto -h $TARGET_URL -o $OUTPUT_DIR/nikto_scan.txt

# Run SQLMap testing
echo "Testing for SQL injection..."
sqlmap -u "$TARGET_URL/v1/auth/login" --data="email=test&password=test" --batch --output-dir=$OUTPUT_DIR

# Run SSL/TLS testing
echo "Testing SSL/TLS configuration..."
testssl.sh $TARGET_URL > $OUTPUT_DIR/ssl_test.txt

# Run custom API security tests
echo "Running API security tests..."
python3 tests/security/api_security_tests.py $TARGET_URL

echo "Penetration testing completed. Reports saved to $OUTPUT_DIR"
```

---

## 5. PERFORMANCE TESTS

### Load Testing with K6
```javascript
// tests/performance/load_test.js
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const apiLatency = new Trend('api_latency');

export let options = {
    stages: [
        { duration: '2m', target: 100 },  // Ramp up to 100 users
        { duration: '5m', target: 100 },  // Stay at 100 users
        { duration: '2m', target: 200 },  // Ramp up to 200 users
        { duration: '5m', target: 200 },  // Stay at 200 users
        { duration: '2m', target: 300 },  // Ramp up to 300 users
        { duration: '5m', target: 300 },  // Stay at 300 users
        { duration: '2m', target: 0 },    // Ramp down to 0 users
    ],
    thresholds: {
        http_req_duration: ['p(95)<500'], // 95% of requests under 500ms
        http_req_failed: ['rate<0.1'],    // Error rate under 10%
        errors: ['rate<0.1'],
    },
};

const BASE_URL = __ENV.BASE_URL || 'https://api.helixflow.com';
const API_KEY = __ENV.API_KEY || 'test-api-key';

export default function () {
    const params = {
        headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${API_KEY}`,
        },
    };

    const payload = JSON.stringify({
        model: 'gpt-3.5-turbo',
        messages: [
            { role: 'user', content: 'Generate a short response about AI technology' }
        ],
        max_tokens: 100,
    });

    const response = http.post(`${BASE_URL}/v1/chat/completions`, payload, params);

    const success = check(response, {
        'status is 200': (r) => r.status === 200,
        'response time < 500ms': (r) => r.timings.duration < 500,
        'response has content': (r) => JSON.parse(r.body).choices[0].message.content.length > 0,
    });

    errorRate.add(!success);
    apiLatency.add(response.timings.duration);

    sleep(1); // Think time between requests
}

export function handleSummary(data) {
    return {
        'performance_report.json': JSON.stringify(data),
        'performance_report.html': generateHTMLReport(data),
    };
}

function generateHTMLReport(data) {
    return `
    <html>
        <head><title>Performance Test Report</title></head>
        <body>
            <h1>HelixFlow Performance Test Report</h1>
            <h2>Summary</h2>
            <ul>
                <li>Total Requests: ${data.metrics.http_reqs.count}</li>
                <li>Failed Requests: ${data.metrics.http_req_failed.count}</li>
                <li>Average Response Time: ${data.metrics.http_req_duration.avg}ms</li>
                <li>95th Percentile: ${data.metrics.http_req_duration['p(95)']}ms</li>
            </ul>
            <h2>Error Rate</h2>
            <p>${(data.metrics.http_req_failed.rate * 100).toFixed(2)}%</p>
        </body>
    </html>`;
}
```

### Stress Testing
```javascript
// tests/performance/stress_test.js
import http from 'k6/http';
import { check, sleep } from 'k6';

export let options = {
    stages: [
        { duration: '5m', target: 1000 },   // Ramp up to 1000 users quickly
        { duration: '10m', target: 1000 },  // Stay at 1000 users
        { duration: '5m', target: 2000 },   // Ramp up to 2000 users
        { duration: '10m', target: 2000 },  // Stay at 2000 users
        { duration: '5m', target: 0 },      // Ramp down
    ],
    thresholds: {
        http_req_duration: ['p(99)<2000'], // 99% under 2 seconds during stress
        http_req_failed: ['rate<0.2'],     // Allow up to 20% errors during stress
    },
};

const BASE_URL = __ENV.BASE_URL || 'https://api.helixflow.com';

export default function () {
    // Test different endpoints
    const endpoints = [
        '/v1/models',
        '/v1/chat/completions',
        '/v1/health',
    ];
    
    const endpoint = endpoints[Math.floor(Math.random() * endpoints.length)];
    
    let response;
    if (endpoint === '/v1/chat/completions') {
        response = http.post(`${BASE_URL}${endpoint}`, JSON.stringify({
            model: 'gpt-3.5-turbo',
            messages: [{ role: 'user', content: 'Stress test message' }],
        }), {
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${__ENV.API_KEY}`,
            },
        });
    } else {
        response = http.get(`${BASE_URL}${endpoint}`);
    }

    check(response, {
        'status is 200 or 429': (r) => r.status === 200 || r.status === 429,
        'response time < 5s': (r) => r.timings.duration < 5000,
    });

    sleep(Math.random() * 2); // Random sleep 0-2 seconds
}
```

### Endurance Testing
```javascript
// tests/performance/endurance_test.js
import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter } from 'k6/metrics';

const errorCounter = new Counter('errors');

export let options = {
    stages: [
        { duration: '30m', target: 100 },   // 30 minutes at 100 users
        { duration: '60m', target: 100 },  // 60 minutes at 100 users
        { duration: '30m', target: 200 },  // 30 minutes at 200 users
        { duration: '60m', target: 200 },  // 60 minutes at 200 users
    ],
    thresholds: {
        http_req_duration: ['p(95)<1000'], // 95% under 1 second
        http_req_failed: ['rate<0.01'],    // Less than 1% errors
        errors: ['count<10'],               // Less than 10 total errors
    },
};

const BASE_URL = __ENV.BASE_URL || 'https://api.helixflow.com';

export default function () {
    const startTime = Date.now();
    
    try {
        const response = http.post(`${BASE_URL}/v1/chat/completions`, JSON.stringify({
            model: 'gpt-3.5-turbo',
            messages: [
                { role: 'user', content: 'Endurance test - message at ' + new Date().toISOString() }
            ],
            max_tokens: 50,
        }), {
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${__ENV.API_KEY}`,
            },
            timeout: 30000, // 30 second timeout
        });

        const success = check(response, {
            'status is 200': (r) => r.status === 200,
            'response time < 2s': (r) => r.timings.duration < 2000,
            'no server errors': (r) => !r.body.includes('error'),
        });

        if (!success) {
            errorCounter.add(1);
        }

    } catch (error) {
        errorCounter.add(1);
        console.error('Request failed:', error.message);
    }

    // Adaptive sleep based on response time
    const responseTime = Date.now() - startTime;
    const baseSleep = 2;
    const adaptiveSleep = responseTime > 1000 ? baseSleep * 2 : baseSleep;
    
    sleep(adaptiveSleep);
}

export function handleSummary(data) {
    const hours = data.state.testRunDurationMs / (1000 * 60 * 60);
    const avgRPS = data.metrics.http_reqs.rate;
    const errorRate = data.metrics.http_req_failed.rate;
    
    console.log(`Endurance Test Summary:`);
    console.log(`Duration: ${hours.toFixed(2)} hours`);
    console.log(`Average RPS: ${avgRPS.toFixed(2)}`);
    console.log(`Error Rate: ${(errorRate * 100).toFixed(2)}%`);
    console.log(`Total Errors: ${data.metrics.http_req_failed.count}`);
    
    return {
        'endurance_report.json': JSON.stringify({
            duration: hours,
            avgRPS: avgRPS,
            errorRate: errorRate,
            totalErrors: data.metrics.http_req_failed.count,
            avgResponseTime: data.metrics.http_req_duration.avg,
            p95ResponseTime: data.metrics.http_req_duration['p(95)'],
        }),
    };
}
```

---

## 6. CHAOS TESTS

### Chaos Engineering with Chaos Monkey
```go
// tests/chaos/chaos_test.go
package chaos

import (
    "context"
    "testing"
    "time"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/suite"
)

type ChaosTestSuite struct {
    suite.Suite
    chaosEngine *ChaosEngine
}

func (s *ChaosTestSuite) SetupSuite() {
    s.chaosEngine = NewChaosEngine()
}

func (s *ChaosTestSuite) TestServiceFailureRecovery() {
    ctx := context.Background()
    
    // Start chaos experiment
    experiment := s.chaosEngine.StartExperiment(ChaosExperiment{
        Name:        "API Gateway Failure",
        Target:      "api-gateway",
        Action:      "stop_service",
        Duration:    30 * time.Second,
        Probability: 1.0,
    })
    
    // Wait for service to fail
    time.Sleep(5 * time.Second)
    
    // Test that requests still succeed through fallback
    successCount := 0
    for i := 0; i < 10; i++ {
        response, err := makeAPIRequest(ctx, "/v1/health")
        if err == nil && response.StatusCode == 200 {
            successCount++
        }
        time.Sleep(1 * time.Second)
    }
    
    // Should have successful requests through fallback
    assert.Greater(s.T(), successCount, 5)
    
    // Stop experiment
    s.chaosEngine.StopExperiment(experiment.ID)
    
    // Wait for recovery
    time.Sleep(10 * time.Second)
    
    // Verify full recovery
    response, err := makeAPIRequest(ctx, "/v1/health")
    assert.NoError(s.T(), err)
    assert.Equal(s.T(), 200, response.StatusCode)
}

func (s *ChaosTestSuite) TestDatabaseConnectionFailure() {
    ctx := context.Background()
    
    // Simulate database failure
    experiment := s.chaosEngine.StartExperiment(ChaosExperiment{
        Name:        "Database Connection Failure",
        Target:      "database",
        Action:      "connection_reset",
        Duration:    60 * time.Second,
        Probability: 1.0,
    })
    
    // Test that service handles connection failures gracefully
    for i := 0; i < 20; i++ {
        response, err := makeAPIRequest(ctx, "/v1/user/profile")
        
        // Should either succeed (from cache) or fail gracefully
        if err != nil {
            // Should return appropriate error, not crash
            assert.NotContains(s.T(), err.Error(), "panic")
            assert.NotContains(s.T(), err.Error(), "nil pointer")
        }
        
        time.Sleep(2 * time.Second)
    }
    
    s.chaosEngine.StopExperiment(experiment.ID)
}

func (s *ChaosTestSuite) TestNetworkPartition() {
    ctx := context.Background()
    
    // Simulate network partition between services
    experiment := s.chaosEngine.StartExperiment(ChaosExperiment{
        Name:        "Network Partition",
        Target:      "network",
        Action:      "partition_services",
        Duration:    45 * time.Second,
        Probability: 1.0,
        Parameters: map[string]interface{}{
            "service_a": "api-gateway",
            "service_b": "inference-pool",
        },
    })
    
    // Test that system continues to function
    successCount := 0
    startTime := time.Now()
    
    for time.Since(startTime) < 40*time.Second {
        response, err := makeAPIRequest(ctx, "/v1/chat/completions")
        if err == nil {
            successCount++
        }
        time.Sleep(1 * time.Second)
    }
    
    // Should have some successful requests through fallback mechanisms
    assert.Greater(s.T(), successCount, 0)
    
    s.chaosEngine.StopExperiment(experiment.ID)
}

func (s *ChaosTestSuite) TestResourceExhaustion() {
    ctx := context.Background()
    
    // Simulate CPU exhaustion
    experiment := s.chaosEngine.StartExperiment(ChaosExperiment{
        Name:        "CPU Exhaustion",
        Target:      "cpu",
        Action:      "cpu_stress",
        Duration:    120 * time.Second,
        Probability: 1.0,
        Parameters: map[string]interface{}{
            "cpu_percent": 90,
        },
    })
    
    // Monitor response times
    responseTimes := []time.Duration{}
    
    for i := 0; i < 30; i++ {
        start := time.Now()
        response, err := makeAPIRequest(ctx, "/v1/health")
        duration := time.Since(start)
        
        if err == nil {
            responseTimes = append(responseTimes, duration)
        }
        
        time.Sleep(2 * time.Second)
    }
    
    // Calculate average response time
    var totalTime time.Duration
    for _, rt := range responseTimes {
        totalTime += rt
    }
    avgResponseTime := totalTime / time.Duration(len(responseTimes))
    
    // During CPU stress, response times may increase but should not fail completely
    assert.Less(s.T(), avgResponseTime, 10*time.Second)
    
    s.chaosEngine.StopExperiment(experiment.ID)
}
```

### Network Chaos Testing
```bash
#!/bin/bash
# tests/chaos/network_chaos.sh

# Network latency chaos
function add_network_latency() {
    local interface=$1
    local latency=$2  # in milliseconds
    
    sudo tc qdisc add dev $interface root netem delay ${latency}ms
    echo "Added ${latency}ms latency to $interface"
}

# Network packet loss chaos
function add_packet_loss() {
    local interface=$1
    local loss_percent=$2
    
    sudo tc qdisc add dev $interface root netem loss ${loss_percent}%
    echo "Added ${loss_percent}% packet loss to $interface"
}

# Network bandwidth limitation
function limit_bandwidth() {
    local interface=$1
    local bandwidth=$2  # e.g., "1mbit"
    
    sudo tc qdisc add dev $interface root tbf rate $bandwidth burst 32kbit latency 400ms
    echo "Limited $interface bandwidth to $bandwidth"
}

# DNS chaos
function break_dns() {
    echo "# Chaos DNS configuration" | sudo tee -a /etc/hosts
    echo "0.0.0.0 auth-service" | sudo tee -a /etc/hosts
    echo "0.0.0.0 inference-pool" | sudo tee -a /etc/hosts
    echo "DNS chaos applied"
}

# Run network chaos experiment
function run_network_chaos() {
    echo "Starting network chaos experiment..."
    
    # Add 100ms latency
    add_network_latency "eth0" 100
    
    # Add 5% packet loss
    add_packet_loss "eth0" 5
    
    # Limit bandwidth to 10mbit
    limit_bandwidth "eth0" "10mbit"
    
    # Wait for experiment duration
    sleep 300
    
    # Clean up
    cleanup_network_chaos
}

# Clean up network chaos
function cleanup_network_chaos() {
    sudo tc qdisc del dev eth0 root 2>/dev/null
    sudo sed -i '/Chaos DNS configuration/,+2d' /etc/hosts
    echo "Network chaos cleaned up"
}

# Run experiment
run_network_chaos
```

---

## TEST AUTOMATION FRAMEWORK

### Continuous Integration Pipeline
```yaml
# .github/workflows/test-pipeline.yml
name: HelixFlow Test Pipeline

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

jobs:
  unit-tests:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        service: [api-gateway, auth-service, inference-pool, monitoring]
    
    steps:
    - uses: actions/checkout@v3
    
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21'
    
    - name: Install dependencies
      run: |
        cd ${{ matrix.service }}
        go mod download
    
    - name: Run unit tests
      run: |
        cd ${{ matrix.service }}
        go test -v -race -coverprofile=coverage.out ./...
        go tool cover -html=coverage.out -o coverage.html
    
    - name: Upload coverage reports
      uses: codecov/codecov-action@v3
      with:
        file: ${{ matrix.service }}/coverage.out
        flags: ${{ matrix.service }}

  integration-tests:
    runs-on: ubuntu-latest
    needs: unit-tests
    
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_PASSWORD: testpass
          POSTGRES_DB: helixflow_test
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
      
      redis:
        image: redis:7
        options: >-
          --health-cmd "redis-cli ping"
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
    
    steps:
    - uses: actions/checkout@v3
    
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21'
    
    - name: Start services
      run: |
        docker-compose -f docker-compose.test.yml up -d
        sleep 30  # Wait for services to start
    
    - name: Run integration tests
      run: |
        cd tests/integration
        go test -v ./...
    
    - name: Stop services
      run: docker-compose -f docker-compose.test.yml down

  contract-tests:
    runs-on: ubuntu-latest
    needs: integration-tests
    
    steps:
    - uses: actions/checkout@v3
    
    - name: Set up Node.js
      uses: actions/setup-node@v3
      with:
        node-version: '18'
    
    - name: Install Pact
      run: npm install -g @pact-foundation/pact-cli
    
    - name: Run consumer contract tests
      run: |
        cd tests/contract
        npm install
        npm run test:consumer
    
    - name: Start provider services
      run: |
        docker-compose -f docker-compose.contract-test.yml up -d
        sleep 30
    
    - name: Run provider contract verification
      run: |
        cd tests/contract
        npm run test:provider
    
    - name: Stop services
      run: docker-compose -f docker-compose.contract-test.yml down

  security-tests:
    runs-on: ubuntu-latest
    needs: contract-tests
    
    steps:
    - uses: actions/checkout@v3
    
    - name: Set up Python
      uses: actions/setup-python@v4
      with:
        python-version: '3.11'
    
    - name: Install security tools
      run: |
        pip install zapv2 bandit safety
        sudo apt-get update
        sudo apt-get install -y nmap nikto
    
    - name: Run static security analysis
      run: |
        bandit -r . -f json -o security-report.json || true
        safety check --json > safety-report.json || true
    
    - name: Run OWASP ZAP scan
      run: |
        # Start ZAP daemon
        docker run -d -u zap -p 8080:8080 -i owasp/zap2docker-stable zap.sh -daemon -port 8080 -host 0.0.0.0
        sleep 30
        
        # Run security tests
        cd tests/security
        python3 security_test.py
    
    - name: Upload security reports
      uses: actions/upload-artifact@v3
      with:
        name: security-reports
        path: |
          security-report.json
          safety-report.json
          zap-report.json

  performance-tests:
    runs-on: ubuntu-latest
    needs: security-tests
    if: github.ref == 'refs/heads/main'
    
    steps:
    - uses: actions/checkout@v3
    
    - name: Install K6
      run: |
        sudo gpg -k
        sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
        echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
        sudo apt-get update
        sudo apt-get install k6
    
    - name: Run load test
      run: |
        cd tests/performance
        k6 run --out json=load-test-results.json load_test.js
    
    - name: Run stress test
      run: |
        cd tests/performance
        k6 run --out json=stress-test-results.json stress_test.js
    
    - name: Upload performance results
      uses: actions/upload-artifact@v3
      with:
        name: performance-results
        path: |
          tests/performance/*-test-results.json
          tests/performance/performance_report.html

  chaos-tests:
    runs-on: ubuntu-latest
    needs: performance-tests
    if: github.ref == 'refs/heads/main'
    
    steps:
    - uses: actions/checkout@v3
    
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.21'
    
    - name: Install chaos engineering tools
      run: |
        go install github.com/chaosiq/chaostoolkit@latest
        pip install chaostoolkit
    
    - name: Run chaos tests
      run: |
        cd tests/chaos
        go test -v ./...
        
        # Run network chaos
        sudo bash network_chaos.sh
    
    - name: Upload chaos test results
      uses: actions/upload-artifact@v3
      with:
        name: chaos-test-results
        path: tests/chaos/chaos-report.json

  test-summary:
    runs-on: ubuntu-latest
    needs: [unit-tests, integration-tests, contract-tests, security-tests, performance-tests, chaos-tests]
    if: always()
    
    steps:
    - uses: actions/checkout@v3
    
    - name: Download all artifacts
      uses: actions/download-artifact@v3
    
    - name: Generate test summary
      run: |
        python3 scripts/generate_test_summary.py
    
    - name: Comment PR with results
      if: github.event_name == 'pull_request'
      uses: actions/github-script@v6
      with:
        script: |
          const fs = require('fs');
          const summary = fs.readFileSync('test-summary.md', 'utf8');
          
          github.rest.issues.createComment({
            issue_number: context.issue.number,
            owner: context.repo.owner,
            repo: context.repo.repo,
            body: summary
          });
```

---

## TEST REPORTING AND METRICS

### Test Coverage Dashboard
```python
# scripts/test_coverage_dashboard.py
import json
import matplotlib.pyplot as plt
import seaborn as sns
from datetime import datetime
import pandas as pd

def generate_coverage_report():
    # Collect coverage data from all services
    coverage_data = {
        'api-gateway': collect_go_coverage('api-gateway'),
        'auth-service': collect_go_coverage('auth-service'),
        'inference-pool': collect_go_coverage('inference-pool'),
        'python-sdk': collect_python_coverage('sdks/python'),
    }
    
    # Generate visualizations
    fig, axes = plt.subplots(2, 2, figsize=(15, 10))
    
    # Coverage by service
    services = list(coverage_data.keys())
    coverages = [data['coverage'] for data in coverage_data.values()]
    
    axes[0, 0].bar(services, coverages)
    axes[0, 0].set_title('Test Coverage by Service')
    axes[0, 0].set_ylabel('Coverage %')
    axes[0, 0].set_ylim(0, 100)
    
    # Coverage trend over time
    for service, data in coverage_data.items():
        dates = [entry['date'] for entry in data['history']]
        coverage_values = [entry['coverage'] for entry in data['history']]
        axes[0, 1].plot(dates, coverage_values, label=service, marker='o')
    
    axes[0, 1].set_title('Coverage Trend Over Time')
    axes[0, 1].set_ylabel('Coverage %')
    axes[0, 1].legend()
    
    # Test execution time
    execution_times = collect_test_execution_times()
    axes[1, 0].bar(execution_times.keys(), execution_times.values())
    axes[1, 0].set_title('Test Execution Time by Type')
    axes[1, 0].set_ylabel('Time (minutes)')
    
    # Error rate by test type
    error_rates = collect_error_rates()
    axes[1, 1].bar(error_rates.keys(), error_rates.values())
    axes[1, 1].set_title('Error Rate by Test Type')
    axes[1, 1].set_ylabel('Error Rate %')
    
    plt.tight_layout()
    plt.savefig('test_coverage_dashboard.png', dpi=300, bbox_inches='tight')
    
    # Generate HTML report
    html_content = f"""
    <!DOCTYPE html>
    <html>
    <head>
        <title>HelixFlow Test Coverage Dashboard</title>
        <style>
            body {{ font-family: Arial, sans-serif; margin: 20px; }}
            .metric {{ display: inline-block; margin: 20px; padding: 20px; border: 1px solid #ccc; }}
            .metric h3 {{ margin-top: 0; }}
            .coverage-high {{ color: green; }}
            .coverage-medium {{ color: orange; }}
            .coverage-low {{ color: red; }}
        </style>
    </head>
    <body>
        <h1>HelixFlow Test Coverage Dashboard</h1>
        <p>Generated on: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}</p>
        
        <div class="metric">
            <h3>Overall Coverage</h3>
            <p class="coverage-high">{calculate_overall_coverage(coverage_data):.1f}%</p>
        </div>
        
        <div class="metric">
            <h3>Unit Tests</h3>
            <p>{coverage_data['unit_tests']['passed']}/{coverage_data['unit_tests']['total']} passed</p>
        </div>
        
        <div class="metric">
            <h3>Integration Tests</h3>
            <p>{coverage_data['integration_tests']['passed']}/{coverage_data['integration_tests']['total']} passed</p>
        </div>
        
        <div class="metric">
            <h3>Security Score</h3>
            <p class="coverage-high">{calculate_security_score():.0f}/100</p>
        </div>
        
        <h2>Coverage by Service</h2>
        <img src="test_coverage_dashboard.png" alt="Test Coverage Dashboard" style="max-width: 100%;">
        
        <h2>Recent Test Failures</h2>
        {generate_recent_failures_table()}
    </body>
    </html>
    """
    
    with open('test_coverage_dashboard.html', 'w') as f:
        f.write(html_content)

if __name__ == '__main__':
    generate_coverage_report()
```

---

## QUALITY GATES AND VALIDATION

### Pre-commit Hooks
```yaml
# .pre-commit-config.yaml
repos:
  - repo: https://github.com/pre-commit/pre-commit-hooks
    rev: v4.4.0
    hooks:
      - id: trailing-whitespace
      - id: end-of-file-fixer
      - id: check-yaml
      - id: check-added-large-files

  - repo: https://github.com/psf/black
    rev: 23.3.0
    hooks:
      - id: black
        language_version: python3.11

  - repo: https://github.com/golangci/golangci-lint
    rev: v1.53.3
    hooks:
      - id: golangci-lint

  - repo: https://github.com/securecodewarrior/sast-scan
    rev: v1.9.0
    hooks:
      - id: sast-scan

  - repo: local
    hooks:
      - id: unit-tests
        name: Unit Tests
        entry: bash -c 'make test-unit'
        language: system
        pass_filenames: false
      
      - id: security-scan
        name: Security Scan
        entry: bash -c 'make security-scan'
        language: system
        pass_filenames: false
```

### Quality Gate Criteria
```yaml
# quality-gates.yml
quality_gates:
  unit_tests:
    coverage_minimum: 95%
    pass_rate: 100%
    execution_time_max: 10_minutes
    
  integration_tests:
    pass_rate: 100%
    execution_time_max: 30_minutes
    services_coverage: 100%
    
  contract_tests:
    pass_rate: 100%
    api_coverage: 100%
    breaking_changes: 0
    
  security_tests:
    high_risk_vulnerabilities: 0
    medium_risk_vulnerabilities: 5
    low_risk_vulnerabilities: 10
    
  performance_tests:
    p95_response_time: 500ms
    p99_response_time: 2000ms
    error_rate: 1%
    throughput_minimum: 1000_rps
    
  chaos_tests:
    recovery_time_max: 30s
    availability_minimum: 99.9%
    data_loss: 0%
```

This comprehensive testing strategy ensures that HelixFlow meets the highest quality standards with complete test coverage across all critical paths, automated testing at multiple levels, and continuous quality monitoring.