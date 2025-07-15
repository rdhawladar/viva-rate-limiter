# Project Memory Initialization Command

Comprehensive project analysis and memory bank creation for maintaining context across development sessions.

## Command: `/project:memory:init`

Initialize or refresh the project memory bank by conducting deep codebase analysis and creating structured documentation domains.

## Context
- Full codebase analysis capabilities
- Write access to create documentation structure
- Integration with existing project patterns
- Memory bank organization following established conventions

## Memory Initialization Workflow

### Phase 1: Codebase Discovery and Analysis
```bash
# Analyze project structure and architecture
1. Examine package.json, tsconfig.json, and configuration files
2. Map out directory structure and file organization
3. Identify technology stack and dependencies
4. Analyze build/development commands
5. Review existing documentation (README, CLAUDE.md)
```

### Phase 2: Architecture Pattern Recognition
```bash
# Deep dive into code patterns and conventions
1. Examine state management patterns (Redux, Context, etc.)
2. Identify navigation and routing architecture
3. Analyze component organization and UI patterns
4. Review service layer and API integration
5. Document real-time communication setup
6. Identify testing and development practices
```

### Phase 3: Memory Bank Structure Creation
```bash
# Create comprehensive memory bank in ./docs/memory_bank/
mkdir -p docs/memory_bank

# Core documentation files
touch docs/memory_bank/README.md           # Directory overview
touch docs/memory_bank/projectbrief.md     # High-level overview
touch docs/memory_bank/productContext.md   # User flows and features
touch docs/memory_bank/techContext.md      # Technical implementation
touch docs/memory_bank/systemPatterns.md   # Architecture and patterns
touch docs/memory_bank/activeContext.md    # Current session work
touch docs/memory_bank/progress.md         # Historical milestones
touch docs/memory_bank/developerNotes.md   # Best practices and tips
```

### Phase 4: Content Generation by Domain

#### **Project Brief Domain**
- Project purpose and value proposition
- Core features and capabilities
- Technology stack overview
- Current version and status

#### **Product Context Domain**
- Target user personas and use cases
- Key user flows and scenarios
- Feature specifications
- Business rules and constraints

#### **Technical Context Domain**
- Complete technology stack details
- Development environment setup
- Build and deployment processes
- Performance considerations
- Security implementation

#### **System Patterns Domain**
- Architectural design patterns
- Code organization conventions
- Component development patterns
- State management patterns
- API integration patterns
- Testing strategies

#### **Active Context Domain**
- Current session goals and tasks
- Work in progress status
- Recent findings and insights
- Next session priorities
- Development context notes

#### **Progress Domain**
- Completed milestones and features
- Major technical achievements
- Lessons learned from implementation
- Historical development context

#### **Developer Notes Domain**
- Critical development practices
- Common commands and workflows
- Troubleshooting guides
- Performance optimization notes
- Security best practices
- Code style guidelines

### Phase 5: CLAUDE.md Enhancement
```bash
# Create or update CLAUDE.md with essential context
1. Essential development commands
2. Architecture overview for future instances
3. Key patterns and conventions
4. Integration points and workflows
```

## Memory Domain Validation

### ✅ **Completeness Check**
- [ ] All 8 memory bank files created
- [ ] Each domain has comprehensive, non-overlapping content
- [ ] Technical patterns documented with examples
- [ ] Development workflows clearly explained
- [ ] Current project status captured

### ✅ **Quality Assurance**
- [ ] Information is project-specific (not generic)
- [ ] Code examples use actual project patterns
- [ ] Commands are verified and tested
- [ ] Architecture reflects actual codebase structure
- [ ] Dependencies and integrations documented

### ✅ **Organizational Structure**
- [ ] Clear separation of concerns between domains
- [ ] Logical information hierarchy
- [ ] Easy navigation and reference
- [ ] Consistent documentation format
- [ ] Future session context preserved

## Usage Examples

```bash
# Initialize memory bank for new project analysis
/project:memory:init

# Refresh memory bank after major changes
/project:memory:init
```

## Success Criteria

After successful execution, the project will have:

1. **Complete Memory Bank**: 8 structured documentation files covering all aspects
2. **Enhanced CLAUDE.md**: Essential context for future Claude instances  
3. **Architecture Documentation**: Clear patterns and conventions
4. **Development Guide**: Practical commands and workflows
5. **Context Preservation**: Current status and progress tracking

## Integration Notes

This command complements existing project management workflows by:
- Providing deep context for `/project:resume` operations
- Supporting `/project:status` with historical context
- Enabling efficient `/project:epic` feature additions
- Maintaining consistency across development sessions

## Output Structure

```
./docs/memory_bank/
├── README.md               # Directory overview and rules
├── projectbrief.md         # Project overview and purpose
├── productContext.md       # Features and user flows  
├── techContext.md          # Technology and implementation
├── systemPatterns.md       # Architecture and conventions
├── activeContext.md        # Current work and session notes
├── progress.md             # Completed milestones
└── developerNotes.md       # Best practices and troubleshooting
```

## Memory Refresh Protocol

The memory bank should be refreshed when:
- Major architectural changes occur
- New technology integrations are added
- Significant features are completed
- Development patterns evolve
- Before starting new major initiatives

This ensures the memory bank remains current and valuable for project continuity.