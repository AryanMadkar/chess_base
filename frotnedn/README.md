# Chess Frontend (`frotnedn`)

React frontend for your chess backend in `chess/backend`.

## Stack

- React 19 + Vite 8
- `chess.js` for move validation and board state analysis
- `react-chessboard` for board UI

## Run

1. Start backend first (default expected at `http://127.0.0.1:3000`).
2. In this folder:

```bash
npm install
npm run dev
```

## API URL

For local development, no `.env` is required because Vite proxies `/api` and `/ws` to `http://127.0.0.1:3000`.

If you want to call a different backend URL, set it in `.env`:

```bash
VITE_API_URL=http://127.0.0.1:3000
```

If omitted, app uses same-origin (`""`) and relies on Vite proxy in dev mode.

## Features

- Create game
- Join by `gameId`
- Join matchmaking queue
- Move pieces on a chessboard (drag and drop)
- Resign game
- Live game metadata panel (status, turn, check, legal move count)
