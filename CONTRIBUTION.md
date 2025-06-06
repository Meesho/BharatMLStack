# Contributing to BharatMLStack

Thank you for your interest in contributing to BharatMLStack! This document provides guidelines and instructions for contributing to the project. By participating in this project, you agree to abide by our [Code of Conduct](CODE_OF_CONDUCT.md) and the terms of our [Business Source License](../LICENSE.md).

## 📊 Table of Contents

- [Contributing to BharatMLStack](#contributing-to-BharatMLStack)
  - [📊 Table of Contents](#-table-of-contents)
  - [🌐 Join the Community](#-join-the-community)
  - [🧩 Types of Contributions](#-types-of-contributions)
    - [Code Contributions](#code-contributions)
    - [Non-Code Contributions](#non-code-contributions)
  - [⚙️ Development Workflow](#️-development-workflow)
    - [Setting Up Your Development Environment](#setting-up-your-development-environment)
    - [Development Cycle](#development-cycle)
  - [📝 Contribution Guidelines](#-contribution-guidelines)
    - [General Guidelines](#general-guidelines)
    - [Issue Guidelines](#issue-guidelines)
    - [Feature Requests](#feature-requests)
  - [💻 Code Style](#-code-style)
  - [🧪 Testing Requirements](#-testing-requirements)
  - [📖 Documentation](#-documentation)
  - [🔄 Submitting Pull Requests](#-submitting-pull-requests)
  - [👀 Review Process](#-review-process)
  - [🎉 Recognition](#-recognition)
  - [📜 Licensing](#-licensing)

## 🌐 Join the Community

The best way to connect with the BharatMLStack community is through our [Discord server](https://discord.gg/474wHtfm). Here, you can:

- Ask questions about usage or development
- Discuss feature ideas and roadmap items
- Coordinate with other contributors
- Get help with setting up your development environment
- Connect with the core team

We encourage all contributors to join our Discord community to stay updated on project developments.

## 🧩 Types of Contributions

We welcome various types of contributions:

### Code Contributions
- New features implementation
- Bug fixes
- Performance optimizations
- Code refactoring
- Test coverage improvements

### Non-Code Contributions
- Documentation improvements
- Design proposals
- Use case writeups
- Bug reports
- Feature requests
- Answering questions in issues or Discord

## ⚙️ Development Workflow

### Setting Up Your Development Environment

1. **Fork the Repository**
   - Visit the [BharatMLStack GitHub repository](https://github.com/Meesho/BharatMLStack)
   - Click the "Fork" button to create your own copy of the repository

2. **Clone Your Fork**
   ```bash
   git clone https://github.com/Meesho/BharatMLStack.git
   cd BharatMLStack
   ```

3. **Add the Upstream Remote**
   ```bash
   git remote add upstream https://github.com/Meesho/BharatMLStack.git
   ```

4. **Install Dependencies**
   - Ensure you have Go 1.22+ installed
   - Follow the setup instructions in the Quick Start Guide

5. **Run the Local Environment**
   ```bash
   cd <service>/quick-start
   ./start.sh
   ```

### Development Cycle

1. **Create a Feature Branch**
   ```bash
   git checkout -b feature/<service>-your-feature-name
   ```

2. **Make Your Changes**
   - Implement your feature or fix
   - Write or update tests
   - Update documentation as needed

3. **Run Tests Locally**
   ```bash
   go test ./...
   ```

4. **Commit Your Changes**
   ```bash
   git add .
   git commit -m "feat: brief description of your changes"
   ```
   
   Please follow the [Conventional Commits](https://www.conventionalcommits.org/) format:
   - `feat:` for new features
   - `fix:` for bug fixes
   - `docs:` for documentation changes
   - `test:` for changes to tests
   - `refactor:` for refactoring code without changing functionality
   - `perf:` for performance improvements
   - `chore:` for changes to the build process or auxiliary tools

5. **Keep Your Branch Updated**
   ```bash
   git fetch upstream
   git rebase upstream/main
   ```

6. **Push Your Changes**
   ```bash
   git push origin feat/<service>-your-feature-name
   ```

## 📝 Contribution Guidelines

### General Guidelines

1. **One Issue, One PR** - Keep your pull requests focused on addressing a single concern.

2. **Discuss Large Changes** - For significant features or changes, please discuss them in an issue or on Discord first.

3. **Be Respectful** - Practice respectful communication in issues, pull requests, and community spaces.

4. **Provide Context** - When submitting issues or pull requests, provide as much context as possible.

### Issue Guidelines

1. **Check Existing Issues** - Before creating a new issue, search for related issues to avoid duplicates.

2. **Issue Templates** - Use the provided issue templates when available.

3. **Clear Problem Statement** - Clearly describe the issue, including steps to reproduce for bugs.

### Feature Requests

1. **Describe the Need** - Explain the use case and why it's valuable to the project.

2. **Propose Solutions** - If you have ideas on how to implement the feature, include them.

## 💻 Code Style

Golang services follows these code style guidelines:

1. **Go Code**
   - Follow the [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
   - Use `gofmt` to format your code
   - Ensure code passes `golint` and `go vet`

2. **Documentation**
   - Write clear comments for public APIs
   - Update documentation when changing functionality

3. **Git Commits**
   - Write clear, meaningful commit messages
   - Follow Conventional Commits format

4. **Tests**
   - Write tests for all new features and bug fixes
   - Maintain or improve test coverage

## 🧪 Testing Requirements

All contributions should maintain or improve test coverage:

1. **Unit Tests** - All new code should have corresponding unit tests.

2. **Integration Tests** - New features should include integration tests where appropriate.

3. **Benchmark Tests** - Performance-critical code should include benchmarks.

4. **Test Coverage** - Aim to maintain or improve overall test coverage percentage.

## 📖 Documentation

Documentation is crucial for BharatMLStack's usability:

1. **Code Documentation**
   - Document all exported functions, types, and constants
   - Include examples where helpful

2. **User Documentation**
   - Update README.md and docs/ when adding features
   - Consider adding examples or tutorials for non-trivial features

3. **Architecture Documentation**
   - For significant changes, update architecture documentation

## 🔄 Submitting Pull Requests

1. **Create a Pull Request**
   - Go to your fork on GitHub and create a new pull request
   - Use our pull request template when available

2. **PR Description**
   - Clearly describe what your PR does
   - Link to related issues using GitHub keywords (e.g., "Fixes #123")
   - Include any necessary background information

3. **PR Checklist**
   - Ensure your code follows our style guidelines
   - Verify all tests pass
   - Update documentation as needed
   - Add your changes to the CHANGELOG.md if applicable

## 👀 Review Process

1. **Code Review**
   - At least one core team member will review your PR
   - Address reviewer feedback promptly
   - Be open to suggestions and changes

2. **Continuous Integration**
   - All PRs must pass CI checks
   - Fix any CI failures before requesting re-review

3. **Merge Process**
   - Once approved, a maintainer will merge your PR
   - Some PRs may require multiple reviews or approvals

## 🎉 Recognition

We value all contributions to BharatMLStack! Contributors will be recognized in the following ways:

- Listed in GitHub contributors
- Mentioned in release notes for significant contributions
- Recognized in the community Discord

## 📜 Licensing

By contributing to BharatMLStack, you agree that your contributions will be licensed under the project's [Business Source License 1.1](../LICENSE.md). This license allows for open source development while providing some commercial protections for the project. Key points to understand:

- All contributions become part of the licensed work
- The license will automatically convert to Apache License 2.0 after the change date (3 years from publication)
- The license includes restrictions on using BharatMLStack as a competing service

Please review the full license before contributing to understand how your work may be used.

---

Thank you for contributing to BharatMLStack! Your efforts help make this project better for everyone.

Join our [Discord community](https://discord.gg/474wHtfm) for discussions, questions, and collaborative development.
