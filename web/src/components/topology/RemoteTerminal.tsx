import { useState, useEffect, useRef, useCallback } from 'react'
import { getRemoteSessionWebSocketUrl, RemoteSessionAgent } from '../../api/client'
import { XMarkIcon, CommandLineIcon } from '@heroicons/react/24/outline'

interface RemoteTerminalProps {
  agent: RemoteSessionAgent
  onClose: () => void
  showHeader?: boolean
}

interface TerminalLine {
  type: 'input' | 'output' | 'error' | 'system'
  content: string
  timestamp: Date
}

export default function RemoteTerminal({ agent, onClose, showHeader = false }: RemoteTerminalProps) {
  const [connected, setConnected] = useState(false)
  const [connecting, setConnecting] = useState(true)
  const [lines, setLines] = useState<TerminalLine[]>([])
  const [command, setCommand] = useState('')
  const [history, setHistory] = useState<string[]>([])
  const [historyIndex, setHistoryIndex] = useState(-1)
  const wsRef = useRef<WebSocket | null>(null)
  const terminalRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)

  const addLine = useCallback((type: TerminalLine['type'], content: string) => {
    setLines(prev => [...prev, { type, content, timestamp: new Date() }])
  }, [])

  useEffect(() => {
    const ws = new WebSocket(getRemoteSessionWebSocketUrl())
    wsRef.current = ws

    let agentConnected = false

    ws.onopen = () => {
      setConnecting(false)
      addLine('system', `Connected to control plane, selecting agent ${agent.nodeName}...`)
      // Wait for initial agent_list message before sending connect_agent
    }

    ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data)

        switch (msg.type) {
          case 'agent_list':
            // Server sends agent list on connect, now we can select our agent
            if (!agentConnected) {
              ws.send(JSON.stringify({
                type: 'connect_agent',
                payload: { agentId: agent.id },
                timestamp: new Date().toISOString()
              }))
            }
            break
          case 'agent_connected':
            agentConnected = true
            setConnected(true)
            addLine('system', `Connected to ${agent.nodeType}: ${agent.nodeName}`)
            break
          case 'output':
            if (msg.payload) {
              const payload = typeof msg.payload === 'string' ? JSON.parse(msg.payload) : msg.payload
              if (payload.output) {
                // Remove trailing newline for cleaner display
                const output = payload.output.replace(/\n$/, '')
                addLine(payload.is_stderr ? 'error' : 'output', output)
              }
              if (payload.done && payload.exit_code !== undefined) {
                addLine('system', `Command completed with exit code ${payload.exit_code}`)
              }
            }
            break
          case 'error':
            if (msg.payload) {
              const payload = typeof msg.payload === 'string' ? JSON.parse(msg.payload) : msg.payload
              addLine('error', payload.message || 'Unknown error')
            }
            break
        }
      } catch (e) {
        console.error('Failed to parse message:', e)
      }
    }

    ws.onclose = () => {
      setConnected(false)
      addLine('system', 'Connection closed')
    }

    ws.onerror = () => {
      setConnecting(false)
      addLine('error', 'WebSocket connection error')
    }

    return () => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({
          type: 'disconnect',
          timestamp: new Date().toISOString()
        }))
      }
      ws.close()
    }
  }, [agent, addLine])

  // Auto-scroll to bottom
  useEffect(() => {
    if (terminalRef.current) {
      terminalRef.current.scrollTop = terminalRef.current.scrollHeight
    }
  }, [lines])

  // Focus input on mount
  useEffect(() => {
    inputRef.current?.focus()
  }, [])

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!command.trim() || !wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) return

    addLine('input', `$ ${command}`)

    wsRef.current.send(JSON.stringify({
      type: 'command',
      payload: { command: command.trim() },
      timestamp: new Date().toISOString()
    }))

    // Add to history
    setHistory(prev => [...prev, command])
    setHistoryIndex(-1)
    setCommand('')
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'ArrowUp') {
      e.preventDefault()
      if (history.length > 0) {
        const newIndex = historyIndex < history.length - 1 ? historyIndex + 1 : historyIndex
        setHistoryIndex(newIndex)
        setCommand(history[history.length - 1 - newIndex])
      }
    } else if (e.key === 'ArrowDown') {
      e.preventDefault()
      if (historyIndex > 0) {
        const newIndex = historyIndex - 1
        setHistoryIndex(newIndex)
        setCommand(history[history.length - 1 - newIndex])
      } else if (historyIndex === 0) {
        setHistoryIndex(-1)
        setCommand('')
      }
    }
  }

  return (
    <div className="flex flex-col h-full bg-gray-900 rounded-lg overflow-hidden">
      {/* Terminal Header - optional, hidden when using tabs */}
      {showHeader && (
        <div className="flex items-center justify-between px-4 py-2 bg-gray-800 border-b border-gray-700">
          <div className="flex items-center gap-2">
            <CommandLineIcon className="w-5 h-5 text-green-400" />
            <span className="text-gray-200 font-medium">
              {agent.nodeType}: {agent.nodeName}
            </span>
            <span className={`w-2 h-2 rounded-full ${connected ? 'bg-green-500' : 'bg-red-500'}`} />
          </div>
          <button
            onClick={onClose}
            className="p-1 hover:bg-gray-700 rounded transition-colors"
          >
            <XMarkIcon className="w-5 h-5 text-gray-400" />
          </button>
        </div>
      )}

      {/* Terminal Content */}
      <div
        ref={terminalRef}
        className="flex-1 p-4 overflow-y-auto font-mono text-sm"
        onClick={() => inputRef.current?.focus()}
      >
        {connecting && (
          <div className="text-yellow-400">Connecting...</div>
        )}

        {lines.map((line, idx) => (
          <div key={idx} className={`whitespace-pre-wrap break-all ${
            line.type === 'input' ? 'text-cyan-400' :
            line.type === 'error' ? 'text-red-400' :
            line.type === 'system' ? 'text-yellow-400' :
            'text-gray-100'
          }`}>
            {line.content}
          </div>
        ))}
      </div>

      {/* Command Input */}
      <form onSubmit={handleSubmit} className="flex items-center px-4 py-2 bg-gray-800 border-t border-gray-700">
        <span className="text-green-400 mr-2">$</span>
        <input
          ref={inputRef}
          type="text"
          value={command}
          onChange={(e) => setCommand(e.target.value)}
          onKeyDown={handleKeyDown}
          disabled={!connected}
          placeholder={connected ? 'Enter command...' : 'Disconnected'}
          className="flex-1 bg-transparent text-gray-100 outline-none placeholder-gray-500 font-mono"
          autoComplete="off"
          autoCorrect="off"
          autoCapitalize="off"
          spellCheck="false"
        />
      </form>
    </div>
  )
}
