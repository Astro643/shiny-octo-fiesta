import Head from 'next/head';

export default function HowItWorks() {
  return (
    <>
      <Head>
        <title>How It Works - Controller</title>
        <meta name="description" content="Learn how Controller works. AI-powered Discord management." />
      </Head>

      <header className="nav">
        <div className="wrap nav-in">
          <a href="/" className="brand">
            <svg className="logo" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
              <path d="M2.5 5h19L12 20.5z" />
            </svg>
            Controller
          </a>
          <nav className="nav-right">
            <nav>
              <a href="/how-it-works" className="on">How it works</a>
              <a href="/faq">FAQ</a>
              <a href="/support">Support</a>
            </nav>
            <a href="/dashboard" className="btn btn-primary">
              Open Console
            </a>
          </nav>
        </div>
      </header>

      <main>
        <section className="page">
          <div className="wrap">
            <span className="eyebrow">How it works</span>
            <h1>Simple. Powerful. Safe.</h1>
            <p className="lead">Controller makes Discord server management effortless.</p>

            <div className="prose">
              <h2>Describe What You Want</h2>
              <p>
                Instead of learning complex bot commands, just describe your goal in plain English. 
                Want to create a welcome category with channels? Just say &quot;Create a welcome category with rules and info channels.&quot;
              </p>

              <h2>The AI Plans It</h2>
              <p>
                Our AI audits your request for safety, then creates a step-by-step plan to achieve your goal. 
                It understands Discord&apos;s structure and can perform multiple actions in sequence.
              </p>

              <h2>The Bot Executes</h2>
              <p>
                Your Discord bot carries out each step of the plan. You can watch the progress in real-time 
                in your Controller console. Each action is logged so you know exactly what was done.
              </p>

              <h2>What Can It Do?</h2>
              <ul>
                <li>Create, rename, and delete channels and categories</li>
                <li>Manage roles (create, delete, assign, remove)</li>
                <li>Moderate members (kick, ban, unban, timeout)</li>
                <li>Send messages and embeds</li>
                <li>Set channel topics and slowmode</li>
                <li>Purge messages</li>
                <li>Lock and unlock channel permissions</li>
                <li>And much more through Discord&apos;s API</li>
              </ul>

              <h2>Safety First</h2>
              <p>
                Every request goes through a rigorous safety check. The AI will refuse requests that:
              </p>
              <ul>
                <li>Target you or the bot itself</li>
                <li>Try to wipe your entire server</li>
                <li>Leak secrets or private data</li>
                <li>Violate Discord&apos;s Terms of Service</li>
                <li>Are vague or unclear</li>
              </ul>

              <h2>Follow-Up Requests</h2>
              <p>
                Controller remembers your recent requests, so you can say things like &quot;Do the same for voice channels&quot; 
                or &quot;Undo that.&quot; The AI understands the context of your conversation.
              </p>

              <p className="updated">Last updated: June 2024</p>
            </div>
          </div>
        </section>

        <section className="agentic landing">
          <div className="a-left">
            <h2>Ready to Try It?</h2>
          </div>
          <div className="a-mid">
            <div className="tile">
              <svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
                <path d="M2.5 5h19L12 20.5z" />
              </svg>
            </div>
          </div>
          <div className="a-right">
            <a href="/dashboard" className="a-chip">
              Open Console
              <span className="a-tip">Start managing your server with AI</span>
            </a>
          </div>
        </section>
      </main>

      <footer>
        <div className="wrap foot">
          <div>
            <a href="/" className="brand">
              <svg className="logo" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
                <path d="M2.5 5h19L12 20.5z" />
              </svg>
              Controller
            </a>
          </div>
          <div>
            <h4>Product</h4>
            <a href="/how-it-works">How it works</a>
            <a href="/faq">FAQ</a>
            <a href="/privacy">Privacy</a>
            <a href="/terms">Terms</a>
          </div>
          <div>
            <h4>Support</h4>
            <a href="/support">Help Center</a>
            <a href="/dashboard">Console</a>
          </div>
          <div>
            <h4>Legal</h4>
            <a href="/privacy">Privacy Policy</a>
            <a href="/terms">Terms of Service</a>
          </div>
        </div>
        <div className="wrap legal">
          <span>© {new Date().getFullYear()} Controller</span>
          <span>All rights reserved</span>
        </div>
      </footer>

      <a href="#" className="scroll" id="scroll-cue">
        <svg viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
          <path d="M12 5v14M5 12l7-7 7 7" />
        </svg>
        SCROLL
      </a>
    </>
  );
}
