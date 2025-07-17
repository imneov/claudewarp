# ClaudeWarp Documentation

Official documentation website for ClaudeWarp, built with [Docusaurus](https://docusaurus.io/).

## Overview

This documentation site provides comprehensive guides, API references, and tutorials for ClaudeWarp - a full-featured bridge that connects Claude with chat platforms.

## Development

### Installation

```bash
cd docs
yarn install
```

### Local Development

```bash
yarn start
```

Starts a local development server at `http://localhost:3000` with hot reload.

### Build

```bash
yarn build
```

Generates static content for production deployment.

### Deployment

For GitHub Pages deployment:

```bash
GIT_USER=<Your GitHub username> yarn deploy
```

## Structure

- `docs/` - Markdown documentation files
- `src/pages/` - React pages and components
- `static/` - Static assets (images, icons, etc.)
- `docusaurus.config.ts` - Site configuration

## Features

- 🎨 FoalTS-inspired design with blue gradient theme
- 📱 Responsive design for all devices
- 🔍 Full-text search capability
- 🌐 Multi-language support (Chinese/English)
- ⚡ Fast static site generation
