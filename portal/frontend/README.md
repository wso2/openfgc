<!--
 * Copyright (c) 2026, WSO2 LLC. (https://www.wso2.com).
 *
 * WSO2 LLC. licenses this file to you under the Apache License,
 * Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 -->

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

If your machine cannot install global Corepack shims due to permission restrictions, use `corepack pnpm` directly.

## Install

```bash
pnpm install
```

## Environment

Create a local `.env` file from `.env.example` before running or building the portal.

| Variable | Description | Example |
| --- | --- | --- |
| `VITE_API_BASE_URL` | Required base URL for the OpenFGC Portal backend API. Vite embeds this value at build time. | `http://localhost:8080` |

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

- **Test files**: Located in `src/__tests__/` with `.test.ts`/`.test.tsx` extensions
- **Setup**: Global setup in `vitest.setup.ts` imports jest-dom matchers
- **Run tests**: `pnpm test` or `pnpm test:watch` for watch mode
- **Coverage**: `pnpm test:coverage` generates HTML coverage report in `coverage/`

## Project Structure

```text
src/
├── components/       # Reusable UI components
├── features/         # Feature-level modules (pages, domains)
├── hooks/            # Custom React hooks
├── i18n/             # i18n initialization and locale resources
├── types/            # TypeScript interfaces and types
├── utils/            # Utility functions and helpers
├── __tests__/        # Test files
├── App.tsx           # Root component
└── main.tsx          # Entry point
```

## AI Instructions

This repository uses VS Code Copilot instruction files to keep AI-generated changes aligned with project and organization standards.

Paths below are relative to the repository root.

- Frontend standards: `portal/frontend/AGENTS.md`
- Copilot workspace entrypoint: `.github/copilot-instructions.md`
- Scoped instructions folder: `.github/instructions/`
- Frontend scope mapping: `portal/frontend/**` -> `.github/instructions/portal-frontend.instructions.md`
- Oxygen UI generated reference: `portal/frontend/.ai/oxygen-ui/AGENTS.md`

Recommended precedence:

1. `portal/frontend/AGENTS.md` for shared frontend standards
2. `.github/copilot-instructions.md` for Copilot-specific defaults
3. `.github/instructions/*.instructions.md` for task and file-type-specific rules
4. `portal/frontend/.ai/oxygen-ui/AGENTS.md` for Oxygen component catalog/examples

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

GitHub Actions CI runs pnpm-based install, lint, and build checks on every push and pull request on portal/frontend directory.
