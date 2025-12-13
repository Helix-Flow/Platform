// HelixFlow Website JavaScript

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

// State
let currentLanguage = 'python';

// Initialize
document.addEventListener('DOMContentLoaded', function() {
    initializeNavigation();
    initializeDemo();
    initializePricingCalculator();
    initializeCodeExamples();
    initializeScrollAnimations();
    initializeFormHandlers();
});

// Navigation Functions
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

// Demo Functions
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

function generateDemoResponse() {
    const message = demoInput.value.trim();
    if (!message) {
        showDemoError('Please enter a message');
        return;
    }
    
    const model = demoModel.value;
    
    // Show loading state
    showDemoLoading();
    
    // Simulate API call
    setTimeout(() => {
        const response = generateMockResponse(message, model);
        showDemoResponse(response);
    }, 1500 + Math.random() * 1000); // Random delay between 1.5-2.5 seconds
}

function generateMockResponse(message, model) {
    const responses = {
        'gpt-3.5-turbo': [
            "That's an excellent question! Let me provide you with a comprehensive answer...",
            "Based on my analysis, here's what I found regarding your query...",
            "I understand you're asking about this topic. Here's a detailed explanation..."
        ],
        'gpt-4': [
            "This is a fascinating topic that requires careful consideration. Let me break it down for you...",
            "Your question touches on several important aspects. Here's my perspective...",
            "This is quite complex, but I'll do my best to explain it clearly..."
        ],
        'claude-v1': [
            "I'd be happy to help you understand this better. Let me share some insights...",
            "This is an interesting topic! Here's what I know about it...",
            "Thank you for asking this question. Here's a thoughtful response..."
        ],
        'llama-2-70b': [
            "Great question! Let me provide you with some helpful information...",
            "I appreciate your curiosity about this subject. Here's what I can tell you...",
            "This is definitely worth exploring. Here's my take on it..."
        ]
    };
    
    const modelResponses = responses[model] || responses['gpt-3.5-turbo'];
    const randomResponse = modelResponses[Math.floor(Math.random() * modelResponses.length)];
    
    return {
        model: model,
        response: randomResponse,
        tokens_used: Math.floor(Math.random() * 100) + 50,
        response_time: (Math.random() * 0.5 + 0.1).toFixed(3)
    };
}

function showDemoLoading() {
    demoButton.innerHTML = '<i class="fas fa-spinner fa-spin mr-2"></i>Generating...';
    demoButton.disabled = true;
    demoOutput.classList.add('hidden');
    demoPlaceholder.innerHTML = '<i class="fas fa-spinner fa-spin mr-2"></i>Generating response...';
}

function showDemoResponse(response) {
    demoButton.innerHTML = '<i class="fas fa-magic mr-2"></i>Generate Response';
    demoButton.disabled = false;
    
    demoPlaceholder.classList.add('hidden');
    demoOutput.classList.remove('hidden');
    
    demoResponse.innerHTML = `
        <div class="mb-3">${response.response}</div>
        <div class="text-xs text-gray-400 flex justify-between">
            <span>Model: ${response.model}</span>
            <span>Tokens: ${response.tokens_used} | Time: ${response.response_time}s</span>
        </div>
    `;
}

function showDemoError(error) {
    demoButton.innerHTML = '<i class="fas fa-magic mr-2"></i>Generate Response';
    demoButton.disabled = false;
    
    demoPlaceholder.innerHTML = `<span class="text-red-400">${error}</span>`;
}

// Pricing Calculator Functions
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

function switchLanguage(language) {
    currentLanguage = language;
    
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
curl -X POST "https://api.helixflow.com/v1/chat/completions" \\
  -H "Authorization: Bearer YOUR_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "gpt-3.5-turbo",
    "messages": [
      {"role": "user", "content": "Hello, world!"}
    ]
  }'`
    };
    
    if (codeExample) {
        codeExample.innerHTML = `<code>${examples[language]}</code>`;
    }
    
    // Update active button
    document.querySelectorAll('[onclick^="switchLanguage"]').forEach(btn => {
        btn.classList.remove('text-white');
        btn.classList.add('text-gray-400');
    });
    
    const activeButton = document.querySelector(`[onclick="switchLanguage('${language}')"]`);
    if (activeButton) {
        activeButton.classList.remove('text-gray-400');
        activeButton.classList.add('text-white');
    }
}

// Scroll Animations
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

// Form Handlers
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
        
        // Show success message (you can customize this)
        alert('Thank you for your submission! We\'ll get back to you soon.');
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
    alert('Demo video would play here. In a real implementation, this would open a video modal or navigate to a demo page.');
}

function startFreeTrial() {
    // Simulate starting free trial
    alert('Free trial signup would open here. This would typically navigate to a signup page or open a registration modal.');
}

function scheduleDemo() {
    // Simulate scheduling a demo
    alert('Demo scheduling would open here. This would typically open a calendar booking system or contact form.');
}

// Utility Functions
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

// Export functions for global access
window.HelixFlow = {
    startDemo,
    playDemo,
    startFreeTrial,
    scheduleDemo,
    switchLanguage,
    generateDemoResponse
};