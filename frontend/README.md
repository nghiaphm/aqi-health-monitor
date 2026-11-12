# AQI Health Monitor — Frontend

## Overview

The `frontend/` directory contains the web client for **AQI Health Monitor**, built with **Next.js 16 (App Router)**, **React 19**, **TypeScript 5**, **Tailwind CSS v4**, and **shadcn/ui**.

### Current Implementation Status

The frontend is currently at the **Project Scaffold & Design System Setup** stage:

- **Implemented**: Next.js 16 App Router configuration (`output: 'standalone'`), Tailwind CSS v4 theme tokens (`app/globals.css`), 17 reusable `shadcn/ui` primitives (`components/ui/*`), mobile breakpoint detection hook (`hooks/use-mobile.ts`), class merging utility (`lib/utils.ts`), and a production multi-stage `Dockerfile`.
- **Scaffolded (`0-byte` placeholders)**: Route pages under `app/(auth)` and `app/(dashboard)`, NextAuth route handler (`app/api/auth/[...nextauth]/route.ts`), feature components (`components/aqi`, `components/alerts`, `components/health-profile`, `components/locations`, `components/layout`), API client modules (`lib/api/*`), auth helpers (`lib/auth/*`), domain hooks (`lib/hooks/*`), state store (`store/auth-store.ts`), and TypeScript domain interfaces (`types/*`).

## Technology Stack

Verified from [`package.json`](./package.json) and [`components.json`](./components.json):

| Category | Package | Version | Purpose |
| :--- | :--- | :--- | :--- |
| **Framework** | `next` | `16.3.3` | React framework with App Router and standalone build output |
| **UI Runtime** | `react`, `react-dom` | `19.2.8` | Core React 19 UI library |
| **Language** | `typescript` | `^5` | Static type checking |
| **Styling** | `tailwindcss`, `@tailwindcss/postcss`, `tw-animate-css` | `^4`, `^1.4.0` | Utility-first CSS and animation utilities |
| **Component Primitives** | `shadcn`, `radix-ui` | `^4.19.0`, `^1.6.7` | Accessible UI primitives (`radix-nova` style) |
| **Class Utilities** | `clsx`, `tailwind-merge`, `class-variance-authority` | `^2.1.1`, `^3.6.0`, `^0.7.1` | Conditional class merging (`cn()`) and component variants |
| **Icons & Date Picker** | `lucide-react`, `react-day-picker`, `date-fns` | `^1.37.0`, `^10.0.1`, `^4.4.0` | Iconography and calendar date selection (`components/ui/calendar.tsx`) |
| **Linter** | `eslint`, `eslint-config-next` | `^9`, `16.3.3` | Code quality and Next.js / TypeScript linting |

> **Note**: Libraries mentioned in high-level architecture documents (such as `next-auth`, Leaflet/Mapbox, Recharts, or Zustand) are **not yet installed** in `package.json`.

## Frontend Architecture

The project is organized around the Next.js App Router with a modular directory layout separating routing (`app/`), presentation components (`components/`), business/API utilities (`lib/`), client state (`store/`), and shared TypeScript contracts (`types/`).

- **Path Alias**: `@/*` is mapped to `./*` in [`tsconfig.json`](./tsconfig.json).
- **shadcn/ui Aliases**: Configured in [`components.json`](./components.json) (`@/components`, `@/components/ui`, `@/lib`, `@/lib/utils`, `@/hooks`).

## Directory Structure

```text
frontend/
├── app/
│   ├── (auth)/
│   │   ├── login/page.tsx                  # Login page placeholder [0 bytes]
│   │   └── register/page.tsx               # Registration page placeholder [0 bytes]
│   ├── (dashboard)/
│   │   ├── layout.tsx                      # Dashboard shell layout placeholder [0 bytes]
│   │   ├── alerts/page.tsx                 # Alert history & settings page placeholder [0 bytes]
│   │   ├── dashboard/page.tsx              # Main AQI dashboard page placeholder [0 bytes]
│   │   ├── history/page.tsx                # Historical AQI chart page placeholder [0 bytes]
│   │   ├── locations/page.tsx              # Saved locations management page placeholder [0 bytes]
│   │   └── profile/page.tsx                # Health profile & threshold page placeholder [0 bytes]
│   ├── api/auth/[...nextauth]/
│   │   └── route.ts                        # NextAuth route handler placeholder [0 bytes]
│   ├── globals.css                         # Tailwind CSS v4 imports & oklch theme variables [Implemented]
│   ├── layout.tsx                          # Root HTML layout with Geist Sans & Mono fonts [Implemented]
│   └── page.tsx                            # Default Next.js starter landing page [Implemented]
├── components/
│   ├── alerts/
│   │   └── alert-list.tsx                  # Alert list component placeholder [0 bytes]
│   ├── aqi/
│   │   ├── aqi-badge.tsx                   # AQI level badge placeholder [0 bytes]
│   │   ├── aqi-gauge.tsx                   # AQI gauge visualization placeholder [0 bytes]
│   │   ├── aqi-history-chart.tsx           # Time-series chart placeholder [0 bytes]
│   │   └── aqi-map.tsx                     # Station/location map placeholder [0 bytes]
│   ├── health-profile/
│   │   ├── health-profile-form.tsx         # Health profile form placeholder [0 bytes]
│   │   └── threshold-slider.tsx            # Custom AQI threshold slider placeholder [0 bytes]
│   ├── layout/
│   │   ├── navbar.tsx                      # Top navigation bar placeholder [0 bytes]
│   │   └── sidebar.tsx                     # App sidebar navigation placeholder [0 bytes]
│   ├── locations/
│   │   └── location-picker.tsx             # GPS / map coordinate picker placeholder [0 bytes]
│   └── ui/                                 # 17 implemented shadcn/ui components
│       ├── alert.tsx
│       ├── badge.tsx
│       ├── button.tsx
│       ├── calendar.tsx
│       ├── card.tsx
│       ├── checkbox.tsx
│       ├── dialog.tsx
│       ├── field.tsx
│       ├── input.tsx
│       ├── label.tsx
│       ├── separator.tsx
│       ├── sheet.tsx
│       ├── sidebar.tsx
│       ├── skeleton.tsx
│       ├── slider.tsx
│       ├── table.tsx
│       └── tooltip.tsx
├── hooks/
│   └── use-mobile.ts                       # Responsive viewport hook (< 768px) [Implemented]
├── lib/
│   ├── api/                                # Backend API client placeholders [All 0 bytes]
│   │   ├── client.ts
│   │   ├── aqi.ts
│   │   ├── alerts.ts
│   │   ├── health-profile.ts
│   │   └── locations.ts
│   ├── auth/                               # Auth configuration placeholders [All 0 bytes]
│   │   ├── auth-client.ts
│   │   └── auth-options.ts
│   ├── hooks/                              # Domain hook placeholders [All 0 bytes]
│   │   ├── use-aqi.ts
│   │   ├── use-current-location.ts
│   │   └── use-push-subscription.ts
│   ├── constants.ts                        # Constants placeholder [0 bytes]
│   └── utils.ts                            # cn() Tailwind class merge helper [Implemented]
├── public/                                 # Static SVG assets
├── store/
│   └── auth-store.ts                       # Auth state store placeholder [0 bytes]
├── types/                                  # Domain TypeScript definition placeholders [All 0 bytes]
│   ├── alert.ts
│   ├── aqi.ts
│   ├── health-profile.ts
│   └── user.ts
├── components.json                         # shadcn/ui configuration
├── Dockerfile                              # Multi-stage standalone production Dockerfile
├── eslint.config.mjs                       # ESLint 9 flat configuration
├── next.config.ts                          # Next.js configuration (output: 'standalone')
├── package.json                            # Dependencies and npm scripts
├── postcss.config.mjs                      # PostCSS configuration (@tailwindcss/postcss)
└── tsconfig.json                           # TypeScript compiler configuration
```

## Routing and Page Organization

The App Router directory structure uses two route groups (`(auth)` and `(dashboard)`) alongside the root landing page:

| Route Path | Source File | Current Status |
| :--- | :--- | :--- |
| `/` | `app/page.tsx` | Default `create-next-app` welcome screen |
| `/login` | `app/(auth)/login/page.tsx` | Placeholder (`0 bytes`) |
| `/register` | `app/(auth)/register/page.tsx` | Placeholder (`0 bytes`) |
| `/dashboard` | `app/(dashboard)/dashboard/page.tsx` | Placeholder (`0 bytes`) |
| `/locations` | `app/(dashboard)/locations/page.tsx` | Placeholder (`0 bytes`) |
| `/history` | `app/(dashboard)/history/page.tsx` | Placeholder (`0 bytes`) |
| `/alerts` | `app/(dashboard)/alerts/page.tsx` | Placeholder (`0 bytes`) |
| `/profile` | `app/(dashboard)/profile/page.tsx` | Placeholder (`0 bytes`) |
| `/api/auth/[...nextauth]` | `app/api/auth/[...nextauth]/route.ts` | Placeholder (`0 bytes`) |

For the intended user journeys, wireframes, and design decisions governing these pages, refer to:
- [`../docs/03-USER-FLOWS.md`](../docs/03-USER-FLOWS.md)
- [`../docs/04-WIREFRAMES.md`](../docs/04-WIREFRAMES.md)
- [`../docs/05-DESIGN-DECISIONS.md`](../docs/05-DESIGN-DECISIONS.md)

## Components and Feature Modules

- **UI Primitives (`components/ui/*`)**: Fully implemented and ready for composition (`Alert`, `Badge`, `Button`, `Calendar`, `Card`, `Checkbox`, `Dialog`, `Field`, `Input`, `Label`, `Separator`, `Sheet`, `Sidebar`, `Skeleton`, `Slider`, `Table`, `Tooltip`).
- **Domain Feature Modules (`components/aqi`, `components/health-profile`, `components/locations`, `components/alerts`, `components/layout`)**: Placeholder files (`0 bytes`) created to match the planned component hierarchy.

## API Integration, State Management, and Authentication Flow

- **API Integration (`lib/api/*`)**: Currently **not implemented** (`client.ts`, `aqi.ts`, `alerts.ts`, `health-profile.ts`, and `locations.ts` are `0-byte` files). No HTTP requests are currently made from the frontend to the Go backend.
- **Backend Contract**: When implementing `lib/api/*`, requests will target the Go API server (`/api/v1/me`, `/api/v1/health-profile`, `/api/v1/locations`, and `/api/v1/aqi/current`) as specified in [`../docs/09-API_CONTRACT.md`](../docs/09-API_CONTRACT.md).
- **CORS / Proxy Consideration**: Neither CORS headers in the Go backend (`backend/cmd/api/main.go`) nor API rewrites in [`next.config.ts`](./next.config.ts) are configured yet. Developers implementing frontend-to-backend calls will need to enable Next.js `rewrites` or add a CORS middleware on the backend.
- **Authentication & State Management (`lib/auth/*`, `store/auth-store.ts`)**: Currently **not implemented** (`0-byte` files). Planned architecture delegates OIDC login to Keycloak and passes the resulting JWT as `Authorization: Bearer <token>` to the backend.

## Environment Variables

No environment variables are currently read inside the `frontend/` source files.

In the commented `frontend` service block of [`../docker-compose.yml`](../docker-compose.yml), the following variable is planned for pointing the client to the Go API server:

```dotenv
NEXT_PUBLIC_API_URL=http://localhost:8000
```

## Installation and Local Development

Open a terminal in the `frontend/` directory:

```bash
cd frontend
npm install
```

Start the Next.js development server:

```bash
npm run dev
```

Open `http://localhost:3000` in your browser.

## Linting and Testing

### ESLint

Run ESLint from the `frontend/` directory:

```bash
npm run lint
```

> **Known Lint Issue**: Running `npm run lint` currently reports 1 error (`react-hooks/set-state-in-effect`) at `hooks/use-mobile.ts:14` (generated by `shadcn/ui`).

### Automated Tests

No unit or integration test runner (such as Jest, Vitest, or Playwright) is currently configured in `package.json`.

## Build and Production Execution

### Local Production Build

[`next.config.ts`](./next.config.ts) is configured with `output: 'standalone'`.

```bash
cd frontend
npm run build
npm run start
```

### Docker Build

[`./Dockerfile`](./Dockerfile) uses a 3-stage build (`deps` → `builder` → `runner` on `node:20-alpine`) that packages `.next/standalone`, `.next/static`, and `public/`, exposing port `3000`:

```bash
cd frontend
docker build -t aqi-frontend:local .
docker run --rm -p 3000:3000 -e NEXT_PUBLIC_API_URL=http://localhost:8000 aqi-frontend:local
```
