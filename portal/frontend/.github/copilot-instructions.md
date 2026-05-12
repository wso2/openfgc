---
applyTo: '**'
---

# OpenFGC Portal Essential Instructions

React + TypeScript + Vite project using Oxygen UI.

Also follow `AGENTS.md` for shared project standards. Use `.ai/oxygen-ui/AGENTS.md` for Oxygen UI component-specific reference.

## Core Rules

- Import UI components only from `@wso2/oxygen-ui` (never `@mui/material`).
- Use `OxygenUIThemeProvider` at app root.
- Use `sx` with theme tokens; avoid hardcoded colors/spacing and inline styles.
- Use functional components only.
- Keep components small and single-purpose; extract reusable logic into hooks.
- Avoid prop drilling; prefer context/state management.
- No `any`; use `unknown` or generics.
- Add explicit return types on function signatures.
- Prefer interfaces for object shapes.
- Do not disable ESLint rules.

## Naming

- Components: `PascalCase.tsx` (default export, one per file).
- Logic/utils: `camelCase.ts`.
- Variables/functions: `camelCase`.
- Interfaces/types: `PascalCase`.
- Constants: `UPPER_SNAKE_CASE`.
- Folders: `kebab-case`.

## Testing

- Every component/hook should have tests.
- Place tests in `src/__tests__/` using `*.test.tsx`.
- Use Vitest + React Testing Library.
- Test happy and error paths; mock network requests.

## i18n

- Externalize all user-facing copy to i18n resource files; avoid hardcoded UI strings.
- Use stable, descriptive translation keys and keep naming consistent across namespaces.
- Provide English defaults/fallbacks for new keys, use locale-aware formatting (date, time, number, currency), and preserve graceful missing-key behavior.
- Add or update tests for translated rendering and fallback behavior when introducing i18n changes.

## Structure

- Keep code under `src/components`, `src/features`, `src/hooks`, `src/types`, `src/utils`, `src/__tests__`.

## Pre-Commit Checks

- `pnpm lint`
- `pnpm format`
- `pnpm test`
- `pnpm build`
