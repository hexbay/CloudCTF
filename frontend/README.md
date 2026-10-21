# CloudCTF Frontend

Unified React + Vite frontend for CloudCTF.

The project keeps admin and player as separate application domains while sharing build tooling, request code, types, and common utilities.

## Structure

```text
src/
  apps/
    admin/
      layouts/
      pages/
      routes.ts
    player/
      layouts/
      pages/
      routes.ts
  shared/
    api/
    pages/
    types/
  styles/
```

## Routes

- Admin: `/admin/*`
- Admin login: `/admin/login`
- Player: `/*`
- Player login: `/login`

The Vite dev server proxies `/api`, `/api-admin`, and `/static` to `http://127.0.0.1:8005`.

## Commands

```bash
npm install
npm run dev
npm run build
```

## Migration Notes

Move old Umi pages gradually:

- `front-admin/src/pages/*` -> `frontend/src/apps/admin/pages/*`
- `front-user/src/pages/*` -> `frontend/src/apps/player/pages/*`
- duplicated API clients -> `frontend/src/shared/api/*`
- duplicated types -> `frontend/src/shared/types/*`

