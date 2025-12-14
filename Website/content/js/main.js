// HelixFlow Website Interactive Features
// Enhanced with API playground, demo functionality, and interactive elements

// Global variables
let apiKey = localStorage.getItem('helixflow_api_key') || '';
let authToken = localStorage.getItem('helixflow_auth_token') || '';
let currentLanguage = 'python';

// DOM Elements
const mobileMenuButton = document.getElementById('mobile-menu-button');
const mobileMenu = document.getElementById('mobile-menu');
const navbar = document.getElementById('navbar');
const demoInput = document.getElementById('demo-input');
const demoModel = document.getElementById('demo-model');
const demoButton = document.getElementById('demo-button');
const demoOutput = document.getElementById('demo-output');
const demoPlaceholder = document.getElementById('demo-placeholder');
const demoResponse = document.getElementById('demo-response');
const requestSlider = document.getElementById('request-slider');
const requestCount = document.getElementById('request-count');
const estimatedCost = document.getElementById('estimated-cost');
const codeExample = document.getElementById('code-example');

// Initialize when DOM is loaded
document.addEventListener('DOMContentLoaded', function() {
    initializeNavigation();
    initializeDemo();
    initializeAPIPlayground();
    initializePricingCalculator();
    initializeAnimations();
    initializeMobileMenu();
    initializeCodeExamples();
    initializeScrollAnimations();
    initializeFormHandlers();
});

// Navigation functionality
function initializeNavigation() {
    // Mobile menu toggle
    if (mobileMenuButton) {
        mobileMenuButton.addEventListener('click', toggleMobileMenu);
    }
    
    // Smooth scrolling for navigation links
    document.querySelectorAll('a[href^="#"]').forEach(anchor => {
        anchor.addEventListener('click', function (e) {
            e.preventDefault();
            const target = document.querySelector(this.getAttribute('href'));
            if (target) {
                target.scrollIntoView({
                    behavior: 'smooth',
                    block: 'start'
                });
                // Close mobile menu if open
                if (mobileMenu) {
                    mobileMenu.classList.add('hidden');
                }
            }
        });
    });
    
    // Navbar scroll effect
    window.addEventListener('scroll', handleNavbarScroll);
}

function toggleMobileMenu() {
    mobileMenu.classList.toggle('hidden');
    
    // Animate hamburger icon
    const icon = mobileMenuButton.querySelector('i');
    if (mobileMenu.classList.contains('hidden')) {
        icon.classList.remove('fa-times');
        icon.classList.add('fa-bars');
    } else {
        icon.classList.remove('fa-bars');
        icon.classList.add('fa-times');
    }
}

function handleNavbarScroll() {
    const scrolled = window.pageYOffset > 50;
    if (scrolled) {
        navbar.classList.add('shadow-xl');
        navbar.classList.remove('shadow-lg');
    } else {
        navbar.classList.remove('shadow-xl');
        navbar.classList.add('shadow-lg');
    }
}

// Mobile menu functionality
function initializeMobileMenu() {
    // Close mobile menu when clicking outside
    document.addEventListener('click', function(event) {
        const isClickInsideMenu = mobileMenu.contains(event.target);
        const isClickOnButton = mobileMenuButton.contains(event.target);
        
        if (!isClickInsideMenu && !isClickOnButton && !mobileMenu.classList.contains('hidden')) {
            mobileMenu.classList.add('hidden');
        }
    });
}

// Demo functionality
function initializeDemo() {
    if (!demoButton) return;
    
    demoButton.addEventListener('click', generateDemoResponse);
    
    // Add some example prompts
    const examplePrompts = [
        "Explain quantum computing in simple terms",
        "Write a Python function to reverse a string",
        "What are the benefits of cloud computing?",
        "Generate a business plan outline for a startup",
        "Explain machine learning to a 5-year-old"
    ];
    
    // Rotate example prompts
    let currentPromptIndex = 0;
    setInterval(() => {
        if (demoInput && demoInput.placeholder && !demoInput.value) {
            currentPromptIndex = (currentPromptIndex + 1) % examplePrompts.length;
            demoInput.placeholder = `Try: ${examplePrompts[currentPromptIndex]}`;
        }
    }, 5000);
}

async function generateDemoResponse() {
    const message = demoInput.value.trim();
    if (!message) {
        showDemoError('Please enter a message');
        return;
    }
    
    const model = demoModel.value;
    
    // Show loading state
    showDemoLoading();
    
    // Simulate API call with streaming
    try {
        const response = await simulateHelixFlowAPI(message, model);
        await streamDemoResponse(response);
    } catch (error) {
        console.error('Demo error:', error);
        showDemoError('Error generating response. Please try again.');
    }
}

async function simulateHelixFlowAPI(message, model) {
    // Simulate API delay
    await new Promise(resolve => setTimeout(resolve, 1000 + Math.random() * 2000));

    // Generate realistic response based on input and model
    const responses = {
        'gpt-3.5-turbo': generateGPT35Response(message),
        'gpt-4': generateGPT4Response(message),
        'claude-v1': generateClaudeResponse(message),
        'llama-2-70b': generateLlamaResponse(message)
    };

    return {
        choices: [{
            message: {
                role: 'assistant',
                content: responses[model] || generateDefaultResponse(message)
            }
        }],
        usage: {
            prompt_tokens: Math.floor(message.length / 4),
            completion_tokens: Math.floor(Math.random() * 100) + 50,
            total_tokens: Math.floor(message.length / 4) + Math.floor(Math.random() * 100) + 50
        }
    };
}

function generateGPT35Response(message) {
    const responses = [
        "Hello! I'm HelixFlow AI, powered by GPT-3.5 Turbo. I'm here to help you with any questions or tasks you have. How can I assist you today?",
        "Great to meet you! I can help with a wide variety of tasks including answering questions, providing explanations, helping with writing, and much more. What would you like to know?",
        "Hi there! I'm ready to help you with your questions. I have knowledge about many topics and can assist with both simple and complex inquiries."
    ];
    return responses[Math.floor(Math.random() * responses.length)];
}

function generateGPT4Response(message) {
    const responses = [
        "Greetings! I'm powered by GPT-4 through HelixFlow's enterprise platform. I offer enhanced reasoning capabilities and can provide more detailed, nuanced responses. How may I help you today?",
        "Hello! As an AI assistant powered by GPT-4, I can help with complex problem-solving, detailed analysis, creative writing, and much more. I'm excited to assist you with your inquiry."
    ];
    return responses[Math.floor(Math.random() * responses.length)];
}

function generateClaudeResponse(message) {
    const responses = [
        "Hello! I'm Claude, accessible through HelixFlow's unified API. I'm designed to be helpful, harmless, and honest in my interactions. How can I assist you today?",
        "Hi there! I'm Claude, and I'm here to provide helpful and thoughtful responses to your questions. I strive to be informative while maintaining safety and accuracy."
    ];
    return responses[Math.floor(Math.random() * responses.length)];
}

function generateLlamaResponse(message) {
    const responses = [
        "Greetings! I'm powered by Llama 2 70B through HelixFlow. I'm an open-source large language model that can help with various tasks including answering questions and providing information.",
        "Hello! I'm accessible via Llama 2, and I'm here to help you with your questions and tasks. I can assist with information, analysis, and general conversation."
    ];
    return responses[Math.floor(Math.random() * responses.length)];
}

function generateDefaultResponse(message) {
    return `Hello! I understand you're asking about: "${message}". I'm here to help you with any questions or tasks you have. How can I assist you further?`;
}

async function streamDemoResponse(response) {
    demoPlaceholder.classList.add('hidden');
    demoOutput.classList.remove('hidden');
    demoResponse.innerHTML = '';
    
    const text = response.choices[0].message.content;
    const words = text.split(' ');
    
    for (let i = 0; i < words.length; i++) {
        demoResponse.innerHTML += words[i] + ' ';
        demoResponse.scrollTop = demoResponse.scrollHeight;
        
        // Simulate typing delay
        await new Promise(resolve => setTimeout(resolve, 50 + Math.random() * 100));
    }
    
    // Show completion info
    const usage = response.usage;
    demoResponse.innerHTML += `
        <div class="text-xs text-blue-200 mt-3 flex justify-between">
            <span>Model: ${demoModel.value}</span>
            <span>Tokens: ${usage.total_tokens} | Time: ${(Math.random() * 0.5 + 0.1).toFixed(3)}s</span>
        </div>
    `;
    
    // Reset button
    demoButton.innerHTML = '<i class="fas fa-magic mr-2"></i>Generate Response';
    demoButton.disabled = false;
}

function showDemoLoading() {
    demoButton.innerHTML = '<i class="fas fa-spinner fa-spin mr-2"></i>Generating...';
    demoButton.disabled = true;
    demoOutput.classList.add('hidden');
    demoPlaceholder.innerHTML = '<i class="fas fa-spinner fa-spin mr-2"></i>Generating response...';
}

function showDemoError(error) {
    demoButton.innerHTML = '<i class="fas fa-magic mr-2"></i>Generate Response';
    demoButton.disabled = false;
    demoPlaceholder.innerHTML = `<span class="text-red-400">${error}</span>`;
}

// API Playground functionality
function initializeAPIPlayground() {
    const executeButton = document.querySelector('#live-demo button[onclick="executeCode()"]');
    if (executeButton) {
        executeButton.addEventListener('click', executeCode);
    }

    // Set up default code examples
    updateCodeExample('python');
}

function executeCode() {
    const codeInput = document.getElementById('code-input');
    const codeOutput = document.getElementById('code-output');
    const executionStatus = document.getElementById('execution-status');
    const responseTime = document.getElementById('response-time');

    if (!codeInput || !codeOutput || !executionStatus || !responseTime) return;

    const code = codeInput.value.trim();
    if (!code) {
        showNotification('Please enter some code to execute', 'error');
        return;
    }

    // Show execution status
    executionStatus.innerHTML = '<i class="fas fa-spinner fa-spin mr-2"></i>Executing...';
    codeOutput.innerHTML = '';

    const startTime = Date.now();

    // Simulate code execution
    setTimeout(() => {
        const responseTimeMs = Date.now() - startTime;
        
        try {
            // Simulate API response
            const response = simulateAPIResponse(code);
            codeOutput.innerHTML = `<pre class="text-green-400">${JSON.stringify(response, null, 2)}</pre>`;
            
            executionStatus.innerHTML = '<i class="fas fa-check-circle text-green-400 mr-2"></i>Success';
            responseTime.textContent = `Response time: ${responseTimeMs}ms`;
            
        } catch (error) {
            codeOutput.innerHTML = `<pre class="text-red-400">Error: ${error.message}</pre>`;
            executionStatus.innerHTML = '<i class="fas fa-exclamation-circle text-red-400 mr-2"></i>Error';
        }
    }, 1000 + Math.random() * 2000);
}

function simulateAPIResponse(code) {
    // Simulate different API responses based on code content
    if (code.includes('chat/completions')) {
        return {
            choices: [{
                message: {
                    role: 'assistant',
                    content: 'This is a simulated response from HelixFlow API. In a real implementation, this would be an actual AI-generated response.'
                }
            }],
            usage: {
                prompt_tokens: 10,
                completion_tokens: 25,
                total_tokens: 35
            }
        };
    } else if (code.includes('models')) {
        return {
            data: [
                { id: 'gpt-3.5-turbo', object: 'model', owned_by: 'openai' },
                { id: 'gpt-4', object: 'model', owned_by: 'openai' },
                { id: 'claude-v1', object: 'model', owned_by: 'anthropic' }
            ]
        };
    } else {
        return {
            message: 'Simulated API response',
            timestamp: new Date().toISOString(),
            status: 'success'
        };
    }
}

function switchLanguage(language) {
    currentLanguage = language;
    updateCodeExample(language);
    
    // Update active button
    const buttons = document.querySelectorAll('#live-demo button[onclick^="switchLanguage"]');
    buttons.forEach(btn => {
        btn.classList.remove('text-white', 'bg-blue-600');
        btn.classList.add('text-gray-400', 'bg-gray-800');
    });
    
    const activeButton = Array.from(buttons).find(btn => btn.textContent.toLowerCase() === language);
    if (activeButton) {
        activeButton.classList.remove('text-gray-400', 'bg-gray-800');
        activeButton.classList.add('text-white', 'bg-blue-600');
    }
}

function updateCodeExample(language) {
    if (!codeExample) return;

    const examples = {
        python: `# Python example
import requests

# Set your API key
headers = {
    'Authorization': 'Bearer YOUR_API_KEY',
    'Content-Type': 'application/json'
}

# Make a chat completion request
data = {
    'model': 'gpt-3.5-turbo',
    'messages': [
        {'role': 'user', 'content': 'Hello, world!'}
    ]
}

response = requests.post('https://api.helixflow.com/v1/chat/completions', 
                        json=data, headers=headers)
print(response.json())`,

        javascript: `// JavaScript example
const axios = require('axios');

// Set your API key
const headers = {
    'Authorization': 'Bearer YOUR_API_KEY',
    'Content-Type': 'application/json'
};

// Make a chat completion request
const data = {
    model: 'gpt-3.5-turbo',
    messages: [
        {role: 'user', content: 'Hello, world!'}
    ]
};

axios.post('https://api.helixflow.com/v1/chat/completions', data, {headers})
    .then(response => console.log(response.data))
    .catch(error => console.error(error));`,

        curl: `# cURL example
curl -X POST "https://api.helixflow.com/v1/chat/completions" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [
      {"role": "user", "content": "Hello, world!"}
    ]
  }'`
    };

    codeExample.innerHTML = `<code class="language-${language}">${examples[language]}</code>`;
    
    // Re-highlight code
    if (window.Prism) {
        Prism.highlightElement(codeExample.querySelector('code'));
    }
}

function copyResponse() {
    const codeOutput = document.getElementById('code-output');
    if (!codeOutput) return;

    const text = codeOutput.textContent;
    navigator.clipboard.writeText(text).then(() => {
        showNotification('Response copied to clipboard!', 'success');
    }).catch(() => {
        showNotification('Failed to copy response', 'error');
    });
}

// Pricing calculator functionality
function initializePricingCalculator() {
    if (!requestSlider) return;
    
    requestSlider.addEventListener('input', updatePricingCalculator);
    updatePricingCalculator(); // Initial update
}

function updatePricingCalculator() {
    const requests = parseInt(requestSlider.value);
    requestCount.textContent = requests.toLocaleString();
    
    // Calculate estimated cost
    let cost = 0;
    let plan = 'Free';
    
    if (requests <= 10000) {
        cost = 0;
        plan = 'Starter (Free)';
    } else if (requests <= 100000) {
        cost = 99;
        plan = 'Pro';
    } else {
        // Custom pricing for high volume
        cost = 99 + (requests - 100000) * 0.001;
        plan = 'Enterprise';
    }
    
    estimatedCost.textContent = cost === 0 ? 'Free' : `$${cost.toFixed(2)}/month`;
    
    // Update plan recommendation
    const costElement = estimatedCost.parentElement;
    const planText = costElement.querySelector('.text-sm');
    if (planText) {
        planText.textContent = `Perfect for ${plan} plan`;
    }
}

// Code Example Functions
function initializeCodeExamples() {
    // Add click handlers for language buttons
    document.querySelectorAll('[onclick^="switchLanguage"]').forEach(button => {
        button.addEventListener('click', function() {
            const language = this.getAttribute('onclick').match(/'([^']+)'/)[1];
            switchLanguage(language);
        });
    });
}

function initializeScrollAnimations() {
    const observerOptions = {
        threshold: 0.1,
        rootMargin: '0px 0px -50px 0px'
    };
    
    const observer = new IntersectionObserver(function(entries) {
        entries.forEach(entry => {
            if (entry.isIntersecting) {
                entry.target.style.opacity = '1';
                entry.target.style.transform = 'translateY(0)';
            }
        });
    }, observerOptions);
    
    // Observe feature cards
    document.querySelectorAll('.feature-card').forEach(card => {
        card.style.opacity = '0';
        card.style.transform = 'translateY(20px)';
        card.style.transition = 'opacity 0.6s ease, transform 0.6s ease';
        observer.observe(card);
    });
    
    // Observe other elements
    document.querySelectorAll('.animate-on-scroll').forEach(element => {
        element.style.opacity = '0';
        element.style.transform = 'translateY(20px)';
        element.style.transition = 'opacity 0.6s ease, transform 0.6s ease';
        observer.observe(element);
    });
}

function initializeFormHandlers() {
    // Newsletter signup
    const newsletterForm = document.getElementById('newsletter-form');
    if (newsletterForm) {
        newsletterForm.addEventListener('submit', handleNewsletterSignup);
    }
    
    // Contact forms
    document.querySelectorAll('form').forEach(form => {
        form.addEventListener('submit', handleFormSubmit);
    });
}

function handleNewsletterSignup(e) {
    e.preventDefault();
    const email = this.querySelector('input[type="email"]').value;
    
    // Simulate signup
    const submitButton = this.querySelector('button[type="submit"]');
    const originalText = submitButton.innerHTML;
    
    submitButton.innerHTML = '<i class="fas fa-spinner fa-spin mr-2"></i>Signing up...';
    submitButton.disabled = true;
    
    setTimeout(() => {
        this.innerHTML = `
            <div class="bg-green-50 border border-green-200 rounded-lg p-4">
                <p class="text-green-800">Thank you for subscribing! Check your email for confirmation.</p>
            </div>
        `;
    }, 2000);
}

function handleFormSubmit(e) {
    e.preventDefault();
    
    const form = e.target;
    const submitButton = form.querySelector('button[type="submit"]');
    const originalText = submitButton.innerHTML;
    
    // Show loading state
    submitButton.innerHTML = '<i class="fas fa-spinner fa-spin mr-2"></i>Submitting...';
    submitButton.disabled = true;
    
    // Simulate form submission
    setTimeout(() => {
        submitButton.innerHTML = originalText;
        submitButton.disabled = false;
        
        // Show success message
        showNotification('Thank you for your submission! We\'ll get back to you soon.', 'success');
    }, 3000);
}

// CTA Functions
function startDemo() {
    // Scroll to demo section
    document.getElementById('hero').scrollIntoView({
        behavior: 'smooth',
        block: 'start'
    });
    
    // Focus on demo input
    setTimeout(() => {
        if (demoInput) {
            demoInput.focus();
        }
    }, 1000);
}

function playDemo() {
    // Simulate playing a demo video
    showNotification('Demo video loading...', 'info');
    setTimeout(() => {
        showNotification('Demo video ready to play!', 'success');
    }, 2000);
}

function startFreeTrial() {
    // Simulate starting free trial
    showNotification('Setting up your free trial...', 'info');
    setTimeout(() => {
        showNotification('Free trial activated! Check your email for setup instructions.', 'success');
    }, 3000);
}

function scheduleDemo() {
    // Simulate scheduling a demo
    showNotification('Opening scheduling calendar...', 'info');
    setTimeout(() => {
        showNotification('Demo scheduled! You will receive a calendar invitation.', 'success');
    }, 3000);
}

function contactSales() {
    showNotification('Opening contact form...', 'info');
    setTimeout(() => {
        showNotification('Sales team will contact you within 24 hours.', 'success');
    }, 2000);
}

function downloadEnterpriseGuide() {
    showNotification('Preparing enterprise guide download...', 'info');
    setTimeout(() => {
        showNotification('Enterprise guide downloaded successfully!', 'success');
    }, 2000);
}

// Utility functions
function showNotification(message, type = 'info') {
    // Create notification element
    const notification = document.createElement('div');
    notification.className = `fixed top-4 right-4 z-50 p-4 rounded-lg shadow-lg transition-all duration-300 transform translate-x-full`;
    
    const colors = {
        success: 'bg-green-500 text-white',
        error: 'bg-red-500 text-white',
        info: 'bg-blue-500 text-white',
        warning: 'bg-yellow-500 text-gray-900'
    };
    
    notification.className += ' ' + (colors[type] || colors.info);
    notification.innerHTML = `
        <div class="flex items-center">
            <span class="mr-2">${message}</span>
            <button onclick="this.parentElement.parentElement.remove()" class="ml-2 text-xl font-bold hover:opacity-70">
                ×
            </button>
        </div>
    `;
    
    document.body.appendChild(notification);
    
    // Animate in
    setTimeout(() => {
        notification.classList.remove('translate-x-full');
    }, 100);
    
    // Auto remove after 5 seconds
    setTimeout(() => {
        notification.classList.add('translate-x-full');
        setTimeout(() => {
            notification.remove();
        }, 300);
    }, 5000);
}

function debounce(func, wait) {
    let timeout;
    return function executedFunction(...args) {
        const later = () => {
            clearTimeout(timeout);
            func(...args);
        };
        clearTimeout(timeout);
        timeout = setTimeout(later, wait);
    };
}

function throttle(func, limit) {
    let inThrottle;
    return function() {
        const args = arguments;
        const context = this;
        if (!inThrottle) {
            func.apply(context, args);
            inThrottle = true;
            setTimeout(() => inThrottle = false, limit);
        }
    };
}

// Error Handling
window.addEventListener('error', function(e) {
    console.error('JavaScript Error:', e.error);
    // You can send errors to your logging service here
});

// Performance Monitoring
if ('PerformanceObserver' in window) {
    const perfObserver = new PerformanceObserver((list) => {
        for (const entry of list.getEntries()) {
            // Monitor performance metrics
            console.log('Performance entry:', entry.name, entry.duration);
        }
    });
    
    perfObserver.observe({ entryTypes: ['measure', 'navigation'] });
}

// Accessibility Functions
function improveAccessibility() {
    // Add keyboard navigation support
    document.addEventListener('keydown', function(e) {
        if (e.key === 'Escape' && !mobileMenu.classList.contains('hidden')) {
            toggleMobileMenu();
        }
    });
    
    // Add focus management for modals and dropdowns
    // Add ARIA labels dynamically
    // Add skip links for screen readers
}

// Initialize accessibility improvements
improveAccessibility();

// CSS animations
const style = document.createElement('style');
style.textContent = `
    .animate-fade-in {
        animation: fadeIn 0.6s ease-out forwards;
    }
    
    @keyframes fadeIn {
        from {
            opacity: 0;
            transform: translateY(20px);
        }
        to {
            opacity: 1;
            transform: translateY(0);
        }
    }
    
    .feature-card {
        opacity: 0;
        transform: translateY(30px);
        transition: all 0.3s ease;
    }
    
    .feature-card.animate-fade-in {
        opacity: 1;
        transform: translateY(0);
    }
    
    .transform-hover:hover {
        transform: translateY(-2px);
    }
    
    .transition-all {
        transition: all 0.3s ease;
    }
    
    .backdrop-blur-lg {
        backdrop-filter: blur(10px);
    }
    
    /* Custom scrollbar for code areas */
    pre::-webkit-scrollbar {
        width: 8px;
        height: 8px;
    }
    
    pre::-webkit-scrollbar-track {
        background: #1e293b;
    }
    
    pre::-webkit-scrollbar-thumb {
        background: #475569;
        border-radius: 4px;
    }
    
    pre::-webkit-scrollbar-thumb:hover {
        background: #64748b;
    }
`;
document.head.appendChild(style);

// Export functions for global use
window.HelixFlow = {
    startDemo,
    playDemo,
    startFreeTrial,
    scheduleDemo,
    switchLanguage,
    generateDemoResponse,
    executeCode,
    copyResponse
};