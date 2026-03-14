import { useEffect, useMemo, useRef, useState } from 'react'
import { Chess } from 'chess.js'
import { Chessboard } from 'react-chessboard'
import './App.css'

const API_BASE = (import.meta.env.VITE_API_URL || '').replace(/\/$/, '')
const START_FEN = 'rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1'
const FILES = ['a', 'b', 'c', 'd', 'e', 'f', 'g', 'h']

function findKingSquare(chess, side) {
  const board = chess.board()
  for (let rankIndex = 0; rankIndex < board.length; rankIndex += 1) {
    for (let fileIndex = 0; fileIndex < board[rankIndex].length; fileIndex += 1) {
      const piece = board[rankIndex][fileIndex]
      if (piece && piece.type === 'k' && piece.color === side[0]) {
        return `${FILES[fileIndex]}${8 - rankIndex}`
      }
    }
  }
  return ''
}

function createWsUrl(playerId, gameId) {
  const base = API_BASE || window.location.origin
  const normalized = base.replace(/^http/i, 'ws').replace(/\/$/, '')
  const params = new URLSearchParams()
  if (playerId) params.set('playerId', playerId)
  if (gameId) params.set('gameId', gameId)
  return `${normalized}/ws/game?${params.toString()}`
}

async function apiRequest(path, options = {}) {
  const response = await fetch(`${API_BASE}${path}`, {
    headers: {
      'Content-Type': 'application/json',
      ...(options.headers || {}),
    },
    ...options,
  })
  const text = await response.text()
  const data = text ? JSON.parse(text) : null
  if (!response.ok) {
    throw new Error(data?.error || `Request failed with ${response.status}`)
  }
  return data
}

function App() {
  const [playerId, setPlayerId] = useState(() => `player-${crypto.randomUUID().slice(0, 8)}`)
  const [joinGameId, setJoinGameId] = useState('')
  const [game, setGame] = useState(null)
  const [role, setRole] = useState('')
  const [info, setInfo] = useState('Create a game or join an existing game to start.')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)
  const [socketState, setSocketState] = useState('connecting')
  const [selectedSquare, setSelectedSquare] = useState('')
  const [legalTargets, setLegalTargets] = useState([])

  const wsRef = useRef(null)
  const reconnectTimerRef = useRef(null)
  const previousMovesRef = useRef(0)

  const boardChess = useMemo(() => new Chess(game?.board || START_FEN), [game?.board])

  const analysis = useMemo(() => {
    const inCheck = boardChess.inCheck()
    return {
      inCheck,
      legalMoves: boardChess.moves().length,
      inCheckmate: inCheck && boardChess.moves().length === 0,
    }
  }, [boardChess])

  const isMyTurn = useMemo(() => {
    if (!game || !role) return false
    const mine = role.toLowerCase() === 'white' ? 'white' : 'black'
    return game.turn === mine
  }, [game, role])

  const myColor = useMemo(() => {
    if (!role) return ''
    return role.toLowerCase() === 'white' ? 'white' : 'black'
  }, [role])

  const checkedKingSquare = useMemo(() => {
    if (!game?.board || !analysis.inCheck || !game.turn) return ''
    return findKingSquare(boardChess, game.turn)
  }, [analysis.inCheck, boardChess, game?.board, game?.turn])

  const squareStyles = useMemo(() => {
    const styles = {}

    if (selectedSquare) {
      styles[selectedSquare] = {
        background: 'radial-gradient(circle, rgba(78, 148, 226, 0.58) 0%, rgba(35, 94, 153, 0.35) 68%, rgba(35, 94, 153, 0.18) 100%)',
      }
    }

    for (const square of legalTargets) {
      styles[square] = {
        background: 'radial-gradient(circle, rgba(44, 197, 127, 0.52) 0%, rgba(44, 197, 127, 0.26) 56%, rgba(44, 197, 127, 0.08) 100%)',
      }
    }

    if (checkedKingSquare) {
      styles[checkedKingSquare] = {
        background: 'radial-gradient(circle, rgba(245, 91, 79, 0.8) 0%, rgba(245, 91, 79, 0.44) 56%, rgba(245, 91, 79, 0.12) 100%)',
      }
    }

    return styles
  }, [checkedKingSquare, legalTargets, selectedSquare])

  const alertMessage = useMemo(() => {
    if (!game) return ''

    if (game.status === 'finished' && (game.reason === 'checkmate' || analysis.inCheckmate)) {
      const winnerName = game.winner ? `${game.winner[0].toUpperCase()}${game.winner.slice(1)}` : 'Winner'
      return `Checkmate. ${winnerName} wins.`
    }

    if (analysis.inCheck && game.status !== 'finished') {
      const checkedSide = game.turn === 'white' ? 'White' : 'Black'
      return `Check on ${checkedSide}.`
    }

    return ''
  }, [analysis.inCheck, analysis.inCheckmate, game])

  const alertType = useMemo(() => {
    if (!alertMessage) return ''
    return alertMessage.toLowerCase().includes('checkmate') ? 'mate' : 'check'
  }, [alertMessage])

  function clearMoveHints() {
    setSelectedSquare('')
    setLegalTargets([])
  }

  function updateMoveHints(square) {
    if (!game?.board || !square) {
      clearMoveHints()
      return
    }

    const chess = new Chess(game.board)
    const piece = chess.get(square)
    if (!piece) {
      clearMoveHints()
      return
    }

    const moves = chess.moves({ square, verbose: true })
    setSelectedSquare(square)
    setLegalTargets(moves.map((move) => move.to))
  }

  function onPieceClick({ square }) {
    updateMoveHints(square)
  }

  function onSquareClick({ square, piece }) {
    if (!piece) {
      clearMoveHints()
      return
    }
    updateMoveHints(square)
  }

  useEffect(() => {
    let cancelled = false

    const connect = () => {
      if (!playerId.trim()) {
        setSocketState('disconnected')
        return
      }

      setSocketState('connecting')
      const socket = new WebSocket(createWsUrl(playerId.trim(), game?.gameId || ''))
      wsRef.current = socket

      socket.onopen = () => {
        if (!cancelled) {
          setSocketState('connected')
          if (game?.gameId) {
            socket.send(JSON.stringify({ type: 'subscribe-game', gameId: game.gameId }))
          }
        }
      }

      socket.onmessage = (event) => {
        if (cancelled) return

        try {
          const payload = JSON.parse(event.data)

          if (payload.type === 'game-update' && payload.game) {
            setGame((current) => {
              if (!current) return payload.game
              if (current.gameId !== payload.game.gameId) return current
              return payload.game
            })
            return
          }

          if (payload.type === 'match-found' && payload.game) {
            const matchedGame = payload.game
            const assignedRole = matchedGame.player1 === playerId.trim() ? 'White' : 'Black'
            setGame(matchedGame)
            setRole(assignedRole)
            setJoinGameId(matchedGame.gameId)
            setInfo(`Match found. You are ${assignedRole}.`)
            setError('')
            return
          }

          if (payload.type === 'queue-waiting') {
            const waiting = payload.queueLength ? ` Queue: ${payload.queueLength}.` : ''
            setInfo(`Queued for matchmaking.${waiting}`)
            return
          }

          if (payload.type === 'error' && payload.error) {
            setError(payload.error)
          }
        } catch {
          // Ignore malformed websocket payloads from transient network glitches.
        }
      }

      socket.onclose = () => {
        if (cancelled) return
        setSocketState('reconnecting')
        reconnectTimerRef.current = setTimeout(connect, 1100)
      }

      socket.onerror = () => {
        if (!cancelled) {
          setSocketState('error')
        }
      }
    }

    connect()

    return () => {
      cancelled = true
      if (reconnectTimerRef.current) {
        clearTimeout(reconnectTimerRef.current)
        reconnectTimerRef.current = null
      }
      if (wsRef.current) {
        wsRef.current.close()
        wsRef.current = null
      }
    }
  }, [playerId, game?.gameId])

  useEffect(() => {
    if (!game?.gameId || socketState === 'connected') {
      return undefined
    }

    const pollState = async () => {
      try {
        const latest = await apiRequest(`/api/game/state?gameId=${encodeURIComponent(game.gameId)}`)
        setGame((current) => {
          if (!current || current.gameId !== latest.gameId) {
            return current
          }
          return latest
        })
      } catch {
        // Keep UI responsive even if fallback polling fails temporarily.
      }
    }

    void pollState()
    const intervalId = setInterval(pollState, 4000)
    return () => clearInterval(intervalId)
  }, [game?.gameId, socketState])

  useEffect(() => {
    const moveCount = game?.moves?.length || 0
    if (moveCount <= previousMovesRef.current) {
      previousMovesRef.current = moveCount
      return
    }

    previousMovesRef.current = moveCount

    if (game?.status === 'finished' && (game?.reason === 'checkmate' || analysis.inCheckmate)) {
      const winnerName = game?.winner ? `${game.winner[0].toUpperCase()}${game.winner.slice(1)}` : 'Winner'
      setInfo(`Checkmate. ${winnerName} wins.`)
      clearMoveHints()
      return
    }

    if (analysis.inCheck && game?.status !== 'finished') {
      const checkedSide = game?.turn === 'white' ? 'White' : 'Black'
      setInfo(`Check on ${checkedSide}.`)
    }
  }, [analysis.inCheck, analysis.inCheckmate, game?.moves, game?.reason, game?.status, game?.turn, game?.winner])

  async function createGame() {
    setBusy(true); setError('')
    try {
      const created = await apiRequest('/api/game/create', { method: 'POST' })
      clearMoveHints()
      setGame(created); setRole(''); setJoinGameId(created.gameId)
      setInfo('Game created. Join as a player to begin moves.')
    } catch (err) { setError(err.message) }
    finally { setBusy(false) }
  }

  async function joinGame() {
    if (!playerId.trim()) { setError('Enter playerId before joining.'); return }
    if (!joinGameId.trim()) { setError('Enter a gameId first.'); return }
    setBusy(true); setError('')
    try {
      const result = await apiRequest('/api/game/join', {
        method: 'POST',
        body: JSON.stringify({ gameId: joinGameId.trim(), playerId: playerId.trim() }),
      })
      clearMoveHints()
      setGame(result.game); setRole(result.role || '')
      setInfo(`Joined game as ${result.role || 'player'}.`)
    } catch (err) { setError(err.message) }
    finally { setBusy(false) }
  }

  async function joinMatchmaking() {
    if (!playerId.trim()) { setError('Enter playerId before matchmaking.'); return }
    setBusy(true); setError('')
    try {
      if (wsRef.current?.readyState === WebSocket.OPEN) {
        wsRef.current.send(JSON.stringify({ type: 'queue', playerId: playerId.trim() }))
        setInfo('Queued for matchmaking. Waiting for opponent...')
        return
      }

      const result = await apiRequest('/api/matchmaking/join', {
        method: 'POST',
        body: JSON.stringify({ playerId: playerId.trim() }),
      })
      if (result.matched) {
        const matchedGame = result.game
        const assignedRole = matchedGame.player1 === playerId.trim() ? 'White' : 'Black'
        clearMoveHints()
        setGame(matchedGame); setRole(assignedRole)
        setJoinGameId(matchedGame.gameId)
        setInfo(`Match found. You are ${assignedRole}.`)
      } else {
        const waiting = result.queueLength ? ` Queue: ${result.queueLength}.` : ''
        setInfo(`${result.message || 'Queued for matchmaking.'}${waiting}`)
      }
    } catch (err) { setError(err.message) }
    finally { setBusy(false) }
  }

  async function resignGame() {
    if (!playerId.trim()) { setError('Enter playerId before resigning.'); return }
    if (!game?.gameId) { setError('No active game to resign.'); return }
    setBusy(true); setError('')
    try {
      const result = await apiRequest('/api/game/resign', {
        method: 'POST',
        body: JSON.stringify({ gameId: game.gameId, playerId: playerId.trim() }),
      })
      clearMoveHints()
      setGame(result); setInfo('Game resigned successfully.')
    } catch (err) { setError(err.message) }
    finally { setBusy(false) }
  }

  function onPieceDrop({ sourceSquare, targetSquare, piece }) {
    // Snapshot BEFORE any setGame() calls can re-compute isMyTurn
    const myTurnNow = isMyTurn

    if (!sourceSquare || !targetSquare || !piece) return false
    if (!game) { setError('Create or join a game first.'); return false }
    if (!role) { setError('Join the game as white or black before making a move.'); return false }
    if (!playerId.trim()) { setError('Enter playerId before making a move.'); return false }
    if (!myTurnNow) {
      const sideToMove = game.turn === 'white' ? 'White' : 'Black'
      setError(`Not your turn. ${sideToMove} must move from that player's session.`)
      return false
    }

    const local = new Chess(game.board)
    const pieceType = piece?.pieceType || ''
    const isPromotionMove =
      (pieceType === 'wP' && targetSquare.endsWith('8')) ||
      (pieceType === 'bP' && targetSquare.endsWith('1'))

    const movePreview = local.move({
      from: sourceSquare,
      to: targetSquare,
      promotion: isPromotionMove ? 'q' : undefined,
    })

    if (!movePreview) { setError('Illegal move.'); return false }

    const uciMove = `${sourceSquare}${targetSquare}${isPromotionMove ? 'q' : ''}`
    const previousGame = game
    const nextTurn = game.turn === 'white' ? 'black' : 'white'
    clearMoveHints()

    // Optimistic update so the board feels instant
    setGame((current) => current ? {
      ...current,
      board: local.fen(),
      turn: nextTurn,
      moves: [...(current.moves || []), uciMove],
    } : current)

    setBusy(true); setError('')

    void (async () => {
      try {
        const updated = await apiRequest('/api/game/move', {
          method: 'POST',
          body: JSON.stringify({
            gameId: game.gameId,
            playerId: playerId.trim(),
            move: uciMove,
          }),
        })
        setGame(updated)
        setInfo(`Move sent: ${uciMove}`)
      } catch (err) {
        setGame(previousGame)  // rollback on server rejection
        setError(err.message)
      } finally {
        setBusy(false)
      }
    })()

    return true
  }

  return (
    <main className="page">
      <header className="topbar">
        <h1>Chess Arena</h1>
        <p>React + Chess.js client connected to your Go backend.</p>
      </header>

      <section className="layout">
        <aside className="panel controls">
          <h2>Session</h2>
          <label htmlFor="playerId">Player ID</label>
          <input
            id="playerId"
            value={playerId}
            onChange={(e) => setPlayerId(e.target.value)}
            placeholder="your player id"
          />
          <label htmlFor="gameId">Game ID</label>
          <input
            id="gameId"
            value={joinGameId}
            onChange={(e) => setJoinGameId(e.target.value)}
            placeholder="paste gameId to join"
          />
          <div className="buttonGrid">
            <button onClick={createGame} disabled={busy}>Create Game</button>
            <button onClick={joinGame} disabled={busy}>Join Game</button>
            <button onClick={joinMatchmaking} disabled={busy}>Matchmaking</button>
            <button onClick={resignGame} disabled={busy || !game}>Resign</button>
          </div>
          <div className="statusBox">
            <div><strong>API:</strong> {API_BASE || '(same origin)'}</div>
            <div><strong>Socket:</strong> {socketState}</div>
            <div><strong>Role:</strong> {role || 'Not joined'}</div>
            <div><strong>Turn:</strong> {game?.turn || '-'}</div>
            <div><strong>Status:</strong> {game?.status || '-'}</div>
            <div><strong>Moves:</strong> {game?.moves?.length || 0}</div>
            <div><strong>In Check:</strong> {analysis.inCheck ? 'Yes' : 'No'}</div>
            <div><strong>Legal Moves:</strong> {analysis.legalMoves}</div>
          </div>
          <p className="info">{info}</p>
          {error ? <p className="error">{error}</p> : null}
        </aside>

        <section className="panel boardPanel">
          <div className="boardHeader">
            <h2>Board</h2>
            {game?.gameId ? <code>{game.gameId}</code> : <code>no game selected</code>}
          </div>

          <div className="boardWrap" onTouchStart={(e) => e.stopPropagation()}>
            {/*
              react-chessboard v5 changed to a single `options` object prop.
              Flat props like position={} onPieceDrop={} are silently ignored in v5.
            */}
            <Chessboard
              id="arena-board"
              options={{
                position: game?.board || START_FEN,
                onPieceDrop: onPieceDrop,
                onPieceClick: onPieceClick,
                onSquareClick: onSquareClick,
                allowDragging: true,
                allowDrawingArrows: true,
                boardOrientation: myColor || 'white',
                squareStyles: squareStyles,
                canDragPiece: ({ piece }) => {
                  if (!isMyTurn || !myColor) return false
                  const pieceCode = piece?.pieceType || ''
                  if (!pieceCode) return false
                  const pieceColor = pieceCode.startsWith('w') ? 'white' : 'black'
                  return pieceColor === myColor
                },
                darkSquareStyle: { backgroundColor: '#1f3c58' },
                lightSquareStyle: { backgroundColor: '#f1dfc7' },
              }}
            />
          </div>

          <div className="meta">
            <span>My turn: {isMyTurn ? 'Yes ✓' : 'No'}</span>
            <span>Path hints: {legalTargets.length > 0 ? `${selectedSquare} -> ${legalTargets.length}` : 'Off'}</span>
            <span>Winner: {game?.winner || '-'}</span>
            <span>Reason: {game?.reason || '-'}</span>
          </div>

          {alertMessage ? <div className={`alertStrip ${alertType}`}>{alertMessage}</div> : null}
        </section>
      </section>
    </main>
  )
}

export default App