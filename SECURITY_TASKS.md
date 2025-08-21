# Security Tasks for WinRamp

## Overview
This document outlines security vulnerabilities identified during code review and provides actionable tasks to remediate them. Tasks are organized by priority level.

## Task Status Legend
- 🔴 **Critical** - Address immediately
- 🟠 **High** - Address within current sprint
- 🟡 **Medium** - Address within next release
- 🟢 **Low** - Address as time permits
- ✅ **Completed** - Task has been resolved

---

## 🔴 Critical Security Tasks

### 1. Fix Path Traversal Vulnerability in Album Art Storage
**Location**: `internal/library/scanner.go:342-358`
**Issue**: Unsanitized artist/album names used in file path construction could allow writing files outside intended directories.

**Tasks**:
- [ ] Move sanitization before path construction in `saveAlbumArt()` function
- [ ] Validate that final path is within the cache directory using `filepath.Clean()` and `filepath.Rel()`
- [ ] Add unit tests for path traversal attempts (e.g., "../../../etc/passwd")
- [ ] Implement path validation helper function for reuse across codebase

**Implementation**:
```go
// Add validation before line 350
cleanedPath := filepath.Clean(filepath.Join(cacheDir, filename))
if !strings.HasPrefix(cleanedPath, cacheDir) {
    return "", fmt.Errorf("invalid path: potential traversal attempt")
}
```

### 2. Strengthen SQL Query Construction
**Location**: `internal/infrastructure/db/track_repository.go:136-141`
**Issue**: String concatenation in LIKE queries could potentially lead to SQL injection.

**Tasks**:
- [ ] Review all database queries for proper parameterization
- [ ] Implement query builder pattern for complex searches
- [ ] Add input validation for search queries (length limits, character restrictions)
- [ ] Create SQL injection test suite
- [ ] Document safe query patterns for team

---

## 🟠 High Priority Security Tasks

### 3. Implement Frontend Input Validation
**Location**: `frontend/src/main.js` (multiple locations)
**Issue**: No input sanitization before sending to backend, innerHTML usage could lead to XSS.

**Tasks**:
- [ ] Replace all `innerHTML` usage with safe alternatives (`textContent`, `createElement`)
- [ ] Implement input validation library (e.g., DOMPurify)
- [ ] Add Content Security Policy headers
- [ ] Create validation functions for all user inputs
- [ ] Add XSS prevention tests

**Implementation**:
```javascript
// Replace line 37
const app = document.getElementById('app');
// Use createElement and appendChild instead of innerHTML
```

### 4. Secure Configuration File Permissions
**Location**: `internal/config/config.go:317`
**Issue**: Config directories created with world-readable permissions (0755).

**Tasks**:
- [ ] Change directory permissions to 0700 (owner only)
- [ ] Change file permissions to 0600 for sensitive configs
- [ ] Implement configuration encryption for sensitive values
- [ ] Add configuration validation on startup
- [ ] Document secure configuration practices

### 5. Fix Buffer Handling in MP3 Decoder
**Location**: `internal/audio/decoder/mp3.go:114-116`
**Issue**: Unchecked bit shifting operations could cause buffer overflow with malformed files.

**Tasks**:
- [ ] Add bounds checking before bit operations
- [ ] Implement safe integer conversion functions
- [ ] Add fuzzing tests for decoder with malformed files
- [ ] Implement file format validation before decoding
- [ ] Add error recovery mechanisms

---

## 🟡 Medium Priority Security Tasks

### 6. Remove Hardcoded URLs and Secrets
**Location**: `internal/network/streaming.go:283-297`
**Issue**: Hardcoded radio station URLs could be replaced or used for tracking.

**Tasks**:
- [ ] Move default stations to configuration file
- [ ] Implement station verification mechanism
- [ ] Add HTTPS enforcement for stream URLs
- [ ] Create station management interface
- [ ] Implement URL signature verification

---

## 🟢 Low Priority Security Tasks

### 7. Implement Comprehensive Logging
**Tasks**:
- [ ] Add security event logging (failed auth, suspicious paths, etc.)
- [ ] Implement log rotation and retention policies
- [ ] Add monitoring for security events
- [ ] Create security dashboard
- [ ] Implement alerting for critical events

### 8. Dependency Management
**Tasks**:
- [ ] Audit all dependencies for known vulnerabilities
- [ ] Implement automated dependency scanning
- [ ] Create dependency update policy
- [ ] Document dependency review process
- [ ] Set up security advisories monitoring

### 9. Testing and Validation
**Tasks**:
- [ ] Create comprehensive security test suite
- [ ] Implement penetration testing schedule
- [ ] Add security checks to CI/CD pipeline
- [ ] Create security regression tests
- [ ] Document security testing procedures

---

## Implementation Guidelines

### For Each Task:
1. Create a feature branch named `security/task-description`
2. Implement the fix with appropriate tests
3. Update documentation as needed
4. Request security-focused code review
5. Test in staging environment
6. Deploy with monitoring enabled

### Security Review Checklist:
- [ ] Input validation implemented
- [ ] Output encoding applied
- [ ] Authentication/authorization checked
- [ ] Sensitive data encrypted
- [ ] Logging implemented (without sensitive data)
- [ ] Error handling doesn't leak information
- [ ] Tests cover security scenarios
- [ ] Documentation updated

### Testing Requirements:
- Unit tests for security functions
- Integration tests for security flows
- Negative test cases (invalid inputs, attacks)
- Performance impact assessment
- Regression test suite updated

---

## Priority Timeline

### Sprint 1 (Immediate)
- All Critical tasks (1-2)
- High priority tasks 3-4

### Sprint 2 (Next 2 weeks)
- High priority task 5
- Medium priority task 6
- Begin low priority task 7

### Sprint 3 (Next month)
- Low priority tasks 7-9
- Security testing and validation

### Ongoing
- Regular security audits
- Dependency updates
- Security monitoring
- Team security training

---

## Contact and Resources

**Security Team Contact**: security@winramp.example.com
**Security Documentation**: `/docs/security/`
**Vulnerability Reporting**: Use private issue tracker with `security` label

### Useful Security Resources:
- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [Go Security Best Practices](https://github.com/OWASP/Go-SCP)
- [CWE Database](https://cwe.mitre.org/)
- [NIST Cybersecurity Framework](https://www.nist.gov/cyberframework)

---

## Revision History

| Date | Version | Changes | Author |
|------|---------|---------|--------|
| 2025-08-20 | 1.0 | Initial security assessment and task creation | Security Review |
| 2025-08-20 | 1.1 | Removed incorrect SSRF vulnerability (Task 2) - Not applicable to local desktop apps | Security Review |
| 2025-08-20 | 1.2 | Removed network streaming authentication (Task 6) - WinRamp doesn't run a server | Security Review |
| 2025-08-20 | 1.3 | Removed Security Hardening section (Task 8) - Web service hardening not applicable to desktop apps | Security Review |

---

*This document should be reviewed and updated regularly as tasks are completed and new security concerns are identified.*