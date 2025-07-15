# Rate Limiter Memory Bank

This memory bank serves as the centralized knowledge repository for the Rate-Limited API Key Manager project. It follows Claude Code's memory bank structure to maintain context and continuity across development sessions.

## Memory Bank Structure

### Core Documentation Files

1. **[projectbrief.md](./projectbrief.md)** - High-level project overview
   - Project mission and goals
   - Key stakeholders and users
   - Success metrics and constraints

2. **[productContext.md](./productContext.md)** - User flows and features
   - User personas and use cases
   - Feature specifications
   - API endpoints and workflows

3. **[techContext.md](./techContext.md)** - Technical implementation details
   - Architecture diagrams
   - Technology stack details
   - Database schemas and data flows

4. **[systemPatterns.md](./systemPatterns.md)** - Architecture patterns and conventions
   - Design patterns used
   - Coding conventions
   - Best practices and standards

5. **[activeContext.md](./activeContext.md)** - Current session work
   - Active development tasks
   - Open issues and blockers
   - Session-specific notes

6. **[progress.md](./progress.md)** - Historical milestones
   - Completed features
   - Architecture decisions
   - Version history

7. **[developerNotes.md](./developerNotes.md)** - Best practices and tips
   - Development setup
   - Common pitfalls
   - Performance optimization tips

## How to Use This Memory Bank

### For New Sessions
1. Start by reading `projectbrief.md` for project overview
2. Review `activeContext.md` for current work status
3. Check `techContext.md` for technical details relevant to your task

### For Feature Development
1. Consult `productContext.md` for user requirements
2. Reference `systemPatterns.md` for coding standards
3. Update `activeContext.md` with your progress

### For Troubleshooting
1. Check `developerNotes.md` for known issues
2. Review `techContext.md` for system architecture
3. Look at `progress.md` for historical context

## Maintenance Guidelines

- Update `activeContext.md` at the end of each session
- Add completed features to `progress.md`
- Document new patterns in `systemPatterns.md`
- Keep `developerNotes.md` updated with learnings

## Project Overview

The Rate-Limited API Key Manager is a production-ready system for managing API keys with sophisticated rate limiting capabilities. It provides:

- Multi-tier rate limiting (free, pro, enterprise)
- Real-time usage tracking and analytics
- Automatic billing and overage handling
- Distributed rate limiting with Redis sharding
- Comprehensive monitoring and alerting

Built with Go, PostgreSQL, Redis, and RabbitMQ, this system is designed to handle millions of requests while maintaining sub-millisecond rate limit checks.