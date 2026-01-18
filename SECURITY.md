# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| latest  | :white_check_mark: |
| < 1.0   | :x:                |

Only the latest release receives security updates. We recommend always running the most recent version.

## Reporting a Vulnerability

**Do not open a public GitHub issue for security vulnerabilities.**

Please report security vulnerabilities via email to **irfnhm@gmail.com** or use [GitHub's private vulnerability reporting](https://github.com/malwarebo/conductor/security/advisories/new).

Include:
- Description of the vulnerability
- Steps to reproduce
- Affected versions
- Potential impact

## Response Timeline

- **Acknowledgment**: Within 48 hours
- **Initial assessment**: Within 7 days
- **Resolution target**: Within 30 days for critical issues

## Disclosure Policy

We follow coordinated disclosure. We will:
1. Confirm receipt of your report
2. Investigate and validate the issue
3. Develop and test a fix
4. Release the fix and publish an advisory
5. Credit you (unless you prefer anonymity)

We ask that you do not publicly disclose the vulnerability until we have released a fix.

## Security Practices

This project:
- Uses bcrypt for password hashing
- Implements AES-256-GCM for encryption
- Pins CI/CD dependencies to specific commit SHAs
- Runs automated security scanning (govulncheck, gosec, Trivy)
- Follows least-privilege principles for GitHub Actions workflows
