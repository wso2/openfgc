# openfgc-portal

The OpenFGC Portal project is designed to create a comprehensive Consent Management Portal, leveraging the [OpenFGC Consent Management API](https://github.com/wso2/openfgc).

React 19 + TypeScript + Vite app using WSO2 Oxygen UI.

## Requirements

- Node.js 20.19+ (or 22.12+)
- Corepack enabled
- pnpm

## Package Manager

This project uses pnpm.

```bash
corepack enable
corepack prepare pnpm@10.6.5 --activate
```

If your machine cannot install global Corepack shims due permission restrictions, use `corepack pnpm` directly.

## Install

```bash
pnpm install
```

## Scripts

```bash
pnpm dev
pnpm lint
pnpm test
pnpm test:watch
pnpm test:coverage
pnpm build
pnpm preview
```

## Testing

Tests are written with [Vitest](https://vitest.dev/) and [React Testing Library](https://testing-library.com/react).

- **Test files**: Located in `src/__tests__/` with `.test.tsx` extension
- **Setup**: Global setup in `vitest.setup.ts` imports jest-dom matchers
- **Run tests**: `pnpm test` or `pnpm test:watch` for watch mode
- **Coverage**: `pnpm test:coverage` generates HTML coverage report in `coverage/`

## Project Structure

```
src/
├── components/       # Reusable UI components
├── features/         # Feature-level modules (pages, domains)
├── hooks/            # Custom React hooks
├── i18n/             # i18n initialization and locale resources
├── types/            # TypeScript interfaces and types
├── utils/            # Utility functions and helpers
├── __tests__/        # Test files
├── App.tsx           # Root component
├── main.tsx          # Entry point
└── index.css         # Global styles
```

## AI Instructions

This repository uses VS Code Copilot instruction files to keep AI-generated changes aligned with project and organization standards.

- Canonical cross-agent project rules: `AGENTS.md`
- Copilot always-on workspace instructions: `.github/copilot-instructions.md`
- Scoped instructions folder: `.github/instructions/`
- Oxygen UI generated reference: `.ai/oxygen-ui/AGENTS.md`

Recommended precedence:

1. `AGENTS.md` for shared project standards
2. `.github/copilot-instructions.md` for Copilot-specific defaults
3. `.github/instructions/*.instructions.md` for task and file-type specific rules
4. `.ai/oxygen-ui/AGENTS.md` for Oxygen component catalog/examples

Copilot instruction files are automatically discovered by Copilot Chat and applied based on their `applyTo` patterns.

## Internationalization

This project uses `i18next` and `react-i18next` for UI translations.

- Add locale resources under `src/i18n/resources/<locale>/`.
- Keep keys grouped by namespace (for example `common`) and feature intent (`app`, `forms`, `buttons`).
- In components, use `useTranslation` and keys instead of hardcoded user-facing text.
- Keep accessibility labels and user-visible messages localized as well.

Example:

```tsx
import { useTranslation } from 'react-i18next'

function Example(): React.JSX.Element {
  const { t } = useTranslation('common')

  return <h1>{t('app.title')}</h1>
}
```

## CI

GitHub Actions CI runs pnpm-based install, lint, and build checks on every push and pull request.
