/**
 * Security utilities for input validation and XSS prevention
 */

export class Security {
    /**
     * Escapes HTML special characters to prevent XSS
     * @param {string} str - String to escape
     * @returns {string} Escaped string
     */
    static escapeHtml(str) {
        if (typeof str !== 'string') return '';
        
        const div = document.createElement('div');
        div.textContent = str;
        return div.innerHTML;
    }
    
    /**
     * Validates and sanitizes search input
     * @param {string} input - Search query
     * @returns {string} Sanitized search query
     */
    static sanitizeSearchInput(input) {
        if (typeof input !== 'string') return '';
        
        // Trim and limit length
        let sanitized = input.trim().substring(0, 100);
        
        // Remove potentially dangerous characters
        sanitized = sanitized.replace(/[<>\"'`]/g, '');
        
        // Remove SQL-like patterns
        sanitized = sanitized.replace(/(\-\-|\/\*|\*\/|;)/g, '');
        
        return sanitized;
    }
    
    /**
     * Validates file paths (for local files only)
     * @param {string} path - File path to validate
     * @returns {boolean} True if path appears safe
     */
    static isValidFilePath(path) {
        if (typeof path !== 'string') return false;
        
        // Check for path traversal attempts
        if (path.includes('..') || path.includes('~')) {
            return false;
        }
        
        // Check for suspicious patterns
        const dangerousPatterns = [
            /\.\.\//g,
            /\.\.\\/g,
            /%2e%2e/gi,
            /%252e%252e/gi,
            /\x00/g,
        ];
        
        for (const pattern of dangerousPatterns) {
            if (pattern.test(path)) {
                return false;
            }
        }
        
        return true;
    }
    
    /**
     * Validates playlist names
     * @param {string} name - Playlist name
     * @returns {string} Sanitized playlist name
     */
    static sanitizePlaylistName(name) {
        if (typeof name !== 'string') return 'Untitled';
        
        // Remove any HTML tags
        name = name.replace(/<[^>]*>/g, '');
        
        // Limit length
        name = name.substring(0, 50);
        
        // Remove control characters
        name = name.replace(/[\x00-\x1F\x7F]/g, '');
        
        return name.trim() || 'Untitled';
    }
    
    /**
     * Creates a safe text node (alternative to innerHTML)
     * @param {string} text - Text content
     * @returns {Text} Safe text node
     */
    static createTextNode(text) {
        return document.createTextNode(text);
    }
    
    /**
     * Safely sets element content
     * @param {HTMLElement} element - Target element
     * @param {string} content - Content to set
     */
    static setTextContent(element, content) {
        if (element && typeof content === 'string') {
            element.textContent = content;
        }
    }
    
    /**
     * Creates DOM elements safely without innerHTML
     * @param {string} tag - HTML tag name
     * @param {Object} attributes - Element attributes
     * @param {string|Node} content - Element content
     * @returns {HTMLElement} Created element
     */
    static createElement(tag, attributes = {}, content = null) {
        const element = document.createElement(tag);
        
        // Set attributes safely
        for (const [key, value] of Object.entries(attributes)) {
            if (key === 'className') {
                element.className = value;
            } else if (key === 'dataset') {
                Object.assign(element.dataset, value);
            } else if (key.startsWith('on')) {
                // Skip event handlers in attributes for security
                console.warn('Event handlers should be added with addEventListener');
            } else {
                element.setAttribute(key, String(value));
            }
        }
        
        // Add content safely
        if (content) {
            if (typeof content === 'string') {
                element.textContent = content;
            } else if (content instanceof Node) {
                element.appendChild(content);
            }
        }
        
        return element;
    }
    
    /**
     * Validates numeric input
     * @param {any} value - Value to validate
     * @param {number} min - Minimum value
     * @param {number} max - Maximum value
     * @returns {number} Validated number or default
     */
    static validateNumber(value, min = 0, max = 100) {
        const num = parseFloat(value);
        if (isNaN(num)) return min;
        return Math.max(min, Math.min(max, num));
    }
    
    /**
     * Sanitizes URL for safe use
     * @param {string} url - URL to sanitize
     * @returns {string|null} Sanitized URL or null if invalid
     */
    static sanitizeUrl(url) {
        if (typeof url !== 'string') return null;
        
        try {
            const parsed = new URL(url);
            
            // Only allow http(s) and file protocols
            if (!['http:', 'https:', 'file:'].includes(parsed.protocol)) {
                return null;
            }
            
            return parsed.toString();
        } catch {
            return null;
        }
    }
    
    /**
     * Content Security Policy for dynamic content
     * @returns {string} CSP header value
     */
    static getCSPHeader() {
        return [
            "default-src 'self'",
            "script-src 'self' 'unsafe-inline'", // Wails requires unsafe-inline
            "style-src 'self' 'unsafe-inline'",
            "img-src 'self' data: blob:",
            "media-src 'self' blob:",
            "connect-src 'self' http: https:",
            "font-src 'self'",
            "object-src 'none'",
            "base-uri 'self'",
            "form-action 'self'",
            "frame-ancestors 'none'",
        ].join('; ');
    }
}

// Export validation functions for easy use
export const {
    escapeHtml,
    sanitizeSearchInput,
    isValidFilePath,
    sanitizePlaylistName,
    createTextNode,
    setTextContent,
    createElement,
    validateNumber,
    sanitizeUrl,
} = Security;