import Head from 'next/head';

export default function FAQ() {
  return (
    <>
      <Head>
        <title>FAQ - Controller</title>
        <meta name="description" content="Frequently asked questions about Controller." />
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
              <a href="/how-it-works">How it works</a>
              <a href="/faq" className="on">FAQ</a>
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
            <span className="eyebrow">FAQ</span>
            <h1>Frequently Asked Questions</h1>
            <p className="lead">Get answers to common questions about Controller.</p>

            <div className="faq">
              <details>
                <summary>How do I get started?</summary>
                <div className="ans">
                  <p>Invite the Controller bot to your Discord server, then run <code>/dashboard</code> in any channel. Click the magic link to open your console.</p>
                </div>
              </details>

              <details>
                <summary>Is Controller safe?</summary>
                <div className="ans">
                  <p>Yes. Every request goes through a rigorous AI safety check. The bot will refuse any request that could harm your server or violate Discord&apos;s rules.</p>
                </div>
              </details>

              <details>
                <summary>What can Controller do?</summary>
                <div className="ans">
                  <p>Controller can create, rename, and delete channels and categories; manage roles; moderate members; send messages; set channel topics and slowmode; purge messages; and much more through Discord&apos;s API.</p>
                </div>
              </details>

              <details>
                <summary>Can Controller delete my server?</summary>
                <div className="ans">
                  <p>No. The safety check explicitly prevents server deletion and other destructive actions that could harm your entire server.</p>
                </div>
              </details>

              <details>
                <summary>How many requests can I make?</summary>
                <div className="ans">
                  <p>There&apos;s no hard limit, but each agent can only handle one request at a time. If all agents are busy, you&apos;ll join a queue and be notified when an agent is available.</p>
                </div>
              </details>

              <details>
                <summary>Can I see what the bot will do before it does it?</summary>
                <div className="ans">
                  <p>Yes! Controller now has a dry run preview feature. Before executing your request, it will show you all the planned steps. You can review them and decide whether to proceed.</p>
                </div>
              </details>

              <details>
                <summary>What if the bot makes a mistake?</summary>
                <div className="ans">
                  <p>The AI is trained to be careful, but mistakes can happen. You can always undo most actions manually. For irreversible actions (like bans), the AI is extra cautious.</p>
                </div>
              </details>

              <details>
                <summary>Can Controller read my messages?</summary>
                <div className="ans">
                  <p>No. Controller only reads the commands you give it in the console. It does not read or store any Discord chat messages.</p>
                </div>
              </details>

              <details>
                <summary>Is my data private?</summary>
                <div className="ans">
                  <p>Yes. Your requests and server data are sent to Google&apos;s Gemini API for processing, but we don&apos;t store them longer than necessary. See our <a href="/privacy">Privacy Policy</a> for details.</p>
                </div>
              </details>

              <details>
                <summary>How do I cancel a request?</summary>
                <div className="ans">
                  <p>If a request is in the &quot;Waiting&quot; state (paused for a scheduled action), you can click the Cancel button. Once a request starts executing, individual steps cannot be cancelled, but you can close the console to free up your agent.</p>
                </div>
              </details>
            </div>
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
    </>
  );
}
