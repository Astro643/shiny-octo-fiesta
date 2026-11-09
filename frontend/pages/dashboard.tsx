import { useState, useEffect, useRef } from 'react';
import Head from 'next/head';
import { useRouter } from 'next/router';

const LABELS: Record<string, string> = {
  processing: 'Processing',
  waiting: 'Waiting',
  cancelled: 'Cancelled',
  completed: 'Completed',
  blocked_by_traffic: 'Traffic stall',
  rejected: 'Rejected',
  failed: 'Failed',
  stalled: 'No response',
  unknown: 'Expired',
};

const FINAL_STATES = new Set(['cancelled', 'completed', 'blocked_by_traffic', 'failed', 'rejected', 'unknown', 'stalled']);

interface FeedItem {
  id: string;
  prompt: string;
  ts: number;
  status: string;
  note: string;
}

interface PreviewStep {
  turn: number;
  action: string;
  description: string;
  label: string;
  parameters?: any;
  delay_seconds?: number;
}

interface PreviewPlan {
  complete: boolean;
  total_turns: number;
  steps: PreviewStep[];
  summary?: string;
  incomplete: boolean;
  question?: string;
}

interface SessionState {
  state: string;
  agent?: number;
  position?: number;
  ahead?: number;
  waiting?: number;
}

export default function Dashboard() {
  const router = useRouter();
  const [items, setItems] = useState<FeedItem[]>([]);
  const [inputValue, setInputValue] = useState('');
  const [hint, setHint] = useState('');
  const [isHintError, setIsHintError] = useState(false);
  const [active, setActive] = useState(false);
  const [sessionAgent, setSessionAgent] = useState<number | null>(null);
  const [gateState, setGateState] = useState<{
    title: string;
    num: string;
    text: string;
    showBar: boolean;
  }>({ title: 'Checking...', num: '', text: 'Finding you an agent.', showBar: true });
  const [showPreview, setShowPreview] = useState(false);
  const [previewData, setPreviewData] = useState<PreviewPlan | null>(null);
  const [currentPrompt, setCurrentPrompt] = useState('');
  const previewTimeoutRef = useRef<NodeJS.Timeout | null>(null);

  // Load history from localStorage
  useEffect(() => {
    try {
      const saved = localStorage.getItem('console_history_v1');
      if (saved) {
        setItems(JSON.parse(saved).slice(-50));
      }
    } catch {}
  }, []);

  // Save to localStorage
  useEffect(() => {
    try {
      localStorage.setItem('console_history_v1', JSON.stringify(items.slice(-50)));
    } catch {}
  }, [items]);

  // Ping for session status
  const ping = async () => {
    try {
      const response = await fetch('/api/session/ping', {
        method: 'POST',
        cache: 'no-store',
        credentials: 'include',
      });

      if (response.status === 401) {
        setGateState({ title: 'Session expired', num: '', text: 'Open a fresh link from your bot to continue.', showBar: false });
        setActive(false);
        return;
      }

      if (!response.ok) throw new Error('ping failed');

      const data: SessionState = await response.json();

      if (data.state === 'active') {
        setActive(true);
        setSessionAgent(data.agent || null);
      } else if (data.state === 'waiting') {
        setActive(false);
        setGateState({
          title: 'You are in line',
          num: `#${data.position}`,
          text: data.ahead === 0
            ? 'You are next. An agent frees up when someone closes their console.'
            : `${data.ahead} ${data.ahead === 1 ? 'person is' : 'people are'} ahead of you. Keep this tab open to hold your place.`,
          showBar: true,
        });
      } else if (data.state === 'idle_released') {
        setActive(false);
        setGateState({ title: 'Agent released', num: '', text: 'You were idle while others were waiting. Rejoining the line...', showBar: true });
      } else if (data.state === 'full') {
        setActive(false);
        setGateState({ title: 'Line is full', num: '', text: 'Too many people are waiting right now. Please try again in a few minutes.', showBar: false });
      } else {
        setActive(false);
        setGateState({ title: 'Unavailable', num: '', text: 'No agent is set up right now. Please try again later.', showBar: false });
      }
    } catch {
      // Retry after delay
    } finally {
      setTimeout(ping, active ? 30000 : 6000);
    }
  };

  // Initial ping
  useEffect(() => {
    ping();
    return () => {
      if (previewTimeoutRef.current) {
        clearTimeout(previewTimeoutRef.current);
      }
    };
  }, []);

  // Poll for status updates
  useEffect(() => {
    const interval = setInterval(() => {
      const pending = items.filter(i => !FINAL_STATES.has(i.status));
      pending.forEach(async (item) => {
        if (Date.now() - item.ts > 26 * 60 * 60 * 1000) {
          item.status = 'stalled';
          setItems([...items]);
          return;
        }
        try {
          const response = await fetch(`/api/status/${encodeURIComponent(item.id)}`, {
            cache: 'no-store',
            credentials: 'include',
          });
          if (!response.ok) return;
          const data = await response.json();
          const note = data.error || data.result || '';
          if (data.status !== item.status || note !== item.note) {
            item.status = data.status;
            item.note = note;
            setItems([...items]);
            if (FINAL_STATES.has(data.status)) {
              ping();
            }
          }
        } catch {}
      });
    }, 1500);

    return () => clearInterval(interval);
  }, [items]);

  // Leave session on page hide
  useEffect(() => {
    const handlePageHide = () => {
      try {
        navigator.sendBeacon('/api/session/leave');
      } catch {}
    };
    window.addEventListener('pagehide', handlePageHide);
    return () => window.removeEventListener('pagehide', handlePageHide);
  }, []);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const prompt = inputValue.trim();
    if (!prompt) return;

    // First, generate preview
    try {
      const previewResponse = await fetch('/api/preview', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ prompt }),
        credentials: 'include',
      });

      if (!previewResponse.ok) {
        const data = await previewResponse.json().catch(() => ({}));
        const msg = typeof data.detail === 'string' ? data.detail : 'Could not generate preview.';
        setHint(msg);
        setIsHintError(true);
        return;
      }

      const previewData: PreviewPlan = await previewResponse.json();
      setPreviewData(previewData);
      setCurrentPrompt(prompt);
      setShowPreview(true);
    } catch {
      setHint('Network error. Please try again.');
      setIsHintError(true);
    }
  };

  const executeWithPrompt = async (prompt: string) => {
    try {
      setShowPreview(false);
      setInputValue('');
      
      const response = await fetch('/api/execute', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ prompt }),
        credentials: 'include',
      });

      if (!response.ok) {
        const data = await response.json().catch(() => ({}));
        if (response.status === 409) {
          ping();
        }
        const msg = typeof data.detail === 'string' 
          ? (response.status === 401 ? 'Session expired. Open a fresh link from your bot.' : data.detail)
          : 'Could not queue that goal.';
        setHint(msg);
        setIsHintError(true);
        return;
      }

      const data = await response.json();
      const newItem: FeedItem = {
        id: data.command_id,
        prompt,
        ts: Date.now(),
        status: data.status || 'processing',
        note: '',
      };
      setItems([...items, newItem]);
      setTimeout(ping, 4000);
    } catch {
      setHint('Network error. Please try again.');
      setIsHintError(true);
    }
  };

  const handleCancel = async (id: string) => {
    try {
      const response = await fetch(`/api/cancel/${encodeURIComponent(id)}`, {
        method: 'POST',
        credentials: 'include',
      });

      if (!response.ok) {
        const data = await response.json().catch(() => ({}));
        const msg = typeof data.detail === 'string' ? data.detail : 'Could not cancel.';
        setHint(msg);
        setIsHintError(true);
        return;
      }

      const data = await response.json();
      setItems(items.map(item => 
        item.id === id ? { ...item, status: 'cancelled', note: '' } : item
      ));
    } catch {
      setHint('Network error. Please try again.');
      setIsHintError(true);
    }
  };

  const renderItem = (item: FeedItem) => (
    <article key={item.id} className="op" data-id={item.id}>
      <div className="bubble">{item.prompt}</div>
      <div className="status-row">
        <span className={`badge s-${item.status}`}>
          <i></i>
          {LABELS[item.status] || item.status}
        </span>
        <span className="op-id">{item.id}</span>
        <span className="op-time">
          {new Date(item.ts).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
        </span>
        {item.status === 'waiting' && (
          <button
            className="op-cancel"
            type="button"
            title="Cancel this waiting request. Steps that already ran are not undone."
            onClick={() => handleCancel(item.id)}
          >
            Cancel
          </button>
        )}
      </div>
      {item.note && <div className="op-note">{item.note}</div>}
    </article>
  );

  return (
    <>
      <Head>
        <title>Controller - Console</title>
      </Head>

      <a className="mark" href="/" aria-label="Home">
        <svg className="logo" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
          <path d="M2.5 5h19L12 20.5z" />
        </svg>
      </a>

      <div className="top-right">
        <div className="top-row">
          {sessionAgent && active && (
            <span className="agent-chip show">Agent {sessionAgent}</span>
          )}
        </div>
      </div>

      <main>
        <section id="feed" aria-live="polite">
          {items.map(renderItem)}
        </section>
      </main>

      {/* Gate Modal */}
      {(!active || !sessionAgent) && (
        <div id="gate" className="show" role="status" aria-live="polite">
          <div className="gate-card">
            <h1 id="gate-title">{gateState.title}</h1>
            {gateState.num && <div className="gate-num">{gateState.num}</div>}
            <p id="gate-text">{gateState.text}</p>
            {gateState.showBar && (
              <div className="gate-bar" id="gate-bar">
                <i></i>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Input Bar */}
      <div id="bar">
        <form id="form" onSubmit={handleSubmit} className={inputValue.trim().length > 0 ? 'ready' : ''}>
          <input
            id="prompt"
            type="text"
            maxLength={2000}
            placeholder="Describe a change to your server"
            aria-label="Goal"
            value={inputValue}
            onChange={(e) => setInputValue(e.target.value)}
            disabled={!active}
          />
          <button
            id="go"
            type="submit"
            aria-label="Execute goal"
            title="Execute goal"
            disabled={!active}
          >
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.4" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
              <path d="M12 19V5" />
              <path d="M5 12l7-7 7 7" />
            </svg>
          </button>
        </form>
        <div id="hint" className={isHintError ? 'err' : ''}>{hint}</div>
      </div>

      {/* Preview Modal */}
      {showPreview && previewData && (
        <div className="preview-modal" id="preview-modal">
          <div className="preview-card">
            <h2>Preview (Dry Run)</h2>
            
            {(previewData.incomplete || !previewData.complete) && (
              <p className="preview-question">
                {previewData.question || 'Can you clarify your request?'}
              </p>
            )}
            
            {previewData.summary && (
              <p className="preview-summary">{previewData.summary}</p>
            )}
            
            <h3>Planned Steps:</h3>
            <div className="preview-steps">
              {previewData.steps && previewData.steps.length > 0 ? (
                previewData.steps.map((step, idx) => (
                  <div key={idx} className="preview-step">
                    <span className="preview-step-num">{idx + 1}</span>
                    <span className="preview-step-label">{step.label || step.action}</span>
                  </div>
                ))
              ) : (
                <p>No steps generated.</p>
              )}
            </div>
            
            <div className="preview-actions">
              <button
                className="preview-btn preview-cancel"
                type="button"
                onClick={() => {
                  if (previewTimeoutRef.current) {
                    clearTimeout(previewTimeoutRef.current);
                  }
                  setShowPreview(false);
                }}
              >
                Cancel
              </button>
              <button
                className="preview-btn preview-confirm"
                type="button"
                onClick={() => {
                  if (previewTimeoutRef.current) {
                    clearTimeout(previewTimeoutRef.current);
                  }
                  setShowPreview(false);
                  executeWithPrompt(currentPrompt);
                }}
              >
                {previewData.incomplete ? 'Answer Question' : 'Execute'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Preview Modal Styles */}
      <style jsx>{`
        .preview-modal {
          position: fixed;
          top: 0;
          left: 0;
          right: 0;
          bottom: 0;
          background: rgba(0, 0, 0, 0.5);
          z-index: 100;
          display: flex;
          justify-content: center;
          align-items: center;
          padding: 20px;
        }
        .preview-card {
          background: var(--surface);
          border: 1px solid var(--border);
          border-radius: 14px;
          padding: 24px;
          max-width: 500px;
          width: 100%;
          max-height: 80vh;
          overflow: auto;
        }
        .preview-card h2 {
          font: var(--display);
          font-size: 24px;
          margin: 0 0 16px;
          text-transform: uppercase;
          letter-spacing: 0.005em;
        }
        .preview-question {
          color: var(--amber);
          margin: 16px 0;
          font-weight: 600;
        }
        .preview-summary {
          color: var(--muted);
          margin: 8px 0 16px;
          font-size: 14px;
        }
        .preview-card h3 {
          font-size: 16px;
          margin: 16px 0 8px;
          color: var(--ink);
        }
        .preview-steps {
          display: flex;
          flex-direction: column;
          gap: 8px;
        }
        .preview-step {
          background: var(--mono-bg);
          border-radius: 8px;
          padding: 10px 12px;
          font-size: 14px;
          display: flex;
          align-items: flex-start;
          gap: 10px;
        }
        .preview-step-num {
          background: var(--dark);
          color: var(--green);
          border-radius: 50%;
          width: 24px;
          height: 24px;
          display: flex;
          align-items: center;
          justify-content: center;
          font-size: 12px;
          font-weight: 700;
          flex-shrink: 0;
        }
        .preview-step-label {
          color: var(--ink);
          flex: 1;
        }
        .preview-actions {
          display: flex;
          gap: 12px;
          margin-top: 20px;
          justify-content: flex-end;
        }
        .preview-btn {
          padding: 10px 20px;
          border-radius: 8px;
          font: var(--mono);
          font-size: 13px;
          cursor: pointer;
          transition: background 0.15s;
        }
        .preview-cancel {
          border: 1px solid var(--border);
          background: transparent;
          color: var(--ink);
        }
        .preview-confirm {
          border: 0;
          background: var(--green);
          color: #000;
          font-weight: 600;
        }
        .gate-card {
          background: var(--surface);
          border: 1px solid var(--border);
          border-radius: 18px;
          padding: 34px 30px 28px;
          text-align: center;
          box-shadow: var(--lift);
          animation: rise 0.5s cubic-bezier(0.2, 0.7, 0.2, 1);
        }
        .gate-card h1 {
          font: 400 34px/1.05 var(--display);
          text-transform: uppercase;
          letter-spacing: 0.01em;
          margin: 0 0 10px;
        }
        .gate-num {
          font: 400 72px/1 var(--display);
          margin: 14px 0 4px;
          font-variant-numeric: tabular-nums;
        }
        .gate-card p {
          margin: 0;
          color: var(--muted);
          font-size: 14px;
        }
        .gate-bar {
          height: 3px;
          border-radius: 3px;
          background: var(--border-soft);
          overflow: hidden;
          margin: 22px 0 0;
        }
        .gate-bar i {
          display: block;
          height: 100%;
          width: 40%;
          background: var(--green);
          border-radius: 3px;
          animation: slide 1.8s ease-in-out infinite;
        }
        .agent-chip {
          font: 600 11px/1 var(--mono);
          text-transform: uppercase;
          letter-spacing: 0.07em;
          color: var(--green-ink);
          background: var(--green-soft);
          border-radius: 999px;
          padding: 10px 12px;
          display: none;
        }
        .agent-chip.show {
          display: block;
        }
        .top-right {
          position: fixed;
          top: 16px;
          right: 24px;
          z-index: 10;
          display: flex;
          flex-direction: column;
          align-items: flex-end;
          gap: 12px;
        }
        .top-row {
          display: flex;
          align-items: center;
          gap: 14px;
        }
        .top-right .agent-chip {
          position: static;
        }
        @keyframes slide {
          0% { transform: translateX(-120%); }
          100% { transform: translateX(320%); }
        }
        @keyframes rise {
          from { opacity: 0; transform: translateY(10px); }
          to { opacity: 1; transform: none; }
        }
        @media (prefers-reduced-motion: reduce) {
          * { animation: none !important; transition: none !important; }
        }
      `}</style>
    </>
  );
}
